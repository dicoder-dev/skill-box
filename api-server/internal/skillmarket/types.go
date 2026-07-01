// Package skillmarket 提供三方 skill 市场的统一接入。
//
// 设计要点(见 docs/project/需求规划.md 第 5.1 节 + 第 7.7 节):
//   - MarketAdapter 是各三方源(skillhub / skills.sh / 自定义)实现的接口
//   - Discover 拉目录;Detail 拉详情;Download 拿到 canonical
//   - 上层 (smarket / frontend) 不直接接触具体源,统一走 orchestrator
//   - 列表查询走 entity.MarketSkill 缓存,避免每次页面刷新都打三方
package skillmarket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ginp-api/internal/skilladapter"
)

// SourceType 三方源类型。
const (
	SourceSkillhub = "skillhub" // skillhub.cn
	SourceSkillsSH = "skillssh" // skills.sh
	SourceCustom   = "custom"   // 用户自定义 HTTP+JSON(预留)
)

// 业务错误。
var (
	ErrSourceNotFound  = errors.New("skillmarket: source not found")
	ErrSourceDisabled  = errors.New("skillmarket: source disabled")
	ErrSourceNotImpl   = errors.New("skillmarket: source not implemented")
	ErrRemoteNotFound  = errors.New("skillmarket: remote skill not found")
	ErrRemoteFetchFail = errors.New("skillmarket: remote fetch failed")
	ErrEmptyRemoteID   = errors.New("skillmarket: empty remote id")
)

// MarketItem 列表里的一项(只列轻量字段;详情走 Detail)。
type MarketItem struct {
	RemoteID    string    `json:"remote_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version,omitempty"`
	Description string    `json:"description,omitempty"`
	Author      string    `json:"author,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	DetailURL   string    `json:"detail_url,omitempty"`
	InstallRef  string    `json:"install_ref,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// MarketDetail 详情(列表 + 文件清单 / 额外元数据)。
type MarketDetail struct {
	MarketItem
	License   string                 `json:"license,omitempty"`
	Homepage  string                 `json:"homepage,omitempty"`
	Extra     map[string]any         `json:"extra,omitempty"`
	Canonical *skilladapter.Canonical `json:"canonical,omitempty"`
}

// MarketAdapter 三方源适配器接口。
//
// 实现方约束:
//   - Discover 在源不可达 / 解析失败时返回 (nil, fmt.Errorf("%w: %v", ErrRemoteFetchFail, err))
//   - Detail / Download 同样,缺失/找不到走 ErrRemoteNotFound
//   - 所有方法接 ctx,实现侧负责 ctx.Err() 检查
type MarketAdapter interface {
	// SourceID 源唯一 ID(skillhub / skillssh / custom),与 entity.MarketSource.Type 对应。
	SourceID() string

	// DisplayName UI 展示名。
	DisplayName() string

	// BaseURL 三方源根地址;可由 entity.MarketSource.ConfigJSON 覆盖。
	BaseURL() string

	// Discover 拉目录(轻量字段,适合列表展示)。
	// keyword 空 = 全量目录(走默认排序);非空 = 三方源搜索语义(由 adapter 决定如何透传)。
	// 2026-07-01 增:keyword 参数,前端搜索框透传到三方源(替代之前 SQL LIKE)。
	Discover(ctx context.Context, baseURL, keyword string) ([]MarketItem, error)

	// Detail 拉详情(包含 canonical 或 canonical 所需文件清单)。
	Detail(ctx context.Context, baseURL, remoteID string) (*MarketDetail, error)

	// Download 拉到本地:返回 canonical,以及(可选)写好的本地目录。
	// 实现侧负责把 source-specific 格式(单个 SKILL.md / tarball / 目录树)转成 canonical。
	Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error)
}

// SanitizeSourceName 把 source.name 规范成 ^[a-z][a-z0-9_-]{1,63}$。
// 用于 entity.MarketSource.Name 与适配器 SourceID 的弱校验。
func SanitizeSourceName(s string) string {
	out := skilladapter.NormalizeName(s)
	if out == "" {
		return ""
	}
	return out
}

// ResolveBaseURL 用 entity.MarketSource.ConfigJSON("base_url") 覆盖 adapter.BaseURL()。
// 约定:ConfigJSON 是 JSON,字段 base_url 优先;否则用 adapter 的默认。
// 此处不引入额外 JSON 解析依赖,直接包一层 fmt.Errorf 让调用方自己解析。
func ResolveBaseURL(adapter MarketAdapter, sourceConfigJSON string) string {
	if sourceConfigJSON == "" {
		return adapter.BaseURL()
	}
	// 简化处理:config_json 形如 `{"base_url":"https://x"}`;不做严格 JSON 解析,
	// 留给 controller 层用 json.Unmarshal 解出 base_url 后再调 Discovery/Detail。
	// 这里只兜底返回默认值。
	return adapter.BaseURL()
}

// WrapErr 统一把 error 包成带前缀的 ErrRemoteFetchFail。
func WrapErr(verb string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", ErrRemoteFetchFail, verb, err)
}
