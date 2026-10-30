package agentspecs

import (
	"fmt"
	"sort"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/magent"
	"ginp-api/internal/skilladapter"

	"gorm.io/gorm"
)

// LoadAllFromDB 从 tools + tool_paths 表加载全部 enabled=true 的 Agent,
// 转成 []*AgentSpec。
//
// 2026-06-30 二改:替代原 LoadAll() 的"读 yaml"实现,Agent 元数据从编译期
// 内嵌变成运行时 DB 拉。每次启动 / 每次 reload 都会调一次。
//
// 关键约束:
//   - 排除 enabled=false 的 Agent(用户禁用,不注册 adapter)
//   - 同一 tool_id 在 DB 层面靠 uniqueIndex 兜底;这里再校验一次(防御
//     race condition / DB 异常)
//   - 每个 agent 的 path 按 (scope, category, path_order) 排序后填入 AgentSpec.Paths
func LoadAllFromDB(db *gorm.DB) ([]*AgentSpec, error) {
	agentM := magent.NewModel(db, db)
	pathM := magent.NewAgentPathModel(db, db)

	agents, err := agentM.ListAllEnabled()
	if err != nil {
		return nil, fmt.Errorf("agentspecs: list enabled agents: %w", err)
	}
	if len(agents) == 0 {
		return nil, nil
	}
	agentIDs := make([]uint, len(agents))
	for i, t := range agents {
		agentIDs[i] = t.ID
	}
	pathsByAgent, err := pathM.FindAllByAgentIDs(agentIDs)
	if err != nil {
		return nil, fmt.Errorf("agentspecs: list paths: %w", err)
	}

	out := make([]*AgentSpec, 0, len(agents))
	seen := make(map[string]bool, len(agents))
	for _, t := range agents {
		if seen[t.AgentID] {
			return nil, fmt.Errorf("agentspecs: duplicate tool_id %q in DB", t.AgentID)
		}
		seen[t.AgentID] = true

		paths := pathsByAgent[t.ID]
		spec, err := toSpec(t, paths)
		if err != nil {
			return nil, fmt.Errorf("agentspecs: convert %s: %w", t.AgentID, err)
		}
		out = append(out, spec)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].AgentID < out[j].AgentID })
	return out, nil
}

// toSpec entity.Agent + []entity.AgentPath → AgentSpec。
// 失败:Validate 不通过(空 tool_id / 空 display_name / 空 mdi_icon / 路径为空)。
func toSpec(t *entity.Agent, paths []*entity.AgentPath) (*AgentSpec, error) {
	spec := &AgentSpec{
		AgentID:     t.AgentID,
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
				return nil, fmt.Errorf("agent %s: unknown category %q for global", t.AgentID, p.Category)
			}
		case skilladapter.ScopeProject:
			switch p.Category {
			case "user":
				spec.Paths.Project.User = append(spec.Paths.Project.User, p.Path)
			case "system":
				spec.Paths.Project.System = append(spec.Paths.Project.System, p.Path)
			default:
				return nil, fmt.Errorf("agent %s: unknown category %q for project", t.AgentID, p.Category)
			}
		default:
			return nil, fmt.Errorf("agent %s: unknown scope %q", t.AgentID, p.Scope)
		}
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return spec, nil
}
