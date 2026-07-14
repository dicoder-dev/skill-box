// Package services 提供给 Wails Webview 调用的桌面服务绑定。
//
// 命名空间约定:
//   app_svc.go / updater_helper.go / 其他服务
//
// 2026-07-14 增:
//   UpdaterSpawnHelper / UpdaterInstallDir / UpdaterManifestURLs 三个导出符号,
//   是 wails_app.go 在 SetDesktopHooks 时注入到 BootstrapHooks.Updater* 字段用的。
//   替身脚本(helper_darwin.sh / helper_windows.ps1 / helper_linux.sh)随 binary
//   //go:embed,实现"下载完后 fork + 父进程 Quit + child 接管 + 重启"的硬规则。
package services

import (
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

//go:embed updater_scripts/helper_darwin.sh updater_scripts/helper_windows.ps1 updater_scripts/helper_linux.sh
var scriptsFS embed.FS

// PickScript 根据当前平台返回替身脚本内容(纯文本)。
// runtime.GOOS 不在预期范围(目前 darwin / windows / linux)时返 "",
// 配合 cdesktop.PostUpdateDownload 的 501 行为。
func PickScript() string {
	switch runtime.GOOS {
	case "darwin":
		b, err := scriptsFS.ReadFile("updater_scripts/helper_darwin.sh")
		if err != nil {
			return ""
		}
		return string(b)
	case "windows":
		b, err := scriptsFS.ReadFile("updater_scripts/helper_windows.ps1")
		if err != nil {
			return ""
		}
		return string(b)
	default:
		// linux 走 sh
		b, err := scriptsFS.ReadFile("updater_scripts/helper_linux.sh")
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// SpawnHelper fork 当前 OS 对应的替身脚本,**不 Wait**;父进程立刻退出 Wails 主循环。
//
// 关键时序(必须遵守):详见 ssupdater.SpawnOrder。
//
// 步骤:
//   1) 把脚本内容写到 os.TempDir()/<random>.sh(或 .ps1),chmod 0755;
//   2) exec.Command 跑 bash <script> <args...>;Start() 不阻塞;
//   3) Start() 返 err == nil 时父进程继续跑;父进程跑完自己就 AppQuit()。
//
// 返回值:启动失败(error)。cdesktop.PostUpdateDownload 检测到非 nil 就返 500,
// 不调 AppQuit(避免应用死但二进制没换)。
func SpawnHelper(args []string) error {
	script := PickScript()
	if script == "" {
		return errUnsupportedOS
	}
	// 把脚本落到临时文件
	ext := ".sh"
	interpreter := "bash"
	if runtime.GOOS == "windows" {
		ext = ".ps1"
		interpreter = "powershell"
	}
	tmp, err := os.CreateTemp("", "skillbox-updater-*"+ext)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		// 不清理:MVP 阶段先保留(helper 启动后很快自己覆盖),后续可改成清理
		_ = tmpPath
	}()
	if _, err := tmp.WriteString(script); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0o755)
	}

	cmd := exec.Command(interpreter, append([]string{tmpPath}, args...)...)
	// helper 启动后完全独立,父进程不需要 stdio。
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	// 不调 cmd.Wait();helper 必须独立存活
	return nil
}

// DefaultInstallDir 返回当前 OS 预期的"应用安装目录"。
//
// macOS: /Applications/SkillBox.app(由 release 阶段固定);
// Windows: 与 wails app.exe 同目录(走 %LOCALAPPDATA% 兜底);
// Linux: ~/bin/skill-box(由 AppImage 解包得到)。
//
// 真正实施时由 desktop.NewApp 注入具体路径,这里只兜底默认。
func DefaultInstallDir() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/SkillBox.app"
	case "windows":
		// 桌面端在 Windows 上的安装位置是 Program Files;MVP 阶段用 %LOCALAPPDATA%
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return dir + "\\Programs\\SkillBox"
		}
		return "C:\\Program Files\\SkillBox"
	default:
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		return home + "/.local/bin/skill-box"
	}
}

// errUnsupportedOS 是 platform 未覆盖时返的错误。
var errUnsupportedOS = &unsupportedOSError{runtime: runtime.GOOS}

type unsupportedOSError struct{ runtime string }

func (e *unsupportedOSError) Error() string {
	return "updater: unsupported runtime.GOOS=" + e.runtime
}

// 内部用:检查 scripts 是否真读出来了(dev 阶段 fs.ReadFile 失败可能是因为路径写错)。
func scriptsOK() bool {
	if _, err := fs.ReadFile(scriptsFS, "updater_scripts/helper_darwin.sh"); err != nil {
		return false
	}
	if _, err := fs.ReadFile(scriptsFS, "updater_scripts/helper_linux.sh"); err != nil {
		return false
	}
	if _, err := fs.ReadFile(scriptsFS, "updater_scripts/helper_windows.ps1"); err != nil {
		return false
	}
	return true
}

// 工具:确保脚本路径字符串能用在 assemble 里;调试用。
func ScriptNames() []string {
	out := []string{}
	entries, _ := fs.ReadDir(scriptsFS, "updater_scripts")
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "helper_") {
			out = append(out, e.Name())
		}
	}
	return out
}

// 外部暴露(给 desktop/wails_app.go 接到 BootstrapHooks.UpdaterSpawnHelper)
// 改 alias 让 import 路径短一些。
var (
	UpdaterSpawnHelper   = SpawnHelper
	UpdaterInstallDir    = DefaultInstallDir
	UpdaterManifestURLs  = defaultManifestURLsExport
)

// defaultManifestURLsExport 桌面端 manifest 多源 url,实际规则:
//   1) 优先读环境变量 SKILLBOX_UPDATER_URLS(逗号分隔,release 阶段注入);
//   2) 否则走 example.json 的 raw url(MVP 默认)。
//
// env 注入在 release 阶段由 scripts/release.sh 的安装后置脚本完成,本期先用
// 内存默认值(dev 模式下也能跑通)。
func defaultManifestURLsExport() []string {
	if raw := os.Getenv("SKILLBOX_UPDATER_URLS"); raw != "" {
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if v := strings.TrimSpace(p); v != "" {
				out = append(out, v)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{
		"https://raw.githubusercontent.com/dicoder/skill-box/main/build/updater/manifest.example.json",
	}
}
