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

// OpenAIProvider 走 OpenAI Chat Completions streaming API。
// 同时支持 KindOpenAI(官方)与 KindOpenAICom(DeepSeek / 硅基 / 月之暗面 等)。
// base URL 在 Chat 时由调用方注入(ai_providers.BaseURL),这里只负责协议。
type OpenAIProvider struct {
	kind        string // KindOpenAI 或 KindOpenAICom
	defaultBase string
}

func NewOpenAIProvider(kind string) *OpenAIProvider {
	switch kind {
	case KindOpenAI:
		return &OpenAIProvider{kind: kind, defaultBase: "https://api.openai.com/v1"}
	case KindOpenAICom:
		return &OpenAIProvider{kind: kind, defaultBase: "https://api.deepseek.com/v1"}
	default:
		return &OpenAIProvider{kind: kind, defaultBase: "https://api.openai.com/v1"}
	}
}

// WithBaseURL 覆盖 base URL(主要给测试用,生产由 ai_providers.BaseURL 决定)。
func (p *OpenAIProvider) WithBaseURL(s string) *OpenAIProvider {
	if s != "" {
		p.defaultBase = s
	}
	return p
}

func (p *OpenAIProvider) Kind() string { return p.kind }

// ChatRequest 体(对齐 OpenAI)。
type openAIReq struct {
	Model       string         `json:"model"`
	Messages    []openAIMsg    `json:"messages"`
	Stream      bool           `json:"stream"`
	StreamOpts  *openAIStream  `json:"stream_options,omitempty"`
	Temperature *float32       `json:"temperature,omitempty"`
	MaxTokens   *int           `json:"max_tokens,omitempty"`
}

type openAIStream struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest, apiKey string, out chan<- StreamEvent) error {
	defer close(out)
	if apiKey == "" {
		writeErr(out, errors.New("openai: empty api key"))
		return nil
	}
	body := openAIReq{
		Model:       nonEmpty(req.Model, "gpt-4o-mini"),
		Stream:      true,
		StreamOpts:  &openAIStream{IncludeUsage: true},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, m := range req.Messages {
		body.Messages = append(body.Messages, openAIMsg{Role: string(m.Role), Content: m.Content})
	}
	payload, err := json.Marshal(body)
	if err != nil {
		writeErr(out, err)
		return nil
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(p.defaultBase, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		writeErr(out, err)
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
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
		writeErr(out, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(b)))
		return nil
	}

	reader := bufio.NewReader(resp.Body)
	var dataLine strings.Builder
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
			raw := dataLine.String()
			dataLine.Reset()
			if raw == "" || raw == "[DONE]" {
				continue
			}
			var chunk openAIChunk
			if jerr := json.Unmarshal([]byte(raw), &chunk); jerr != nil {
				continue
			}
			for _, c := range chunk.Choices {
				if c.Delta.Content != "" {
					out <- StreamEvent{Kind: "chunk", Text: c.Delta.Content}
				}
			}
			if chunk.Usage != nil {
				finalUsage = chunk.Usage
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLine.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	out <- StreamEvent{Kind: "done", Usage: finalUsage}
	return nil
}

func writeErr(out chan<- StreamEvent, err error) {
	out <- StreamEvent{Kind: "error", Err: err.Error()}
}

func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
