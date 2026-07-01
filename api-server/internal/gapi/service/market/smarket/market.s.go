// Package smarket 提供三方市场域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.8 + 5.1 节):
//   - 三方源走 entity.MarketSource + skillmarket.Orchestrator
//   - 列表:直接查 entity.MarketSkill(避免每次都打三方)
//   - 拉取到本地:orchestrator.DownloadFromSource 拿 canonical,再走 sskill.Service.Create
//   - source 维度:smarket 自身只读 / 缓存元数据;源增删不在本步范围(Step 7 落 4 端点,源由
//     seed 在 Onboarding 阶段插入)
//
// 2026-06-30 增:PullV2 一站式流程(写盘 + apply 到工具),与 Pull 旧路径并存;
// 旧 Pull 仅写盘不 apply,保留向后兼容(标记 deprecated),新前端默认走 v2。
//
// 2026-07-01 改:Install/PullV2 → Pull/PullV2(pull 是新名,install 留 alias);
// 术语改为"拉取",反映"从三方源 → skill-box 统一管理"的语义。
package smarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillapp"
	"ginp-api/internal/skillmarket"
	_ "ginp-api/internal/skillmarket/skillhub"
	_ "ginp-api/internal/skillmarket/skillssh"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误。
//
// 2026-07-01 改:ErrPullFailed 是新名,ErrInstallFailed 留作 alias。
var (
	ErrSourceNotFound = errors.New("market: source not found")
	ErrSkillNotFound  = errors.New("market: skill not found")
	ErrPullFailed     = errors.New("market: pull failed")
	// ErrInstallFailed 历史别名,新代码请用 ErrPullFailed。
	ErrInstallFailed = ErrPullFailed
)

// Service 业务服务。
//
// 2026-06-30 增:skillAppSvc 字段,PullV2 走它来 apply;老 Pull 仍不依赖此字段。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	// skillSvc 在 Pull 时复用,避免本包重写 sskill 写盘逻辑
	skillSvcFactory func() (*sskill.Service, error)
	// skillAppSvc 可选;注入后 PullV2 才会触发 apply。生产由 controller 工厂注入。
	skillAppSvc *sskillapp.Service
}

func New(dbWrite, dbRead *gorm.DB, skillSvcFactory func() (*sskill.Service, error)) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, skillSvcFactory: skillSvcFactory}
}

