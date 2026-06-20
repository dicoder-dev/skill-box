package desktop

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// NewAppMenu 构造桌面端的应用主菜单(macOS 顶部 / Windows 窗口栏)。
// 菜单项的快捷键仅在窗口聚焦时生效;真正的"全局快捷键"需要平台特定 API,
// 在 shortcut.go 中按需补齐。
func NewAppMenu(app *application.App, onShow func(), onQuit func()) *application.Menu {
	menu := application.NewMenu()

	// 视图
	menu.Add("显示主窗口", application.CmdOrCtrl+"+Shift+S").
		OnClick(func(_ *application.Context) {
			if onShow != nil {
				onShow()
			}
		})
	menu.Add("切换置顶", application.CmdOrCtrl+"+Shift+T").
		OnClick(func(_ *application.Context) {
			if w := app.Window.Current(); w != nil {
				opts := w.Options()
				w.SetAlwaysOnTop(!opts.AlwaysOnTop)
			}
		})

	menu.AddSeparator()

	// 帮助
	menu.Add("关于", application.CmdOrCtrl+"+I").
		OnClick(func(_ *application.Context) {
			app.Dialog.InfoDialog().
				SetTitle("关于").
				SetMessage("Skill Box\n桌面端 + Web 端双部署\nhttp://127.0.0.1 后端本地服务").
				Show()
		})

	menu.AddSeparator()

	// 退出
	menu.Add("退出", application.CmdOrCtrl+"+Q").
		OnClick(func(_ *application.Context) {
			if onQuit != nil {
				onQuit()
			}
			app.Quit()
		})

	return menu
}
