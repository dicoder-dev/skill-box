// Package sskillapp 提供 Skill Apply / Undo / 批量 / 更新检测 的业务层封装。
//
// 2026-06-24 改造:skill 不再用 entity.Skill 表示,改用 (scope, name) 作为唯一键;
// ApplyInput 不再需要 SkillID(从 store 直接 Load 即可)。
package sskillapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/internal/gapi/service/skill/sskill"
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
type Service struct {
	dbWrite         *gorm.DB
	dbRead          *gorm.DB
	skillSvcFactory func() (*sskill.Service, error)
	adapterRegistry *skilladapter.Registry
	updater         *skillapp.Updater
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

// WithAdapterRegistry 替换 adapter registry(测试用)。
func (s *Service) WithAdapterRegistry(reg *skilladapter.Registry) *Service {
	s.adapterRegistry = reg
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
	return skillapp.NewApplier(s.adapterRegistry)
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

// Apply 跑一次 apply:从 sskill 拿 canonical,逐 tool apply。
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
		return nil, err
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
			SkillName: full.Manifest.Name,
			Scope:     scope,
			ProjectID: in.ProjectID,
			Tools:     []string{tool},
			Canonical: full,
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
			row := &entity.SkillApply{
				Scope:       scope,
				ProjectID:   in.ProjectID,
				Name:        full.Manifest.Name,
				Tool:        tool,
				Status:      res.Status,
				TargetPath:  res.TargetPath,
				PreSnapshot: res.PreSnapshot.Marshal(),
				AppliedAt:   res.FinishedAt,
			}
			created, _ := s.applyModel().Create(row)
			if created != nil {
				res.ApplyID = created.ID
				if res.Status == skillapp.StatusApplied {
					s.audit(0, full.Manifest.Name, map[string]any{
						"action":     "apply",
						"tool":       tool,
						"scope":      scope,
						"target_path": res.TargetPath,
						"apply_id":   created.ID,
					})
				}
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
		for _, tool := range it.Tools {
			scope := it.Scope
			if scope == "" {
				scope = skilladapter.ScopeGlobal
			}
			items = append(items, skillapp.BatchItem{
				SkillName:  full.Manifest.Name,
				SkillVersion: full.Manifest.Version,
				Scope:      scope,
				ProjectID:  it.ProjectID,
				Tool:       tool,
				Canonical:  full,
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
		row := &entity.SkillApply{
			Scope:       bir.Scope,
			ProjectID:   bir.ProjectID,
			Name:        bir.SkillName,
			Tool:        bir.Tool,
			Status:      bir.Result.Status,
			TargetPath:  bir.Result.TargetPath,
			PreSnapshot: bir.Result.PreSnapshot.Marshal(),
			AppliedAt:   bir.Result.FinishedAt,
		}
		created, _ := s.applyModel().Create(row)
		if created != nil {
			bir.Result.ApplyID = created.ID
			if bir.Result.Status == skillapp.StatusApplied {
				s.audit(0, bir.SkillName, map[string]any{
					"action":      "apply",
					"tool":        bir.Tool,
					"scope":       bir.Scope,
					"target_path": bir.Result.TargetPath,
					"apply_id":    created.ID,
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

// localSkillAsMarketLike 把 sskill 列出的 skill 转成 skillapp.UpdateItem 期望的
// local 形态。skillapp.updater.CheckUpdates 期望的签名是 []*entity.Skill,
// 我们直接实现一个简化的 map:仅 name + version 字段。
func (s *Service) localSkillAsMarketLike(scope string, projectID uint) ([]*entity.Skill, error) {
	store, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	_ = store
	// 简化:用 sskill.List(keyword="") 列全部,然后过滤
	canonicals, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	_ = canonicals
	// 这里直接走 store 拉:用 NewStore 工厂
	fullStore, err := sskill.NewStore()
	if err != nil {
		return nil, err
	}
	cs, err := fullStore.List("")
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Skill, 0, len(cs))
	for _, c := range cs {
		out = append(out, &entity.Skill{
			Name:    c.Manifest.Name,
			Version: c.Manifest.Version,
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

// Suppress unused imports.
var (
	_ = context.Background
	_ = mmarketsource.NewModel
)