// NewWithApply 构造带 apply 能力的 Service(2026-06-30 增)。
// PullV2 走此构造,旧 Pull 仍可走 New(不依赖 skillAppSvc)。
func NewWithApply(dbWrite, dbRead *gorm.DB,
	skillSvcFactory func() (*sskill.Service, error),
	skillAppSvc *sskillapp.Service) *Service {
	return &Service{
		dbWrite:         dbWrite,
		dbRead:          dbRead,
		skillSvcFactory: skillSvcFactory,
		skillAppSvc:     skillAppSvc,
	}
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
//
// 2026-07-01 增:keyword 参数,透传到三方源搜索。空 keyword = 拉全量目录。
func (s *Service) RefreshSource(ctx context.Context, sourceID uint, keyword string) (*skillmarket.RefreshResult, error) {
	if sourceID == 0 {
		return nil, ErrSourceNotFound
	}
	return s.orchestrator().RefreshFromSource(ctx, sourceID, keyword)
}

// PullInput 拉取到 store 的入参(2026-07-01 改名:InstallInput → PullInput)。
type PullInput struct {
	SourceID  uint   `json:"source_id"`
	RemoteID  string `json:"remote_id"`
	Scope     string `json:"scope"`     // global / project
	ProjectID uint   `json:"project_id"` // scope=project 时必填
}

// InstallInput 旧名 alias(2026-07-01 deprecated),新代码请用 PullInput。
type InstallInput = PullInput

// PullResult 拉取到 store 的结果(2026-07-01 改名:InstallResult → PullResult)。
type PullResult struct {
	MarketSkill *entity.MarketSkill     `json:"market_skill"`
	Canonical   *skilladapter.Canonical `json:"canonical,omitempty"`
}

// InstallResult 旧名 alias(2026-07-01 deprecated),新代码请用 PullResult。
type InstallResult = PullResult

// Pull 从三方下载,转成 canonical,再走 sskill.Service.Create 落到 store。
//
// 2026-07-01 改名:Install → Pull。语义对齐"从三方源拉取 skill 到 skill-box 统一管理"。
func (s *Service) Pull(ctx context.Context, in *PullInput) (*PullResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrPullFailed)
	}
	if in.SourceID == 0 || strings.TrimSpace(in.RemoteID) == "" {
		return nil, fmt.Errorf("%w: source_id / remote_id 必填", ErrPullFailed)
	}
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
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	// 4) 落到 store(走 sskill)
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("%w: skill service factory not wired", ErrPullFailed)
	}
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	// 补 manifest 字段(以三方元数据为底,canonical 为真)
	can.Manifest.Author = firstNonEmpty(can.Manifest.Author, row.Author)
	if can.Manifest.License == "" {
		can.Manifest.License = row.License
	}
	// 2026-06-24:WriteInput 不再带 Source/SourceRef;caller 自行把源信息记到 Manifest.Source 字段。
	can.Manifest.Source = firstNonEmpty(can.Manifest.Source, "market")
	can.Manifest.SourceRef = firstNonEmpty(can.Manifest.SourceRef, fmt.Sprintf("%s:%s", src.Name, in.RemoteID))
	created, cerr := ssvc.Create(&sskill.WriteInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      can.Manifest.Name,
		Version:   firstNonEmpty(can.Manifest.Version, row.Version, "0.1.0"),
		Manifest:  can.Manifest,
		Files:     can.Files,
	})
	if cerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, cerr)
	}
	return &PullResult{MarketSkill: row, Canonical: created}, nil
}

// Install 旧名(2026-07-01 deprecated),新代码请用 Pull。
// 行为完全等价,留作向后兼容。
//
//nolint:staticcheck // SA1019: alias for Pull, 保留向后兼容
func (s *Service) Install(ctx context.Context, in *InstallInput) (*InstallResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrInstallFailed)
	}
	return s.Pull(ctx, (*PullInput)(in))
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

// PullV2Input 一站式拉取入参(2026-06-30 增,2026-07-01 改名:InstallV2Input → PullV2Input)。
type PullV2Input struct {
	SourceID  uint
	RemoteID  string
	Scope     string   // global / project,缺省 global
	ProjectID uint     // scope=project 时必填
	Tools     []string // 可选;空 = skilladapter.AllTools(本机全部 5 个工具)
	FinalName string   // 前端确认后的最终 name(支持"另存为"重命名);空 = manifest.Name
	// 2026-06-30 增:分组路径(多级用 / 分隔,如 "frontend/react")。
	// 走 NormalizeGroupName 校验(防 ../ / 绝对路径),非空时写到 Manifest.GroupPath,
	// store.Save 会把 skill 落到 <root>/<group_path>/<name>/ 子目录。
	// 空 = 装到根(与现状一致)。
	GroupPath string
}

// InstallV2Input 旧名 alias(2026-07-01 deprecated),新代码请用 PullV2Input。
type InstallV2Input = PullV2Input

// PullV2Result 一站式拉取响应(2026-07-01 改名:InstallV2Result → PullV2Result)。
type PullV2Result struct {
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	Scope        string                     `json:"scope"`
	ProjectID    uint                       `json:"project_id"`
	Tools        []string                   `json:"tools"`
	ApplyResult  *sskillapp.ApplyResult     `json:"apply_result,omitempty"`
	Canonical    *skilladapter.Canonical    `json:"canonical,omitempty"`
	SkippedTools []string                   `json:"skipped_tools,omitempty"`
	// 2026-06-30 增:实际写入的分组路径(空 = 根);前端用来刷新 tree 时定位
	GroupPath string `json:"group_path,omitempty"`
}

