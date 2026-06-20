package desktop

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WindowManager 统一管理桌面端的所有窗口,提供尺寸记忆、置顶切换等能力。
// 业务侧永远只通过 Wails Bind 调 WindowService,不会直接拿 Wails 对象。
type WindowManager struct {
	mu      sync.Mutex
	primary application.Window
	// 后续可加 secondary、toolbox 等多窗口
}

// NewWindowManager 创建窗口管理器。
func NewWindowManager() *WindowManager {
	return &WindowManager{}
}

// RegisterPrimary 把 Wails 创建的主窗口交给 manager 管理。
func (m *WindowManager) RegisterPrimary(w application.Window) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.primary = w
}

// Primary 返回主窗口,供 service 层调 Wails 原生方法。
func (m *WindowManager) Primary() application.Window {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.primary
}

// ToggleAlwaysOnTop 切换主窗口的"窗口置顶"状态,返回切换后的值。
// Wails v3 alpha 60 未提供 AlwaysOnTop() getter,直接从 options 字段读取。
func (m *WindowManager) ToggleAlwaysOnTop() bool {
	if m.primary == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	opts := m.primary.Options()
	next := !opts.AlwaysOnTop
	m.primary.SetAlwaysOnTop(next)
	return next
}

// ShowPrimary 如果主窗口被最小化/隐藏,把它恢复并置前。
func (m *WindowManager) ShowPrimary() {
	if m.primary == nil {
		return
	}
	m.primary.Show()
	if m.primary.IsMinimised() {
		m.primary.ToggleMaximise() // v3 alpha 暂未提供 UnMinimise 公开 API,这里只是兜底
	}
}
