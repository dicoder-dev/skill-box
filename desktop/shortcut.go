package desktop

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ShortcutManager 全局快捷键占位实现。
//
// 说明:Wails v3 alpha 60 没有暴露跨平台全局快捷键 API;
// macOS 需要 CGEventTap,Windows 需要 RegisterHotKey,Linux 需要 X11/Wayland 协议,
// 这些都涉及 cgo 与平台特定代码。短期方案:仅通过菜单加速键(CmdOrCtrl+Shift+S)
// 提供"窗口聚焦时"显示主窗口的能力,这是 v3 已支持的最自然形式。
//
// 后续若需要"应用未聚焦时也能唤起",在 platform_darwin.go / platform_windows.go
// 里用 cgo 实现 CGEventTap / RegisterHotKey,本文件保留抽象以保证上层调用稳定。
type ShortcutManager struct {
	mu       sync.Mutex
	handlers map[string]func()
	app      *application.App
}

// NewShortcutManager 构造快捷键管理器。
func NewShortcutManager(app *application.App) *ShortcutManager {
	return &ShortcutManager{
		handlers: make(map[string]func()),
		app:      app,
	}
}

// Register 注册一个全局快捷键(当前 alpha 仅在窗口菜单中生效,真正全局键位需要后续平台实现)。
// combo 示例:"CmdOrCtrl+Shift+S"。
func (s *ShortcutManager) Register(combo string, handler func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[combo] = handler
	// 占位:仅记录,不实际绑定全局快捷键。菜单 accelerator 已在 menu.go 里覆盖相同组合。
}
