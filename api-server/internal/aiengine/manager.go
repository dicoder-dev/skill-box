package aiengine

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"ginp-api/internal/gapi/entity"
)

// ErrNoProvider 没有可用 provider。
var ErrNoProvider = errors.New("aiengine: no enabled provider")

// Config Manager 用到的"已解析"配置,不含 API key(API key 走 SecretStore 现场取)。
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

// Factory 把 Config 变成可用的 Provider 实例(允许后续注册自定义 kind)。
type Factory func(cfg Config) Provider

// Manager 选 provider + 拼凭据;无状态,共享。
type Manager struct {
	factories map[string]Factory
	secrets   SecretStore
}

func NewManager(secrets SecretStore) *Manager {
	m := &Manager{
		factories: map[string]Factory{},
		secrets:   secrets,
	}
	// 注册内置 kind
	m.Register(KindOpenAI, func(cfg Config) Provider {
		p := NewOpenAIProvider(KindOpenAI)
		if cfg.BaseURL != "" {
			p.defaultBase = cfg.BaseURL
		}
		return p
	})
	m.Register(KindOpenAICom, func(cfg Config) Provider {
		p := NewOpenAIProvider(KindOpenAICom)
		if cfg.BaseURL != "" {
			p.defaultBase = cfg.BaseURL
		}
		return p
	})
	m.Register(KindAnthropic, func(cfg Config) Provider {
		p := NewAnthropicProvider()
		if cfg.BaseURL != "" {
			p.defaultBase = cfg.BaseURL
		}
		return p
	})
	return m
}

// Register 注册自定义 factory(供第三方 / 单测 mock)。
func (m *Manager) Register(kind string, f Factory) {
	m.factories[kind] = f
}

// Select 从候选里选一个。name 非空时按 name 精确匹配;否则按 priority 升序、name 字典序。
func (m *Manager) Select(providers []*entity.AIProvider, name string) (*entity.AIProvider, error) {
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

// BuildFromConfig 接受 Config 构造 Provider(用于 service 层没有 entity 行的场景)。
// 不解析 api key,由 caller 自行管理。
func (m *Manager) BuildFromConfig(cfg Config) (Provider, error) {
	f, ok := m.factories[strings.ToLower(cfg.Kind)]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, cfg.Kind)
	}
	return f(cfg), nil
}

// Build 把选中的 row 转成 Provider + api key。
func (m *Manager) Build(p *entity.AIProvider) (Provider, string, error) {
	f, ok := m.factories[strings.ToLower(p.Kind)]
	if !ok {
		return nil, "", fmt.Errorf("%w: %q", ErrUnknownKind, p.Kind)
	}
	prov := f(Config{Name: p.Name, Kind: p.Kind, BaseURL: p.BaseURL, Model: p.Model})
	apiKey, err := m.secrets.Resolve(p.Name)
	if err != nil {
		return nil, "", fmt.Errorf("aiengine: resolve key for %s: %w", p.Name, err)
	}
	return prov, apiKey, nil
}
