// Package caiprovider - chat_stream.a.go
// POST /api/skillbox/ai/chat
//
// SSE 流式对话;协议兼容 OpenAI 的 text/event-stream:
//   - 每条事件:  data: {"kind":"chunk","text":"..."}\n\n
//   - 结束标记:  data: [DONE]\n\n
//   - 错误事件:  data: {"kind":"error","err":"..."}\n\n
//
// 入参两种风格(任选其一):
//   1) { provider?, model?, messages:[...], temperature?, max_tokens? }
//   2) { provider?, preset_id, vars:{...} }  ← 自动渲染 prompt
package caiprovider

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestChat 流式对话入参。
// Provider / Model 留空时由 manager 按 priority 选默认;PresetID + Vars 触发预设。
type RequestChat struct {
	Provider    string             `json:"provider"`
	Model       string             `json:"model"`
	Messages    []aiengine.Message `json:"messages"`
	PresetID    string             `json:"preset_id"`
	Vars        map[string]string  `json:"vars"`
	Temperature *float32           `json:"temperature,omitempty"`
	MaxTokens   *int               `json:"max_tokens,omitempty"`
}

// ChatStream POST /api/skillbox/ai/chat(SSE)
func ChatStream(c *gin.Context) {
	var req RequestChat
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}

	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)

	ctx := c.Request.Context()
	var (
		ch  <-chan aiengine.StreamEvent
		err error
	)
	if req.PresetID != "" {
		ch, err = svc.ChatWithPreset(ctx, req.PresetID, req.Provider, req.Vars)
	} else {
		ch, err = svc.Chat(ctx, aiengine.ChatRequest{
			Provider:    req.Provider,
			Model:       req.Model,
			Messages:    req.Messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxTokens,
		}, req.Provider)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SSE 头必须在写 body 前 set;gin.Status() == 200 后 Flush 才能稳定触发浏览器逐条渲染。
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	// 写一条 comment 行(心跳)让浏览器立即知道连接建立。
	if flusher != nil {
		_, _ = c.Writer.WriteString(": open\n\n")
		flusher.Flush()
	}

	for ev := range ch {
		if ctx.Err() != nil {
			return
		}
		switch ev.Kind {
		case "chunk":
			writeSSE(c.Writer, ev)
		case "error":
			writeSSE(c.Writer, ev)
			if flusher != nil {
				flusher.Flush()
			}
			return
		case "done":
			writeSSE(c.Writer, ev)
			if flusher != nil {
				flusher.Flush()
			}
			// 协议约定:写 [DONE] 终止
			_, _ = c.Writer.WriteString("data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return
		default:
			logger.Warn("ai chat: unknown event kind=%q", ev.Kind)
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// writeSSE 把 StreamEvent 序列化为 SSE 帧。Err 字段已被 StreamEvent 自身序列化,
// 这里直接走 json.Marshal(已经在 struct tag 上有 json tag)。
func writeSSE(w http.ResponseWriter, ev aiengine.StreamEvent) {
	b, err := json.Marshal(ev)
	if err != nil {
		_, _ = fmt.Fprintf(w, "data: {\"kind\":\"error\",\"err\":\"marshal failed: %s\"}\n\n", err.Error())
		return
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(b))
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:    "/api/skillbox/ai/chat",
		Handler: ChatStream,
		// 必须 POST 才能带 body;不能用 HttpGet 也不走 BindParamsHandler,因为要手动写 SSE
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.chat",
		Swagger: &ginp.SwaggerInfo{
			Title:       "ai.chat",
			Description: "SSE 流式对话;event-stream 协议,data: {json}\\n\\n,结束 data: [DONE]\\n\\n",
		},
	})
}
