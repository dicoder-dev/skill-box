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
//
// Messages 兼容两种格式(都是 Message 的合法子集):
//   1) 标准: { role, parts:[{text:...}] }           ← 喂给 ModelProvider
//   2) 简化: { role, content: "..." }                ← 前端 chatStream 走这个
//
// 2026-07-14 加:实测前端发的是简化格式(只填 content 不填 parts)。
// *blades.Message 直接 JSON 反序列化会把 content 字段丢掉(没有 json tag),
// 导致传入 Anthropic SDK 后 messages[0].content 为空,
// 国产 anthropic 兼容端点(MiniMax 等)校验失败,返 400 "input json is empty"。
// 修复:用中间结构 ChatMessage 接收入参,再手工转 *blades.Message。
type RequestChat struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Messages []ChatMessage      `json:"messages"`
	PresetID string            `json:"preset_id"`
	Vars     map[string]string `json:"vars"`
}

// ChatMessage 中间结构,兼容"标准"和"简化"两种入参。
type ChatMessage struct {
	Role    string          `json:"role"`
	Content string          `json:"content"`
	Parts   []ChatMessagePart `json:"parts,omitempty"`
}

// ChatMessagePart parts 的简化接收(只关心 text 字段)。
type ChatMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// toBladesMessages 把入参统一转成 *blades.Message。
// 优先用 parts(标准格式),否则用 content(简化格式),都没有就给个空 TextPart 兜底。
func toBladesMessages(in []ChatMessage) []*blades.Message {
	out := make([]*blades.Message, 0, len(in))
	for _, m := range in {
		role := m.Role
		if role == "" {
			role = string(blades.RoleUser)
		}
		var parts []blades.Part
		if len(m.Parts) > 0 {
			for _, p := range m.Parts {
				parts = append(parts, blades.TextPart{Text: p.Text})
			}
		} else if m.Content != "" {
			parts = []blades.Part{blades.TextPart{Text: m.Content}}
		} else {
			parts = []blades.Part{blades.TextPart{Text: ""}}
		}
		msg := &blades.Message{
			Role:  blades.Role(role),
			Parts: parts,
		}
		out = append(out, msg)
	}
	return out
}

// ChatStream POST /api/skillbox/ai/chat(SSE)
func ChatStream(c *gin.Context) {
	var req RequestChat
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数有误: " + err.Error()})
		return
	}

	// 2026-07-14 增:护栏 system。
	// 防止 AI 主动读取 .skill-box/ 目录下的运行时数据(chat history 等)并泄露给用户。
	// 仅在用户没显式带 system role 且没用 preset 时追加,不覆盖用户配置;
	// preset 路径走 ChatWithPreset,内部已经过 buildSkillMDForPrompt 过滤 hidden,
	// 这里不用重复加。
	guardSkillBoxSystem(&req.Messages)

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
		chat, err = svc.Chat(ctx, toBladesMessages(req.Messages), req.Provider)
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

// skillboxSystemGuard 追加到 messages 顶部的 system prompt。
// 告诉 AI 不要读取 .skill-box/ 目录(2026-07-14 增)。
//
// 用字符串拼接而非 raw string,避免文本里的目录名需要转义反引号。
const skillboxSystemGuard = "You MUST NOT read, quote, summarize, or otherwise reference " +
	"files under any '.skill-box/' directory. '.skill-box/' is Skill Box " +
	"runtime data (local chat history, caches, etc.) that is NOT part of " +
	"the skill's instruction. Treat it as opaque to you. If the user asks " +
	"about it, decline politely."

// guardSkillBoxSystem 仅在 user 没显式带 system role 时,把 skillboxSystemGuard
// 追加到 messages 前面。保留用户配置优先。
func guardSkillBoxSystem(messages *[]ChatMessage) {
	if messages == nil {
		return
	}
	for _, m := range *messages {
		if m.Role == "system" {
			return
		}
	}
	*messages = append([]ChatMessage{{Role: "system", Content: skillboxSystemGuard}}, *messages...)
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
