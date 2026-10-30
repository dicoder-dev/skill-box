// Package sagent 提供工具(AI 编程工具元数据)的业务层封装。
//
// 2026-06-30 二改:替代原 agentspecs/specs/*.yaml 编译期配置,工具元数据
// 全部走 e_tool + e_tool_path 表,本服务负责 CRUD + Reload。
//
// 设计要点:
//   - Create / Update / Delete 走 magent(model 层)
//   - 系统工具(is_system=true):tool_id / is_system 不可改,行不可删;
//     其他字段(display_name / mdi_icon / maturity / note / enabled / paths)可改
//   - 改完业务数据后,业务层调 Reload() 一次性刷新 skilladapter.DefaultRegistry
//   - 删 agent 时事务里级联删 e_tool_path
package sagent

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/magent"
	"ginp-api/internal/gapi/service/agent/agenticon"
	"ginp-api/internal/skilladapter/agentspecs"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrEmptyAgentID      = errors.New("agent: tool_id is empty")
	ErrAgentIDConflict   = errors.New("agent: tool_id already exists")
	ErrNotFound         = errors.New("agent: not found")
	ErrSystemAgentFrozen = errors.New("agent: system agent cannot be deleted or have tool_id changed")
	ErrEmptyDisplay     = errors.New("agent: display_name is empty")
	// ErrEmptyMdi 仅在"没有 icon_file 兜底"时报 — 即 mdi_icon 和 icon_file 都为空时,必须有 mdi_icon。
	ErrEmptyMdi         = errors.New("agent: mdi_icon and icon_file cannot both be empty")
	ErrBadIconFile      = errors.New("agent: icon_file must be basename with allowed extension (.png/.svg/.jpg/.jpeg/.webp/.ico)")
	ErrBadMaturity      = errors.New("agent: maturity must be stable|experimental|deprecated")
	ErrBadCategory      = errors.New("agent: category must be user|system")
	ErrBadScope         = errors.New("agent: scope must be global|project")
	ErrEmptyPath        = errors.New("agent: path is empty")
	// ErrPathExisted 同一 (agent, scope, category) 已存在一条 path。
	// 2026-07-04 改:单 path 模型下,(scope, category) 只能对应 1 条 path。
	ErrPathExisted = errors.New("agent: a path already exists for (scope, category)")
)

// Service 工具管理服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func New(dbWrite, dbRead *gorm.DB) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead}
}

func (s *Service) agentM() *magent.Model        { return magent.NewModel(s.dbWrite, s.dbRead) }
func (s *Service) pathM() *magent.AgentPathModel { return magent.NewAgentPathModel(s.dbWrite, s.dbRead) }

// CreateInput 新建工具入参。
type CreateInput struct {
	AgentID      string
	DisplayName string
	MdiIcon     string // 可空 — 若 IconFile 非空则允许为空
	IconFile    string // 可空 — 自定义图标文件名(basename),存于 ~/.skill-box/agent-icons/
	Maturity    string
	Note        string
	Enabled     bool
	SortOrder   int
	Paths       []PathInput
}

// PathInput 单条路径入参(写库用)。
type PathInput struct {
	Scope     string
	Category  string
	Path      string
	PathOrder int
}

// Create 新建一个用户工具(is_system 强制 false)。
func (s *Service) Create(in *CreateInput) (*entity.Agent, error) {
	if err := validateBase(in.AgentID, in.DisplayName, in.MdiIcon, in.IconFile, in.Maturity); err != nil {
		return nil, err
	}
	if err := validatePathUniqueness(in.Paths); err != nil {
		return nil, err
	}
	for i, p := range in.Paths {
		if err := validatePath(p); err != nil {
			return nil, fmt.Errorf("paths[%d]: %w", i, err)
		}
	}
	// tool_id 唯一
	if _, err := s.agentM().FindByAgentID(in.AgentID); err == nil {
		return nil, fmt.Errorf("%w: %q", ErrAgentIDConflict, in.AgentID)
	}
	agent := &entity.Agent{
		AgentID:      strings.TrimSpace(in.AgentID),
		DisplayName: strings.TrimSpace(in.DisplayName),
		MdiIcon:     strings.TrimSpace(in.MdiIcon),
		IconFile:    strings.TrimSpace(in.IconFile),
		Maturity:    in.Maturity,
		Note:        in.Note,
		IsSystem:    false, // 用户新建,永远非系统工具
		Enabled:     in.Enabled,
		SortOrder:   in.SortOrder,
	}
	created, err := s.agentM().Create(agent)
	if err != nil {
		return nil, fmt.Errorf("agent: create: %w", err)
	}
	if err := s.replacePaths(created.ID, in.Paths); err != nil {
		return nil, fmt.Errorf("agent: create paths: %w", err)
	}
	return created, nil
}

