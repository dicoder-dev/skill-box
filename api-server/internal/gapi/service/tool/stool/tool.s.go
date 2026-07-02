// Package stool 提供工具(AI 编程工具元数据)的业务层封装。
//
// 2026-06-30 二改:替代原 toolspecs/specs/*.yaml 编译期配置,工具元数据
// 全部走 e_tool + e_tool_path 表,本服务负责 CRUD + Reload。
//
// 设计要点:
//   - Create / Update / Delete 走 mtool(model 层)
//   - 系统工具(is_system=true):tool_id / is_system 不可改,行不可删;
//     其他字段(display_name / mdi_icon / maturity / note / enabled / paths)可改
//   - 改完业务数据后,业务层调 Reload() 一次性刷新 skilladapter.DefaultRegistry
//   - 删 tool 时事务里级联删 e_tool_path
package stool

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/mtool"
	"ginp-api/internal/gapi/service/tool/toolicon"
	"ginp-api/internal/skilladapter/toolspecs"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrEmptyToolID      = errors.New("tool: tool_id is empty")
	ErrToolIDConflict   = errors.New("tool: tool_id already exists")
	ErrNotFound         = errors.New("tool: not found")
	ErrSystemToolFrozen = errors.New("tool: system tool cannot be deleted or have tool_id changed")
	ErrEmptyDisplay     = errors.New("tool: display_name is empty")
	// ErrEmptyMdi 仅在"没有 icon_file 兜底"时报 — 即 mdi_icon 和 icon_file 都为空时,必须有 mdi_icon。
	ErrEmptyMdi         = errors.New("tool: mdi_icon and icon_file cannot both be empty")
	ErrBadIconFile      = errors.New("tool: icon_file must be basename with allowed extension (.png/.svg/.jpg/.jpeg/.webp/.ico)")
	ErrBadMaturity      = errors.New("tool: maturity must be stable|experimental|deprecated")
	ErrBadCategory      = errors.New("tool: category must be user|system")
	ErrBadScope         = errors.New("tool: scope must be global|project")
	ErrEmptyPath        = errors.New("tool: path is empty")
)

// Service 工具管理服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func New(dbWrite, dbRead *gorm.DB) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead}
}

func (s *Service) toolM() *mtool.Model        { return mtool.NewModel(s.dbWrite, s.dbRead) }
func (s *Service) pathM() *mtool.ToolPathModel { return mtool.NewToolPathModel(s.dbWrite, s.dbRead) }

