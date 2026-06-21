// Package sai 提供 AI 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 7.3 节):
//   - AIProvider 表只放元数据(name / kind / model / base_url / priority / enabled)
//   - 真实 API key 放 settings KV,key 约定 "ai:<provider_name>:api_key"(v1 明文,P1 换 keychain)
//   - Chat 不落库,只把流式事件转发给 controller 的 SSE
//   - Preset 渲染 + 选 provider 一并做,controller 只传"我要用哪个 preset + 替换变量"
package sai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/entity"
	maiprovider "ginp-api/internal/gapi/model/skillbox/maiprovider"
	"ginp-api/internal/settings"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

const apiKeyPrefix = "ai:" // settings key 形如 "ai:openai-prod:api_key"

// 业务错误。
var (
	ErrEmptyName   = errors.New("ai: name is empty")
	ErrEmptyKind   = errors.New("ai: kind is empty")
	ErrUnknownKind = errors.New("ai: unknown kind")
	ErrNotFound    = errors.New("ai: provider not found")
)

// Service 业务服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	settings *settings.Service
	manager  *aiengine.Manager
}

func New(dbWrite, dbRead *gorm.DB, st *settings.Service, mgr *aiengine.Manager) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, settings: st, manager: mgr}
}

// 业务层用的 settings 实现 aiengine.SecretStore。
type secretAdapter struct{ s *settings.Service }

func (a *secretAdapter) Resolve(providerName string) (string, error) {
	v, _, err := a.s.Get(apiKeyPrefix + providerName + ":api_key")
	return v, err
}

// NewManager 工厂方法:用 settings 构造 SecretStore 后包出 Manager。
func NewManager(st *settings.Service) *aiengine.Manager {
	return aiengine.NewManager(&secretAdapter{s: st})
}

func (s *Service) model() *maiprovider.Model {
	return maiprovider.NewModel(s.dbWrite, s.dbRead)
}

// Create 新建一个 provider(name / kind 必填;api key 走 SetKey 单独设置)。
func (s *Service) Create(in *entity.AIProvider) (*entity.AIProvider, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	if in.Name == "" {
		return nil, ErrEmptyName
	}
	if !validKind(in.Kind) {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKind, in.Kind)
	}
	// name 唯一
	if _, err := s.model().FindOne(where.New("name", "=", in.Name).Conditions()); err == nil {
		return nil, fmt.Errorf("ai: name %q already exists", in.Name)
	}
	created, err := s.model().Create(in)
	if err != nil {
		return nil, fmt.Errorf("ai: create: %w", err)
	}
	return created, nil
}

// SetKey 单独设置 api key(写 settings,不进 ai_providers 表)。
func (s *Service) SetKey(name, key string) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmptyName
	}
	return s.settings.Set(apiKeyPrefix+name+":api_key", key)
}

// DeleteKey 删 api key(幂等)。
func (s *Service) DeleteKey(name string) error {
	s.settings.Delete(apiKeyPrefix + name + ":api_key")
	return nil
}

// GetKey 读 api key(测试 / 调试用;前端不应直接调)。
func (s *Service) GetKey(name string) (string, error) {
	v, _, err := s.settings.Get(apiKeyPrefix + name + ":api_key")
	return v, err
}

func (s *Service) Update(id uint, in *entity.AIProvider) (*entity.AIProvider, error) {
	cur, err := s.model().FindOneById(id)
	if err != nil {
		return nil, ErrNotFound
	}
	if in.Kind != "" {
		k := strings.ToLower(strings.TrimSpace(in.Kind))
		if !validKind(k) {
			return nil, fmt.Errorf("%w: %s", ErrUnknownKind, k)
		}
		cur.Kind = k
	}
	if in.BaseURL != "" {
		cur.BaseURL = in.BaseURL
	}
	if in.Model != "" {
		cur.Model = in.Model
	}
	if in.Name != "" && strings.TrimSpace(in.Name) != cur.Name {
		newName := strings.TrimSpace(in.Name)
		if _, err := s.model().FindOne(where.New("name", "=", newName).Conditions()); err == nil {
			return nil, fmt.Errorf("ai: name %q already exists", newName)
		}
		// 改名后把 key 也迁过去
		if oldKey, _, _ := s.settings.Get(apiKeyPrefix + cur.Name + ":api_key"); oldKey != "" {
			_ = s.settings.Set(apiKeyPrefix+newName+":api_key", oldKey)
			_ = s.settings.Delete(apiKeyPrefix + cur.Name + ":api_key")
		}
		cur.Name = newName
	}
	cur.Priority = in.Priority
	cur.Enabled = in.Enabled
	if err := s.model().Update(where.New("id", "=", id).Conditions(), cur); err != nil {
		return nil, fmt.Errorf("ai: update: %w", err)
	}
	return cur, nil
}

