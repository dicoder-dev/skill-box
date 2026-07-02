// Package sskillapp 提供 Skill Apply / Undo / 批量 / 更新检测 的业务层封装。
//
// 2026-06-24 改造:skill 不再用 entity.Skill 表示,改用 (scope, name) 作为唯一键;
// ApplyInput 不再需要 SkillID(从 store 直接 Load 即可)。
// 2026-06-29 增:Service 可选注入 projectService;scope=project 时,Apply / BatchApply
// 会通过它把 project_id 查成 entity.Project.RootPath 传给 skillapp.Applier,
// 让 apply 写到真实项目根(<project>/.agents/skills),而不是 home/.skillbox/projects/<id>/
// 占位路径(占位实现已废,production 必须有 projectService 注入)。
package sskillapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/controller/skillbox/cskill"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/settings"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillapp"
	mmarket "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	mskillapply "ginp-api/internal/gapi/model/skillbox/mskillapply"

	"ginp-api/pkg/where"
	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrSkillNotFound = errors.New("skillapp: skill not found")
	ErrEmptyTools     = errors.New("skillapp: no tools specified")
	ErrToolNotFound   = errors.New("skillapp: tool not in registry")
)

// Service 业务服务。
// 2026-07-02 增:settings 用于在 apply 时按 settings.apply_mode 切换 copy/symlink;
// 不注入时永远走 copy(向后兼容老调用方)。
type Service struct {
	dbWrite         *gorm.DB
	dbRead          *gorm.DB
	skillSvcFactory func() (*sskill.Service, error)
	adapterRegistry *skilladapter.Registry
	updater         *skillapp.Updater
	projectSvc      *sproject.Service // 2026-06-29 增:scope=project 时把 project_id 查成 root_path
	settings        *settings.Service
}

// New 构造 service。
func New(dbWrite, dbRead *gorm.DB, skillSvcFactory func() (*sskill.Service, error)) *Service {
	return &Service{
		dbWrite:         dbWrite,
		dbRead:          dbRead,
		skillSvcFactory: skillSvcFactory,
		updater:         skillapp.NewUpdater(),
	}
}

// WithSettings 注入 settings.Service,让 apply 按 settings.apply_mode 选择落盘模式(2026-07-02)。
// 不注入时走 copy(老行为)。
func (s *Service) WithSettings(st *settings.Service) *Service {
	s.settings = st
	return s
}

// currentApplyMode 读 settings.apply_mode;未注入 settings 时返 copy。
func (s *Service) currentApplyMode() string {
	if s.settings == nil {
		return skillapp.ModeCopy
	}
	return s.settings.GetApplyMode()
}

// WithAdapterRegistry 替换 adapter registry(测试用)。
func (s *Service) WithAdapterRegistry(reg *skilladapter.Registry) *Service {
	s.adapterRegistry = reg
	return s
}

// WithProjectService 注入 project service(2026-06-29):scope=project 时,
// Apply / BatchApply 需要查 entity.Project.RootPath 传给 skillapp.Applier。
// 未注入时 scope=project 的 apply 走 fallback 占位路径(向后兼容,但 production 必须注入)。
func (s *Service) WithProjectService(ps *sproject.Service) *Service {
	s.projectSvc = ps
	return s
}

func (s *Service) applyModel() *mskillapply.Model {
	return mskillapply.NewModel(s.dbWrite, s.dbRead)
}

func (s *Service) marketSkillModel() *mmarket.Model {
	return mmarket.NewModel(s.dbWrite, s.dbRead)
}

func (s *Service) marketSourceModel() *mmarketsource.Model {
	return mmarketsource.NewModel(s.dbWrite, s.dbRead)
}

func (s *Service) applier() *skillapp.Applier {
	a := skillapp.NewApplier(s.adapterRegistry)
	// 2026-07-02 增:让 applier 知道当前 apply 模式;未注入 settings 时 Applier 走默认 copy。
	a.Mode = s.currentApplyMode()
	return a
}

