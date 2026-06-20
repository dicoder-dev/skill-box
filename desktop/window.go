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

	alwaysOnTop bool // v3 alpha 60 Window 接口无 getter,自己维护
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
func (m *WindowManager) ToggleAlwaysOnTop() bool {
	if m.primary == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alwaysOnTop = !m.alwaysOnTop
	m.primary.SetAlwaysOnTop(m.alwaysOnTop)
	return m.alwaysOnTop
}

// ShowPrimary 如果主窗口被最小化/隐藏,把它恢复并置前。
func (m *WindowManager) ShowPrimary() {
	if m.primary == nil {
		return
	}
	m.primary.Show()
	if m.primary.IsMinimised() {
		m.primary.UnMinimise()
	}
	m.primary.Focus()
}
