// Package sskilltest 提供 Skill 测试器域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 6.4 节):
//   - 一次 run = 一次 SkillTestRun 记录 + 若干 SkillTestResult(3 个 check)
//   - AI 走查走 aiengine.Manager + 注入的 SecretStore(沿用 sai 同款实现)
//   - store 物理文件读不出来 = errored,DB 写失败 = 回滚
//
// 2026-06-24 改造:不再走 mskill 表(已弃用),直接用 store.Load(name) 拿 canonical;
// skill_id 关联键用 0 占位(下游表 scope+name 才是真正的定位键)。
package sskilltest

import (
	"errors"
	"sort"
	"strings"

	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/entity"
	maiprovider "ginp-api/internal/gapi/model/skillbox/maiprovider"
	mskilltestresult "ginp-api/internal/gapi/model/skillbox/mskilltestresult"
	mskilltestrun "ginp-api/internal/gapi/model/skillbox/mskilltestrun"
	"ginp-api/internal/settings"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
	"ginp-api/internal/skilltester"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrEmptyKey  = errors.New("skilltest: skill key is empty")
	ErrNotFound  = errors.New("skilltest: run not found")
	ErrStoreLoad = errors.New("skilltest: store load failed")
	ErrDBPersist = errors.New("skilltest: db persist failed")
)

// Service 测试服务。dbWrite / dbRead / store / settings / aiEngine 全套依赖。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	store   *skillstore.Store
	st      *settings.Service
	mgr     *aiengine.Manager
}

func New(dbWrite, dbRead *gorm.DB, store *skillstore.Store, st *settings.Service, mgr *aiengine.Manager) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, store: store, st: st, mgr: mgr}
}

func (s *Service) runModel() *mskilltestrun.Model {
	return mskilltestrun.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) resultModel() *mskilltestresult.Model {
	return mskilltestresult.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) aiModel() *maiprovider.Model {
	return maiprovider.NewModel(s.dbWrite, s.dbRead)
}

// RunRequest 测试入参。
type RunRequest struct {
	Scope     string
	ProjectID uint
	Name      string
	Version   string
	Trigger   string // manual / auto,空 = manual
	Options   skilltester.Options
}

// RunResult 业务层返回(已落库)。
type RunResult struct {
	Run     *entity.SkillTestRun     `json:"run"`
	Results []*entity.SkillTestResult `json:"results"`
}

// Run 触发一次测试,落 DB,返回 Run + Results。
func (s *Service) Run(req *RunRequest) (*RunResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrEmptyKey
	}
	scope, projectID, name, version := strings.ToLower(strings.TrimSpace(req.Scope)), req.ProjectID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Version)
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return nil, ErrEmptyKey
	}
	if version == "" {
		version = "0.1.0"
	}

	// 1) 读 canonical(从 store 直接拿,不再查 mskill 表)
	c, err := s.store.Load(name)
	if err != nil {
		return nil, ErrStoreLoad
	}

	// 2) 准备 AI walker
	walker := s.buildWalker(req.Options.AIProvider)

	// 3) 跑测试器
	trigger := req.Trigger
	if trigger == "" {
		trigger = skilltester.TriggerManual
	}
	tester := skilltester.New()
	report := tester.Run(*c, skilltester.Options{
		ScriptCommand:    req.Options.ScriptCommand,
		ScriptWorkDir:    req.Options.ScriptWorkDir,
		ScriptTimeoutSec: req.Options.ScriptTimeoutSec,
		AIProvider:       req.Options.AIProvider,
		AIPreset:         req.Options.AIPreset,
		Trigger:          trigger,
	}, walker)

	// 4) 落 run
	runRow := &entity.SkillTestRun{
		Scope:      scope,
		ProjectID:  projectID,
		Name:       name,
		Version:    version,
		Status:     report.Status,
		Trigger:    trigger,
		Summary:    report.Summary,
		StartedAt:  report.StartedAt,
		FinishedAt: report.FinishedAt,
	}
	created, err := s.runModel().Create(runRow)
	if err != nil {
		return nil, ErrDBPersist
	}

	// 5) 落 result(check 一条一行)
	results := make([]*entity.SkillTestResult, 0, len(report.Results))
	for _, r := range report.Results {
		cres, err := s.resultModel().Create(&entity.SkillTestResult{
			RunID:   created.ID,
			Check:   r.Check,
			Status:  r.Status,
			Message: r.Message,
			Detail:  r.Detail,
		})
		if err != nil {
			// 容忍:result 落库失败不影响主流程(以 run 状态为准)
			continue
		}
		results = append(results, cres)
	}
	return &RunResult{Run: created, Results: results}, nil
}