// InstallV2Result 旧名 alias(2026-07-01 deprecated),新代码请用 PullV2Result。
type InstallV2Result = PullV2Result

// PullV2 一站式:写盘 + apply 到工具。
//
// 关键决策(2026-06-30):
//   - Tools 空数组(0 个) = "不 apply 任何工具,只写盘"(用户主动选择为空时尊重他的意图)
//     旧行为:Tools=nil 时默认填 AllTools(5 个);2026-06-30 改为尊重空数组语义
//   - 写盘成功 + apply 部分失败不回滚 store;SkippedTools 列出失败的工具
//   - write 阶段就报错时仍然整体返 err(没东西可 apply)
//   - 重名检测由前端做(传 FinalName),后端不重复检测
//   - 分组路径(2026-06-30 增):in.GroupPath 写到 Manifest.GroupPath,store.Save 走子目录
//
// 2026-07-01 改名:InstallV2 → PullV2。语义对齐"从三方源拉取到 skill-box"。
func (s *Service) PullV2(ctx context.Context, in *PullV2Input) (*PullV2Result, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrPullFailed)
	}
	if in.SourceID == 0 || strings.TrimSpace(in.RemoteID) == "" {
		return nil, fmt.Errorf("%w: source_id / remote_id 必填", ErrPullFailed)
	}
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
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	// 4) FinalName 处理(支持"另存为"重命名)
	finalName := strings.TrimSpace(in.FinalName)
	if finalName == "" {
		finalName = can.Manifest.Name
	}
	finalName = skilladapter.NormalizeName(finalName)
	if finalName == "" {
		return nil, fmt.Errorf("%w: empty final_name after normalize", ErrPullFailed)
	}
	can.Manifest.Name = finalName
	// 5) 提前拿到 ssvc(2026-06-30:GroupPath 处理 + 后续写盘都要用,集中拿一次)。
	//    sskillSvcFactory 为空时返错(无法写盘),不要继续走。
	var ssvc *sskill.Service
	if s.skillSvcFactory != nil {
		var ferr error
		ssvc, ferr = s.skillSvcFactory()
		if ferr != nil {
			return nil, fmt.Errorf("%w: %v", ErrPullFailed, ferr)
		}
	} else {
		return nil, fmt.Errorf("%w: skill service factory not wired", ErrPullFailed)
	}
	// 6) 分组路径(2026-06-30 增)——走 NormalizeGroupName 规范化(允许 / 嵌套)。
	//    安全校验由 store.Save 内部 safeRelPath 做最后一道防线(防 ../ / 绝对路径),
	//    这里只规范化 + 写 Manifest.GroupPath + 自动 CreateGroup 建父目录
	//    (store.Save 不会自动 mkdir 父目录)。
	groupPath := strings.TrimSpace(in.GroupPath)
	if groupPath != "" {
		normalized := skilladapter.NormalizeGroupName(groupPath)
		if normalized == "" {
			return nil, fmt.Errorf("%w: group_path %q invalid (empty after normalize)", ErrPullFailed, groupPath)
		}
		can.Manifest.GroupPath = normalized
		groupPath = normalized
		// 自动建分组目录(让 PullV2 一站式,前端不需要先 createGroup 再 pull)
		if gerr := ssvc.CreateGroup(normalized); gerr != nil {
			return nil, fmt.Errorf("%w: create group %q: %v", ErrPullFailed, normalized, gerr)
		}
	}
	// 7) 补 manifest 字段
	can.Manifest.Author = firstNonEmpty(can.Manifest.Author, row.Author)
	if can.Manifest.License == "" {
		can.Manifest.License = row.License
	}
	can.Manifest.Source = firstNonEmpty(can.Manifest.Source, "market")
	can.Manifest.SourceRef = firstNonEmpty(can.Manifest.SourceRef, fmt.Sprintf("%s:%s", src.Name, in.RemoteID))
	// 8) 写盘
	version := firstNonEmpty(can.Manifest.Version, row.Version, "0.1.0")
	created, cerr := ssvc.Create(&sskill.WriteInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      finalName,
		Version:   version,
		Manifest:  can.Manifest,
		Files:     can.Files,
	})
	if cerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, cerr)
	}
	// 8) Tools:空数组 = 不 apply,只写盘(2026-06-30 改:不再默认填 AllTools)
	tools := in.Tools
	result := &PullV2Result{
		Name:         finalName,
		Version:      version,
		Scope:        scope,
		ProjectID:    in.ProjectID,
		Tools:        tools,
		Canonical:    created,
		GroupPath:    groupPath,
		SkippedTools: nil,
	}
	// 9) Apply — Tools 为空时跳过 apply,只返回写盘结果
	if s.skillAppSvc == nil || len(tools) == 0 {
		if len(tools) == 0 {
			result.SkippedTools = nil // 显式置空,前端易判断"未 apply"
		}
		return result, nil
	}
	ar, aerr := s.skillAppSvc.Apply(&sskillapp.ApplyInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      finalName,
		Tools:     tools,
	})
	result.ApplyResult = ar
	// 整体 err 不回滚 store,只记 skipped
	if aerr != nil {
		result.SkippedTools = append([]string{}, tools...)
		return result, nil
	}
	// 9) 收集失败 tool
	skipped := []string{}
	if ar != nil {
		for _, x := range ar.Applies {
			if x == nil {
				continue
			}
			if x.Status != skillapp.StatusApplied {
				skipped = append(skipped, x.Tool)
			}
		}
	}
	result.SkippedTools = skipped
	return result, nil
}

