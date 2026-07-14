package aiengine

import (
	"github.com/go-kratos/blades"
	"github.com/go-kratos/blades/contrib/anthropic"
	"github.com/go-kratos/blades/contrib/openai"
)

// registerDefaults 把 KindOpenAI / KindAnthropic / KindOpenAICom 三个内置 kind
// 注册到 blades 的对应工厂上。
//
// 设计说明:
//   - KindOpenAI / KindOpenAICom 都走 contrib/openai.NewModel;
//     OpenAI-Compat 协议相同(DeepSeek / 硅基 / 月之暗面 / 阿里 DashScope),
//     区别只在 BaseURL + 可选非默认 Model
//   - KindAnthropic 走 contrib/anthropic.NewModel
//   - contrib/openai 内部用官方 openai-go SDK,contrib/anthropic 用 anthropic-sdk-go,
//     协议细节、SSE 解析、token usage 全由 SDK 处理
func (e *Engine) registerDefaults() {
	e.Register(KindOpenAI, func(cfg Config, apiKey string) (blades.ModelProvider, error) {
		return openai.NewModel(nonEmpty(cfg.Model, "gpt-4o-mini"), openai.Config{
			BaseURL: cfg.BaseURL,
			APIKey:  apiKey,
		}), nil
	})
	e.Register(KindOpenAICom, func(cfg Config, apiKey string) (blades.ModelProvider, error) {
		return openai.NewModel(nonEmpty(cfg.Model, "deepseek-chat"), openai.Config{
			BaseURL: cfg.BaseURL,
			APIKey:  apiKey,
		}), nil
	})
	e.Register(KindAnthropic, func(cfg Config, apiKey string) (blades.ModelProvider, error) {
		return anthropic.NewModel(nonEmpty(cfg.Model, "claude-3-5-sonnet-20241022"), anthropic.Config{
			BaseURL: cfg.BaseURL,
			APIKey:  apiKey,
		}), nil
	})
}

// nonEmpty 空串返回 def,否则返回 s(本地小工具,避免再 import strings)。
func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
