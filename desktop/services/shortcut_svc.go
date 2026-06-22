// Package services 桌面端服务绑定 — ShortcutService。
package services

// ShortcutManager 抽象 desktop.ShortcutManager 的最小能力。
type ShortcutManager interface {
	Register(combo string, handler func()) error
	List() []string
}

// ShortcutService 暴露给前端的全局快捷键服务。
// 业务侧只关心 combo 字符串,具体平台实现(macOS Carbon / Windows RegisterHotKey)
// 由 desktop.ShortcutManager 内部消化。
type ShortcutService struct {
	mgr ShortcutManager
}

// NewShortcutService 构造 ShortcutService。
func NewShortcutService(mgr ShortcutManager) *ShortcutService {
	return &ShortcutService{mgr: mgr}
}

// Register 注册一个全局快捷键;成功返回 true。
// 失败原因(平台不支持 / 缺权限 / combo 不合法)由 mgr.Register 的 error 描述。
func (s *ShortcutService) Register(combo string) bool {
	if s.mgr == nil {
		return false
	}
	return s.mgr.Register(combo, func() {
		// 这里是 hotkey 触发的回调入口,后续 V2 可加 emit
		// (把"快捷键被按"事件推到前端,前端可决定跳哪)。
	}) == nil
}

// List 返回当前已注册的所有 combo。
func (s *ShortcutService) List() []string {
	if s.mgr == nil {
		return []string{}
	}
	return s.mgr.List()
}
