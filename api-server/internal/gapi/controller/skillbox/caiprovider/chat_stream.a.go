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
//
// 2026-07-14 改造:aiengine 全量切到 go-kratos/blades。
// controller 不再关心 aiengine 内部实现,只把 *blades.ModelResponse 转 SSE。
package caiprovider

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kratos/blades"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestChat 流式对话入参。
// Provider 留空时由 engine 按 priority 选默认;PresetID + Vars 触发预设。
// Messages 直接是 blades 原生格式:每条 { role, parts:[{text:...}] }。
type RequestChat struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Messages []*blades.Message  `json:"messages"`
	PresetID string            `json:"preset_id"`
	Vars     map[string]string `json:"vars"`
}

// ChatStream POST /api/skillbox/ai/chat(SSE)
func ChatStream(c *gin.Context) {
	var req RequestChat
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}

	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	eng := sai.NewEngine(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, eng)

	ctx := c.Request.Context()
	var (
		chat *sai.ChatResult
		err  error
	)
	if req.PresetID != "" {
		chat, err = svc.ChatWithPreset(ctx, req.PresetID, req.Provider, req.Vars)
	} else {
		chat, err = svc.Chat(ctx, req.Messages, req.Provider)
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

	for resp, err := range chat.Stream {
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			writeSSEError(c.Writer, err)
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		if resp == nil || resp.Message == nil {
			continue
		}
		// 增量文本
		if text := resp.Message.Text(); text != "" {
			writeSSEChunk(c.Writer, text)
		}
		// 最终帧:status == completed 表示流自然结束
		if resp.Message.Status == blades.StatusCompleted {
			writeSSEDone(c.Writer, resp.Message.TokenUsage)
			if flusher != nil {
				flusher.Flush()
			}
			_, _ = c.Writer.WriteString("data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
	// provider 自己把 generator 关闭,没遇到 StatusCompleted 兜底写 done
	writeSSEDone(c.Writer, blades.TokenUsage{})
	_, _ = c.Writer.WriteString("data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

// writeSSEChunk 写一条 chunk 帧。旧 aiengine 的 SSE 协议约定
// { kind:"chunk", text:"..." },前端 chatStream 解析按这个来,所以我们保留。
func writeSSEChunk(w http.ResponseWriter, text string) {
	b, _ := json.Marshal(map[string]any{"kind": "chunk", "text": text})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(b))
}

// writeSSEDone 写 done 帧(带 token usage)。
func writeSSEDone(w http.ResponseWriter, usage blades.TokenUsage) {
	b, _ := json.Marshal(map[string]any{
		"kind":  "done",
		"usage": usage,
	})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(b))
}

// writeSSEError 写 error 帧。
func writeSSEError(w http.ResponseWriter, err error) {
	b, _ := json.Marshal(map[string]any{"kind": "error", "err": err.Error()})
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(b))
	logger.Warn("ai chat: stream error: %v", err)
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
