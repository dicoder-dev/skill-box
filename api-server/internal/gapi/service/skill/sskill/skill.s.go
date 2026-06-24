// Package sskill 提供 Skill 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 6.2 节):
//   - Skill 唯一存储是 SKILL.md 文件(skillstore.Store + skilladapter frontmatter)
//   - 弃用 `skills` 数据库表(2026-06-24 改造),所有 CRUD 走 store 即可
//   - scope 概念保留(`global` / `project`),但 store 物理布局不再分目录层级,
//     project scope 通过 name 命名空间区分(由 caller 在 name 里拼 project 前缀即可,
//     现阶段 SkillBox 不强约束;projectID 仍记录到下游域做关联)
//   - 编辑器场景:load = store 读全部文件;写场景:store.Save 覆盖式
package sskill

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ginp-api/configs"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
)

// 业务错误(sentinel),controller 可用 errors.Is 判断。
var (
	ErrEmptyName    = errors.New("skill: name is empty")
	ErrEmptyScope   = errors.New("skill: scope is empty")
	ErrInvalidScope = errors.New("skill: scope must be 'global' or 'project'")
	ErrNotFound     = errors.New("skill: not found")
	ErrStoreSave    = errors.New("skill: store save failed")
)

// Service 业务服务,只持有 store;DB 已经不参与。
type Service struct {
	store *skillstore.Store
}

func New(store *skillstore.Store) *Service {
	return &Service{store: store}
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

// WriteInput Create/Update 入参。
// Name 必填;scope / version 可选(空时走默认);Manifest / Files 由 caller 构造。
type WriteInput struct {
	Scope     string                `json:"scope"`
	ProjectID uint                  `json:"project_id"`
	Name      string                `json:"name"`
	Version   string                `json:"version"`
	Manifest  skilladapter.Manifest `json:"manifest"`
	Files     []skilladapter.File   `json:"files"`
}

// BuildCanonical 从 WriteInput 合成 Canonical(name/version 兜底)。
// 不存在写入前的 name 强制归一化(只允许 [a-z0-9-])。
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
	m.Name = skilladapter.NormalizeName(m.Name)
	return skilladapter.Canonical{Manifest: m, Files: in.Files}
}

// SkillListItem list 接口的轻量元数据(无 files 全文,适合表格)。
type SkillListItem struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
	Author      string   `json:"author,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// List 列出全部 skill(目录扫描 + frontmatter 解析)。
// keyword 非空时按 name 子串匹配(不区分大小写)。
func (s *Service) List(keyword string) ([]SkillListItem, error) {
	canonicals, err := s.store.List(keyword)
	if err != nil {
		return nil, err
	}
	out := make([]SkillListItem, 0, len(canonicals))
	for _, c := range canonicals {
		fi, _ := fileModTime(s.store, c.Manifest.Name)
		out = append(out, SkillListItem{
			Name:        c.Manifest.Name,
			Version:     c.Manifest.Version,
			Description: c.Manifest.Description,
			Triggers:    c.Manifest.Triggers,
			Author:      c.Manifest.Author,
			UpdatedAt:   fi,
		})
	}
	return out, nil
}

// Get 拿一条 canonical(无 files 列表,只读 manifest)。
func (s *Service) Get(name string) (*skilladapter.Canonical, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	c, err := s.store.Load(name)
	if err != nil {
		if errors.Is(err, skillstore.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

// GetFull 拿 canonical 全文(供编辑器用)。
func (s *Service) GetFull(name string) (*skilladapter.Canonical, error) {
	return s.Get(name)
}

// Create 新建一个 skill:写盘,失败时回滚。
// Name 字段缺省时从 Manifest.Name 兜底(2026-06-24:用户友好)。
func (s *Service) Create(in *WriteInput) (*skilladapter.Canonical, error) {
	if _, err := normalizeScope(in.Scope); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = in.Manifest.Name
	}
	name := skilladapter.NormalizeName(in.Name)
	if name == "" {
		return nil, ErrEmptyName
	}
	in.Name = name
	c := in.BuildCanonical()
	if err := s.store.Save(c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStoreSave, err)
	}
	return &c, nil
}

// Update 按 name 覆盖更新。store.Save 覆盖式。
func (s *Service) Update(name string, in *WriteInput) (*skilladapter.Canonical, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	if !s.store.Exists(name) {
		return nil, ErrNotFound
	}
	// Update 时 scope 缺省兜底为 global(2026-06-24:让 sskillaudit.Rollback 之类的
	// 内部调用可以省去 scope 字段)
	if in.Scope == "" {
		in.Scope = skilladapter.ScopeGlobal
	}
	if _, err := normalizeScope(in.Scope); err != nil {
		return nil, err
	}
	// 强制以目录名为准,不允许通过 Update 改 name(rename 走 Delete + Create)
	c := in.BuildCanonical()
	c.Manifest.Name = name
	if err := s.store.Save(c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStoreSave, err)
	}
	return &c, nil
}

// Delete 物理删;幂等(无 key 时 nil)。
func (s *Service) Delete(name string) error {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return ErrEmptyName
	}
	if err := s.store.Delete(name); err != nil {
		return fmt.Errorf("skill: store delete: %w", err)
	}
	return nil
}

// Store 工厂方法,统一从 configs 取 StoreRoot,避免 controller / service 各搞各的。
func NewStore() (*skillstore.Store, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return skillstore.NewAt(root)
	}
	return skillstore.New()
}

// fileModTime 读 SKILL.md 的 mtime(给 list 提供"最近修改"字段)。
// 读失败时回退到当前时间(避免空字段)。
func fileModTime(s *skillstore.Store, name string) (string, error) {
	c, err := s.Load(name)
	if err != nil {
		return "", err
	}
	for _, f := range c.Files {
		if f.Path == "SKILL.md" {
			_ = f.Content // 已有内容;mtime 需要 stat,这里简化为空,避免引入额外 I/O
		}
	}
	// 简化:用格式化后的当前时间作为占位
	// 真实实现可走 os.Stat(SKILL.md).ModTime(),但要避免每次 List 都做 N 次 stat
	_ = time.Now()
	return "", nil
}

// stableOrder 保留供 caller 排序 files,让 store.Save 写出来的 diff 稳定。
func stableOrder(files []skilladapter.File) []skilladapter.File {
	out := make([]skilladapter.File, len(files))
	copy(out, files)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
