// Package services 提供给 Wails Webview 调用的桌面服务绑定。
//
// 命名空间约定(由 Wails 自动从 package path 派生):
//   window.go.notify.NotifyService    ← NotifyService(本文件)
//   window.go.shortcut.ShortcutService ← ShortcutService(shortcut_svc.go)
//   window.go.prefs.PrefsService       ← PrefsService(prefs_svc.go)
//   window.go.app.AppService           ← AppService(app_svc.go,已有)
//   window.go.desktop.WindowService    ← WindowService(window_svc.go,已有)
//   window.go.platform.PlatformService ← PlatformService(platform_svc.go,已有)
package services

import (
	"ginp-api/cmd/bootstrap"
)

// Backend 描述桌面端后端的能力(端口查询)。
// 这里用接口定义,避免 services 反向依赖 desktop 或 bootstrap 包的具体实现。
type Backend interface {
	Port() int
	URL() string
	// NewSettings 桌面端按需构造 *settings.Service,供 PrefsService 用。
	NewSettings() interface{ Get(string) (string, bool, error); Set(string, string) error; GetAll() (map[string]string, error) }
}

// _ 触发 import bootstrap 包;Backend 实际使用时由调用方注入 *bootstrap.Backend。
var _ = bootstrap.DefaultConfigFile
