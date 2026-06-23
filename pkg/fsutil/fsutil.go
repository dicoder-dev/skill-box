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
