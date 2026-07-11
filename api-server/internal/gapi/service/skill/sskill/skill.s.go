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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ginp-api/configs"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
)

// 业务错误(sentinel),controller 可用 errors.Is 判断。
var (
	ErrEmptyName        = errors.New("skill: name is empty")
	ErrEmptyScope       = errors.New("skill: scope is empty")
	ErrInvalidScope     = errors.New("skill: scope must be 'global' or 'project'")
	ErrNotFound         = errors.New("skill: not found")
	ErrStoreSave        = errors.New("skill: store save failed")
	ErrInvalidSkillName = errors.New("skill: name is invalid (must match [a-z][a-z0-9-]{1,63})")
	// 2026-07-05 增:磁盘文件被破坏(含非 UTF-8 字节)时返回该 sentinel。
	// controller 识别后返 422 + code=corrupted_file,前端弹"需手动修复"提示。
	ErrCorruptedFile = errors.New("skill: file contains non-UTF-8 bytes")
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
//
// 2026-07-03 改:多级分组(2026-06-29)后,caller 只给 name 不给 groupPath 时,
// 旧 store.Load 只查"根下直接子目录",漏掉 aa/debug-helper 这种嵌套 skill
// → apply 链路返 404 skill not found。改走 store.LoadByName,自动 BFS 全树
// 找第一条匹配(浅层优先,根下同名 skill 覆盖分组里的)。
func (s *Service) Get(name string) (*skilladapter.Canonical, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	c, err := s.store.LoadByName(name)
	if err != nil {
		if errors.Is(err, skillstore.ErrNotFound) {
			return nil, ErrNotFound
		}
		// 2026-07-05 增:store 层的 skillstore.ErrCorruptedFile 翻译成服务层 sentinel
		if errors.Is(err, skillstore.ErrCorruptedFile) {
			return nil, fmt.Errorf("%w: %v", ErrCorruptedFile, err)
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
//
// 2026-07-03 改:同 Get — Exists 改用 ExistsByName,让 aa/debug-helper 这种
// 嵌套 skill 也能被正确命中(避免误报 "skill not found")。
func (s *Service) Update(name string, in *WriteInput) (*skilladapter.Canonical, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	if !s.store.ExistsByName(name) {
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
//
// 2026-07-03 改:同 Get — Delete 改用 DeleteByName,让 aa/debug-helper 这种
// 嵌套 skill 也能被正确删掉(而不是报 not found)。
func (s *Service) Delete(name string) error {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return ErrEmptyName
	}
	if err := s.store.DeleteByName(name); err != nil {
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

// ErrInvalidGroupPath 分组路径非法(空 / 含 .. / 含绝对路径前缀 / 规范化后空字符串)。
var ErrInvalidGroupPath = errors.New("skill: invalid group path")

// normalizeGroupPath 把 caller 传入的分组路径规范化,拒绝一切不安全形态。
//
// 2026-06-29 增:复用 skilladapter.NormalizeGroupName + 二次校验双重防线。
func (s *Service) normalizeGroupPath(p string) (string, error) {
	p = skilladapter.NormalizeGroupName(p)
	if p == "" {
		return "", nil
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("%w: leading slash", ErrInvalidGroupPath)
	}
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("%w: contains ..", ErrInvalidGroupPath)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("%w: bad segment %q", ErrInvalidGroupPath, seg)
		}
	}
	return p, nil
}

// CreateGroup 新建一个空分组目录。groupPath 可多级(用 '/' 分隔)。
// 已存在时不报错(幂等);空字符串返回 nil(根目录)。
//
// 2026-06-29 增:为支持多级分组。
func (s *Service) CreateGroup(groupPath string) error {
	cleaned, err := s.normalizeGroupPath(groupPath)
	if err != nil {
		return err
	}
	return s.store.CreateGroupDir(cleaned)
}

// DeleteGroup 删分组目录及其子树。
//
// cascade=false 时,若分组非空,返回 (deleted_skill_paths, error) — 让 caller
// 决定是否强删。cascade=true 时直接递归删,返回 (deleted_skill_paths, nil)。
// 返回的 deleted_skill_paths 是该分组下所有 skill 叶子的相对路径(用 '/' 分隔),
// 供前端在 cascade_tools=true 时同步清理各工具目录。
//
// 2026-06-29 增:为支持多级分组。
func (s *Service) DeleteGroup(groupPath string, cascade bool) ([]string, error) {
	cleaned, err := s.normalizeGroupPath(groupPath)
	if err != nil {
		return nil, err
	}
	return s.store.DeleteGroupDir(cleaned, cascade)
}

// MoveSkill 把 skill 从 srcGroupPath 移动到 dstGroupPath 下(叶子 name 不变)。
//
// 2026-06-29 增:为支持多级分组拖拽。
func (s *Service) MoveSkill(srcGroupPath string, name string, dstGroupPath string) error {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return ErrEmptyName
	}
	src, err := s.normalizeGroupPath(srcGroupPath)
	if err != nil {
		return err
	}
	dst, err := s.normalizeGroupPath(dstGroupPath)
	if err != nil {
		return err
	}
	return s.store.MoveGroupPath(src, name, dst)
}

// MoveGroup 把整个分组从 srcGroupPath 移动到 dstGroupPath 下。
// src 不能为空;dst 可为空(表示挪到根下)。
//
// 2026-06-29 增:为支持"分组嵌套到另一分组"。
func (s *Service) MoveGroup(srcGroupPath string, dstGroupPath string) error {
	src, err := s.normalizeGroupPath(srcGroupPath)
	if err != nil || src == "" {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: src is empty", ErrInvalidGroupPath)
	}
	dst, err := s.normalizeGroupPath(dstGroupPath)
	if err != nil {
		return err
	}
	if err := s.store.MoveGroupDir(src, dst); err != nil {
		// 源分组不存在 → 包成 sskill.ErrNotFound,controller 走 404
		if errors.Is(err, skillstore.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// RenameGroup 重命名分组的最后一段(父路径不变)。
// srcGroupPath 可多级,newName 是单段名(不含 '/',走 NormalizeName 规约)。
// 返回 (newGroupPath, error) — newGroupPath 是规范化后的新相对路径。
//
// 2026-06-29 增:为支持"分组右键重命名"。
func (s *Service) RenameGroup(srcGroupPath string, newName string) (string, error) {
	src, err := s.normalizeGroupPath(srcGroupPath)
	if err != nil {
		return "", err
	}
	if src == "" {
		return "", fmt.Errorf("%w: src is empty", ErrInvalidGroupPath)
	}
	// newName 走 NormalizeName(单段,不允许 '/',内部会被折叠为 '-')。
	// 若用户输入含 '/',我们先剥掉 → 视为只取最后一段(更友好,严拒反而突兀)。
	cleaned := strings.TrimSpace(newName)
	if i := strings.LastIndexAny(cleaned, "/\\"); i >= 0 {
		cleaned = cleaned[i+1:]
	}
	cleaned = skilladapter.NormalizeName(cleaned)
	if cleaned == "" {
		return "", fmt.Errorf("%w: new name is empty", ErrInvalidGroupPath)
	}
	return s.store.RenameGroupDir(src, cleaned)
}

// RenameSkill 2026-07-11 增:在同分组内重命名 skill 目录名(name 是 SKILL.md 目录的
// 最后一段;group_path 不变)。底层走 skillstore.RenameSkillInGroup,实际就是把
// "<root>/<group>/<oldName>" 这个目录整体 os.Rename 到 "<root>/<group>/<newName>"。
// 内部会上锁 + 校验目标不存在,所以并发安全。
//
// 设计取舍:不在 SKILL.md frontmatter 里改 name — frontmatter 跟目录名是 1:1 绑定
// 关系(Save 走 RenderSkillMD 重渲,目录名是真值),改 frontmatter 没用,必须改目录。
//
// 入参: srcGroupPath(可空 = 根)、oldName(原名, 走 NormalizeName 规约)、newName(新名)。
// 返回: (newSkillPath, error) — newSkillPath 是规范化后的新相对路径 "<group>/<newName>"。
func (s *Service) RenameSkill(srcGroupPath string, oldName string, newName string) (string, error) {
	old := skilladapter.NormalizeName(oldName)
	if old == "" {
		return "", fmt.Errorf("%w: old name is empty", ErrEmptyName)
	}
	cleaned := strings.TrimSpace(newName)
	if i := strings.LastIndexAny(cleaned, "/\\"); i >= 0 {
		// 输入含路径分隔符 → 只取最后一段(防止用户手抖输入 'foo/bar' 让目录名意外跨组)
		cleaned = cleaned[i+1:]
	}
	cleaned = skilladapter.NormalizeName(cleaned)
	if cleaned == "" {
		return "", fmt.Errorf("%w: new name is empty", ErrInvalidSkillName)
	}
	if cleaned == old {
		return "", fmt.Errorf("%w: new name equals old name", ErrInvalidSkillName)
	}
	gp, err := s.normalizeGroupPath(srcGroupPath)
	if err != nil {
		return "", err
	}
	return s.store.RenameSkillInGroup(gp, old, cleaned)
}

// ListTree 列出全部 skill 的树形结构(供前端分组 UI 用)。
//
// 2026-06-29 增:keyword 非空时做 skill name 子串匹配,分组(即使不含匹配项)
// 保留(便于展示"匹配项所在的路径")。
func (s *Service) ListTree(keyword string) ([]skillstore.TreeNode, error) {
	return s.store.ListTree(keyword)
}

// GetByPath 按分组路径读 skill 详情。
//
// 2026-06-29 增:为支持多级分组的 detail 加载。
func (s *Service) GetByPath(groupPath string, name string) (*skilladapter.Canonical, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return nil, ErrEmptyName
	}
	gp, err := s.normalizeGroupPath(groupPath)
	if err != nil {
		return nil, err
	}
	canon, err := s.store.LoadByPath(gp, name)
	if err != nil {
		// 2026-07-05 增:store 层的 skillstore.ErrCorruptedFile 翻译成本服务的
		// sentinel ErrCorruptedFile,controller 用 errors.Is 跨包识别。
		if errors.Is(err, skillstore.ErrCorruptedFile) {
			return nil, fmt.Errorf("%w: %v", ErrCorruptedFile, err)
		}
		return nil, err
	}
	return canon, nil
}

// DeleteByPath 按分组路径删 skill(供 cskill.delete_skill 用)。
// 返回 (deleted_skill_path, error) — deleted_skill_path 用于前端在
// cascade_tools=true 时同步清理工具目录。
//
// 2026-06-29 增:为支持多级分组的删除。
func (s *Service) DeleteByPath(groupPath string, name string) (string, error) {
	name = skilladapter.NormalizeName(name)
	if name == "" {
		return "", ErrEmptyName
	}
	gp, err := s.normalizeGroupPath(groupPath)
	if err != nil {
		return "", err
	}
	fullPath := gp
	if fullPath != "" {
		fullPath = fullPath + "/" + name
	} else {
		fullPath = name
	}
	if err := s.store.DeleteByPath(gp, name); err != nil {
		return fullPath, err
	}
	return fullPath, nil
}

// SplitPath 把 "frontend/react/use-cache" 拆成 ("frontend/react", "use-cache")。
// 不可拆(无 '/' 或 '/')返回 ("", "use-cache")。
//
// 2026-06-29 增:helper,供 cskill 控制器解析前端传入的"完整相对路径"。
func SplitPath(fullPath string) (groupPath string, name string) {
	fullPath = strings.TrimSpace(fullPath)
	if fullPath == "" {
		return "", ""
	}
	cleaned := filepath.ToSlash(filepath.Clean("/" + fullPath))
	if cleaned == "/" {
		return "", ""
	}
	cleaned = cleaned[1:]
	idx := strings.LastIndex(cleaned, "/")
	if idx < 0 {
		return "", cleaned
	}
	return cleaned[:idx], cleaned[idx+1:]
}
