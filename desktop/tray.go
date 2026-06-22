package desktop

import (
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TrayManager 包装 Wails 系统托盘,统一通过它创建/管理托盘。
//
// 菜单项:显示主窗口 / 测试通知 / 偏好设置 / 关于 / 退出。
// "测试通知" / "偏好设置" 是 V1 新增;前者调 notifier.Notify,后者跳到
// /settings/desktop 路由(SettingsView 末尾的桌面端 section)。
type TrayManager struct {
	tray     *application.SystemTray
	notifier *Notifier
}

// TrayCallbacks 托盘菜单回调。onShow / onQuit 必须;onOpenSettings 可选。
type TrayCallbacks struct {
	OnShow         func()
	OnQuit         func()
	OnOpenSettings func()
}

// NewTrayManager 构造并创建托盘。
//
// notifier 可为 nil(测试菜单项 no-op);backend 也不强依赖,只是偏好设置菜单
// 跳转需要它。
func NewTrayManager(app *application.App, cb TrayCallbacks, notifier *Notifier) *TrayManager {
	t := app.SystemTray.New()
	t.SetLabel("Skill Box")
	t.SetTooltip("Skill Box")
	t.OnClick(cb.OnShow)
	t.OnDoubleClick(cb.OnShow)
	t.SetMenu(buildTrayMenu(app, cb, notifier))
	t.Show()
	return &TrayManager{tray: t, notifier: notifier}
}

// buildTrayMenu 构造托盘菜单。
func buildTrayMenu(app *application.App, cb TrayCallbacks, notifier *Notifier) *application.Menu {
	menu := application.NewMenu()

	menu.Add("显示主窗口").
		OnClick(func(_ *application.Context) {
			if cb.OnShow != nil {
				cb.OnShow()
			}
		})

	menu.Add("测试通知").
		OnClick(func(_ *application.Context) {
			if notifier == nil {
				log.Printf("tray: notifier not initialized, skip test notify")
				return
			}
			err := notifier.Notify(
				"",
				"Skill Box",
				"托盘测试通知 — "+time.Now().Format("15:04:05"),
			)
			if err != nil {
				log.Printf("tray: notify failed: %v", err)
			}
		})

	if cb.OnOpenSettings != nil {
		menu.Add("偏好设置").
			OnClick(func(_ *application.Context) {
				if cb.OnOpenSettings != nil {
					cb.OnOpenSettings()
				}
			})
	}

	menu.AddSeparator()
	menu.Add("关于 Skill Box").
		OnClick(func(_ *application.Context) {
			app.Dialog.Info().
				SetTitle("关于").
				SetMessage("Skill Box\n桌面端 + Web 端双部署\n本地后端走 http://127.0.0.1").
				Show()
		})
	menu.AddSeparator()
	menu.Add("退出").
		OnClick(func(_ *application.Context) {
			if cb.OnQuit != nil {
				cb.OnQuit()
			}
			app.Quit()
		})
	return menu
}
