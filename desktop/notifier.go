package desktop

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Notifier 系统通知包装。
//
// 说明:Wails v3 alpha 60 暂未提供跨平台系统通知 API,
// 这里先通过 InfoDialog 提供视觉反馈,后续 v3 GA 后
// 替换为系统通知 API(macOS NSUserNotification / Windows Toast)。
type Notifier struct {
	app *application.App
}

// NewNotifier 构造 Notifier。
func NewNotifier(app *application.App) *Notifier {
	return &Notifier{app: app}
}

// Notify 触发一次提示(后台 goroutine,不阻塞调用方)。
func (n *Notifier) Notify(title, body string) {
	if n.app == nil {
		return
	}
	go func() {
		n.app.Dialog.Info().
			SetTitle(title).
			SetMessage(body).
			Show()
	}()
}