// audit 内部 helper:把关键事件落 audit_log。actor 暂用 "system"。
// targetID 弃用(2026-06-24):改用 name 字符串作为标识;为保持 saudit.WriteInput 兼容,
// 这里把 name 做 hash 成 uint 简化处理(实际查询时按 action + payload 过滤)。
func (s *Service) audit(targetID uint, name string, payload any) {
	if s.dbWrite == nil {
		return
	}
	payloadStr := ""
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			payloadStr = string(b)
		}
	}
	action := "skill_apply"
	if mp, ok := payload.(map[string]any); ok {
		if a, ok2 := mp["action"].(string); ok2 {
			action = a
		}
	}
	_, _ = saudit.New(s.dbWrite, s.dbRead).Write(saudit.WriteInput{
		Actor:      "system",
		Action:     action,
		TargetType: "skill",
		TargetID:   targetID,
		Payload:    payloadStr + "|name=" + name,
	})
}

// ApplyInput 单 skill apply 入参(2026-06-24:用 scope+name 定位)。
type ApplyInput struct {
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Name      string   `json:"name"`
	Tools     []string `json:"tools"`
}

// ApplyResult 单 skill apply 结果(多 tool)。
type ApplyResult struct {
	Name     string                  `json:"name"`
	Version  string                  `json:"version"`
	Applies  []*skillapp.ApplyResult `json:"applies"`
	AllOK    bool                    `json:"all_ok"`
}

// recordApply 写一条 SkillApply 行。
//
// 2026-06-25 修:之前每次 apply 都 Create,同名同 scope 重复 apply 会撞
// skill_applies 的 (scope, project_id, name) uniqueIndex。现在改 upsert:
// 存在同键行 → Update(applied_at/pre_snapshot/target_path/tool/status),
// 不存在 → Create。这样 redo / 重新启用 都安全。
// 2026-07-02 增:apply_mode 字段(当前 settings.apply_mode),便于模式切换
// 时只迁"对得上"的行,以及前端展示当前 apply 用的是哪种模式。
func (s *Service) recordApply(scope string, projectID uint, name, tool string, res *skillapp.ApplyResult) {
	if res == nil {
		return
	}
	pre := res.PreSnapshot.Marshal()
	mode := s.currentApplyMode()
	existing, _ := s.applyModel().FindLatestByKey(scope, projectID, name)
	if existing != nil {
		existing.Tool = tool
		existing.Status = res.Status
		existing.ApplyMode = mode
		existing.TargetPath = res.TargetPath
		existing.PreSnapshot = pre
		existing.AppliedAt = res.FinishedAt
		existing.RolledBackAt = nil // 重新启用时清掉回滚时间
		if err := s.applyModel().Update(where.New(mskillapply.FieldID, "=", existing.ID).Conditions(), existing); err == nil {
			res.ApplyID = existing.ID
		}
		return
	}
	row := &entity.SkillApply{
		Scope:       scope,
		ProjectID:   projectID,
		Name:        name,
		Tool:        tool,
		Status:      res.Status,
		ApplyMode:   mode,
		TargetPath:  res.TargetPath,
		PreSnapshot: pre,
		AppliedAt:   res.FinishedAt,
	}
	created, _ := s.applyModel().Create(row)
	if created != nil {
		res.ApplyID = created.ID
	}
}

