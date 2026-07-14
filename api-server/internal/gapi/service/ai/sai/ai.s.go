// Package sai 提供 AI 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 7.3 节):
//   - AIProvider 表只放元数据(name / kind / model / base_url / priority / enabled)
//   - 真实 API key 放 settings KV,key 约定 "ai:<provider_name>:api_key"(v1 明文,P1 换 keychain)
//   - Chat 直接走 blades 原生 *Message / ModelProvider,controller 拿到 Generator
//   - Preset 渲染走 aiengine.RenderPreset(用 blades.Prompt 占位符)
package sai

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/blades"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/entity"
	maiprovider "ginp-api/internal/gapi/model/skillbox/maiprovider"
	"ginp-api/internal/settings"
	"ginp-api/internal/skillboxdata"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/logger"
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
	dbWrite  *gorm.DB
	dbRead   *gorm.DB
	settings *settings.Service
	engine   *aiengine.Engine
}

func New(dbWrite, dbRead *gorm.DB, st *settings.Service, engine *aiengine.Engine) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, settings: st, engine: engine}
}

// 业务层用的 settings 实现 aiengine.SecretStore。
type secretAdapter struct{ s *settings.Service }

func (a *secretAdapter) Resolve(providerName string) (string, error) {
	v, _, err := a.s.Get(apiKeyPrefix + providerName + ":api_key")
	return v, err
}