// CreateInput 新建工具入参。
type CreateInput struct {
	ToolID      string
	DisplayName string
	MdiIcon     string // 可空 — 若 IconFile 非空则允许为空
	IconFile    string // 可空 — 自定义图标文件名(basename),存于 ~/.skill-box/tool-icons/
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
func (s *Service) Create(in *CreateInput) (*entity.Tool, error) {
	if err := validateBase(in.ToolID, in.DisplayName, in.MdiIcon, in.IconFile, in.Maturity); err != nil {
		return nil, err
	}
	for i, p := range in.Paths {
		if err := validatePath(p); err != nil {
			return nil, fmt.Errorf("paths[%d]: %w", i, err)
		}
	}
	// tool_id 唯一
	if _, err := s.toolM().FindByToolID(in.ToolID); err == nil {
		return nil, fmt.Errorf("%w: %q", ErrToolIDConflict, in.ToolID)
	}
	tool := &entity.Tool{
		ToolID:      strings.TrimSpace(in.ToolID),
		DisplayName: strings.TrimSpace(in.DisplayName),
		MdiIcon:     strings.TrimSpace(in.MdiIcon),
		IconFile:    strings.TrimSpace(in.IconFile),
		Maturity:    in.Maturity,
		Note:        in.Note,
		IsSystem:    false, // 用户新建,永远非系统工具
		Enabled:     in.Enabled,
		SortOrder:   in.SortOrder,
	}
	created, err := s.toolM().Create(tool)
	if err != nil {
		return nil, fmt.Errorf("tool: create: %w", err)
	}
	if err := s.replacePaths(created.ID, in.Paths); err != nil {
		return nil, fmt.Errorf("tool: create paths: %w", err)
	}
	return created, nil
}

// UpdateInput 更新入参(零值表示"不改";tool_id 不可改)。
type UpdateInput struct {
	ToolID      string // locator,不改
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
func (s *Service) Update(in *UpdateInput) (*entity.Tool, error) {
	cur, err := s.toolM().FindByToolID(in.ToolID)
	if err != nil {
		return nil, ErrNotFound
	}
	upd := &entity.Tool{
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
		if icon != "" && !toolicon.ValidIconFileName(icon) {
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
		mtool.FieldDisplayName, mtool.FieldMdiIcon, mtool.FieldIconFile, mtool.FieldMaturity, mtool.FieldNote,
		mtool.FieldEnabled, mtool.FieldSortOrder,
	}
	if err := s.toolM().Update(where.New(mtool.FieldID, "=", cur.ID).Conditions(), upd, cols...); err != nil {
		return nil, fmt.Errorf("tool: update: %w", err)
	}
	if in.Paths != nil {
		for i, p := range *in.Paths {
			if err := validatePath(p); err != nil {
				return nil, fmt.Errorf("paths[%d]: %w", i, err)
			}
		}
		if err := s.replacePaths(cur.ID, *in.Paths); err != nil {
			return nil, fmt.Errorf("tool: replace paths: %w", err)
		}
	}
	return s.toolM().FindOneByID(cur.ID)
}

// Delete 删一个工具。系统工具(is_system=true)不可删。
func (s *Service) Delete(toolID string) error {
	cur, err := s.toolM().FindByToolID(toolID)
	if err != nil {
		return ErrNotFound
	}
	if cur.IsSystem {
		return fmt.Errorf("%w: %s", ErrSystemToolFrozen, toolID)
	}
	err = s.dbWrite.Transaction(func(tx *gorm.DB) error {
		pathM := mtool.NewToolPathModel(tx, tx)
		if err := pathM.DeleteByToolID(cur.ID); err != nil {
			return fmt.Errorf("tool: delete paths: %w", err)
		}
		toolM := mtool.NewModel(tx, tx)
		if err := toolM.DeleteByID(cur.ID); err != nil {
			return fmt.Errorf("tool: delete: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 级联删除自定义图标文件(若指定)。失败不报错 — icon_file 文件不存在
	// 只是孤儿,不影响业务;但要确保 basename 校验过,防止越界删除任意文件。
	if cur.IconFile != "" && toolicon.ValidIconFileName(cur.IconFile) {
		_ = toolicon.Delete(cur.IconFile) // best-effort:清理用户上传的图标
	}
	return nil
}

// List 列出所有工具(给前端用,含 path)。
func (s *Service) List() ([]ToolView, error) {
	tools, _, err := s.toolM().FindList(nil, nil)
	if err != nil {
		return nil, err
	}
	if len(tools) == 0 {
		return nil, nil
	}
	ids := make([]uint, len(tools))
	for i, t := range tools {
		ids[i] = t.ID
	}
	pathsByTool, err := s.pathM().FindAllByToolIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]ToolView, 0, len(tools))
	for _, t := range tools {
		ps := pathsByTool[t.ID]
		views := make([]PathView, 0, len(ps))
		for _, p := range ps {
			views = append(views, PathView{
				Scope: p.Scope, Category: p.Category, Path: p.Path, PathOrder: p.PathOrder,
			})
		}
		out = append(out, ToolView{
			ID: t.ID, ToolID: t.ToolID, DisplayName: t.DisplayName,
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
	if err := toolspecs.ReloadAllFromDB(s.dbRead); err != nil {
		return err
	}
	log.Printf("stool: registry reloaded")
	return nil
}

// ─── 内部辅助 ───────────────────────────────────────────────────────

// replacePaths 事务里删旧 path + 写新 path(覆盖式)。
func (s *Service) replacePaths(toolID uint, paths []PathInput) error {
	return s.dbWrite.Transaction(func(tx *gorm.DB) error {
		pathM := mtool.NewToolPathModel(tx, tx)
		if err := pathM.DeleteByToolID(toolID); err != nil {
			return err
		}
		for _, p := range paths {
			if _, err := pathM.Create(&entity.ToolPath{
				ToolID: toolID, Scope: p.Scope, Category: p.Category,
				Path: strings.TrimSpace(p.Path), PathOrder: p.PathOrder,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func validateBase(toolID, display, mdi, iconFile, maturity string) error {
	if strings.TrimSpace(toolID) == "" {
		return ErrEmptyToolID
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
	if iconT != "" && !toolicon.ValidIconFileName(iconT) {
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

func validMaturity(s string) bool {
	switch s {
	case "stable", "experimental", "deprecated":
		return true
	}
	return false
}

// ─── 视图结构 ───────────────────────────────────────────────────────

// ToolView 工具视图(给前端)。
type ToolView struct {
	ID          uint       `json:"id"`
	ToolID      string     `json:"tool_id"`
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