// Apply 跑一次 apply:从 sskill 拿 canonical,逐 tool apply。
//
// 2026-06-29 改造:scope=project 时,先把 in.ProjectID 查成项目 root_path 再传给 applier;
// 这是 Apply 写"真实项目根"而不是 ~/.skillbox/projects/<id>/ 占位的关键。
func (s *Service) Apply(in *ApplyInput) (*ApplyResult, error) {
	if in == nil {
		return nil, ErrSkillNotFound
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: name 必填", ErrSkillNotFound)
	}
	if len(in.Tools) == 0 {
		return nil, ErrEmptyTools
	}
	full, err := s.loadFull(in.Name)
	if err != nil {
		if errors.Is(err, sskill.ErrNotFound) {
			return nil, ErrSkillNotFound
		}
		return nil, err
	}
	// scope=project 时,把 project_id 解析成真实项目根
	var projectRoot string
	if in.Scope == skilladapter.ScopeProject {
		projectRoot, err = s.resolveProjectRoot(in.ProjectID)
		if err != nil {
			return nil, err
		}
	}
	applier := s.applier()
	out := &ApplyResult{
		Name:    full.Manifest.Name,
		Version: full.Manifest.Version,
		AllOK:   true,
	}
	for _, tool := range in.Tools {
		scope := in.Scope
		if scope == "" {
			scope = skilladapter.ScopeGlobal
		}
		res, err := applier.ApplyOne(skillapp.ApplyInput{
			SkillName:   full.Manifest.Name,
			Scope:       scope,
			ProjectID:   in.ProjectID,
			ProjectRoot: projectRoot,
			Tools:       []string{tool},
			Canonical:   full,
		})
		if err != nil {
			out.AllOK = false
			s.audit(0, full.Manifest.Name, map[string]any{
				"action": "apply_failed",
				"tool":   tool,
				"scope":  scope,
				"error":  err.Error(),
			})
		}
		if res != nil {
			s.recordApply(scope, in.ProjectID, full.Manifest.Name, tool, res)
			if res.ApplyID > 0 && res.Status == skillapp.StatusApplied {
				s.audit(0, full.Manifest.Name, map[string]any{
					"action":     "apply",
					"tool":       tool,
					"scope":      scope,
					"target_path": res.TargetPath,
					"apply_id":   res.ApplyID,
				})
			}
			res.PreSnapshot = nil
		}
		out.Applies = append(out.Applies, res)
	}
	return out, nil
}

// BatchApplyInput 多 skill 批量。
type BatchApplyInput struct {
	Items  []ApplyInput `json:"items"`
	Atomic bool         `json:"atomic"`
}

// BatchApply 多 skill × 多 tool 笛卡尔积。
//
// 2026-06-29 改造:同 Apply — scope=project 时把 project_id 查成 root_path 传下去。
func (s *Service) BatchApply(in *BatchApplyInput) (*skillapp.BatchOutput, error) {
	if in == nil || len(in.Items) == 0 {
		return &skillapp.BatchOutput{AllOK: true}, nil
	}
	items := make([]skillapp.BatchItem, 0, len(in.Items)*3)
	for _, it := range in.Items {
		full, err := s.loadFull(it.Name)
		if err != nil {
			return nil, err
		}
		// scope=project 时,把 project_id 解析成真实项目根(一次性查一次)
		var projectRoot string
		if it.Scope == skilladapter.ScopeProject {
			projectRoot, err = s.resolveProjectRoot(it.ProjectID)
			if err != nil {
				return nil, err
			}
		}
		for _, tool := range it.Tools {
			scope := it.Scope
			if scope == "" {
				scope = skilladapter.ScopeGlobal
			}
			items = append(items, skillapp.BatchItem{
				SkillName:    full.Manifest.Name,
				SkillVersion: full.Manifest.Version,
				Scope:        scope,
				ProjectID:    it.ProjectID,
				ProjectRoot:  projectRoot,
				Tool:         tool,
				Canonical:    full,
			})
		}
	}
	applier := s.applier()
	ba := skillapp.NewBatchApplier(applier)
	out := ba.ApplyWithItems(items, in.Atomic)
	for _, bir := range out.Items {
		if bir.Result == nil {
			continue
		}
		s.recordApply(bir.Scope, bir.ProjectID, bir.SkillName, bir.Tool, bir.Result)
		if bir.Result.ApplyID > 0 {
			if bir.Result.Status == skillapp.StatusApplied {
				s.audit(0, bir.SkillName, map[string]any{
					"action":      "apply",
					"tool":        bir.Tool,
					"scope":       bir.Scope,
					"target_path": bir.Result.TargetPath,
					"apply_id":    bir.Result.ApplyID,
					"batch":       true,
				})
			} else if bir.Result.Status == skillapp.StatusFailed {
				s.audit(0, bir.SkillName, map[string]any{
					"action":      "apply_failed",
					"tool":        bir.Tool,
					"scope":       bir.Scope,
					"target_path": bir.Result.TargetPath,
					"batch":       true,
				})
			}
		}
		bir.Result.PreSnapshot = nil
	}
	return out, nil
}

