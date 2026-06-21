// Package sskill 提供 Skill 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 6.2 节):
//   - Skill 同时落库(entity.Skill)+ 落盘(skillstore.Store);两边要保持一致
//   - scope + project_id + name + version 是唯一组合;version 缺省 0.1.0
//   - ManifestJSON 是冗余字段,方便列表/详情接口不出 N+1
//   - 编辑器场景:load = DB 拿到元数据 + skillstore.Load 拿全部文件
//   - 写场景:先 skillstore.Save(覆盖式)再写 DB;store 失败时 DB 不动
package sskill

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ginp-api/configs"
	"ginp-api/internal/gapi/entity"
	mskill "ginp-api/internal/gapi/model/skillbox/mskill"
	mskillfile "ginp-api/internal/gapi/model/skillbox/mskillfile"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误(sentinel),controller 可用 errors.Is 判断。
var (
	ErrEmptyName   = errors.New("skill: name is empty")
	ErrEmptyScope  = errors.New("skill: scope is empty")
	ErrInvalidScope = errors.New("skill: scope must be 'global' or 'project'")
	ErrNotFound    = errors.New("skill: not found")
	ErrStoreSave   = errors.New("skill: store save failed")
)

// Service 业务服务。dbWrite / dbRead / store 三件套。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	store   *skillstore.Store
}

func New(dbWrite, dbRead *gorm.DB, store *skillstore.Store) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, store: store}
}

func (s *Service) skillModel() *mskill.Model {
	return mskill.NewModel(s.dbWrite, s.dbRead)
}

func (s *Service) fileModel() *mskillfile.Model {
	return mskillfile.NewModel(s.dbWrite, s.dbRead)
}

// normalizeScope 把空 / 大写 scope 规范化成 'global' / 'project'。
func normalizeScope(scope string) (string, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return "", ErrEmptyScope
	}
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return "", ErrInvalidScope
	}
	return scope, nil
}

// Create / Update 入参(可同时传 canonical 字段和 file 列表)。
type WriteInput struct {
	Scope     string                 `json:"scope"`
	ProjectID uint                   `json:"project_id"`
	Name      string                 `json:"name"`
	Version   string                 `json:"version"`
	Source    string                 `json:"source"`     // local / imported / market
	SourceRef string                 `json:"source_ref"` // 可选
	Manifest  skilladapter.Manifest  `json:"manifest"`
	Files     []skilladapter.File    `json:"files"`
}

// BuildCanonical 从 WriteInput 合成 Canonical(name/version 兜底)。
func (in *WriteInput) BuildCanonical() skilladapter.Canonical {
	m := in.Manifest
	if in.Name != "" {
		m.Name = in.Name
	}
	if in.Version != "" {
		m.Version = in.Version
	}
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	return skilladapter.Canonical{Manifest: m, Files: in.Files}
}

// Create 新建一个 skill:先 store.Save 写盘,再插 DB。
// store 失败时 DB 不动(写盘是 source of truth)。
func (s *Service) Create(in *WriteInput) (*entity.Skill, error) {
	scope, err := normalizeScope(in.Scope)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimSpace(in.Manifest.Name)
	}
	if name == "" {
		return nil, ErrEmptyName
	}
	in.Name = name
	c := in.BuildCanonical()
	if err := s.store.Save(c, scope, in.ProjectID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStoreSave, err)
	}
	row := &entity.Skill{
		Scope:        scope,
		ProjectID:    in.ProjectID,
		Name:         c.Manifest.Name,
		Version:      c.Manifest.Version,
		Source:       defaultSource(in.Source),
		SourceRef:    in.SourceRef,
		ManifestJSON: marshalManifest(c.Manifest),
	}
	created, err := s.skillModel().Create(row)
	if err != nil {
		// 回滚物理文件,避免孤儿
		_ = s.store.Delete(scope, c.Manifest.Name, c.Manifest.Version, in.ProjectID)
		return nil, fmt.Errorf("skill: db create: %w", err)
	}
	return created, nil
}

// Update 按 (scope, project_id, name, version) 更新。store.Save 覆盖式。
// version 不允许改(唯一键的一部分);改 version 走"新建一条 + 删除旧条"。
func (s *Service) Update(scope, name, version string, projectID uint, in *WriteInput) (*entity.Skill, error) {
	scope, err := normalizeScope(scope)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}
	row, err := s.findRow(scope, name, version, projectID)
	if err != nil {
		return nil, err
	}
	c := in.BuildCanonical()
	if err := s.store.Save(c, scope, projectID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStoreSave, err)
	}
	row.ManifestJSON = marshalManifest(c.Manifest)
	row.Source = defaultSource(in.Source)
	if in.SourceRef != "" {
		row.SourceRef = in.SourceRef
	}
	row.UpdatedAt = time.Now()
	if err := s.skillModel().Update(where.New(mskill.FieldID, "=", row.ID).Conditions(), row); err != nil {
		return nil, fmt.Errorf("skill: db update: %w", err)
	}
	return row, nil
}