func (s *Service) Delete(id uint) error {
	cur, err := s.model().FindOneById(id)
	if err != nil {
		return ErrNotFound
	}
	if err := s.model().DeleteById(id); err != nil {
		return err
	}
	// 顺手清 key
	_ = s.settings.Delete(apiKeyPrefix + cur.Name + ":api_key")
	return nil
}

func (s *Service) GetByID(id uint) (*entity.AIProvider, error) {
	row, err := s.model().FindOneById(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return row, nil
}

func (s *Service) GetByName(name string) (*entity.AIProvider, error) {
	row, err := s.model().FindOne(where.New("name", "=", name).Conditions())
	if err != nil {
		return nil, ErrNotFound
	}
	return row, nil
}

// ListProviders 列全部;含 has_key 标记(用于前端 UI 提示"未配置 API key")。
type ProviderView struct {
	*entity.AIProvider
	HasKey bool `json:"has_key"`
}

func (s *Service) ListProviders() ([]*ProviderView, error) {
	rows, _, err := s.model().FindList(nil, nil)
	if err != nil {
		return nil, err
	}
	views := make([]*ProviderView, 0, len(rows))
	for _, r := range rows {
		v, _, _ := s.settings.Get(apiKeyPrefix + r.Name + ":api_key")
		views = append(views, &ProviderView{AIProvider: r, HasKey: v != ""})
	}
	return views, nil
}

// Presets 暴露给前端(直接复用 aiengine.AllPresets 的快照)。
func (s *Service) Presets() []aiengine.Preset {
	out := make([]aiengine.Preset, len(aiengine.AllPresets))
	copy(out, aiengine.AllPresets)
	return out
}

// Chat 选 provider + 启动流。返回 aiengine.StreamEvent channel,controller 透传给 SSE。
// providerName 留空 = 由 Manager 按 priority 选。
func (s *Service) Chat(ctx context.Context, req aiengine.ChatRequest, providerName string) (<-chan aiengine.StreamEvent, error) {
	rows, _, err := s.model().FindList(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ai: list providers: %w", err)
	}
	row, err := s.manager.Select(rows, providerName)
	if err != nil {
		return nil, err
	}
	prov, key, err := s.manager.Build(row)
	if err != nil {
		return nil, err
	}
	if req.Model == "" {
		req.Model = row.Model
	}
	out := make(chan aiengine.StreamEvent, 32)
	go func() {
		_ = prov.Chat(ctx, req, key, out)
	}()
	return out, nil
}

// ChatWithPreset:preset + 变量一次性合成。
func (s *Service) ChatWithPreset(ctx context.Context, presetID, providerName string, vars map[string]string) (<-chan aiengine.StreamEvent, error) {
	preset, ok := findPreset(presetID)
	if !ok {
		return nil, fmt.Errorf("ai: unknown preset %q", presetID)
	}
	req := aiengine.ChatRequest{Messages: aiengine.RenderPreset(preset, vars)}
	return s.Chat(ctx, req, providerName)
}

func findPreset(id string) (aiengine.Preset, bool) {
	for _, p := range aiengine.AllPresets {
		if p.ID == id {
			return p, true
		}
	}
	return aiengine.Preset{}, false
}

func validKind(k string) bool {
	for _, v := range aiengine.AllKinds {
		if v == k {
			return true
		}
	}
	return false
}