// UpdateInput 更新入参(零值表示"不改";tool_id 不可改)。
type UpdateInput struct {
	AgentID      string // locator,不改
	DisplayName *string
	MdiIcon     *string
	IconFile    *string
	Maturity    *string
	Note        *string
	Enabled     *bool
	SortOrder   *int
	Paths       *[]PathInput // nil 表示不改 paths;非 nil 表示"用此组覆盖"
}

// Update 改一个工具的元数据。系统工具的 tool_id / is_system 不可改,
// 其他字段(本函数)以及 path(通过 Paths 替换)可改。
func (s *Service) Update(in *UpdateInput) (*entity.Agent, error) {
	cur, err := s.agentM().FindByAgentID(in.AgentID)
	if err != nil {
		return nil, ErrNotFound
	}
	upd := &entity.Agent{
		DisplayName: cur.DisplayName,
		MdiIcon:     cur.MdiIcon,
		IconFile:    cur.IconFile,
		Maturity:    cur.Maturity,
		Note:        cur.Note,
		Enabled:     cur.Enabled,
		SortOrder:   cur.SortOrder,
	}
	if in.DisplayName != nil {
		if strings.TrimSpace(*in.DisplayName) == "" {
			return nil, ErrEmptyDisplay
		}
		upd.DisplayName = strings.TrimSpace(*in.DisplayName)
	}
	// mdi_icon 和 icon_file 至少要有一个非空;如果 client 同时清空两者,拒绝。
	if in.MdiIcon != nil {
		mdi := strings.TrimSpace(*in.MdiIcon)
		// 允许空串(清空 mdi_icon),但若新的 mdi_icon 为空且 icon_file 也空/被清空 → 报错
		upd.MdiIcon = mdi
	}
	if in.IconFile != nil {
		icon := strings.TrimSpace(*in.IconFile)
		if icon != "" && !agenticon.ValidIconFileName(icon) {
			return nil, fmt.Errorf("%w: %q", ErrBadIconFile, icon)
		}
		upd.IconFile = icon
	}
	// 终态校验:改完后 mdi_icon + icon_file 不能都为空
	if upd.MdiIcon == "" && upd.IconFile == "" {
		return nil, ErrEmptyMdi
	}
	if in.Maturity != nil {
		if !validMaturity(*in.Maturity) {
			return nil, ErrBadMaturity
		}
		upd.Maturity = *in.Maturity
	}
	if in.Note != nil {
		upd.Note = *in.Note
	}
	if in.Enabled != nil {
		upd.Enabled = *in.Enabled
	}
	if in.SortOrder != nil {
		upd.SortOrder = *in.SortOrder
	}
	cols := []string{
		magent.FieldDisplayName, magent.FieldMdiIcon, magent.FieldIconFile, magent.FieldMaturity, magent.FieldNote,
		magent.FieldEnabled, magent.FieldSortOrder,
	}
	if err := s.agentM().Update(where.New(magent.FieldID, "=", cur.ID).Conditions(), upd, cols...); err != nil {
		return nil, fmt.Errorf("agent: update: %w", err)
	}
	if in.Paths != nil {
		if err := validatePathUniqueness(*in.Paths); err != nil {
			return nil, err
		}
		for i, p := range *in.Paths {
			if err := validatePath(p); err != nil {
				return nil, fmt.Errorf("paths[%d]: %w", i, err)
			}
		}
		if err := s.replacePaths(cur.ID, *in.Paths); err != nil {
			return nil, fmt.Errorf("agent: replace paths: %w", err)
		}
	}
	return s.agentM().FindOneByID(cur.ID)
}