// Get 拿一条元数据。
func (s *Service) Get(scope, name, version string, projectID uint) (*entity.Skill, error) {
	scope, err := normalizeScope(scope)
	if err != nil {
		return nil, err
	}
	return s.findRow(scope, name, version, projectID)
}

// GetFull 拿元数据 + 物理文件(给编辑器用)。
type FullSkill struct {
	*entity.Skill
	Canonical skilladapter.Canonical `json:"canonical"`
}

func (s *Service) GetFull(scope, name, version string, projectID uint) (*FullSkill, error) {
	scope, err := normalizeScope(scope)
	if err != nil {
		return nil, err
	}
	row, err := s.findRow(scope, name, version, projectID)
	if err != nil {
		return nil, err
	}
	c, err := s.store.Load(scope, name, version, projectID)
	if err != nil {
		return nil, fmt.Errorf("skill: store load: %w", err)
	}
	return &FullSkill{Skill: row, Canonical: *c}, nil
}

// Delete 物理删 + DB 删;幂等(无 key 时 nil)。
func (s *Service) Delete(scope, name, version string, projectID uint) error {
	scope, err := normalizeScope(scope)
	if err != nil {
		return err
	}
	// 先找行(不要求存在,允许幂等)
	row, err := s.findRow(scope, name, version, projectID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err := s.store.Delete(scope, name, version, projectID); err != nil {
		return fmt.Errorf("skill: store delete: %w", err)
	}
	if row != nil {
		if err := s.skillModel().Delete(where.New(mskill.FieldID, "=", row.ID).Conditions()); err != nil {
			return fmt.Errorf("skill: db delete: %w", err)
		}
		// 同步清 skill_files 表
		_ = s.fileModel().Delete(where.New(mskillfile.FieldSkillID, "=", row.ID).Conditions())
	}
	return nil
}

// ListQuery 列表过滤。
type ListQuery struct {
	Scope     string // 留空 = 全部
	ProjectID uint   // 0 = 全局
	Keyword   string // 模糊匹配 name
	Page      int
	Size      int
}

// ListResult 列表结果。
type ListResult struct {
	Items []*entity.Skill `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Size  int             `json:"size"`
}

func (s *Service) List(q ListQuery) (*ListResult, error) {
	var conds []*where.Condition
	if scope := strings.TrimSpace(q.Scope); scope != "" {
		conds = append(conds, where.New(mskill.FieldScope, "=", scope).Conditions()...)
	}
	if q.ProjectID > 0 {
		conds = append(conds, where.New(mskill.FieldProjectID, "=", q.ProjectID).Conditions()...)
	}
	if k := strings.TrimSpace(q.Keyword); k != "" {
		conds = append(conds, where.New(mskill.FieldName, "LIKE", "%"+k+"%").Conditions()...)
	}
	page := q.Page
	size := q.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	items, total, err := s.skillModel().FindList(conds, &where.Extra{
		PageNum:       page,
		PageSize:      size,
		OrderByColumn: mskill.FieldUpdatedAt,
		OrderByDesc:   true,
	})
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(total), Page: page, Size: size}, nil
}

// findRow 内部 helper。
func (s *Service) findRow(scope, name, version string, projectID uint) (*entity.Skill, error) {
	conds := []*where.Condition{}
	conds = append(conds, where.New(mskill.FieldScope, "=", scope).Conditions()...)
	conds = append(conds, where.New(mskill.FieldName, "=", name).Conditions()...)
	conds = append(conds, where.New(mskill.FieldProjectID, "=", projectID).Conditions()...)
	if version != "" {
		conds = append(conds, where.New(mskill.FieldVersion, "=", version).Conditions()...)
	}
	row, err := s.skillModel().FindOne(conds)
	if err != nil {
		return nil, ErrNotFound
	}
	return row, nil
}

// marshalManifest 把 manifest 序列化成 JSON 字符串(冗余存 DB)。
func marshalManifest(m skilladapter.Manifest) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// defaultSource 给 source 字段兜底。
func defaultSource(s string) string {
	if strings.TrimSpace(s) == "" {
		return "local"
	}
	return s
}

// Store 工厂方法,统一从 configs 取 StoreRoot,避免 controller / service 各搞各的。
func NewStore() (*skillstore.Store, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return skillstore.NewAt(root)
	}
	return skillstore.New()
}

// stableOrder 把 WriteInput.Files 按 Path 排序,让 store.Save 写出来的 diff 稳定。
func stableOrder(files []skilladapter.File) []skilladapter.File {
	out := make([]skilladapter.File, len(files))
	copy(out, files)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
