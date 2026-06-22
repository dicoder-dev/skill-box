package desktop

import "sync"

// ShortcutManager 全局快捷键管理。
//
// V1:macOS 走 Carbon RegisterEventHotKey(见 globalhotkey_darwin.go);
//    其它平台降级到只走菜单 accelerator。
//
// 限制:macOS 首次需要用户授予 Accessibility 权限,否则
// RegisterEventHotKey 静默失败(回调不触发)。
// 错误会通过 Register 返回,业务侧根据 err 提示用户去系统设置。
type ShortcutManager struct {
	mu        sync.Mutex
	ghk       *GlobalHotKeyManager
	handlers  map[string]func()
	enabled   bool
}

// NewShortcutManager 构造 ShortcutManager。
func NewShortcutManager() *ShortcutManager {
	return &ShortcutManager{
		ghk:      NewGlobalHotKeyManager(),
		handlers: make(map[string]func()),
		enabled:  true,
	}
}

// SetEnabled 启用 / 停用全局快捷键;为 false 时 Register 不真绑,只记录。
func (m *ShortcutManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.enabled == enabled {
		return
	}
	m.enabled = enabled
	if !enabled {
		// 全部解绑
		for combo := range m.handlers {
			_ = m.ghk.Unregister(combo)
		}
	} else {
		// 重新绑回去
		for combo, h := range m.handlers {
			_ = m.ghk.Register(combo, h)
		}
	}
}

// Register 注册一个全局快捷键。
// combo 当前 V1 只支持 "Cmd+Shift+S"(macOS);其它平台 / 其它 combo 返回 error。
// handler 异步执行,不会阻塞 CFRunLoop / 平台事件循环。
func (m *ShortcutManager) Register(combo string, handler func()) error {
	if m == nil || handler == nil {
		return nil
	}
	m.mu.Lock()
	m.handlers[combo] = handler
	enabled := m.enabled
	m.mu.Unlock()
	if !enabled {
		return nil
	}
	return m.ghk.Register(combo, handler)
}

// List 返回当前已注册的所有 combo。
func (m *ShortcutManager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.handlers))
	for c := range m.handlers {
		out = append(out, c)
	}
	return out
}
