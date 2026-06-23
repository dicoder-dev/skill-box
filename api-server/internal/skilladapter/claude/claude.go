// Package claude 是 Claude Code 的 Adapter 实现。
//
// Claude Code 本机(2026-06 探测)的 skill 实际放在两处:
//   1) ~/.claude/skills/<name>             ← 用户日常用的 skill(以 symlink 形式存在,目标在 ~/.agents/skills 等)
//   2) ~/.claude/plugins/marketplaces/*/plugins/<plugin>/skills/<name>/SKILL.md
//                                            ← plugin 自带的 skill(深度 ~6 层)
//
// 分档:
//   - user   : ~/.claude/skills 下的 symlink skill(默认勾选,可取消)
//   - system : ~/.claude/plugins/marketplaces 下的 plugin skill(只读参考,不可勾选)
//
// 全都加进扫描根目录,BaseAdapter.Scan 会递归 walk;symlink 也会被 BaseAdapter.Scan 跟随。
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
			// ~/.claude/skills 是 Claude Code 真正的用户 skill 目录,
			// 里面是 symlink(目标一般在 ~/.agents/skills/...),BaseAdapter.Scan 会跟随。
			global = append(global, filepath.Join(home, ".claude", "skills"))
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
