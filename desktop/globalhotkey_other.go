//go:build !darwin

// 其它平台全局快捷键占位实现。
//
// V1 只支持 macOS(Carbon RegisterEventHotKey);Windows / Linux 留 no-op,
// Register 返回 error 让调用方走降级路径(只靠菜单 accelerator)。
//
// V2 计划:
//   - Windows: golang.org/x/sys/windows + RegisterHotKey(HWND_MESSAGE, id, mods, vk)
//   - Linux: xgb + XGrabKey(需 X server)/ portal GlobalShortcuts(Wayland)
package desktop

import (
	"fmt"
	"sync"
)

// GlobalHotKeyManager 全局快捷键管理占位。
type GlobalHotKeyManager struct {
	mu       sync.Mutex
	handlers map[string]func()
}

// NewGlobalHotKeyManager 构造占位实例。
func NewGlobalHotKeyManager() *GlobalHotKeyManager {
	return &GlobalHotKeyManager{
		handlers: make(map[string]func()),
	}
}

// Register 在非 darwin 平台直接返回 error,业务侧降级到菜单 accelerator。
func (g *GlobalHotKeyManager) Register(combo string, h func()) error {
	if g == nil || h == nil {
		return fmt.Errorf("globalhotkey: invalid args")
	}
	return fmt.Errorf("globalhotkey: not implemented on this platform (V1 macOS only)")
}

// Unregister 占位 no-op。
func (g *GlobalHotKeyManager) Unregister(combo string) error {
	return nil
}
