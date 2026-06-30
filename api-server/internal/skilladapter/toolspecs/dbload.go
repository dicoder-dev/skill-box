package toolspecs

import (
	"fmt"
	"sort"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/mtool"
	"ginp-api/internal/skilladapter"

	"gorm.io/gorm"
)

// LoadAllFromDB 从 e_tool + e_tool_path 表加载全部 enabled=true 的工具,
// 转成 []*ToolSpec。
//
// 2026-06-30 二改:替代原 LoadAll() 的"读 yaml"实现,工具元数据从编译期
// 内嵌变成运行时 DB 拉。每次启动 / 每次 reload 都会调一次。
//
// 关键约束:
//   - 排除 enabled=false 的工具(用户禁用,不注册 adapter)
//   - 同一 tool_id 在 DB 层面靠 uniqueIndex 兜底;这里再校验一次(防御
//     race condition / DB 异常)
//   - 每个 tool 的 path 按 (scope, category, path_order) 排序后填入 ToolSpec.Paths
func LoadAllFromDB(db *gorm.DB) ([]*ToolSpec, error) {
	toolM := mtool.NewModel(db, db)
	pathM := mtool.NewToolPathModel(db, db)

	tools, err := toolM.ListAllEnabled()
	if err != nil {
		return nil, fmt.Errorf("toolspecs: list enabled tools: %w", err)
	}
	if len(tools) == 0 {
		return nil, nil
	}
	toolIDs := make([]uint, len(tools))
	for i, t := range tools {
		toolIDs[i] = t.ID
	}
	pathsByTool, err := pathM.FindAllByToolIDs(toolIDs)
	if err != nil {
		return nil, fmt.Errorf("toolspecs: list paths: %w", err)
	}

	out := make([]*ToolSpec, 0, len(tools))
	seen := make(map[string]bool, len(tools))
	for _, t := range tools {
		if seen[t.ToolID] {
			return nil, fmt.Errorf("toolspecs: duplicate tool_id %q in DB", t.ToolID)
		}
		seen[t.ToolID] = true

		paths := pathsByTool[t.ID]
		spec, err := toSpec(t, paths)
		if err != nil {
			return nil, fmt.Errorf("toolspecs: convert %s: %w", t.ToolID, err)
		}
		out = append(out, spec)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ToolID < out[j].ToolID })
	return out, nil
}

// toSpec entity.Tool + []entity.ToolPath → ToolSpec。
// 失败:Validate 不通过(空 tool_id / 空 display_name / 空 mdi_icon / 路径为空)。
func toSpec(t *entity.Tool, paths []*entity.ToolPath) (*ToolSpec, error) {
	spec := &ToolSpec{
		ToolID:      t.ToolID,
		DisplayName: t.DisplayName,
		MdiIcon:     t.MdiIcon,
		Maturity:    t.Maturity,
		Note:        t.Note,
	}
	for _, p := range paths {
		switch p.Scope {
		case skilladapter.ScopeGlobal:
			switch p.Category {
			case "user":
				spec.Paths.Global.User = append(spec.Paths.Global.User, p.Path)
			case "system":
				spec.Paths.Global.System = append(spec.Paths.Global.System, p.Path)
			default:
				return nil, fmt.Errorf("tool %s: unknown category %q for global", t.ToolID, p.Category)
			}
		case skilladapter.ScopeProject:
			switch p.Category {
			case "user":
				spec.Paths.Project.User = append(spec.Paths.Project.User, p.Path)
			case "system":
				spec.Paths.Project.System = append(spec.Paths.Project.System, p.Path)
			default:
				return nil, fmt.Errorf("tool %s: unknown category %q for project", t.ToolID, p.Category)
			}
		default:
			return nil, fmt.Errorf("tool %s: unknown scope %q", t.ToolID, p.Scope)
		}
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return spec, nil
}
