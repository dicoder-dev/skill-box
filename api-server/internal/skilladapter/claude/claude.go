// Package claude 是 Claude Code 的 Adapter 实现。
//
// Claude Code 本机(2026-06 探测)的 skill 实际放在 plugin marketplaces 下:
//   ~/.claude/plugins/marketplaces/*/plugins/<plugin>/skills/<name>/SKILL.md
// Claude 没有自己的 ~/.claude/skills/ 目录(只有 plugins / projects / cache 等)。
// 我们把整个 marketplaces 根作为 scan 起点,BaseAdapter.Scan 会逐层 walk。
//
// 全部按 BaseAdapter 通用逻辑处理(目录 + SKILL.md + YAML frontmatter)。
package claude

import (
	"os"
	"strings"
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
		if home != "" {
			for _, p := range []string{"~/.claude/plugins/marketplaces"} {
				p = strings.TrimPrefix(p, "~/")
				global = append(global, filepath.Join(home, p))
			}
		}
		Adapter.base = &skilladapter.BaseAdapter{
			ID:        id,
			Display:   "Claude Code",
			IconEmoji: "ð¤",
			Tools: map[string][]string{
				skilladapter.ScopeGlobal:  global,
				skilladapter.ScopeProject: []string{".claude/skills"},
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
