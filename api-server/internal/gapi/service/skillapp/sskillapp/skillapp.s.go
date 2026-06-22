// Package sskillapp 提供 Skill Apply / Undo / 批量 / 更新检测 的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.3 + 5.1 节):
//   - Apply / Undo 走 internal/skillapp.Applier + 写 entity.SkillApply
//   - 批量走 BatchApplier,任一失败原子回滚
//   - 更新检测:本地 skill 列表 vs market_skills 缓存,输出 UpdateItem 列表
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
	mskill "ginp-api/internal/gapi/model/skillbox/mskill"
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
func (s *Service) skillModel() *mskill.Model {
	return mskill.NewModel(s.dbWrite, s.dbRead)
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

// audit 内部 helper:把关键事件落 audit_log。
// actor 暂用 "system"(P1 接入登录后改成当前用户);payload 走 JSON 字符串。
func (s *Service) audit(action string, targetID uint, payload any) {
	if s.dbWrite == nil {
		return
	}
	payloadStr := ""
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			payloadStr = string(b)
		}
	}
	_, _ = saudit.New(s.dbWrite, s.dbRead).Write(saudit.WriteInput{
		Actor:      "system",
		Action:     action,
		TargetType: "skill",
		TargetID:   targetID,
		Payload:    payloadStr,
	})
}

// ApplyInput 单 skill apply 入参。
type ApplyInput struct {
	SkillID   uint     `json:"skill_id"`
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Tools     []string `json:"tools"`
}

// ApplyResult 单 skill apply 结果(多 tool)。
type ApplyResult struct {
	SkillID   uint                    `json:"skill_id"`
	SkillName string                  `json:"skill_name"`
	Applies   []*skillapp.ApplyResult `json:"applies"`
	AllOK     bool                    `json:"all_ok"`
}

// Apply 跑一次 apply:把 sskill 的 full skill 拉出来,逐 tool apply。
func (s *Service) Apply(in *ApplyInput) (*ApplyResult, error) {
	if in == nil {
		return nil, ErrSkillNotFound
	}
	if in.SkillID == 0 {
		return nil, fmt.Errorf("%w: skill_id 必填", ErrSkillNotFound)
	}
	if len(in.Tools) == 0 {
		return nil, ErrEmptyTools
	}
	full, err := s.loadFull(in.SkillID)
	if err != nil {
		return nil, err
	}
	applier := s.applier()
	out := &ApplyResult{SkillID: in.SkillID, SkillName: full.Canonical.Manifest.Name, AllOK: true}
	for _, tool := range in.Tools {
		scope := in.Scope
		if scope == "" {
			scope = full.Skill.Scope
		}
		res, err := applier.ApplyOne(skillapp.ApplyInput{
			SkillID:   in.SkillID,
			Scope:     scope,
			ProjectID: in.ProjectID,
			Tools:     []string{tool},
			Canonical: &full.Canonical,
		})
		if err != nil {
			out.AllOK = false
			s.audit("apply_failed", in.SkillID, map[string]any{
				"tool":   tool,
				"scope":  scope,
				"error":  err.Error(),
			})
		}
		if res != nil {
			row := &entity.SkillApply{
				SkillID:     in.SkillID,
				Tool:        tool,
				Status:      res.Status,
				TargetPath:  res.TargetPath,
				PreSnapshot: res.PreSnapshot.Marshal(),
				AppliedAt:   res.FinishedAt,
			}
			created, _ := s.applyModel().Create(row)
			if created != nil && res.Status == skillapp.StatusSuccess {
				s.audit("apply", in.SkillID, map[string]any{
					"tool":         tool,
					"scope":        scope,
					"target_path":  res.TargetPath,
					"apply_id":     created.ID,
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
func (s *Service) BatchApply(in *BatchApplyInput) (*skillapp.BatchOutput, error) {
	if in == nil || len(in.Items) == 0 {
		return &skillapp.BatchOutput{AllOK: true}, nil
	}
	items := make([]skillapp.BatchItem, 0, len(in.Items)*3)
	for _, it := range in.Items {
		full, err := s.loadFull(it.SkillID)
		if err != nil {
			return nil, err
		}
		for _, tool := range it.Tools {
			scope := it.Scope
			if scope == "" {
				scope = full.Skill.Scope
			}
			items = append(items, skillapp.BatchItem{
				SkillID:   it.SkillID,
				SkillName: full.Canonical.Manifest.Name,
				Scope:     scope,
				ProjectID: it.ProjectID,
				Tool:      tool,
				Canonical: &full.Canonical,
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
			SkillID:     bir.SkillID,
			Tool:        bir.Tool,
			Status:      bir.Result.Status,
			TargetPath:  bir.Result.TargetPath,
			PreSnapshot: bir.Result.PreSnapshot.Marshal(),
			AppliedAt:   bir.Result.FinishedAt,
		}
		_, _ = s.applyModel().Create(row)
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
		return nil, fmt.Errorf("skillapp: undo file: %w", err)
	}
	now := time.Now()
	row.Status = skillapp.StatusRolledBack
	row.RolledBackAt = &now
	if err := s.applyModel().Update(where.New(mskillapply.FieldID, "=", row.ID).Conditions(), row); err != nil {
		return nil, fmt.Errorf("skillapp: update apply row: %w", err)
	}
	return &UndoResult{ApplyID: applyID, NewStatus: row.Status, RolledBackAt: now}, nil
}

// ListInput 列表过滤。
type ListInput struct {
	SkillID uint
	Tool    string
	Status  string
	Page    int
	Size    int
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
	if in.SkillID > 0 {
		conds = append(conds, where.New(mskillapply.FieldSkillID, "=", in.SkillID).Conditions()...)
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

// CheckUpdates 拿本地 vs 市场。
func (s *Service) CheckUpdates(scope string, projectID uint) ([]skillapp.UpdateItem, error) {
	conds := []*where.Condition{}
	if scope = strings.ToLower(strings.TrimSpace(scope)); scope != "" {
		conds = append(conds, where.New(mskill.FieldScope, "=", scope).Conditions()...)
	}
	if projectID > 0 {
		conds = append(conds, where.New(mskill.FieldProjectID, "=", projectID).Conditions()...)
	}
	extra := &where.Extra{PageNum: 1, PageSize: 10000}
	local, _, err := s.skillModel().FindList(conds, extra)
	if err != nil {
		return nil, err
	}
	mkt, _, err := s.marketSkillModel().FindList(nil, &where.Extra{PageNum: 1, PageSize: 10000})
	if err != nil {
		return nil, err
	}
	return s.updater.CheckUpdates(local, mkt), nil
}

// loadFull 走 sskill 拿 full skill(含 canonical files)。
func (s *Service) loadFull(skillID uint) (*sskill.FullSkill, error) {
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("skillapp: skillSvcFactory not wired")
	}
	ssvc, err := s.skillSvcFactory()
	if err != nil {
		return nil, err
	}
	row, err := s.skillModel().FindOneById(skillID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrSkillNotFound, skillID)
	}
	version := row.Version
	if version == "" {
		version = "0.1.0"
	}
	return ssvc.GetFull(row.Scope, row.Name, version, row.ProjectID)
}

// Suppress unused imports.
var (
	_ = context.Background
	_ = mmarketsource.NewModel
	_ = entity.SkillApply{}
)
