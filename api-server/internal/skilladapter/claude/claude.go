// Package claude 是 Claude Code 的 Adapter 实现。
//
// Claude Code 本机(2026-06 探测)的 skill 实际放在两处:
//   1) ~/.agents/skills/<name>             ← Agent Skills 标准个人级路径(Anthropic 推行),
//                                            用户日常把 ~/.claude/skills/<name> 用 symlink 指向这里
//   2) ~/.claude/plugins/marketplaces/*/plugins/<plugin>/skills/<name>/SKILL.md
//                                            ← plugin 自带的 skill(深度 ~6 层)
//
// 分档:
//   - user   : ~/.agents/skills 下的 skill(默认勾选,可取消)
//   - system : ~/.claude/plugins/marketplaces 下的 plugin skill(只读参考,不可勾选)
//
// 写盘根 = ~/.agents/skills:Agent Skills 标准个人级路径。Claude / Codex / Trae 三工具
// 共享这一目录,通过各自 symlink 入口(用户日常的 ~/.claude/skills、~/.codex/skills、
// ~/.trae/skills)互相可见。这样 apply 写入后,三个工具都能读取,且不会破坏用户的
// symlink 布局(MkdirAll 写的是真实根,不是 symlink 链上的目录)。
//
// 扫描根 = ~/.agents/skills(写盘根即扫描根,避免 chip 重复):BaseAdapter.Scan 会
// 跟随 symlink,如果用户在 ~/.claude/skills/find-skills 这种 symlink 入口创建了独立
// 真实目录,也会被 Scan 走到(只要该目录最终含 SKILL.md)。
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
				skilladapter.ScopeProject: []string{".agents/skills"},
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
