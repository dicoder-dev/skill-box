package desktop

import (
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	wnotif "github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// Notifier 桌面端系统通知管理器。
//
// 实现要点:
//   - macOS 走 NSUserNotificationCenter(由 v3 内置 notifications service 提供),
//     首次启动调 RequestAuthorization 触发系统弹窗;用户拒绝后,后续 SendNotification 静默失败。
//   - 平台要求应用必须打包 + 签名,见 build/darwin/Taskfile.yml 的 package 任务。
//   - SendNotification 需要 ID + Title(ID 用于去重 / 关闭;Title 必填)。
//   - OnNotificationResponse 注册回调,把用户点通知的事件 emit 到前端
//     (事件名 "notify:clicked",data: [notifID, actionID])。
type Notifier struct {
	mu      sync.Mutex
	svc     *wnotif.NotificationService
	app     *application.App
	enabled bool // 跟随 desktop.notify_enabled 偏好
}

// NewNotifier 构造 Notifier 并挂上 OnNotificationResponse 回调。
//
// app 不能为 nil;服务已通过 wnotif.New() 内部 sync.Once 全局单例,
// 多次 NewNotifier 共享同一 service,OnNotificationResponse 是 set 语义
// (只有最后一次注册生效),所以桌面端单实例化即可。
func NewNotifier(app *application.App) *Notifier {
	n := &Notifier{
		svc:     wnotif.New(),
		app:     app,
		enabled: true,
	}
	n.svc.OnNotificationResponse(func(result wnotif.NotificationResult) {
		if app != nil {
			app.Event.Emit("notify:clicked", result.Response.ID, result.Response.ActionIdentifier)
		}
	})
	return n
}

// SetEnabled 设置通知开关;为 false 时 SendNotification 直接 no-op,不调系统 API。
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// HasPermission 查询当前通知授权状态。Web 端平台 / 未打包的 dev binary 可能返回 false。
func (n *Notifier) HasPermission() bool {
	if n == nil || n.svc == nil {
		return false
	}
	ok, _ := n.svc.CheckNotificationAuthorization()
	return ok
}

// RequestAuthorization 调系统 API 弹授权窗(macOS 首次启动会触发系统弹窗)。
// 用户允许返回 true,拒绝返回 false。失败时返回 error。
func (n *Notifier) RequestAuthorization() (bool, error) {
	if n == nil || n.svc == nil {
		return false, fmt.Errorf("notifier: not initialized")
	}
	return n.svc.RequestNotificationAuthorization()
}

// Notify 触发一次系统通知;title 必填。
//
// id 留空时自动用 time.Now().UnixNano() 拼一个(必须唯一);
// body 为通知正文;为简化调用,subtitle / category 暂不暴露给上层。
//
// SetEnabled(false) 时直接返回 nil,不发任何通知。
func (n *Notifier) Notify(id, title, body string) error {
	if n == nil || n.svc == nil {
		return fmt.Errorf("notifier: not initialized")
	}
	n.mu.Lock()
	enabled := n.enabled
	n.mu.Unlock()
	if !enabled {
		return nil
	}
	if id == "" {
		id = fmt.Sprintf("skill-box-%d", time.Now().UnixNano())
	}
	return n.svc.SendNotification(wnotif.NotificationOptions{
		ID:    id,
		Title: title,
		Body:  body,
	})
}
