// Package services 提供给 Wails Webview 调用的桌面服务绑定。
//
// 命名空间约定(由 Wails 自动从 package path 派生):
//   window.go.notify.NotifyService    ← NotifyService(本文件)
//   window.go.shortcut.ShortcutService ← ShortcutService(shortcut_svc.go)
//   window.go.prefs.PrefsService       ← PrefsService(prefs_svc.go)
//   window.go.app.AppService           ← AppService(app_svc.go,已有)
//   window.go.desktop.WindowService    ← WindowService(window_svc.go,已有)
//   window.go.platform.PlatformService ← PlatformService(platform_svc.go,已有)
//
// 业务调用请走 HTTP,不暴露在这里。
package services

// Notifier 抽象 desktop.Notifier 的最小能力,避免 services 反向依赖 desktop。
type Notifier interface {
	HasPermission() bool
	RequestAuthorization() (bool, error)
	Notify(id, title, body string) error
}

// NotifyService 暴露给前端的通知服务。
// 仅做"通知层"的事,业务逻辑不允许塞到这里。
type NotifyService struct {
	n Notifier
}

// NewNotifyService 构造 NotifyService。
func NewNotifyService(n Notifier) *NotifyService {
	return &NotifyService{n: n}
}

// Show 发送一条系统通知。
// id 留空时由 Notifier 内部生成;title 必填;body 可选。
func (s *NotifyService) Show(id, title, body string) error {
	if s.n == nil {
		return nil
	}
	return s.n.Notify(id, title, body)
}

// HasPermission 查询当前通知授权状态。
func (s *NotifyService) HasPermission() bool {
	if s.n == nil {
		return false
	}
	return s.n.HasPermission()
}

// RequestAuthorization 触发系统弹授权窗(macOS 首次启动会弹系统对话框)。
// 返回 true = 用户允许,false = 拒绝或失败。
func (s *NotifyService) RequestAuthorization() (bool, error) {
	if s.n == nil {
		return false, nil
	}
	return s.n.RequestAuthorization()
}
