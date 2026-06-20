package desktop

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TrayManager 包装 Wails 系统托盘,统一通过它创建/管理托盘。
type TrayManager struct {
	tray *application.SystemTray
}

// NewTrayManager 构造并创建托盘。
func NewTrayManager(app *application.App, onShow func(), onQuit func()) *TrayManager {
	t := app.SystemTray.New()
	t.SetLabel("Skill Box")
	t.SetTooltip("Skill Box")
	t.OnClick(onShow)
	t.OnDoubleClick(onShow)
	t.SetMenu(buildTrayMenu(app, onShow, onQuit))
	t.Show()
	return &TrayManager{tray: t}
}

// buildTrayMenu 构造托盘菜单:显示主窗口 / 关于 / 退出。
func buildTrayMenu(app *application.App, onShow func(), onQuit func()) *application.Menu {
	menu := application.NewMenu()
	menu.Add("显示主窗口").
		OnClick(func(_ *application.Context) {
			if onShow != nil {
				onShow()
			}
		})
	menu.AddSeparator()
	menu.Add("关于 Skill Box").
		OnClick(func(_ *application.Context) {
			app.Dialog.Info().
				SetTitle("关于").
				SetMessage("Skill Box\n桌面端 + Web 端双部署").
				Show()
		})
	menu.AddSeparator()
	menu.Add("退出").
		OnClick(func(_ *application.Context) {
			if onQuit != nil {
				onQuit()
			}
			app.Quit()
		})
	return menu
}
