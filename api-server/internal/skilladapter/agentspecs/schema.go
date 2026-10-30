package toolspecs

import (
	"errors"
	"fmt"
	"strings"
)

// ToolSpec 单个 AI 编程工具的静态元数据,从 specs/<tool>.yaml 加载。
//
// 字段定义与 skilladapter.BaseAdapter 字段一一对应,加载后由
// skilladapter.NewSpecAdapter 转成 BaseAdapter。
type ToolSpec struct {
	// ToolID 工具唯一 ID,用于路由、落库、前端 mdi 查找。
	// 一旦定下不要轻易改 — DB / 前端都靠它做关联。
	ToolID string `yaml:"tool_id" json:"tool_id"`

	// DisplayName UI 展示名(中英混合,前端 i18n 可覆盖)。
	DisplayName string `yaml:"display_name" json:"display_name"`

	// MdiIcon 前端 iconify 用的 mdi 图标名,格式 "mdi:xxx"。
	// 后端 Adapter.Icon() 直接返回这个字符串,前端 SkillsView 的 TOOL_ICON_MAP
	// 不再需要。
	// 命名约定:可读语义优先(如 claude → mdi:robot-outline,codex → mdi:console),
	// 找不到合适的用 mdi:puzzle-outline 占位。
	MdiIcon string `yaml:"mdi_icon" json:"mdi_icon"`

	// Maturity 工具的稳定度,可选值:stable / experimental / deprecated。
	// stable:已实测可正常工作(读、写、扫都能找到 SKILL.md);
	// experimental:路径是社区约定/官方未明确,可能在用户机器上找不到;
	// deprecated:工具已停维,保留只是不破坏旧数据。
	Maturity string `yaml:"maturity,omitempty" json:"maturity,omitempty"`

	// Note 自由文本,用于在 YAML 注释之外补充一些"为什么这样配"的说明。
	// 前端不展示,仅供阅读 / 日志排查。
	Note string `yaml:"note,omitempty" json:"note,omitempty"`

	// Paths 路径配置,scope × category 笛卡尔积。
	//   - Scope: global(用户级,挂在 $HOME 下) / project(项目级,挂在 <project>/.xxx/skills)
	//   - Category: user(用户自己装,可读可写) / system(工具自带 / vendor,只读)
	//
	// 任一 scope × category 都可省略,意味该档位没有扫描根(仍允许 scope 的其它档位工作)。
	Paths ToolPaths `yaml:"paths" json:"paths"`
}

// ToolPaths 全局 / 项目级的路径配置。
type ToolPaths struct {
	Global  CategoryPaths `yaml:"global" json:"global"`
	Project CategoryPaths `yaml:"project" json:"project"`
}

// CategoryPaths user / system 两类扫描根。
// 至少一个为非空,否则该 scope 没意义。
type CategoryPaths struct {
	User   []string `yaml:"user,omitempty" json:"user,omitempty"`
	System []string `yaml:"system,omitempty" json:"system,omitempty"`
}

// Validate 校验 spec 合法性 — 加载阶段必须全部通过,否则 fail-fast。
func (s *ToolSpec) Validate() error {
	if strings.TrimSpace(s.ToolID) == "" {
		return errors.New("tool_id is required")
	}
	if strings.TrimSpace(s.DisplayName) == "" {
		return fmt.Errorf("%s: display_name is required", s.ToolID)
	}
	if strings.TrimSpace(s.MdiIcon) == "" {
		return fmt.Errorf("%s: mdi_icon is required", s.ToolID)
	}
	if s.Maturity != "" {
		switch s.Maturity {
		case "stable", "experimental", "deprecated":
			// ok
		default:
			return fmt.Errorf("%s: maturity must be stable|experimental|deprecated, got %q", s.ToolID, s.Maturity)
		}
	}
	if err := s.Paths.Validate(s.ToolID); err != nil {
		return err
	}
	return nil
}

// Validate 校验路径配置:每个 scope 至少有一个非空 category。
func (p *ToolPaths) Validate(toolID string) error {
	if err := p.Global.validateOne(toolID, "global"); err != nil {
		return err
	}
	if err := p.Project.validateOne(toolID, "project"); err != nil {
		return err
	}
	return nil
}

func (c *CategoryPaths) validateOne(toolID, scope string) error {
	if len(c.User) == 0 && len(c.System) == 0 {
		return fmt.Errorf("%s: paths.%s is empty (need at least one user or system path)", toolID, scope)
	}
	return nil
}
