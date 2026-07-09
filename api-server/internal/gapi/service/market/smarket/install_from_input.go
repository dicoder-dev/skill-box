package smarket

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	"ginp-api/pkg/where"
)

// InstallFromInputInput 用户输入框 → 下载入参(2026-07-09 增)。
//
// 与 PullInput 的差别:不要求 caller 提前知道 sourceID + remoteID,而是把"用户原文"
// (slug / 详情页 URL / owner/repo@skill / GitHub URL)交给 service 解析。
type InstallFromInputInput struct {
	// SourceHint 前端传下来的当前 tab source_type("skillhub" / "skillssh");
	// 空 = auto(由 service 解析 input 推断);非空时对纯 slug / owner/repo@skill 格式
	// 按此 source 解释;URL 输入时此字段被忽略(URL 域名优先)。
	SourceHint string
	Input      string
	Scope      string // global(默认) / project
	ProjectID  uint
}

// InstallFromInputResult 落盘结果(2026-07-09 增)。
type InstallFromInputResult struct {
	SourceType    string `json:"source_type"`
	SourceName    string `json:"source_name"`
	RemoteID      string `json:"remote_id"`
	ResolvedURL   string `json:"resolved_url,omitempty"`
	SkillName     string `json:"skill_name"`
	SkillVersion  string `json:"skill_version"`
	Scope         string `json:"scope"`
	MarketSkillID uint   `json:"market_skill_id,omitempty"`
}

// InstallFromInput 用户输入框 → 解析 → 下载 → 落 store(2026-07-09 增)。
//
// 流程:
//   1) ResolveInstallInput 解析 input → (source_type, remote_id)
//   2) 找 source / 或注册一个新 source(skillhub / skillssh 默认已 seed)
//   3) Orchestrator.DownloadFromSource 拿 canonical
//   4) sskill.Service.Create 落 store(skillstore DB)
//
// 错误:
//   - ErrInvalidInput 解析失败(400)
//   - ErrSourceNotFound 源未注册(404)
//   - ErrPullFailed 下载 / 写盘失败(500)
//
// 不做 apply:本接口只负责把 skill 装到本地 store,apply 到具体工具(scope)
// 由用户在首页 Onboarding / Skill detail 触发,跟现有 Pull / PullV2 行为一致。
func (s *Service) InstallFromInput(ctx context.Context, in *InstallFromInputInput) (*InstallFromInputResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrPullFailed)
	}
	input := strings.TrimSpace(in.Input)
	if input == "" {
		return nil, fmt.Errorf("%w: input 必填", ErrInvalidInput)
	}

	// 1) 解析 input
	resolved, err := ResolveInstallInput(input, in.SourceHint)
	if err != nil {
		return nil, err
	}

	// 2) scope 校验
	scope := strings.ToLower(strings.TrimSpace(in.Scope))
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return nil, fmt.Errorf("%w: scope 必须是 global / project", ErrPullFailed)
	}
	if scope == skilladapter.ScopeProject && in.ProjectID == 0 {
		return nil, fmt.Errorf("%w: project scope 需要 project_id", ErrPullFailed)
	}

	// 3) 找 source(skillhub / skillssh 默认已 seed)
	src, err := s.findOrCreateSourceByType(resolved.SourceType)
	if err != nil {
		return nil, err
	}

	// 4) 拿缓存里的 market_skill(若已 refresh 过则有,无也不影响下载)
	conds := append(where.New(mmarketskill.FieldSourceID, "=", src.ID).Conditions(),
		where.New(mmarketskill.FieldRemoteID, "=", resolved.RemoteID).Conditions()...)
	row, _ := s.skillModel().FindOne(conds)

	// 5) 下载
	can, err := s.orchestrator().DownloadFromSource(ctx, src.ID, resolved.RemoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	if can == nil {
		return nil, fmt.Errorf("%w: empty canonical for %s", ErrPullFailed, resolved.RemoteID)
	}

	// 6) 写盘:走 sskill.Service.Create
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("%w: skill service factory not wired", ErrPullFailed)
	}
	ssvc, ferr := s.skillSvcFactory()
	if ferr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, ferr)
	}

	// 7) 补 manifest(以三方元数据为底,canonical 为真)
	if row != nil {
		can.Manifest.Author = firstNonEmpty(can.Manifest.Author, row.Author)
		if can.Manifest.License == "" {
			can.Manifest.License = row.License
		}
	}
	can.Manifest.Source = firstNonEmpty(can.Manifest.Source, "market")
	can.Manifest.SourceRef = firstNonEmpty(can.Manifest.SourceRef, fmt.Sprintf("%s:%s", src.Name, resolved.RemoteID))

	version := firstNonEmpty(can.Manifest.Version, "0.1.0")
	if row != nil {
		version = firstNonEmpty(version, row.Version, "0.1.0")
	}

	created, cerr := ssvc.Create(&sskill.WriteInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      can.Manifest.Name,
		Version:   version,
		Manifest:  can.Manifest,
		Files:     can.Files,
	})
	if cerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, cerr)
	}

	out := &InstallFromInputResult{
		SourceType:   resolved.SourceType,
		SourceName:   resolved.SourceName,
		RemoteID:     resolved.RemoteID,
		ResolvedURL:  resolved.ResolvedURL,
		SkillName:    created.Manifest.Name,
		SkillVersion: created.Manifest.Version,
		Scope:        scope,
	}
	if row != nil {
		out.MarketSkillID = row.ID
	}
	return out, nil
}

// findOrCreateSourceByType 按 source_type 找已有 source,没有就 seed 一条(2026-07-09 增)。
//
// skillhub / skillssh 默认由 EnsureDefaultSources 在 ListSources 时 seed;
// 但 InstallFromInput 路径不一定先经过 ListSources,这里兜底自 seed。
func (s *Service) findOrCreateSourceByType(sourceType string) (*entity.MarketSource, error) {
	list, _, err := s.sourceModel().FindList(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: list sources: %v", ErrSourceNotFound, err)
	}
	for _, e := range list {
		if e != nil && e.Type == sourceType && e.Enabled {
			return e, nil
		}
	}
	// 兜底:注册一个临时 enabled source(seed 流程尚未跑过)
	name := sourceType
	switch sourceType {
	case skillmarket.SourceSkillhub:
		name = "skillhub"
	case skillmarket.SourceSkillsSH:
		name = "skills.sh"
	}
	created, cerr := s.sourceModel().Create(&entity.MarketSource{
		Name:    name,
		Type:    sourceType,
		Enabled: true,
	})
	if cerr != nil {
		// 并发 race:另一个请求可能同时 seed,FindOneByName 兜底再找一次
		conds := append(where.New(mmarketsource.FieldName, "=", name).Conditions(),
			where.New(mmarketsource.FieldType, "=", sourceType).Conditions()...)
		if again, aerr := s.sourceModel().FindOne(conds); aerr == nil && again != nil {
			return again, nil
		}
		return nil, fmt.Errorf("%w: create source %s: %v", ErrSourceNotFound, sourceType, cerr)
	}
	return created, nil
}

// firstNonEmpty 已在 PullV2 上下文存在(此文件外);这里别名引用避免重复声明。
//
// 本文件其它地方使用 firstNonEmpty 时仍走 package-level 函数。
var _ = errors.Is