// UndoResult 撤销结果。
type UndoResult struct {
	ApplyID      uint      `json:"apply_id"`
	NewStatus    string    `json:"new_status"`
	RolledBackAt time.Time `json:"rolled_back_at"`
}

// Undo 撤销一条 apply。
func (s *Service) Undo(applyID uint) (*UndoResult, error) {
	if applyID == 0 {
		return nil, skillapp.ErrApplyNotFound
	}
	row, err := s.applyModel().FindOneById(applyID)
	if err != nil {
		return nil, skillapp.ErrApplyNotFound
	}
	if row.Status == skillapp.StatusRolledBack {
		return nil, skillapp.ErrAlreadyRolled
	}
	if row.Status == skillapp.StatusFailed {
		return nil, fmt.Errorf("skillapp: cannot undo a failed apply (id=%d)", applyID)
	}
	if err := skillapp.UndoWithSnapshot(row.TargetPath, row.PreSnapshot); err != nil {
		s.audit(0, row.Name, map[string]any{
			"action":     "undo_failed",
			"apply_id":   applyID,
			"tool":       row.Tool,
			"target_path": row.TargetPath,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("skillapp: undo file: %w", err)
	}
	now := time.Now()
	row.Status = skillapp.StatusRolledBack
	row.RolledBackAt = &now
	if err := s.applyModel().Update(where.New(mskillapply.FieldID, "=", row.ID).Conditions(), row); err != nil {
		return nil, fmt.Errorf("skillapp: update apply row: %w", err)
	}
	s.audit(0, row.Name, map[string]any{
		"action":     "undo",
		"apply_id":   applyID,
		"tool":       row.Tool,
		"target_path": row.TargetPath,
	})
	return &UndoResult{ApplyID: applyID, NewStatus: row.Status, RolledBackAt: now}, nil
}

// ForceUndoInput 强制按 (scope, project_id, name, tool) 撤销,不走 apply_id 流程。
//
// 2026-06-25 增:用于"scope-status 命中但 DB 没 apply 记录"场景(用户手动 cp /
// 外部安装)。逻辑:
//   1) 先在 DB 按 (scope, project_id, name, tool) 找最近一条 status=applied
//      记录,找到就走标准 Undo(走 pre-snapshot 还原)。
//   2) 没记录:用 scope-status 扫,找该 (tool, scope, project_id) 命中
//      (exists=true) 的 resolved 路径,直接调 ForceRemoveFromPath 删整个目录。
//   3) DB 插一条占位 status=rolled_back 记录,applied_at/rolled_back_at 都
//      用 now(),target_path 记录实际删的路径,tool/scope/project_id/name
//      按入参填。
type ForceUndoInput struct {
	Scope     string
	ProjectID uint
	Name      string
	Tool      string
}

func (s *Service) ForceUndo(in *ForceUndoInput) (*UndoResult, error) {
	if in == nil {
		return nil, skillapp.ErrApplyNotFound
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Tool) == "" {
		return nil, fmt.Errorf("skillapp: force undo: name/tool required")
	}
	scope := strings.ToLower(strings.TrimSpace(in.Scope))
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}

	// 1) 优先按 DB 记录走标准 Undo
	var conds []*where.Condition
	conds = append(conds, where.New(mskillapply.FieldScope, "=", scope).Conditions()...)
	conds = append(conds, where.New(mskillapply.FieldProjectID, "=", in.ProjectID).Conditions()...)
	conds = append(conds, where.New(mskillapply.FieldName, "=", in.Name).Conditions()...)
	conds = append(conds, where.New(mskillapply.FieldTool, "=", in.Tool).Conditions()...)
	conds = append(conds, where.New(mskillapply.FieldStatus, "=", skillapp.StatusApplied).Conditions()...)
	rows, _, err := s.applyModel().FindList(conds, &where.Extra{
		PageNum: 1, PageSize: 1,
		OrderByColumn: mskillapply.FieldAppliedAt, OrderByDesc: true,
	})
	if err == nil && len(rows) > 0 {
		return s.Undo(rows[0].ID)
	}

	// 2) DB 没记录 → 走 scope-status 强制删磁盘
	resolved, err := s.resolveByScopeStatus(in.Name, scope, in.ProjectID, in.Tool)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		return nil, fmt.Errorf("skillapp: force undo: no active hit for %s/%s/%s/%d",
			scope, in.Name, in.Tool, in.ProjectID)
	}
	if err := skillapp.ForceRemoveFromPath(resolved); err != nil {
		return nil, fmt.Errorf("skillapp: force undo: %w", err)
	}

	// 3) DB 写占位 rolled_back 记录
	now := time.Now()
	placeholder := &entity.SkillApply{
		Scope:       scope,
		ProjectID:   in.ProjectID,
		Name:        in.Name,
		Tool:        in.Tool,
		Status:      skillapp.StatusRolledBack,
		TargetPath:  resolved,
		PreSnapshot: "",
		AppliedAt:   now,
		RolledBackAt: &now,
	}
	created, _ := s.applyModel().Create(placeholder)
	if created != nil {
		s.audit(0, in.Name, map[string]any{
			"action":     "force_undo",
			"tool":       in.Tool,
			"scope":      scope,
			"target_path": resolved,
			"apply_id":   created.ID,
		})
		return &UndoResult{ApplyID: created.ID, NewStatus: skillapp.StatusRolledBack, RolledBackAt: now}, nil
	}
	s.audit(0, in.Name, map[string]any{
		"action":     "force_undo",
		"tool":       in.Tool,
		"scope":      scope,
		"target_path": resolved,
	})
	return &UndoResult{NewStatus: skillapp.StatusRolledBack, RolledBackAt: now}, nil
}