// ListRequest 列表入参。
type ListRequest struct {
	Scope     string
	ProjectID uint
	Name      string // 0 = 全部
	Page      int
	Size      int
}

// ListResult 列表结果。
type ListResult struct {
	Items []*entity.SkillTestRun `json:"items"`
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
}

// List 列出测试 run(可选按 scope+name 过滤)。
func (s *Service) List(req *ListRequest) (*ListResult, error) {
	var conds []*where.Condition
	if sc := strings.TrimSpace(req.Scope); sc != "" {
		conds = append(conds, where.New(mskilltestrun.FieldScope, "=", sc).Conditions()...)
	}
	if n := strings.TrimSpace(req.Name); n != "" {
		conds = append(conds, where.New(mskilltestrun.FieldName, "=", n).Conditions()...)
	}
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	items, total, err := s.runModel().FindList(conds, &where.Extra{
		PageNum:       page,
		PageSize:      size,
		OrderByColumn: mskilltestrun.FieldStartedAt,
		OrderByDesc:   true,
	})
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(total), Page: page, Size: size}, nil
}

// Detail 拿 run + 关联 results。
type Detail struct {
	Run     *entity.SkillTestRun      `json:"run"`
	Results []*entity.SkillTestResult `json:"results"`
}

// Get 拿 run 详情。
func (s *Service) Get(id uint) (*Detail, error) {
	row, err := s.runModel().FindOneById(id)
	if err != nil {
		return nil, ErrNotFound
	}
	results, _, err := s.resultModel().FindList(
		where.New(mskilltestresult.FieldRunID, "=", id).Conditions(),
		&where.Extra{OrderByColumn: mskilltestresult.FieldID, OrderByDesc: false},
	)
	if err != nil {
		return nil, err
	}
	return &Detail{Run: row, Results: results}, nil
}

// buildWalker 准备 AI 走查所需的闭包:把 ai_providers 行转成 aiengine.Config,按 priority 排序。
func (s *Service) buildWalker(providerName string) *skilltester.AIWalker {
	if s.mgr == nil || s.st == nil {
		return nil
	}
	rows, _, err := s.aiModel().FindList(nil, nil)
	if err != nil || len(rows) == 0 {
		return &skilltester.AIWalker{
			Providers: nil,
			Secret:    s.secretForAI(),
			Build:     s.buildForAI(),
		}
	}
	// 按 priority 升序,disabled 排除
	cands := make([]*entity.AIProvider, 0, len(rows))
	for _, r := range rows {
		if r.Enabled {
			cands = append(cands, r)
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].Priority != cands[j].Priority {
			return cands[i].Priority < cands[j].Priority
		}
		return cands[i].Name < cands[j].Name
	})
	cfgs := make([]*aiengine.Config, 0, len(cands))
	for _, r := range cands {
		cfgs = append(cfgs, &aiengine.Config{Name: r.Name, Kind: r.Kind, BaseURL: r.BaseURL, Model: r.Model})
	}
	return &skilltester.AIWalker{
		Providers: cfgs,
		Secret:    s.secretForAI(),
		Build:     s.buildForAI(),
	}
}

// secretForAI 拿 provider name 对应的 api key(settings KV)。
func (s *Service) secretForAI() func(string) (string, error) {
	return func(name string) (string, error) {
		v, _, err := s.st.Get("ai:" + name + ":api_key")
		return v, err
	}
}

// buildForAI 走 aiengine.Manager 构造 Provider。
func (s *Service) buildForAI() func(aiengine.Config) (aiengine.Provider, error) {
	return func(cfg aiengine.Config) (aiengine.Provider, error) {
		if s.mgr == nil {
			return nil, errors.New("aiengine: manager is nil")
		}
		return s.mgr.BuildFromConfig(cfg)
	}
}

// NewManagerForTester 构造一个绑定了 settings SecretStore 的 aiengine.Manager。
// 给 cskilltest 在没有完整 sai 依赖时复用。
func NewManagerForTester(st *settings.Service) *aiengine.Manager {
	if st == nil {
		return aiengine.NewManager(nil)
	}
	return aiengine.NewManager(secretAdapterForTester{s: st})
}

type secretAdapterForTester struct{ s *settings.Service }

func (a secretAdapterForTester) Resolve(name string) (string, error) {
	v, _, err := a.s.Get("ai:" + name + ":api_key")
	return v, err
}
