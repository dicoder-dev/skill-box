// Package trae 是 Trae 的 Adapter 实现。
//
// Trae 本机(2026-06 探测)已装 5 个 skill:ui-ux-pro-max / find-skills / flutter-*
//
// 全部按 BaseAdapter 通用逻辑处理(目录 + SKILL.md + YAML frontmatter)。
package trae

import (
	"os"
	"strings"
	"path/filepath"
	"sync"

	"ginp-api/internal/skilladapter"
)

const id = skilladapter.ToolTrae

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
			for _, p := range []string{"~/.trae/skills"} {
				p = strings.TrimPrefix(p, "~/")
				global = append(global, filepath.Join(home, p))
			}
		}
		Adapter.base = &skilladapter.BaseAdapter{
			ID:        id,
			Display:   "Trae",
			IconEmoji: "", // 已废弃:项目规范禁止 emoji 作为图标,前端按 tool_id 映射 mdi 图标。
			Tools: map[string][]string{
				skilladapter.ScopeGlobal:  global,
				skilladapter.ScopeProject: []string{".trae/skills"},
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