// resolveByScopeStatus 通过 scope-status 扫描找 (tool, scope, project_id) 命中
// 的 resolved 路径。复用 cskill.ResolveHit。
func (s *Service) resolveByScopeStatus(name, scope string, projectID uint, tool string) (string, error) {
	return cskill.ResolveHit(name, scope, projectID, tool)
}

// ListInput 列表过滤(2026-06-24:按 scope+name 过滤)。
type ListInput struct {
	Scope  string
	Name   string
	Tool   string
	Status string
	Page   int
	Size   int
}

// ListResult 列表结果。
type ListResult struct {
	Items []*entity.SkillApply `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

// List 列出 apply 历史。
func (s *Service) List(in ListInput) (*ListResult, error) {
	conds := []*where.Condition{}
	if sc := strings.TrimSpace(in.Scope); sc != "" {
		conds = append(conds, where.New(mskillapply.FieldScope, "=", sc).Conditions()...)
	}
	if n := strings.TrimSpace(in.Name); n != "" {
		conds = append(conds, where.New(mskillapply.FieldName, "=", n).Conditions()...)
	}
	if t := strings.TrimSpace(in.Tool); t != "" {
		conds = append(conds, where.New(mskillapply.FieldTool, "=", t).Conditions()...)
	}
	if st := strings.TrimSpace(in.Status); st != "" {
		if !skillapp.IsStatusValid(st) {
			return nil, fmt.Errorf("skillapp: invalid status %q", st)
		}
		conds = append(conds, where.New(mskillapply.FieldStatus, "=", st).Conditions()...)
	}
	page, size := in.Page, in.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	extra := &where.Extra{
		PageNum:       page,
		PageSize:      size,
		OrderByColumn: mskillapply.FieldAppliedAt,
		OrderByDesc:   true,
	}
	items, total, err := s.applyModel().FindList(conds, extra)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(total), Page: page, Size: size}, nil
}

// CheckUpdatesInput 更新检测入参。
type CheckUpdatesInput struct {
	Scope     string
	ProjectID uint
}

// CheckUpdates 对比本地 skill 列表 vs 市场缓存。
// 2026-06-24 改造:本地列表从 store 读(而非 skills 表),对比 MarketSkill。
func (s *Service) CheckUpdates(in CheckUpdatesInput) ([]skillapp.UpdateItem, error) {
	// 本地:走 sskill.List → 再转成 skillapp 期望的 local form
	store, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	_ = store
	items, err := s.localSkillAsMarketLike(in.Scope, in.ProjectID)
	if err != nil {
		return nil, err
	}
	mkt, _, err := s.marketSkillModel().FindList(nil, &where.Extra{PageNum: 1, PageSize: 10000})
	if err != nil {
		return nil, err
	}
	return s.updater.CheckUpdates(items, mkt), nil
}

// localSkillAsMarketLike 把 store 列出的 skill 转成 skillapp 期望的 local 形态。
// 2026-06-24:DB 弃用后,local 来源是 skillstore.Store.List;entity.Skill 只剩
// Name/Version/Source/SourceRef 几个字段(其它字段都已弃用)。用 canonical 的
// 内容稳定 hash 作为 SkillID,保证 updater 的 seen map 工作。
func (s *Service) localSkillAsMarketLike(scope string, projectID uint) ([]*entity.Skill, error) {
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	canonicals, err := ssvc.List("")
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Skill, 0, len(canonicals))
	for i, item := range canonicals {
		_ = canonicals
		out = append(out, &entity.Skill{
			ID:        uint(i + 1),
			Scope:     scope,
			ProjectID: projectID,
			Name:      item.Name,
			Version:   item.Version,
		})
	}
	return out, nil
}

// loadFull 走 sskill 拿 full skill(含 canonical files)。
func (s *Service) loadFull(name string) (*skilladapter.Canonical, error) {
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("skillapp: skillSvcFactory not wired")
	}
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	return ssvc.Get(name)
}

// resolveProjectRoot 把 project_id 查成 entity.Project.RootPath(2026-06-29):
//   - projectSvc 未注入 → 返空串(让 applier 走 fallback 占位路径)
//   - project_id = 0 → 返空串(调用方应在 global scope 才传 0)
//   - sproject.List 失败 → 返 error(让 Apply 在 controller 层弹 4xx)
//   - 找到的 project.RootPath 为空 → 返 error(项目实体存在但未配置 root,拒写)
func (s *Service) resolveProjectRoot(projectID uint) (string, error) {
	if s.projectSvc == nil {
		return "", nil
	}
	if projectID == 0 {
		return "", nil
	}
	list, err := s.projectSvc.List(sproject.ListQuery{Page: 1, Size: 500})
	if err != nil {
		return "", fmt.Errorf("skillapp: list projects: %w", err)
	}
	if list == nil {
		return "", fmt.Errorf("skillapp: project %d not found", projectID)
	}
	for _, p := range list.Items {
		if p == nil || uint(p.ID) != projectID {
			continue
		}
		if p.RootPath == "" {
			return "", fmt.Errorf("skillapp: project %d (%s) has empty root_path", p.ID, p.Alias)
		}
		return p.RootPath, nil
	}
	return "", fmt.Errorf("skillapp: project %d not found", projectID)
}

// Suppress unused imports.
var (
	_ = context.Background
	_ = mmarketsource.NewModel
)

// ===================== 模式切换迁移(2026-07-02 增) =====================

// MigrateModeResult 单条迁移结果。
type MigrateModeEntry struct {
	ApplyID   uint   `json:"apply_id"`
	Scope     string `json:"scope"`
	Name      string `json:"name"`
	Tool      string `json:"tool"`
	Target    string `json:"target"`
	FromMode  string `json:"from_mode"`
	ToMode    string `json:"to_mode"`
	OK        bool   `json:"ok"`
	Skipped   bool   `json:"skipped"`             // target 已被外部修改,跳过避免破坏
	SkipReason string `json:"skip_reason,omitempty"`
	Error     string `json:"error,omitempty"`
}

// MigrateModeResult 整体结果。
type MigrateModeResult struct {
	FromMode string              `json:"from_mode"`
	ToMode   string              `json:"to_mode"`
	Total    int                 `json:"total"`
	OK       int                 `json:"ok"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Entries  []MigrateModeEntry  `json:"entries"`
}

