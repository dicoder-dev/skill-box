// Package desktop 桌面端 Wails 应用的设置 key 常量。
//
// 所有桌面端偏好统一以 "desktop." 前缀,避免和业务设置冲突。
// 这些 key 在 internal/settings.Service 持久化(SQLite entity.Setting 表),
// 启动期 / 运行时都被 desktop.WailsApp 读 / 写。
package desktop

// 偏好 key 常量。前端 + 后端都通过 desktop.prefs.get/set 读写。
const (
	// PrefKeyStartMinimized 启动时是否最小化到托盘(关窗不退出,只到托盘)。
	// 取值:"true" / "false",默认 "false"。
	PrefKeyStartMinimized = "desktop.start_minimized"

	// PrefKeyNotifyEnabled 是否允许系统通知(用户在 Settings 页面可关)。
	// 取值:"true" / "false",默认 "true"。
	PrefKeyNotifyEnabled = "desktop.notify_enabled"

	// PrefKeyShortcutEnabled 是否启用全局快捷键(打开主窗口)。
	// macOS 首次需用户在"系统设置 → 隐私与安全 → 辅助功能"手动授权。
	// 取值:"true" / "false",默认 "true"。
	PrefKeyShortcutEnabled = "desktop.shortcut_enabled"

	// PrefKeyGlobalHotKey 快捷键组合,默认 "Cmd+Shift+S"。
	// 格式遵循 Wails accelerator 字符串。
	PrefKeyGlobalHotKey = "desktop.global_hotkey"
)

// PrefsDefaults 返回所有偏好的默认值。Get 返回不存在键时使用。
func PrefsDefaults() map[string]string {
	return map[string]string{
		PrefKeyStartMinimized: "false",
		PrefKeyNotifyEnabled:  "true",
		PrefKeyShortcutEnabled: "true",
		PrefKeyGlobalHotKey:    "Cmd+Shift+S",
	}
}