// NewEngine 工厂方法:用 settings 构造 SecretStore 后包出 Engine。
func NewEngine(st *settings.Service) *aiengine.Engine {
	return aiengine.NewEngine(&secretAdapter{s: st})
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

// ChatResult 流式 chat 一次的结果(model + 选中的 provider 元数据)。
// controller 拿到这个后,只需把 Generator 转 SSE 推给前端。
type ChatResult struct {
	Model     blades.ModelProvider
	Provider  string // 选中的 provider 名字(显示用)
	ModelName string
	Stream    iter.Seq2[*blades.ModelResponse, error]
}

// Chat 选 provider + 启动流。返回 model + 流,controller 透传给 SSE。
// providerName 留空 = 由 Engine 按 priority 选。
func (s *Service) Chat(ctx context.Context, messages []*blades.Message, providerName string) (*ChatResult, error) {
	rows, _, err := s.model().FindList(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ai: list providers: %w", err)
	}
	row, err := s.engine.Select(rows, providerName)
	if err != nil {
		return nil, err
	}
	model, _, err := s.engine.Build(row)
	if err != nil {
		return nil, err
	}
	req := &blades.ModelRequest{Messages: messages}
	return &ChatResult{
		Model:     model,
		Provider:  row.Name,
		ModelName: row.Model,
		Stream:    model.NewStreaming(ctx, req),
	}, nil
}

// ChatWithPreset:preset + 变量一次性合成。
func (s *Service) ChatWithPreset(ctx context.Context, presetID, providerName string, vars map[string]string) (*ChatResult, error) {
	preset, ok := findPreset(presetID)
	if !ok {
		return nil, fmt.Errorf("ai: unknown preset %q", presetID)
	}
	return s.Chat(ctx, aiengine.RenderPreset(preset, vars), providerName)
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

// TestParams 单次测试请求体(可以指 provider_id 用已存的,或直接传裸 kind/base_url/model/api_key)。
//
// 设计原因:
//   - 设置界面:用户在表单里改完想"先试一下能不能通"再保存 —— api key 此时还没落盘
//   - 列表页:用户想验证已存 provider 的 key 没失效
// 两类用法合并为一个 controller,避免 controller 数量膨胀。
type TestParams struct {
	// ProviderID 非空 = 用 ai_providers 表里的元数据(name/kind/base_url/model)+ settings 里的 api_key
	ProviderID uint
	// 或者以下裸参数(优先于 ProviderID),方便"还没保存就试一下"
	Name    string
	Kind    string
	BaseURL string
	Model   string
	APIKey  string
}

// TestResult 测试结果。
type TestResult struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`          // 失败原因 或 成功说明
	Sample    string `json:"sample,omitempty"` // 成功时截取前 80 字片段作为回执
	LatencyMS int64  `json:"latency_ms"`       // 端到端耗时(不含本次本身)
}

// TestConnection 探测 provider 是否真的能跑通。
//
// 行为:
//   - kind 不合法 / API key 空 → 立即返 ok=false + 清晰错误
//   - 用最小 prompt 探测,只要拿到任意 chunk 就视为"成功"
//   - 30s 兜底超时,防止 provider 卡死拖死 controller
//   - 拿到 provider 真实错误原文(状态码 + body)原样回传,设置界面直接展示
func (s *Service) TestConnection(p TestParams) (*TestResult, error) {
	// 1) 解析最终参数:ProviderID 优先拉库里的,否则用裸参数
	cfg := aiengine.Config{}
	apiKey := p.APIKey
	if p.ProviderID != 0 {
		row, err := s.model().FindOneById(p.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("ai: provider %d not found: %w", p.ProviderID, ErrNotFound)
		}
		cfg.Kind = strings.ToLower(strings.TrimSpace(row.Kind))
		cfg.BaseURL = row.BaseURL
		cfg.Model = row.Model
		if apiKey == "" {
			v, _, gerr := s.settings.Get(apiKeyPrefix + row.Name + ":api_key")
			if gerr == nil {
				apiKey = v
			}
		}
	}
	if p.Kind != "" {
		cfg.Kind = strings.ToLower(strings.TrimSpace(p.Kind))
	}
	if p.BaseURL != "" {
		cfg.BaseURL = p.BaseURL
	}
	if p.Model != "" {
		cfg.Model = p.Model
	}

	// 2) 基础校验
	if !validKind(cfg.Kind) {
		return &TestResult{OK: false, Message: fmt.Sprintf("未知的 provider kind: %q(仅支持 %v)", cfg.Kind, aiengine.AllKinds)}, nil
	}
	if strings.TrimSpace(apiKey) == "" {
		return &TestResult{OK: false, Message: "API Key 为空,请先在表单填写或保存后再试"}, nil
	}
	if cfg.Model == "" {
		cfg.Model = defaultTestModel(cfg.Kind)
	}

	// 3) 构造 model
	model, err := s.engine.BuildFromConfig(cfg, strings.TrimSpace(apiKey))
	if err != nil {
		return &TestResult{OK: false, Message: err.Error()}, nil
	}

	// 4) 探测:发一条极短 user prompt,只关心能不能拿到任意 chunk
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req := &blades.ModelRequest{
		Messages: []*blades.Message{blades.UserMessage("hi")},
	}
	stream := model.NewStreaming(ctx, req)
	start := time.Now()

	var (
		sample strings.Builder
		gotAny bool
		gotErr string
	)
	for resp, err := range stream {
		if err != nil {
			gotErr = err.Error()
			break
		}
		if resp == nil {
			continue
		}
		if text := resp.Message.Text(); text != "" {
			gotAny = true
			if sample.Len() < 80 {
				sample.WriteString(text)
			}
		}
	}
	latency := time.Since(start).Milliseconds()
	if ctx.Err() != nil {
		return &TestResult{OK: false, Message: fmt.Sprintf("30s 超时未回应: %v", ctx.Err()), LatencyMS: latency}, nil
	}
	if gotErr != "" {
		return &TestResult{OK: false, Message: gotErr, LatencyMS: latency}, nil
	}
	if gotAny {
		return &TestResult{OK: true, Message: "测试成功", Sample: truncate(sample.String(), 80), LatencyMS: latency}, nil
	}
	return &TestResult{OK: false, Message: "provider 已关闭 stream,但没有任何事件(dial 成功但未给出内容)", LatencyMS: latency}, nil
}

// defaultTestModel 给没填 model 的探测兜底(各 kind 用主流默认)。
func defaultTestModel(kind string) string {
	switch kind {
	case aiengine.KindAnthropic:
		return "claude-3-5-sonnet-20241022"
	case aiengine.KindOpenAICom:
		return "deepseek-chat"
	default:
		return "gpt-4o-mini"
	}
}

// truncate 简单按字节截(不算严谨,展示给用户看足够)。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// =====================================================================
// AI 历史对话持久化(2026-07-14 增)。
//
// 把每个 skill 的 AI 对话历史保存到 "<source_path>/.skill-box/history.json",
// 既给前端"历史对话"Modal 一个跨设备/跨刷新持久化通道,也是 user-mode /
// link-mode 共享同一份源端。
//
// 安全性:source_path 必须位于 skillstore.root 之下 + 包含 SKILL.md,避免
// 任意磁盘路径被写入。
// =====================================================================

// ErrEmptySourcePath source_path 缺失或非法(2026-07-14 增)。
var ErrEmptySourcePath = errors.New("ai.history: source_path is required")

// ErrSourcePathNotInStore source_path 不在 skillstore.root 之下,或不含 SKILL.md(2026-07-14 增)。
var ErrSourcePathNotInStore = errors.New("ai.history: source_path is not a skill in this store")

// resolveSkillDirBySourcePath 把前端传的 disk 绝对 source_path 解析为 skill 目录,
// 并校验它"在 store.root 之下 + 是合法 skill"。
//
// source_path 前端来源:`node.skill_meta?.source_path` 是 skillstore 通过
// EvalSymlinks 后的绝对目录(可能含 symlink)。这里也再 EvalSymlinks 一次,处理
// source_path 处于 symlink 链上的情况。
func (s *Service) resolveSkillDirBySourcePath(sourcePath string) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", ErrEmptySourcePath
	}
	// 解析 symlink 真实路径(source_path 可能被外层再 symlink 包了一层)。
	real, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		// 2026-07-14 增:eval 失败不静默,留给 caller 弹日志(诊断使用率高的字段)
		real = sourcePath
	}
	store, err := skillstore.New()
	if err != nil {
		return "", fmt.Errorf("ai.history: open store: %w", err)
	}
	rootReal, err := filepath.EvalSymlinks(store.Root())
	if err != nil {
		rootReal = store.Root()
	}
	// 必须在 root 之下(real 路径前缀匹配,避免 prefix-bug)
	if real != rootReal && !strings.HasPrefix(real, rootReal+string(filepath.Separator)) {
		// 2026-07-14 增:返错前打日志,便于前端排错
		logger.Warn("ai.history: source_path not in store: source=%q real=%q root=%q", sourcePath, real, rootReal)
		return "", ErrSourcePathNotInStore
	}
	// 必须含 SKILL.md
	if _, err := os.Stat(filepath.Join(real, "SKILL.md")); err != nil {
		logger.Warn("ai.history: SKILL.md missing: source=%q real=%q err=%v", sourcePath, real, err)
		return "", ErrSourcePathNotInStore
	}
	return real, nil
}

// SaveHistory 把 items 写入 source_path/.skill-box/history.json。
// items 由前端 POST 过来,每条含 id/title/preview/ts/provider/model/messages。
//
// 空 items 视为"清空本地历史",仍会写一份空文件(便于前端显示)。
func (s *Service) SaveHistory(sourcePath string, items []skillboxdata.HistoryItem) error {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return err
	}
	if err := skillboxdata.Ensure(dir); err != nil {
		return err
	}
	return skillboxdata.WriteHistory(dir, items)
}

// ListHistory 读 source_path/.skill-box/history.json;
// 不存在返空 History(由 ReadHistory 兜底),不算 error。
func (s *Service) ListHistory(sourcePath string) (*skillboxdata.History, error) {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return nil, err
	}
	return skillboxdata.ReadHistory(dir)
}

// =====================================================================
// v2 单 conv 文件 API(2026-07-14 增,替代 v1 单文件 history.json 粒度)
//
// 设计:一个对话 = 一份 .skill-box/history/<conv-id>.json。
// 列表只返 metadata,按需 GET 单条 messages;
// SAVE 单条 upsert;DELETE 按 id 删。
// =====================================================================

// ListConvs metadata-only 列表;目录不存在返空(nil);source_path 非法走 v1 错误流(2026-07-14 增)。
func (s *Service) ListConvs(sourcePath string) ([]skillboxdata.ConvMeta, error) {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return nil, err
	}
	return skillboxdata.ListConvs(dir)
}

// GetConv 按 conv_id 拉单条完整;不存在返 (nil, nil),让上层判 404(2026-07-14 增)。
func (s *Service) GetConv(sourcePath, convID string) (*skillboxdata.HistoryItem, error) {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return nil, err
	}
	return skillboxdata.ReadConv(dir, convID)
}

// SaveConv 单条 upsert;source_path 非法 / conv_id 不合法 / 文件超 2MB
// 分别返 ErrEmptySourcePath / ErrSourcePathNotInStore / ErrInvalidConvID / ErrConvTooLarge(2026-07-14 增)。
func (s *Service) SaveConv(sourcePath string, item skillboxdata.HistoryItem) error {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return err
	}
	return skillboxdata.WriteConv(dir, item)
}

// DeleteConv 按 id 删单条;不存在幂等返 nil(2026-07-14 增)。
func (s *Service) DeleteConv(sourcePath, convID string) error {
	dir, err := s.resolveSkillDirBySourcePath(sourcePath)
	if err != nil {
		return err
	}
	return skillboxdata.DeleteConv(dir, convID)
}
