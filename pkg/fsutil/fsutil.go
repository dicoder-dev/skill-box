// Package fsutil 提供跨平台的本地文件能力(读文本 + 系统文件管理器 reveal)。
//
// 设计:
//   - 与 cdesktop 解耦:既可被 cdesktop HTTP 端点使用,也可被桌面端 wails_app
//     直接 import,不形成循环依赖。
//   - 位置:放在 root module 的 pkg/ 下(api-server + desktop 都能 import,
//     internal/ 不能跨 module 共享)。
//
// 两个能力:
//   - ReadText(path): 读文本文件(1 MB 上限,超过返 error)
//   - Reveal(path):   在系统文件管理器中显示该路径
//
// Reveal 在 macOS 走 `open -R`(高亮),Windows 走 `explorer /select,`,
// Linux 兜底 `xdg-open` 父目录。

package fsutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// MaxReadBytes 1 MB 上限,超过返 error。
const MaxReadBytes = 1 << 20

// ReadText 读文件文本内容,大小上限 1 MB。
func ReadText(path string) (string, error) {
	cleaned := filepath.Clean(path)
	fi, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	if fi.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", cleaned)
	}
	if fi.Size() > MaxReadBytes {
		return "", fmt.Errorf("file too large: %d bytes (limit %d)", fi.Size(), MaxReadBytes)
	}
	buf := bytes.NewBuffer(nil)
	f, err := os.Open(cleaned)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	// 即使 stat 报小,也用 ReadFrom 拿到真实大小,防止 TOCTOU
	if _, err := buf.ReadFrom(f); err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	if buf.Len() > MaxReadBytes {
		return "", fmt.Errorf("file too large: read %d bytes (limit %d)", buf.Len(), MaxReadBytes)
	}
	return buf.String(), nil
}

// Reveal 在系统文件管理器中显示给定路径。
func Reveal(path string) error {
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("abs: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return exec.Command("open", abs).Start()
		}
		return exec.Command("open", "-R", abs).Start()
	case "windows":
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return exec.Command("explorer", abs).Start()
		}
		return exec.Command("explorer", "/select,", abs).Start()
	default:
		return exec.Command("xdg-open", "file://"+filepath.Dir(abs)).Start()
	}
}

// ProjectHint 是从目录路径推断出来的"项目元信息",供前端"导入项目"流程预填表单。
//
//   - Name:取目录的 basename,如 /Users/x/repo/foo → "foo"
//   - Alias:取 Name 的 slug 化结果,小写 + 非字母数字替换为 "-" + 折叠重复 "-"。
//     例如 "My App" → "my-app","foo_bar.baz" → "foo-bar-baz"。
//
// 解析原则:
//   - Name 直接取 basename(显示名,用户可在前端覆盖)
//   - Alias 是"机器友好"的标识,前端仍允许用户改成自己想要的
//   - 不读 package.json / pyproject / Cargo.toml,保持"零依赖、轻启发";
//     真要做元数据探测应放到独立 service 里,这里只做兜底
type ProjectHint struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// InspectProject 根据给定目录路径推断项目元信息。
//
// 失败语义:
//   - path 不存在 / 不是目录:返 error
//   - path 是目录但 basename 为空(理论不会发生,root "/" 算 1 段):返 error
func InspectProject(path string) (*ProjectHint, error) {
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", abs)
	}
	base := filepath.Base(abs)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return nil, fmt.Errorf("cannot infer project name from path: %s", abs)
	}
	return &ProjectHint{
		Name:  base,
		Alias: slugify(base),
	}, nil
}

// RemovePath 删除给定路径(文件或目录,带目录树)。
//
// 安全策略:
//   - 拒绝删除根目录 "/" / "." / ""(避免误删整盘)
//   - path 必须是绝对路径(前端传来的 source_path 已是 EvalSymlinks 后的真实绝对路径,
//     但 fsutil 仍然走 filepath.Abs 双保险,确保后续 filepath.Rel(parent) 比较有意义)
//   - 调用方必须自己确认"该路径确实要删"(二次确认、scope 校验等),fsutil
//     不做语义层校验,只做物理层防护(防 root / 防空)
//   - 失败语义:不存在 → 不报错(幂等,跟 Reveal 一致);权限不足/路径不可达 → 报错
func RemovePath(path string) error {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return fmt.Errorf("path is empty")
	}
	if cleaned == "/" || cleaned == "." || cleaned == ".." {
		return fmt.Errorf("refusing to delete root-like path: %q", cleaned)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("abs: %w", err)
	}
	// 双保险:绝对路径化后再次判定,防 cleaned="/./" 这类绕过
	if abs == "/" || abs == "." {
		return fmt.Errorf("refusing to delete root-like path: %q", abs)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			// 幂等:已经不存在就当成功,不报 404(前端调完再 reload 不会因为 race 翻车)
			return nil
		}
		return fmt.Errorf("stat: %w", err)
	}
	if fi.IsDir() {
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("remove dir: %w", err)
		}
		return nil
	}
	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// slugify 把任意字符串规整成"小写 + 字母数字保留,其它替成 '-',折叠重复 '-' 并 trim"。
//
// 例:
//   - "My App"        → "my-app"
//   - "foo_bar.baz"   → "foo-bar-baz"
//   - "Hello World!!" → "hello-world"
//   - "你好-world"     → "world"
//   - "---"           → ""  (调用方需要保证非空别名,这里只是字符串规整)
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := b.String()
	out = strings.TrimRight(out, "-")
	return out
}
