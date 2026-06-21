package push

import (
	"fmt"
	"ginp-api/pkg/httpclient"
	"ginp-api/pkg/logger"
)

// HuaweiPushClient 华为推送客户端
type HuaweiPushClient struct {
	Config        *HuaweiPushConfig
	serviceConfig *HuaweiServiceAccountConfig // 可选的服务账号配置，用于自动获取token
}

// NewHuaweiPushClient 创建华为推送客户端
func NewHuaweiPushClient(config *HuaweiPushConfig) *HuaweiPushClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://push-api.cloud.huawei.com"
	}
	return &HuaweiPushClient{
		Config: config,
	}
}

// SendNotification 发送通知消息
func (c *HuaweiPushClient) SendNotification(req *HuaweiPushRequest) (*HuaweiPushResponse, error) {
	if c.Config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	if c.Config.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if len(req.Target.Token) == 0 {
		return nil, fmt.Errorf("at least one push token is required")
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/v3/%s/messages:send", c.Config.BaseURL, c.Config.ProjectID)

	// 构建请求头
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.Config.AccessToken),
		"push-type":     "0", // 0表示Alert消息(通知消息)
	}

	// 构建请求数据
	requestData := map[string]any{
		"payload": map[string]any{
			"notification": map[string]any{
				"category":       req.Payload.Notification.Category,
				"title":          req.Payload.Notification.Title,
				"body":           req.Payload.Notification.Body,
				"clickAction":    req.Payload.Notification.ClickAction,
				"foregroundShow": req.Payload.Notification.ForegroundShow,
				"notifyId":       req.Payload.Notification.NotifyID,
			},
		},
		"target": map[string]any{
			"token": req.Target.Token,
		},
		"pushOptions": map[string]any{
			"testMessage": req.PushOptions.TestMessage,
			"ttl":         req.PushOptions.TTL,
		},
	}

	// 发送HTTP请求
	postParams := &httpclient.PostParams{
		Url:           url,
		Data:          requestData,
		Header:        headers,
		TimeoutSecond: 30,
	}

	var response HuaweiPushResponse
	err := httpclient.Post(postParams, &response)
	if err != nil {
		logger.Errorf("华为推送请求失败: %v", err)
		return nil, fmt.Errorf("华为推送请求失败: %w", err)
	}

	logger.Info("华为推送请求成功, RequestID: %s", response.RequestID)
	return &response, nil
}

// SendSimpleNotification 发送简单通知消息的便捷方法
func (c *HuaweiPushClient) SendSimpleNotification(tokens []string, title, body, category string) (*HuaweiPushResponse, error) {
	req := &HuaweiPushRequest{
		Payload: PushPayload{
			Notification: NotificationMessage{
				Category: category,
				Title:    title,
				Body:     body,
				ClickAction: ClickAction{
					ActionType: 0, // 默认进入应用首页
				},
				ForegroundShow: true, // 默认前台也展示
			},
		},
		Target: PushTarget{
			Token: tokens,
		},
		PushOptions: PushOptions{
			TestMessage: false,
			TTL:         86400, // 默认缓存1天
		},
	}

	return c.SendNotification(req)
}

// SendTestNotification 发送测试通知消息
func (c *HuaweiPushClient) SendTestNotification(tokens []string, title, body, category string) (*HuaweiPushResponse, error) {
	req := &HuaweiPushRequest{
		Payload: PushPayload{
			Notification: NotificationMessage{
				Category: category,
				Title:    title,
				Body:     body,
				ClickAction: ClickAction{
					ActionType: 0,
				},
				ForegroundShow: true,
			},
		},
		Target: PushTarget{
			Token: tokens,
		},
		PushOptions: PushOptions{
			TestMessage: true, // 标记为测试消息
			TTL:         86400,
		},
	}

	return c.SendNotification(req)
}