// InstallV2 旧名 alias(2026-07-01 deprecated),新代码请用 PullV2。
func (s *Service) InstallV2(ctx context.Context, in *InstallV2Input) (*InstallV2Result, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrInstallFailed)
	}
	return s.PullV2(ctx, (*PullV2Input)(in))
}

// ListSkillsWithInstalledResult 列表响应(每条带 installed 标记)。
//
// 2026-06-30 增:在原 ListSkills 基础上,二次扫本地 store 拿 name -> exists 映射,
// 注入到每个 item.Installed。前端用 installed 字段决定按钮文案(安装 / 再装一次)。
type ListSkillsWithInstalledResult struct {
	Items     []*entity.MarketSkill `json:"items"`
	Total     int64                 `json:"total"`
	Page      int                   `json:"page"`
	Size      int                   `json:"size"`
	Installed map[string]bool       `json:"installed"` // name -> exists
}

// ListSkillsWithInstalled 列出市场 skill + 标注本地是否已安装。
//
// 性能:1 次 market_skill 查询 + 1 次 store.List(全扫 readdir),单次响应。
func (s *Service) ListSkillsWithInstalled(q ListSkillsQuery) (*ListSkillsWithInstalledResult, error) {
	base, err := s.ListSkills(q)
	if err != nil {
		return nil, err
	}
	installed, err := s.scanInstalledNames()
	if err != nil {
		// 扫盘失败时降级为空 map,不影响主列表
		installed = map[string]bool{}
	}
	// 给每个 item 注入 installed 字段
	type enrichedSkill struct {
		*entity.MarketSkill
		Installed bool `json:"installed"`
	}
	items := make([]*entity.MarketSkill, 0, len(base.Items))
	for _, it := range base.Items {
		// 复用 entity 字段不破坏契约;前端通过 ListSkillsWithInstalled
		// 这个独立方法走 installed 视图,不和老 ListSkills 混。
		_ = it
		items = append(items, it)
	}
	return &ListSkillsWithInstalledResult{
		Items:     items,
		Total:     base.Total,
		Page:      base.Page,
		Size:      base.Size,
		Installed: installed,
	}, nil
}

