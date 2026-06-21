// Package smarket 提供三方市场域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.8 + 5.1 节):
//   - 三方源走 entity.MarketSource + skillmarket.Orchestrator
//   - 列表:直接查 entity.MarketSkill(避免每次都打三方)
//   - 装到本地:orchestrator.DownloadFromSource 拿 canonical,再走 sskill.Service.Create
//   - source 维度:smarket 自身只读 / 缓存元数据;源增删不在本步范围(Step 7 落 4 端点,源由
//     seed 在 Onboarding 阶段插入)
package smarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	_ "ginp-api/internal/skillmarket/skillhub"
	_ "ginp-api/internal/skillmarket/skillssh"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrSourceNotFound = errors.New("market: source not found")
	ErrSkillNotFound  = errors.New("market: skill not found")
	ErrInstallFailed  = errors.New("market: install failed")
)

// Service 业务服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	// skillSvc 在 Install 时复用,避免本包重写 sskill 写盘逻辑
	skillSvcFactory func() (*sskill.Service, error)
}

func New(dbWrite, dbRead *gorm.DB, skillSvcFactory func() (*sskill.Service, error)) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, skillSvcFactory: skillSvcFactory}
}

func (s *Service) sourceModel() *mmarketsource.Model {
	return mmarketsource.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) skillModel() *mmarketskill.Model {
	return mmarketskill.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) orchestrator() *skillmarket.Orchestrator {
	return skillmarket.NewOrchestrator(s.sourceModel(), s.skillModel())
}

// ListSources 列出所有源(不做 enabled 过滤,前端按需展示)。
type ListSourcesResult struct {
	Items []*entity.MarketSource `json:"items"`
	Total int64                  `json:"total"`
}

func (s *Service) ListSources() (*ListSourcesResult, error) {
	items, total, err := s.sourceModel().FindList(nil, &where.Extra{
		PageNum: 1, PageSize: 100, OrderByColumn: mmarketsource.FieldID, OrderByDesc: false,
	})
	if err != nil {
		return nil, err
	}
	return &ListSourcesResult{Items: items, Total: int64(total)}, nil
}

// ListSkillsQuery 列表过滤。
type ListSkillsQuery struct {
	SourceID uint
	Keyword  string
	Page     int
	Size     int
}

