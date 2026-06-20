package desktop

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// alwaysOnTopState 维护每个窗口的置顶状态,Wails v3 alpha 60
// 的 Window 接口未提供 getter,只能由我们维护。
type alwaysOnTopState struct {
	mu    sync.Mutex
	state map[uint]bool
}

func newAlwaysOnTopState() *alwaysOnTopState {
	return &alwaysOnTopState{state: make(map[uint]bool)}
}

func (a *alwaysOnTopState) toggle(w application.Window) bool {
	if w == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	next := !a.state[w.ID()]
	a.state[w.ID()] = next
	w.SetAlwaysOnTop(next)
	return next
}

// NewAppMenu 构造桌面端的应用主菜单(macOS 顶部 / Windows 窗口栏)。
// 菜单项的快捷键仅在窗口聚焦时生效;真正的"全局快捷键"需要平台特定 API,
// 在 shortcut.go 中按需补齐。
func NewAppMenu(app *application.App, onShow func(), onQuit func()) *application.Menu {
	state := newAlwaysOnTopState()
	menu := application.NewMenu()

	show := menu.Add("显示主窗口")
	show.SetAccelerator("CmdOrCtrl+Shift+S")
	show.OnClick(func(_ *application.Context) {
		if onShow != nil {
			onShow()
		}
	})

	toggleTop := menu.Add("切换置顶")
	toggleTop.SetAccelerator("CmdOrCtrl+Shift+T")
	toggleTop.OnClick(func(_ *application.Context) {
		w := app.Window.Current()
		if w != nil {
			state.toggle(w)
		}
	})

	menu.AddSeparator()

	about := menu.Add("关于")
	about.SetAccelerator("CmdOrCtrl+I")
	about.OnClick(func(_ *application.Context) {
		app.Dialog.Info().
			SetTitle("关于").
			SetMessage("Skill Box\n桌面端 + Web 端双部署\n本地后端走 http://127.0.0.1").
			Show()
	})

	menu.AddSeparator()

	quit := menu.Add("退出")
	quit.SetAccelerator("CmdOrCtrl+Q")
	quit.OnClick(func(_ *application.Context) {
		if onQuit != nil {
			onQuit()
		}
		app.Quit()
	})

	return menu
}
