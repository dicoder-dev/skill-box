package toolspecs

import (
	"os"
	"path/filepath"
	"strings"

	"ginp-api/internal/skilladapter"
)

// NewSpecAdapter 根据 ToolSpec 构造 skilladapter.BaseAdapter。
//
// 取代 5 个旧 adapter 子包(claude/codex/cursor/opencode/trae)的 boilerplate,
// 由 toolspecs.init() 在启动期一次性调用,把全部 spec 注册到 default registry。
//
// 关键转换:
//
//   - ToolSpec.Paths.*.User / System → BaseAdapter.Tools / SystemPaths
//   - 把 "~/" 展开为 $HOME 绝对路径(原 5 个 adapter 子包自己做,
//     集成后由本工厂统一处理,避免每个 spec 文件夹出现 os.UserHomeDir 重复代码)
//   - MdiIcon 写入 BaseAdapter.IconEmoji 字段 — 字段名沿用,内容换成 mdi 字符串
//     (BaseAdapter.Icon() 直接返回 IconEmoji)
//
// 项目级路径(.claude/skills、.agents/skills)是相对路径,不展开,原样透传。
func NewSpecAdapter(spec *ToolSpec) *skilladapter.BaseAdapter {
	home, _ := os.UserHomeDir()
	expand := func(p string) string {
		if strings.HasPrefix(p, "~/") {
			if home == "" {
				return p // 退化,跟旧 adapter 行为一致
			}
			return filepath.Join(home, strings.TrimPrefix(p, "~/"))
		}
		return p
	}

	user := make(map[string][]string, 2)
	system := make(map[string][]string, 2)

	for _, scope := range []string{skilladapter.ScopeGlobal, skilladapter.ScopeProject} {
		cp := pathsForScope(spec, scope)
		user[scope] = expandAll(cp.User, expand)
		if len(cp.System) > 0 {
			system[scope] = expandAll(cp.System, expand)
		}
	}

	return &skilladapter.BaseAdapter{
		ID:      spec.ToolID,
		Display: spec.DisplayName,
		// 字段名沿用 IconEmoji,语义已统一为 mdi 名字符串(2026-06-30 改造)。
		// 旧 emoji 内容已废弃,BaseAdapter.Icon() 直接返回该字段。
		IconEmoji: spec.MdiIcon,
		Tools:     user,
		SystemPaths: system,
	}
}

func pathsForScope(spec *ToolSpec, scope string) CategoryPaths {
	switch scope {
	case skilladapter.ScopeGlobal:
		return spec.Paths.Global
	case skilladapter.ScopeProject:
		return spec.Paths.Project
	default:
		return CategoryPaths{}
	}
}

func expandAll(paths []string, expand func(string) string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = expand(p)
	}
	return out
}