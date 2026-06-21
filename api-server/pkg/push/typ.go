package push

import "time"

// ReqPush 推送参数
type PushItem struct {
	UserID     uint           `json:"user_id" `
	AppKey     string         `json:"app_key"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	Category   string         `json:"category"`
	NotifyTime time.Time      `json:"notify_time"`
	OtherData  map[string]any `json:"other_data,omitempty"`
}