// Delete 删一个工具。系统工具(is_system=true)不可删。
func (s *Service) Delete(agentID string) error {
	cur, err := s.agentM().FindByAgentID(agentID)
	if err != nil {
		return ErrNotFound
	}
	if cur.IsSystem {
		return fmt.Errorf("%w: %s", ErrSystemAgentFrozen, agentID)
	}
	err = s.dbWrite.Transaction(func(tx *gorm.DB) error {
		pathM := magent.NewAgentPathModel(tx, tx)
		if err := pathM.DeleteByAgentID(cur.ID); err != nil {
			return fmt.Errorf("agent: delete paths: %w", err)
		}
		agentM := magent.NewModel(tx, tx)
		if err := agentM.DeleteByID(cur.ID); err != nil {
			return fmt.Errorf("agent: delete: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 级联删除自定义图标文件(若指定)。失败不报错 — icon_file 文件不存在
	// 只是孤儿,不影响业务;但要确保 basename 校验过,防止越界删除任意文件。
	if cur.IconFile != "" && agenticon.ValidIconFileName(cur.IconFile) {
		_ = agenticon.Delete(cur.IconFile) // best-effort:清理用户上传的图标
	}
	return nil
}

// List 列出所有工具(给前端用,含 path)。
//
// 2026-07-18 改:加 ORDER BY sort_order ASC, id ASC 兜底,保证前端首页按
// "用户最常用在前"的稳定顺序展示;此前未排序,依赖 DB 物理插入序,新工具加进来
// 会落到表尾或随机位置,体验不一致。
func (s *Service) List() ([]AgentView, error) {
	agents, _, err := s.agentM().FindList(nil, &where.Extra{
		OrderByColumn: "sort_order",
		OrderByDesc:   false, // ASC:数字越小越靠前
	})
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, nil
	}
	ids := make([]uint, len(agents))
	for i, t := range agents {
		ids[i] = t.ID
	}
	pathsByAgent, err := s.pathM().FindAllByAgentIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]AgentView, 0, len(agents))
	for _, t := range agents {
		ps := pathsByAgent[t.ID]
		views := make([]PathView, 0, len(ps))
		for _, p := range ps {
			views = append(views, PathView{
				Scope: p.Scope, Category: p.Category, Path: p.Path, PathOrder: p.PathOrder,
			})
		}
		out = append(out, AgentView{
			ID: t.ID, AgentID: t.AgentID, DisplayName: t.DisplayName,
			MdiIcon: t.MdiIcon, IconFile: t.IconFile,
			Maturity: t.Maturity, Note: t.Note, IsSystem: t.IsSystem, Enabled: t.Enabled,
			SortOrder: t.SortOrder, Paths: views,
			CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		})
	}
	return out, nil
}

// Reload 业务数据改完后,重新从 DB 拉一次工具元数据,刷 skilladapter.Registry。
func (s *Service) Reload() error {
	if err := agentspecs.ReloadAllFromDB(s.dbRead); err != nil {
		return err
	}
	log.Printf("sagent: registry reloaded")
	return nil
}

// ─── 内部辅助 ───────────────────────────────────────────────────────

// replacePaths 事务里删旧 path + 写新 path(覆盖式)。
func (s *Service) replacePaths(agentID uint, paths []PathInput) error {
	return s.dbWrite.Transaction(func(tx *gorm.DB) error {
		pathM := magent.NewAgentPathModel(tx, tx)
		if err := pathM.DeleteByAgentID(agentID); err != nil {
			return err
		}
		for _, p := range paths {
			if _, err := pathM.Create(&entity.AgentPath{
				AgentID: agentID, Scope: p.Scope, Category: p.Category,
				Path: strings.TrimSpace(p.Path), PathOrder: p.PathOrder,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateBase(agentID, display, mdi, iconFile, maturity string) error {
	if strings.TrimSpace(agentID) == "" {
		return ErrEmptyAgentID
	}
	if strings.TrimSpace(display) == "" {
		return ErrEmptyDisplay
	}
	// mdi_icon 和 icon_file 至少要有一个
	mdiT := strings.TrimSpace(mdi)
	iconT := strings.TrimSpace(iconFile)
	if mdiT == "" && iconT == "" {
		return ErrEmptyMdi
	}
	// 若给了 mdi_icon,必须以 mdi: 开头(走 Iconify 解析)
	if mdiT != "" && !strings.HasPrefix(mdiT, "mdi:") {
		return fmt.Errorf("%w: %q", ErrEmptyMdi, mdi)
	}
	// 若给了 icon_file,必须是合法 basename
	if iconT != "" && !agenticon.ValidIconFileName(iconT) {
		return fmt.Errorf("%w: %q", ErrBadIconFile, iconFile)
	}
	if maturity != "" && !validMaturity(maturity) {
		return ErrBadMaturity
	}
	return nil
}

func validatePath(p PathInput) error {
	if strings.TrimSpace(p.Path) == "" {
		return ErrEmptyPath
	}
	if p.Scope != "global" && p.Scope != "project" {
		return fmt.Errorf("%w: %q", ErrBadScope, p.Scope)
	}
	if p.Category != "user" && p.Category != "system" {
		return fmt.Errorf("%w: %q", ErrBadCategory, p.Category)
	}
	return nil
}

// validatePathUniqueness 校验输入的 paths 列表中,同一 (scope, category)
// 不重复(单 path 模型下)。
//
// 2026-07-04 改:之前允许同 (scope, category) 多 path(靠 path_order 排序),
// 现在收口为 0~1 条,Service 层先报错返回 ErrPathExisted,避免依赖
// DB uniqueIndex 的错误信息(更稳定)。
func validatePathUniqueness(paths []PathInput) error {
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		key := p.Scope + "|" + p.Category
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: (%s, %s)", ErrPathExisted, p.Scope, p.Category)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// AddOnePath 给已存在工具追加一条 path,严格校验 (scope, category) 唯一。
//
// 2026-07-04 改:原 controller 直接调 model.Create,依赖 DB uniqueIndex 报错,
// 现在 Service 层先查一次避免空跑事务,DB uniqueIndex 作为最终兜底。
//
// 入参 agentID 是工具的 tool_id(非 DB 主键);失败时返回 ErrNotFound /
// ErrPathExisted / 其它基础校验错。
func (s *Service) AddOnePath(agentID string, in PathInput) (*entity.AgentPath, error) {
	if err := validatePath(in); err != nil {
		return nil, err
	}
	agent, err := s.agentM().FindByAgentID(agentID)
	if err != nil {
		return nil, ErrNotFound
	}
	// 先查 (tool_id, scope, category) 是否已有,命中就返 ErrPathExisted
	existing, lerr := s.pathM().FindOne(where.New(magent.FieldPathAgentID, "=", agent.ID).
		And(magent.FieldPathScope, "=", in.Scope).
		And(magent.FieldPathCategory, "=", in.Category).Conditions())
	if lerr == nil && existing != nil {
		return nil, fmt.Errorf("%w: (%s, %s)", ErrPathExisted, in.Scope, in.Category)
	}
	// gorm.ErrRecordNotFound 不算冲突,继续 Create。
	out, err := s.pathM().Create(&entity.AgentPath{
		AgentID:    agent.ID,
		Scope:     in.Scope,
		Category:  in.Category,
		Path:      strings.TrimSpace(in.Path),
		PathOrder: in.PathOrder,
	})
	if err != nil {
		// DB 兜底:uniqueIndex 也会触发,统一映射为 ErrPathExisted
		if isUniqueConflict(err) {
			return nil, fmt.Errorf("%w: (%s, %s)", ErrPathExisted, in.Scope, in.Category)
		}
		return nil, fmt.Errorf("agent: add path: %w", err)
	}
	return out, nil
}

// isUniqueConflict 简单判断 GORM 报的 duplicate key 错误(跨 DB 兼容)。
func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{"UNIQUE constraint failed", "Duplicate entry", "duplicate key", "unique constraint"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

func validMaturity(s string) bool {
	switch s {
	case "stable", "experimental", "deprecated":
		return true
	}
	return false
}

// ─── 视图结构 ───────────────────────────────────────────────────────

// AgentView 工具视图(给前端)。
type AgentView struct {
	ID          uint       `json:"id"`
	AgentID      string     `json:"agent_id"`
	DisplayName string     `json:"display_name"`
	MdiIcon     string     `json:"mdi_icon"`
	IconFile    string     `json:"icon_file"`
	Maturity    string     `json:"maturity"`
	Note        string     `json:"note"`
	IsSystem    bool       `json:"is_system"`
	Enabled     bool       `json:"enabled"`
	SortOrder   int        `json:"sort_order"`
	Paths       []PathView `json:"paths"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PathView 路径视图(给前端)。
type PathView struct {
	Scope     string `json:"scope"`
	Category  string `json:"category"`
	Path      string `json:"path"`
	PathOrder int    `json:"path_order"`
}