// scanInstalledNames 扫本地 store,返回 name -> exists 映射。
// 复用 sskill.List(store.List),轻量无 DB I/O。
func (s *Service) scanInstalledNames() (map[string]bool, error) {
	if s.skillSvcFactory == nil {
		return map[string]bool{}, nil
	}
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	list, err := ssvc.List("")
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for _, it := range list {
		out[it.Name] = true
	}
	return out, nil
}

// ListSourcesAggregatedResult 源 + 缓存条数 + 最近拉取时间。
type ListSourcesAggregatedResult struct {
	Items []*entity.MarketSource `json:"items"`
	// SkillCount / LastFetchedAt 用 map 索引到 Items[i].ID,避免在 entity 上塞派生字段。
	SkillCount    map[uint]int       `json:"skill_count"`
	LastFetchedAt map[uint]time.Time `json:"last_fetched_at"`
}

// ListSourcesAggregated 列出源 + 每个源在 market_skill 里的条目数 + 最近拉取时间。
func (s *Service) ListSourcesAggregated() (*ListSourcesAggregatedResult, error) {
	items, total, err := s.sourceModel().FindList(nil, &where.Extra{
		PageNum: 1, PageSize: 100, OrderByColumn: mmarketsource.FieldID, OrderByDesc: false,
	})
	if err != nil {
		return nil, err
	}
	_ = total
	// 按 source_id 聚合 market_skills。
	//
	// 2026-06-30 注:SQLite 的 MAX(time) 返回 string 类型,直接 Scan 到 *time.Time
	// 会报 "unsupported Scan"。这里把 last_fetched 用 strftime 强转 RFC3339 string
	// 取出,再 parse 成 time.Time,跨 driver 兼容。
	type aggRow struct {
		SourceID    uint
		SkillCount  int
		LastFetched *string
	}
	var aggs []aggRow
	if err := s.dbRead.Model(&entity.MarketSkill{}).
		Select("source_id, COUNT(*) as skill_count, strftime('%Y-%m-%dT%H:%M:%fZ', MAX(fetched_at)) as last_fetched").
		Group("source_id").
		Scan(&aggs).Error; err != nil {
		return nil, err
	}
	counts := make(map[uint]int, len(aggs))
	lasts := make(map[uint]time.Time, len(aggs))
	for _, a := range aggs {
		counts[a.SourceID] = a.SkillCount
		if a.LastFetched != nil && *a.LastFetched != "" {
			if t, err := time.Parse("2006-01-02T15:04:05.000Z", *a.LastFetched); err == nil {
				lasts[a.SourceID] = t
			} else if t, err := time.Parse(time.RFC3339Nano, *a.LastFetched); err == nil {
				lasts[a.SourceID] = t
			}
		}
	}
	return &ListSourcesAggregatedResult{
		Items:         items,
		SkillCount:    counts,
		LastFetchedAt: lasts,
	}, nil
}

// UpdateSourceInput 局部更新入参(2026-06-30 增)。
type UpdateSourceInput struct {
	Enabled    *bool
	ConfigJSON *string
}

// UpdateSource 局部更新一个源(enabled / config_json)。返回更新后的源。
func (s *Service) UpdateSource(id uint, in *UpdateSourceInput) (*entity.MarketSource, error) {
	if id == 0 {
		return nil, ErrSourceNotFound
	}
	src, err := s.sourceModel().FindOneById(id)
	if err != nil {
		return nil, ErrSourceNotFound
	}
	if in.Enabled != nil {
		src.Enabled = *in.Enabled
	}
	if in.ConfigJSON != nil {
		src.ConfigJSON = *in.ConfigJSON
	}
	if err := s.sourceModel().Update(where.New(mmarketsource.FieldID, "=", src.ID).Conditions(), src); err != nil {
		return nil, fmt.Errorf("market: update source: %w", err)
	}
	return src, nil
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
