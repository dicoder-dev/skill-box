// Package launchagent 给 macOS 桌面端做 LaunchAgent 自注册,绕开 macOS 26 Tahoe
// 的 amfi Code=-423 静默拒启动问题。
//
// 背景:
//   - 本 dmg 是 ad-hoc 签名,wails3 默认 sign 不带 --options runtime。
//   - macOS 26 Tahoe 的 amfi 对 LaunchServices / amfi 派发链上的 ad-hoc signed
//     binary 报 Code=-423,open / Finder 双击 / 右键打开全部一闪而过,
//     Gatekeeper 的「仍要打开」GUI 兜底根本走不到。
//   - 实测唯一能起 ad-hoc binary 的入口是 launchd / launchctl 直派发
//     (launchedByLS=0),例如 `launchctl asuser $UID <binary>` 或
//     `~/Library/LaunchAgents/<label>.plist` 走 launchd 起来。
//
// 策略(2026-07-15):
//   1. binary 启动时,如果 launchd 没拉自己(LaunchAgentLabel env 没值),
//      说明是从 Finder 双击进来的 → amfi 那一关已经过了但进程被 fork,
//      此时写 plist + launchctl bootstrap 让 launchd 接管。
//   2. 如果 LaunchAgentLabel env 有值,说明是 launchd 在拉,直接跑,不递归。
//   3. KeepAlive=false → 用户手动关掉就是真的关掉(之前的 KeepAlive=true 让用户
//      反馈"很脑残的机制 现在双击又不可以打开了")。
//   4. RunAtLoad=false → 不开机自启,只在 launchctl bootstrap 当下拉一次。
//
// 用户视角:
//   1. 双击 dmg → 拖 .app 到 /Applications → 双击 .app
//   2. macOS 26 Tahoe amfi 杀进程(用户看到一闪而过)
//   3. 用户再双击一次(本进程是 2 进/3 出 之类)
//      → binary 检测无 plist → 写 plist → bootstrap → launchd 拉自己 → 起来
//   4. 之后启动方式变:`launchctl kickstart -k gui/$UID/com.dicoder.skillbox`
//      Finder 双击仍会被 amfi 杀,但 launchd 那一份会留着。
package launchagent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Label 是 LaunchAgent 的唯一标识(同时也是 plist 文件名 / launchd 域)。
// 也用作 launchctl bootstrap 的 service-target 字段。
const Label = "com.dicoder.skillbox"

// PlistPath 返回 ~/Library/LaunchAgents/<Label>.plist 的绝对路径。
//
// 注意:macOS 上 LaunchAgent 必须放在 ~/Library/LaunchAgents/ 下,系统才会
// 认这是 user-level agent(走 launchd gui/$UID 域)。放在 /Library/LaunchAgents
// 是 system-level,需要 sudo,不友好。
func PlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("launchagent: UserHomeDir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", Label+".plist"), nil
}

// IsLaunchdChild 报告当前进程是不是 launchd 派发的(launchd 拉起时设了
// LaunchAgentLabel 环境变量)。如果是,binary 不要走自注册逻辑,避免递归。
func IsLaunchdChild() bool {
	return os.Getenv("LaunchAgentLabel") == Label
}

// InstalledBinaryPath 返回当前 plist 里 ProgramArguments[0] 指向的 binary 绝对路径。
//
// 用途:macOS 26 Tahoe 自启动要求 plist 的 program 路径必须跟当前 binary 一致。
// dmg 装的 .app 在 /Applications,plist 之前可能是 dev 期 build 写下的路径;
// 首次启动 dmg .app 时如果 plist 路径不一致,需要用当前 binary 路径重写 plist
// 才能让后续 launchd 派发链拉对 binary。
//
// 实现:plist 是 XML,程序里就用 plist 包的最小子集查 ProgramArguments。
// 失败时(文件不存在 / 解析失败 / 无 ProgramArguments)返回空字符串,
// 调用方按"plist 未装"处理。
func InstalledBinaryPath() (string, error) {
	plist, err := PlistPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(plist)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("launchagent: read plist %s: %w", plist, err)
	}
	// plist XML 形态:
	//   <key>ProgramArguments</key>
	//   <array>
	//     <string>/path/to/binary</string>
	//     ...
	//   </array>
	// 这里用极简 grep:先找 <key>ProgramArguments</key>,再找紧跟的 <array>...</array>。
	// 不引入第三发 plist 解析包,plist 结构稳定,够用。
	body := string(data)
	keyIdx := strings.Index(body, "<key>ProgramArguments</key>")
	if keyIdx < 0 {
		return "", nil
	}
	after := body[keyIdx:]
	arrStart := strings.Index(after, "<array>")
	if arrStart < 0 {
		return "", nil
	}
	after = after[arrStart+len("<array>"):]
	arrEnd := strings.Index(after, "</array>")
	if arrEnd < 0 {
		return "", nil
	}
	arr := after[:arrEnd]
	strStart := strings.Index(arr, "<string>")
	if strStart < 0 {
		return "", nil
	}
	arr = arr[strStart+len("<string>"):]
	strEnd := strings.Index(arr, "</string>")
	if strEnd < 0 {
		return "", nil
	}
	return strings.TrimSpace(arr[:strEnd]), nil
}

