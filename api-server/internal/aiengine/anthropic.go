package aiengine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AnthropicProvider 走 Anthropic Messages streaming API。
// 协议区别于 OpenAI:system 单独字段、messages 只含 user/assistant、event 格式不同。
type AnthropicProvider struct {
	defaultBase string
}

func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{defaultBase: "https://api.anthropic.com"}
}

func (p *AnthropicProvider) Kind() string { return KindAnthropic }

type anthropicReq struct {
	Model       string             `json:"model"`
	Messages    []anthropicMsg     `json:"messages"`
	System      string             `json:"system,omitempty"`
	Stream      bool               `json:"stream"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float32           `json:"temperature,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic SSE 事件类型(我们只关心 content_block_delta 与 message_delta / message_stop)。
type anthropicEvent struct {
	Type  string          `json:"type"`
	Delta *anthropicDelta `json:"delta,omitempty"`
	Usage *Usage          `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
	StopReason *string `json:"stop_reason,omitempty"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest, apiKey string, out chan<- StreamEvent) error {
	defer close(out)
	if apiKey == "" {
		writeErr(out, errors.New("anthropic: empty api key"))
		return nil
	}
	// 拆 system 与 messages(Anthropic 要求 system 独立,只剩 user/assistant)
	var systemParts []string
	var msgs []anthropicMsg
	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			systemParts = append(systemParts, m.Content)
		case RoleUser, RoleAssistant:
			msgs = append(msgs, anthropicMsg{Role: string(m.Role), Content: m.Content})
		}
	}
	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	body := anthropicReq{
		Model:       nonEmpty(req.Model, "claude-3-5-sonnet-20241022"),
		Messages:    msgs,
		System:      strings.Join(systemParts, "\n\n"),
		Stream:      true,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		writeErr(out, err)
		return nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.defaultBase+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		writeErr(out, err)
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(out, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		writeErr(out, fmt.Errorf("anthropic: http %d: %s", resp.StatusCode, string(b)))
		return nil
	}

	reader := bufio.NewReader(resp.Body)
	var dataBuf strings.Builder
	var finalUsage *Usage
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				writeErr(out, err)
			}
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			raw := dataBuf.String()
			dataBuf.Reset()
			if raw == "" {
				continue
			}
			var ev anthropicEvent
			if jerr := json.Unmarshal([]byte(raw), &ev); jerr != nil {
				continue
			}
			switch ev.Type {
			case "content_block_delta":
				if ev.Delta != nil && ev.Delta.Text != "" {
					out <- StreamEvent{Kind: "chunk", Text: ev.Delta.Text}
				}
			case "message_delta":
				if ev.Usage != nil {
					finalUsage = ev.Usage
				}
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataBuf.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	out <- StreamEvent{Kind: "done", Usage: finalUsage}
	return nil
}