// ListSkillsResult 列表结果。
type ListSkillsResult struct {
	Items []*entity.MarketSkill `json:"items"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

func (s *Service) ListSkills(q ListSkillsQuery) (*ListSkillsResult, error) {
	items, total, err := s.orchestrator().ListSkills(q.SourceID, q.Keyword, q.Page, q.Size)
	if err != nil {
		return nil, err
	}
	page, size := q.Page, q.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	return &ListSkillsResult{Items: items, Total: total, Page: page, Size: size}, nil
}

// RefreshSource 触发一个源的刷新(走 orchestrator → adapter → upsert)。
func (s *Service) RefreshSource(ctx context.Context, sourceID uint) (*skillmarket.RefreshResult, error) {
	if sourceID == 0 {
		return nil, ErrSourceNotFound
	}
	return s.orchestrator().RefreshFromSource(ctx, sourceID)
}

// InstallInput 装到 store 的入参。
type InstallInput struct {
	SourceID  uint   `json:"source_id"`
	RemoteID  string `json:"remote_id"`
	Scope     string `json:"scope"`     // global / project
	ProjectID uint   `json:"project_id"` // scope=project 时必填
}

// InstallResult 装到 store 的结果。
type InstallResult struct {
	MarketSkill *entity.MarketSkill `json:"market_skill"`
	Skill       *entity.Skill       `json:"skill"`
}

// Install 从三方下载,转成 canonical,再走 sskill.Service.Create 落到 store。
func (s *Service) Install(ctx context.Context, in *InstallInput) (*InstallResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrInstallFailed)
	}
	if in.SourceID == 0 || strings.TrimSpace(in.RemoteID) == "" {
		return nil, fmt.Errorf("%w: source_id / remote_id 必填", ErrInstallFailed)
	}
	scope := strings.ToLower(strings.TrimSpace(in.Scope))
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return nil, fmt.Errorf("%w: scope 必须是 global / project", ErrInstallFailed)
	}
	if scope == skilladapter.ScopeProject && in.ProjectID == 0 {
		return nil, fmt.Errorf("%w: project scope 需要 project_id", ErrInstallFailed)
	}
	// 1) 找源
	src, err := s.sourceModel().FindOneById(in.SourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %d", ErrSourceNotFound, in.SourceID)
	}
	// 2) 找缓存里的 market_skill
	conds := append(where.New(mmarketskill.FieldSourceID, "=", in.SourceID).Conditions(),
		where.New(mmarketskill.FieldRemoteID, "=", in.RemoteID).Conditions()...)
	row, err := s.skillModel().FindOne(conds)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, in.RemoteID)
	}
	// 3) 下载
	can, err := s.orchestrator().DownloadFromSource(ctx, in.SourceID, in.RemoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInstallFailed, err)
	}
	// 4) 落到 store(走 sskill)
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("%w: skill service factory not wired", ErrInstallFailed)
	}
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInstallFailed, err)
	}
	// 补 manifest 字段(以三方元数据为底,canonical 为真)
	can.Manifest.Author = firstNonEmpty(can.Manifest.Author, row.Author)
	if can.Manifest.License == "" {
		can.Manifest.License = row.License
	}
	created, cerr := ssvc.Create(&sskill.WriteInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      can.Manifest.Name,
		Version:   firstNonEmpty(can.Manifest.Version, row.Version, "0.1.0"),
		Source:    "market",
		SourceRef: fmt.Sprintf("%s:%s", src.Name, in.RemoteID),
		Manifest:  can.Manifest,
		Files:     can.Files,
	})
	if cerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrInstallFailed, cerr)
	}
	return &InstallResult{MarketSkill: row, Skill: created}, nil
}

// GetMarketSkill 拿单个缓存记录。
func (s *Service) GetMarketSkill(id uint) (*entity.MarketSkill, error) {
	if id == 0 {
		return nil, ErrSkillNotFound
	}
	return s.skillModel().FindOneById(id)
}

// UpdateSourceConfig 改写一个 source 的 ConfigJSON(测试用,生产走 Settings 或 admin 端点)。
// 返回更新后的 source。
func (s *Service) UpdateSourceConfig(sourceID uint, configJSON string) (*entity.MarketSource, error) {
	src, err := s.sourceModel().FindOneById(sourceID)
	if err != nil {
		return nil, ErrSourceNotFound
	}
	src.ConfigJSON = configJSON
	if err := s.sourceModel().Update(where.New(mmarketsource.FieldID, "=", src.ID).Conditions(), src); err != nil {
		return nil, fmt.Errorf("market: update source config: %w", err)
	}
	return src, nil
}

// DefaultSources 内置的 source(seed 时用,首启自动注册)。
// 不在 service init 里跑,由 cmd/bootstrap 或首次 Onboarding 调用。
func DefaultSources() []*entity.MarketSource {
	mk := func(name, t string) *entity.MarketSource {
		return &entity.MarketSource{
			Name:    name,
			Type:    t,
			Enabled: true,
		}
	}
	return []*entity.MarketSource{
		mk("skillhub", skillmarket.SourceSkillhub),
		mk("skills.sh", skillmarket.SourceSkillsSH),
	}
}

// EnsureDefaultSources seed 默认 source(只插不存在的)。幂等。
func (s *Service) EnsureDefaultSources() error {
	existing, _, err := s.sourceModel().FindList(nil, &where.Extra{PageNum: 1, PageSize: 100})
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, e := range existing {
		have[e.Name] = true
	}
	for _, def := range DefaultSources() {
		if have[def.Name] {
			continue
		}
		if _, err := s.sourceModel().Create(def); err != nil {
			return fmt.Errorf("seed source %s: %w", def.Name, err)
		}
	}
	return nil
}

// SanityJSON 调试用:把 entity 序列化成可读 JSON。
func SanityJSON(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
