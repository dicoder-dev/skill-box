// Package claude 是 Claude Code 的 Adapter 实现。
//
// Claude Code 本机(2026-06 探测)的 skill 实际放在两处:
//   1) ~/.agents/skills/<name>             ← Agent Skills 标准个人级路径(Anthropic 推行),
//                                            用户日常把 ~/.claude/skills/<name> 用 symlink 指向这里
//   2) ~/.claude/plugins/marketplaces/*/plugins/<plugin>/skills/<name>/SKILL.md
//                                            ← plugin 自带的 skill(深度 ~6 层)
//   3) <project>/.claude/skills/<name>      ← 项目级(Claude Code 官方文档明确这条)
//
// 分档:
//   - user   : ~/.agents/skills 下的 skill(默认勾选,可取消)
//   - system : ~/.claude/plugins/marketplaces 下的 plugin skill(只读参考,不可勾选)
//
// 写盘根:
//   - global  = ~/.agents/skills(Agent Skills 标准个人级路径;~/.claude/skills 是 symlink
//     入口,写入真实根避免 MkdirAll 破坏 symlink,且三工具 Claude/Codex/Trae 共享同一目录)
//   - project = <project>/.claude/skills(Claude Code 官方文档要求,跟 Codex 的
//     .agents/skills 不同 — 不要因为"个人级统一"就盲目推广到项目级)
//
// 全部按 BaseAdapter 通用逻辑处理(目录 + SKILL.md + YAML frontmatter)。
package claude

import (
	"os"
	"path/filepath"
	"sync"

	"ginp-api/internal/skilladapter"
)

const id = skilladapter.ToolClaude

type adapter struct{ base *skilladapter.BaseAdapter }

var (
	registerOnce sync.Once
	Adapter      = &adapter{}
)

func init() { Register() }

// Register 在 init() 与测试里都会调,内部用 sync.Once 防重复。
func Register() {
	registerOnce.Do(func() {
		home, _ := os.UserHomeDir()
		var global []string
		var system []string
		if home != "" {
			// 写盘 + 扫描根 = ~/.agents/skills(Agent Skills 标准)。
			global = append(global, filepath.Join(home, ".agents", "skills"))
			// ~/.claude/plugins/marketplaces 下是 plugin 自带的 skill(深度 6 层),
			// 在 phase2 列表里归为 system,只读展示、不可勾选。
			system = append(system, filepath.Join(home, ".claude", "plugins", "marketplaces"))
		}
		Adapter.base = &skilladapter.BaseAdapter{
			ID:        id,
			Display:   "Claude Code",
			IconEmoji: "", // 已废弃:项目规范禁止 emoji 作为图标,前端按 tool_id 映射 mdi 图标。
			Tools: map[string][]string{
				skilladapter.ScopeGlobal:  global,
				skilladapter.ScopeProject: []string{".claude/skills"},
			},
			SystemPaths: map[string][]string{
				skilladapter.ScopeGlobal: system,
			},
		}
		skilladapter.Register(Adapter)
	})
}

func (a *adapter) ToolID() string      { return a.base.ToolID() }
func (a *adapter) DisplayName() string { return a.base.DisplayName() }
func (a *adapter) Icon() string        { return a.base.Icon() }
func (a *adapter) DiscoverPaths(s string) ([]string, error) {
	return a.base.DiscoverPaths(s)
}
func (a *adapter) Scan(dir string) ([]skilladapter.Canonical, error) {
	return a.base.Scan(dir)
}
func (a *adapter) Apply(c skilladapter.Canonical, targetDir string) error {
	return a.base.Apply(c, targetDir)
}
func (a *adapter) LocalName(c skilladapter.Canonical) string {
	return a.base.LocalName(c)
}
func (a *adapter) Validate(c skilladapter.Canonical) error {
	return a.base.Validate(c)
}
func (a *adapter) IsSystemPath(p string) bool { return a.base.IsSystemPath(p) }