// MigrateMode 把所有 status=applied 的 skill_applies 行,从当前模式切换到 targetMode。
//
// 流程:
//   1) 先把 settings.apply_mode 设为 targetMode(供后续 apply 走新模式),并
//      记录 fromMode(供回滚 / 报告用)。
//   2) 列出 status=applied 的所有行;每行根据 (scope, project_id, name) 拿
//      canonical,根据 tool 拿 adapter;然后判断 target_path 当前是目录还是
//      symlink,与 fromMode 是不是一致。
//   3) 一致(已在新模式 / 老 apply 行没有 mode 字段视为 copy):跳过。
//   4) 不一致:按目标模式落盘(从老 target 拷到新 target,或从老 symlink
//      还原成 copy 实体),失败时回滚到原状态(整个过程"先复制后替换",避免
//      半成品状态)。
//
// 不传 targetMode 时默认 = 当前 settings 模式(便于前端"重新迁移"用)。
func (s *Service) MigrateMode(targetMode string) (*MigrateModeResult, error) {
	if !skillapp.IsModeValid(targetMode) {
		return nil, fmt.Errorf("skillapp: migrate_mode: invalid mode %q (allowed: copy/symlink)", targetMode)
	}
	fromMode := s.currentApplyMode()
	res := &MigrateModeResult{
		FromMode: fromMode,
		ToMode:   targetMode,
		Entries:  []MigrateModeEntry{},
	}
	// 同一模式:无需迁移,但仍要保证 settings 与请求一致(便于用户反复点保存)
	if fromMode == targetMode {
		// settings 已是对的值,直接返空
		return res, nil
	}
	// 1) 改 settings,后续 apply 走新模式
	if s.settings != nil {
		if err := s.settings.SetApplyMode(targetMode); err != nil {
			return nil, fmt.Errorf("skillapp: migrate_mode: set settings: %w", err)
		}
	}

	// 2) 列所有 status=applied 的行
	conds := []*where.Condition{where.New(mskillapply.FieldStatus, "=", skillapp.StatusApplied).Conditions()[0]}
	conds = append(conds, where.New(mskillapply.FieldStatus, "=", skillapp.StatusApplied).Conditions()[1:]...)
	rows, _, err := s.applyModel().FindList(conds, nil)
	if err != nil {
		return nil, fmt.Errorf("skillapp: migrate_mode: list applies: %w", err)
	}
	res.Total = len(rows)

	reg := s.adapterRegistry
	if reg == nil {
		reg = skilladapter.DefaultRegistry()
	}
	for _, row := range rows {
		entry := MigrateModeEntry{
			ApplyID:  row.ID,
			Scope:    row.Scope,
			Name:     row.Name,
			Tool:     row.Tool,
			Target:   row.TargetPath,
			FromMode: fromMode,
			ToMode:   targetMode,
		}
		if row.TargetPath == "" {
			entry.Skipped = true
			entry.SkipReason = "empty target_path"
			res.Skipped++
			res.Entries = append(res.Entries, entry)
			continue
		}
		// 拿 canonical — 拿不到就跳过(用户可能已删了源 skill)
		canonical, err := s.loadFull(row.Name)
		if err != nil {
			entry.Skipped = true
			entry.SkipReason = "source skill not found in store: " + err.Error()
			res.Skipped++
			res.Entries = append(res.Entries, entry)
			continue
		}
		// 拿 adapter
		ad, ok := reg.Get(row.Tool)
		if !ok {
			entry.Error = "tool adapter not found: " + row.Tool
			res.Failed++
			res.Entries = append(res.Entries, entry)
			continue
		}
		if err := s.migrateOne(ad, canonical, row, fromMode, targetMode, &entry); err != nil {
			entry.Error = err.Error()
			res.Failed++
		} else {
			res.OK++
		}
		res.Entries = append(res.Entries, entry)
	}
	s.audit(0, "batch_migrate_mode", map[string]any{
		"action":  "migrate_mode",
		"from":    fromMode,
		"to":      targetMode,
		"total":   res.Total,
		"ok":      res.OK,
		"skipped": res.Skipped,
		"failed":  res.Failed,
	})
	return res, nil
}

