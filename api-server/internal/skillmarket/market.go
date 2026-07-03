package skillmarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/skilladapter"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	"ginp-api/pkg/where"
)

// SourceConfigJSON 允许 source config_json 携带的私有字段(当前只支持 base_url)。
type SourceConfigJSON struct {
	BaseURL string `json:"base_url,omitempty"`
}

// Orchestrator 把 DB(model) + adapter 编排起来:刷新 / 下载。
type Orchestrator struct {
	mu          sync.Mutex
	refreshing  map[uint]bool
	sourceModel *mmarketsource.Model
	skillModel  *mmarketskill.Model
	registry    *Registry
}

// NewOrchestrator 构造 orchestrator。
func NewOrchestrator(sourceModel *mmarketsource.Model, skillModel *mmarketskill.Model) *Orchestrator {
	return &Orchestrator{
		refreshing:  make(map[uint]bool),
		sourceModel: sourceModel,
		skillModel:  skillModel,
		registry:    defaultRegistry,
	}
}

// WithRegistry 替换 registry(测试用)。
func (o *Orchestrator) WithRegistry(reg *Registry) *Orchestrator {
	o.registry = reg
	return o
}

// RefreshResult 一次刷新的产出。
type RefreshResult struct {
	SourceID    uint      `json:"source_id"`
	SourceName  string    `json:"source_name"`
	PulledCount int       `json:"pulled_count"`
	Inserted    int       `json:"inserted"`
	Updated     int       `json:"updated"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Error       string    `json:"error,omitempty"`
}

// RefreshFromSource 拉一个源(走 adapter.Discover),把结果 upsert 到 market_skills。
// 同一 sourceID 短时间内并发触发会被 ignore(防止"刷新风暴")。
//
// 2026-07-01 增:keyword 参数,透传到 adapter.Discover。三方源按自己的语义搜索;
// 空 keyword = 拉全量目录(走 adapter 默认排序)。
func (o *Orchestrator) RefreshFromSource(ctx context.Context, sourceID uint, keyword string) (*RefreshResult, error) {
	if sourceID == 0 {
		return nil, ErrSourceNotFound
	}
	o.mu.Lock()
	if o.refreshing[sourceID] {
		o.mu.Unlock()
		return nil, ErrSourceBusy
	}
	o.refreshing[sourceID] = true
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		delete(o.refreshing, sourceID)
		o.mu.Unlock()
	}()

	started := time.Now()
	res := &RefreshResult{SourceID: sourceID, StartedAt: started}
	src, err := o.sourceModel.FindOneById(sourceID)
	if err != nil {
		res.FinishedAt = time.Now()
		return res, fmt.Errorf("%w: %d", ErrSourceNotFound, sourceID)
	}
	res.SourceName = src.Name
	if !src.Enabled {
		res.FinishedAt = time.Now()
		return res, ErrSourceDisabled
	}
	ad, ok := o.registry.Get(src.Type)
	if !ok {
		res.FinishedAt = time.Now()
		return res, fmt.Errorf("%w: type=%s", ErrSourceNotImpl, src.Type)
	}
	baseURL := resolveBaseFromConfig(src.ConfigJSON, ad.BaseURL())
	items, derr := ad.Discover(ctx, baseURL, strings.TrimSpace(keyword))
	if derr != nil {
		res.FinishedAt = time.Now()
		res.Error = derr.Error()
		return res, derr
	}
	inserted, updated := 0, 0
	for _, it := range items {
		row := itemToRow(src, ad, baseURL, it)
		conds := append(where.New(mmarketskill.FieldSourceID, "=", src.ID).Conditions(),
			where.New(mmarketskill.FieldRemoteID, "=", row.RemoteID).Conditions()...)
		prev, _ := o.skillModel.FindOne(conds)
		if err := o.skillModel.Upsert(row); err != nil {
			continue
		}
		if prev == nil || prev.ID == 0 {
			inserted++
		} else {
			updated++
		}
	}
	res.PulledCount = len(items)
	res.Inserted = inserted
	res.Updated = updated
	res.FinishedAt = time.Now()
	return res, nil
}

// DownloadFromSource 走 adapter.Download,返回 canonical。
// 装到 store 的动作由 service 层调 sskill.Service.Create 完成。
func (o *Orchestrator) DownloadFromSource(ctx context.Context, sourceID uint, remoteID string) (*skilladapter.Canonical, error) {
	if remoteID == "" {
		return nil, ErrEmptyRemoteID
	}
	src, err := o.sourceModel.FindOneById(sourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %d", ErrSourceNotFound, sourceID)
	}
	if !src.Enabled {
		return nil, ErrSourceDisabled
	}
	ad, ok := o.registry.Get(src.Type)
	if !ok {
		return nil, fmt.Errorf("%w: type=%s", ErrSourceNotImpl, src.Type)
	}
	baseURL := resolveBaseFromConfig(src.ConfigJSON, ad.BaseURL())
	can, derr := ad.Download(ctx, baseURL, remoteID)
	if derr != nil {
		return nil, derr
	}
	if can == nil {
		return nil, fmt.Errorf("%w: empty canonical", ErrRemoteFetchFail)
	}
	return can, nil
}

// DiscoverFromSource 走 adapter.Discover,纯拉不写(2026-07-01 增)。
//
// 用于 ListSkillsRemote:不依赖 market_skills 缓存,每次都打三方源,响应永远最新。
// 与 RefreshFromSource 的差别:本方法不 upsert 到 DB、不返回 RefreshResult,
// 仅返回 []MarketItem 给调用方做 in-memory 分页 / 过滤。
func (o *Orchestrator) DiscoverFromSource(ctx context.Context, sourceID uint, keyword string) ([]MarketItem, error) {
	items, _, err := o.DiscoverFromSourceWithMeta(ctx, sourceID, keyword)
	return items, err
}

// DiscoverFromSourceWithMeta 走 adapter.Discover,带 source 标记 (2026-07-03 增)。
//
// 返回 (items, source, err):
//   - source="remote":真实打三方源成功
//   - source="fallback":三方源不可达 / 网络层失败,返回的是 adapter 内置兜底列表
//   - source="":adapter 没识别(理论上不会发生,留空兜底)
//
// 推断方式:看 ctx 是否被 deadline 触发 / 看 adapter 内是否触发 fallback 路径。
// 这里用 err 信息 + items 内容结合判断:
//   1. err 是 context.DeadlineExceeded / context.Canceled / 网络层错误 → fallback
//   2. items 与 adapter 的 knownFallback 完全重合 → fallback(强信号)
//   3. 否则 → remote
//
// 注意:adapter 内部超时通常在 setStop + out=0 时主动 fallback,err 为 nil;
// 此时用 (2) 的"与 knownFallback 重合"判断。
func (o *Orchestrator) DiscoverFromSourceWithMeta(ctx context.Context, sourceID uint, keyword string) ([]MarketItem, string, error) {
	src, err := o.sourceModel.FindOneById(sourceID)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %d", ErrSourceNotFound, sourceID)
	}
	if !src.Enabled {
		return nil, "", ErrSourceDisabled
	}
	ad, ok := o.registry.Get(src.Type)
	if !ok {
		return nil, "", fmt.Errorf("%w: type=%s", ErrSourceNotImpl, src.Type)
	}
	baseURL := resolveBaseFromConfig(src.ConfigJSON, ad.BaseURL())
	items, derr := ad.Discover(ctx, baseURL, strings.TrimSpace(keyword))

	// 判定 source 标记
	source := "remote"
	if derr != nil {
		// 2026-07-03 增:任何 err 都视为远端不可达,即便 adapter 仍返 items
		// (skillhub 全量分支 err 时已走 fallback 返 items,这时也算 fallback)
		source = "fallback"
	} else if isFallbackItems(ad, items) {
		// 2026-07-03 增:items 完全等于 adapter 内置 knownFallback → 走兜底
		source = "fallback"
	}
	return items, source, derr
}

// isFallbackItems 判断 items 是否完全等于 adapter 的 knownFallback 列表(2026-07-03 增)。
//
// 强信号:每个 item.RemoteID 都能在 adapter.KnownFallbackIDs() 找到 → fallback。
// 不同 adapter 的 knownFallback 不同;没有这个方法时,只判断 items 是 fallback
// 路径的特征组合(短 + 全是固定 slug)即可。
func isFallbackItems(ad MarketAdapter, items []MarketItem) bool {
	if ad == nil || len(items) == 0 {
		return false
	}
	known := ad.KnownFallbackIDs()
	if len(known) == 0 {
		return false
	}
	if len(items) != len(known) {
		return false
	}
	have := make(map[string]bool, len(known))
	for _, it := range items {
		have[it.RemoteID] = true
	}
	for _, id := range known {
		if !have[id] {
			return false
		}
	}
	return true
}

// ListSources 简化的"所有源"读出。
func (o *Orchestrator) ListSources() ([]*entity.MarketSource, error) {
	list, _, err := o.sourceModel.FindList(nil, nil)
	return list, err
}

// ListSkills 按 source / keyword 过滤列出。
func (o *Orchestrator) ListSkills(sourceID uint, keyword string, page, size int) ([]*entity.MarketSkill, int64, error) {
	conds := []*where.Condition{}
	if sourceID > 0 {
		conds = append(conds, where.New(mmarketskill.FieldSourceID, "=", sourceID).Conditions()...)
	}
	if k := strings.TrimSpace(keyword); k != "" {
		conds = append(conds, where.New(mmarketskill.FieldName, "LIKE", "%"+k+"%").Conditions()...)
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	extra := &where.Extra{
		PageNum:       page,
		PageSize:      size,
		OrderByColumn: mmarketskill.FieldFetchedAt,
		OrderByDesc:   true,
	}
	list, total, err := o.skillModel.FindList(conds, extra)
	if err != nil {
		return nil, 0, err
	}
	return list, int64(total), nil
}

// ItemToRow 把 MarketItem + source 拼成 entity.MarketSkill(2026-07-01 导出)。
//
// 原本为 RefreshFromSource 内部 helper,2026-07-01 暴露给 smarket.ListSkillsRemote 用:
// ListSkillsRemote 不写 DB,但仍需要把 MarketItem 映射成 entity.MarketSkill,
// 让前端继续用统一 schema(items 数组里的字段保持 remote_id / name / version / author / tags 等)。
func (o *Orchestrator) ItemToRow(src *entity.MarketSource, ad MarketAdapter, baseURL string, it MarketItem) *entity.MarketSkill {
	return itemToRow(src, ad, baseURL, it)
}

// itemToRow 把 MarketItem + source 拼成 entity.MarketSkill。
func itemToRow(src *entity.MarketSource, ad MarketAdapter, baseURL string, it MarketItem) *entity.MarketSkill {
	detail := it.DetailURL
	if detail == "" {
		detail = baseURL
	}
	install := it.InstallRef
	if install == "" {
		install = detail
	}
	tags := strings.Join(it.Tags, ",")
	extra, _ := json.Marshal(map[string]any{
		"source_display": ad.DisplayName(),
		"source_type":    ad.SourceID(),
	})
	return &entity.MarketSkill{
		SourceID:   src.ID,
		SourceName: src.Name,
		RemoteID:   it.RemoteID,
		Name:       it.Name,
		Version:    it.Version,
		Description: it.Description,
		Author:     it.Author,
		Tags:       tags,
		InstallRef: install,
		DetailURL:  detail,
		ExtraJSON:  string(extra),
		FetchedAt:  time.Now(),
	}
}

// resolveBaseFromConfig 解析 source.ConfigJSON 里的 base_url;失败 fallback。
func resolveBaseFromConfig(configJSON, def string) string {
	if strings.TrimSpace(configJSON) == "" {
		return def
	}
	var cfg SourceConfigJSON
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return def
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return def
	}
	return cfg.BaseURL
}

// ErrSourceBusy 源正在被另一个 goroutine 刷新。
var ErrSourceBusy = errors.New("skillmarket: source is refreshing")