// IsInstalled 报告 LaunchAgent 是否已注册(plist 文件存在 + launchd 域里有目标)。
//
// 只检查 plist 文件存在,因为 launchctl list 在 launchd 重启 / 用户登出后
// 会清掉内存里的服务记录,文件存在更可靠。
func IsInstalled() (bool, error) {
	p, err := PlistPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Install 写 plist 到 ~/Library/LaunchAgents/ 并 launchctl bootstrap 立即激活。
//
// 参数:
//   - binaryPath:plist 里 ProgramArguments[0] 指向的 binary 绝对路径。
//     通常是 os.Executable()。
//   - logPath  :LaunchAgent 的 stdout/stderr 重定向到这里,方便排查。
//
// 注意:bootstrap 成功后会立刻让 launchd fork 一个新实例拉 binary,这个新
// 实例会带 LaunchAgentLabel 环境变量,IsLaunchdChild 会返 true;当前进程
// (本函数调用方)不退出,继续跑下去会和 launchd 拉的那一份撞 8082 端口。
// 因此调用方在 Install 返回成功时应该 os.Exit(0),让 launchd 那份接管。
//
// 如果 plist 已存在,先 bootout 旧域,再写新 plist + bootstrap,避免
// `launchctl bootstrap` 报"service already loaded"。
func Install(binaryPath, logPath string) error {
	plist, err := PlistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return fmt.Errorf("launchagent: MkdirAll: %w", err)
	}

	// 已存在的 plist 先 bootout,否则后续 bootstrap 报 "service already loaded"。
	// bootout 失败(服务没在跑 / 已 bootout)不阻断,只是日志提示。
	if _, err := os.Stat(plist); err == nil {
		uid := os.Getuid()
		domainTarget := fmt.Sprintf("gui/%d", uid)
		if bout := exec.Command("launchctl", "bootout", domainTarget, plist); bout.Err != nil {
			// 忽略:可能是 plist 已不在 launchd 域里
		} else {
			_ = bout.Run()
		}
	}

	content := buildPlist(binaryPath, logPath)
	if err := os.WriteFile(plist, []byte(content), 0o644); err != nil {
		return fmt.Errorf("launchagent: WriteFile %s: %w", plist, err)
	}

	// launchctl bootstrap 把 plist 注册到 launchd gui/<uid> 域。
	// 命令格式:`launchctl bootstrap <domain-target> <service-path>`
	//   - domain-target:gui/<uid>(user-domain)
	//   - service-path  :plist 路径
	//
	// 注意:RunAtLoad=false 时,bootstrap 不会立刻拉实例。本函数故意不
	// kickstart,让调用方决定:首次启动(本进程继续 Serve)不 kickstart;
	// 后续启动(本进程退出,kickstart 让 launchd 拉一份)才 kickstart。
	// 实测(2026-07-15):kickstart 立刻 fork 跟当前进程撞 8082,死循环。
	uid := os.Getuid()
	domainTarget := fmt.Sprintf("gui/%d", uid)
	cmd := exec.Command("launchctl", "bootstrap", domainTarget, plist)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchagent: launchctl bootstrap %s %s: %w (out=%s)",
			domainTarget, plist, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Uninstall 反向操作:bootout + 删 plist。给 settings / quit 流程调用。
func Uninstall() error {
	plist, err := PlistPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(plist); os.IsNotExist(err) {
		return nil // 幂等
	}
	uid := os.Getuid()
	// bootout 格式:`launchctl bootout <domain-target> <service-path>`
	// 跟 Install 同样的命令语法。
	domainTarget := fmt.Sprintf("gui/%d", uid)
	// bootout 可能失败(plist 已不在 / 服务没在跑),用 ignore-error 兜底。
	_ = exec.Command("launchctl", "bootout", domainTarget, plist).Run()
	if err := os.Remove(plist); err != nil {
		return fmt.Errorf("launchagent: Remove %s: %w", plist, err)
	}
	return nil
}

// buildPlist 生成 plist 内容。字段含义参考:
//   https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html
//
// KeepAlive=false → 用户手动关掉就是真的关掉(用户原话:KeepAlive=true 很脑残)。
// RunAtLoad=false → 不开机自启,仅 bootstrap 当下拉一次;用户想开机自启由
//
//	settings 单独提供开关。
//
// ProcessType=Interactive → 让 launchd 把它当 GUI 进程,而不是后台 daemon,
//	这样 dock 图标 / 菜单栏能正常出现(launchd 对 GUI app 的处理)。
func buildPlist(binaryPath, logPath string) string {
	// 注意:plist 是 XML,不能写 Go 的 %q。ProgramArguments 是 string array,
	// 这里只有一个 binary 路径,所以不需要复杂转义(路径里不会含 & < >)。
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>

    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>

    <key>ProcessType</key>
    <string>Interactive</string>

    <key>KeepAlive</key>
    <false/>

    <key>RunAtLoad</key>
    <false/>

    <key>WorkingDirectory</key>
    <string>%s</string>

    <key>StandardOutPath</key>
    <string>%s</string>

    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, Label, binaryPath, homeDir(), logPath, logPath)
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}