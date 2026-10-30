---
name: wails-v3-tray-darwin-doubleclick-deadcode
description: Wails v3 在 macOS 上托盘 OnDoubleClick 永远不触发(到 v3.0.0 正式版仍未修),darwin processClick 只 case left/right button
metadata:
  type: project
---

# Wails v3 macOS 托盘双击是死代码(截至 v3.0.0 正式版)

Wails v3 的 `SystemTray.OnDoubleClick(handler)` 在 macOS 上**永远不会被触发**。

## 根因

darwin 路径的点击派发实现 `v3/pkg/application/systemtray_darwin.go` 的 `processClick(b button)`:

```go
func (s *macosSystemTray) processClick(b button) {
	switch b {
	case leftButtonDown:  // 单击左键
	case rightButtonDown: // 单击右键
	}
	// 没有 doubleClick 分支
}
```

而 `doubleClickHandler` 字段定义在 `systemtray.go:92`,赋值在 `systemtray.go:336`
(`OnDoubleClick` 的 setter),**darwin 侧从未读取它**。

实测确认(2026-10-30,升级到 v3.0.0 正式版后重新核查源码):
- `systemtray_darwin.go` 的 `processClick` 仍然只 `case leftButtonDown` / `case rightButtonDown`
- 全仓 grep `doubleClickHandler` 只有 `systemtray.go`(定义+setter)、`systemtray_windows.go:474`(真正 dispatch)
- **只有 Windows 实现真正派发双击**,Linux 和 macOS 都没有

## 项目里的处理

`desktop/tray.go` 的 `NewTrayManager` **故意不绑** `OnDoubleClick`,并在注释里写明原因。
留着绑定就是误导性死代码 —— 开发者以为双击可用,实际 macOS 上毫无反应。

跨平台行为差异(项目已适配):
- **darwin**: 左键单击 = 弹出托盘菜单(`SetTemplateIcon` 单色图标,随系统 Light/Dark 自动反色);`SetLabel` 留空避免图标+文字并存
- **windows / linux**: 左键单击 = 显示主窗口,右键 = 弹出菜单(`SetIcon` 彩色)

## How to apply

- **不要**在 darwin 上依赖托盘双击做交互(比如"双击打开主窗口"),它不会触发。
- 需要 macOS 上"双击托盘图标显示窗口",只能自己在 `OnClick` 里自己计时实现
  (记录上次点击时间戳,间隔 < 400ms 视为双击),或者用 `OnClick` 直接显示窗口。
- 项目侧的窗口显示入口统一走 `WindowManager.ShowPrimary()`,托盘回调调它即可。
- 升到新版本 wails 后**重新 grep 一次** `systemtray_darwin.go` 的 `processClick`,
  确认上游是否修复了双击派发再决定要不要启用。

相关:[[wails-v3-getscreen-primary-resize]](窗口尺寸/屏幕 DIP 相关)、[[wails-v3-upgrade-3-0-0]](升级到正式版)
