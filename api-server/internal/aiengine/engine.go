// Package aiengine 提供多 LLM provider 抽象 + 流式对话能力。
//
// 2026-07-14 改造:从自研 openai.go / anthropic.go 切到 go-kratos/blades。
//   - 上层(controller / service / skilltester)直接用 blades 原生 *Message / ModelProvider
//   - Engine 把 ai_providers 表 + settings KV 拼成 blades.ModelProvider + apiKey
//   - Preset 走 blades.Prompt 占位符替换
//   - Agent / Chain 留给 example/chain.go 演示,业务侧按需启用
package aiengine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-kratos/blades"
	"ginp-api/internal/gapi/entity"
)

// ErrNoProvider 没有可用 provider。
var ErrNoProvider = errors.New("aiengine: no enabled provider")

// ErrUnknownKind 未知 provider kind。
var ErrUnknownKind = errors.New("aiengine: unknown provider kind")

// Kind provider 类型。
const (
	KindOpenAI    = "openai"        // OpenAI 官方
	KindAnthropic = "anthropic"     // Anthropic 官方
	KindOpenAICom = "openai_compat" // OpenAI 协议兼容(DeepSeek / 硅基 / 月之暗面等)
)

// AllKinds v1 支持的全部 kind。
var AllKinds = []string{KindOpenAI, KindAnthropic, KindOpenAICom}

// Config Engine 用到的"已解析"配置,不含 API key(API key 走 SecretStore 现场取)。
//
// 保留 Config 类型是因为 skilltester 的 buildForAI 闭包已经在用它,
// 改造后改成 ModelProvider 暴露面更窄,但为了让 skilltester.Providers
// 这条数据流不重写,Config 仍然充当"传输载体"。
type Config struct {
	Name    string
	Kind    string
	BaseURL string
	Model   string
}

// SecretStore API key 拿取抽象。v1 由 sai 用 settings 实现;P1 可换 OS keychain。
type SecretStore interface {
	// Resolve 拿 provider_name 对应的 api key;空 = 没有配置
	Resolve(providerName string) (string, error)
}

// ModelFactory 把 Config 变成可用的 blades.ModelProvider。允许后续注册自定义 kind。
type ModelFactory func(cfg Config, apiKey string) (blades.ModelProvider, error)

// Engine 选 provider + 拼凭据 + 出 blades.ModelProvider。无状态,共享。
type Engine struct {
	factories map[string]ModelFactory
	secrets   SecretStore
}

func NewEngine(secrets SecretStore) *Engine {
	e := &Engine{
		factories: map[string]ModelFactory{},
		secrets:   secrets,
	}
	e.registerDefaults()
	return e
}

// Register 注册自定义 factory(供第三方 / 单测 mock)。
func (e *Engine) Register(kind string, f ModelFactory) {
	e.factories[strings.ToLower(kind)] = f
}

// Select 从候选里选一个。name 非空时按 name 精确匹配;否则按 priority 升序、name 字典序。
func (e *Engine) Select(providers []*entity.AIProvider, name string) (*entity.AIProvider, error) {
	var candidates []*entity.AIProvider
	for _, p := range providers {
		if p.Enabled {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return nil, ErrNoProvider
	}
	if name != "" {
		for _, p := range candidates {
			if p.Name == name {
				return p, nil
			}
		}
		return nil, fmt.Errorf("aiengine: provider %q not found or disabled", name)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		return candidates[i].Name < candidates[j].Name
	})
	return candidates[0], nil
}

// Build 把选中的 row 转成 blades.ModelProvider + api key。
func (e *Engine) Build(p *entity.AIProvider) (blades.ModelProvider, string, error) {
	f, ok := e.factories[strings.ToLower(p.Kind)]
	if !ok {
		return nil, "", fmt.Errorf("%w: %q", ErrUnknownKind, p.Kind)
	}
	apiKey, err := e.secrets.Resolve(p.Name)
	if err != nil {
		return nil, "", fmt.Errorf("aiengine: resolve key for %s: %w", p.Name, err)
	}
	prov, err := f(Config{Name: p.Name, Kind: p.Kind, BaseURL: p.BaseURL, Model: p.Model}, apiKey)
	if err != nil {
		return nil, "", fmt.Errorf("aiengine: build provider %s: %w", p.Name, err)
	}
	return prov, apiKey, nil
}

// BuildFromConfig 接受 Config 构造 ModelProvider(用于 service 层没有 entity 行的场景)。
// 不解析 api key,由 caller 自行管理。
func (e *Engine) BuildFromConfig(cfg Config, apiKey string) (blades.ModelProvider, error) {
	f, ok := e.factories[strings.ToLower(cfg.Kind)]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, cfg.Kind)
	}
	return f(cfg, apiKey)
}

// validKind 校验 kind 是否在支持清单。
func validKind(k string) bool {
	for _, v := range AllKinds {
		if v == k {
			return true
		}
	}
	return false
}