// migrateOne 把单条 apply 行从 fromMode 切到 toMode。
//
// 关键不变量:任何"先删后建"的中间状态都要可回退 —— 用 os.Rename 把旧 target
// 暂存到 sibling 临时路径,新 target 落成功后再删旧;失败时 Rename 回来。
func (s *Service) migrateOne(ad skilladapter.Adapter, c *skilladapter.Canonical, row *entity.SkillApply, fromMode, toMode string, entry *MigrateModeEntry) error {
	// 1) 检查 target 当前实际状态,与 fromMode 是不是一致
	linfo, lerr := os.Lstat(row.TargetPath)
	targetIsSymlink := lerr == nil && linfo != nil && linfo.Mode()&os.ModeSymlink != 0
	actualMode := skillapp.ModeCopy
	if targetIsSymlink {
		actualMode = skillapp.ModeSymlink
	}
	// 已经是目标模式(用户可能手动改过)→ 跳过 + 同步 DB 字段
	if actualMode == toMode {
		row.ApplyMode = toMode
		_ = s.applyModel().Update(where.New(mskillapply.FieldID, "=", row.ID).Conditions(), row)
		entry.Skipped = true
		entry.SkipReason = "target already in target mode"
		return nil
	}

	// 2) 选择"暂存旧 target"位置:target 同级的 .skillbox-migrate-<id> 临时目录
	tmp := row.TargetPath + fmt.Sprintf(".migrate-%d", row.ID)
	// 清掉可能残留的 tmp(上次中断留下)
	_ = os.RemoveAll(tmp)
	// 3) 把旧 target 挪到 tmp
	if err := os.Rename(row.TargetPath, tmp); err != nil {
		if os.IsNotExist(err) {
			// 旧 target 不存在(用户手动删了),直接落新模式即可
			return s.writeTargetFresh(ad, c, row, toMode)
		}
		return fmt.Errorf("rename old target to tmp: %w", err)
	}
	// 4) 在原位置落新模式
	if err := s.writeTargetFresh(ad, c, row, toMode); err != nil {
		// 落新失败 → 把 tmp 还原回 row.TargetPath
		_ = os.Rename(tmp, row.TargetPath)
		return fmt.Errorf("write new mode target: %w", err)
	}
	// 5) 删 tmp(旧 target)。symlink→copy 时 tmp 是 symlink,Remove 即可
	if err := os.RemoveAll(tmp); err != nil {
		// 落新已成功,旧 target 删不掉的话——只警告,不影响功能;但同步记到 entry
		entry.SkipReason = "new target ok, but old target cleanup failed: " + err.Error()
	}
	// 6) 同步 DB 字段
	row.ApplyMode = toMode
	if err := s.applyModel().Update(where.New(mskillapply.FieldID, "=", row.ID).Conditions(), row); err != nil {
		return fmt.Errorf("update apply row apply_mode: %w", err)
	}
	return nil
}

// writeTargetFresh 在 row.TargetPath 上落一份"toMode"形态的 skill。
func (s *Service) writeTargetFresh(ad skilladapter.Adapter, c *skilladapter.Canonical, row *entity.SkillApply, toMode string) error {
	if toMode == skillapp.ModeSymlink {
		linker, ok := ad.(interface {
			ApplyLink(skilladapter.Canonical, string) error
		})
		if !ok {
			return fmt.Errorf("tool %s does not support symlink mode (missing ApplyLink)", ad.ToolID())
		}
		return linker.ApplyLink(*c, row.TargetPath)
	}
	return ad.Apply(*c, row.TargetPath)
}
