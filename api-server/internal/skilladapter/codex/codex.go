// Package codex 是 Codex 的 Adapter 实现。
//
// Codex 在本机(2026-06 探测)的 skill 目录:
//
//	~/.codex/skills/                          ← 用户/链接的 skill
//	~/.codex/skills/.system/<name>/SKILL.md   ← 系统内置
//	~/.codex/vendor_imports/skills/skills/.curated/<name>/SKILL.md
//	<project>/.codex/skills/<name>/SKILL.md   ← 项目级
//
// 全部按 BaseAdapter 通用逻辑处理(目录 + SKILL.md + YAML frontmatter)。
package codex

import (
	"os"
	"path/filepath"
	"sync"

	"ginp-api/internal/skilladapter"
)

const id = skilladapter.ToolCodex

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
			global = append(global,
				filepath.Join(home, ".codex", "skills"),
				filepath.Join(home, ".codex", "skills", ".system"),
				filepath.Join(home, ".codex", "vendor_imports", "skills", "skills", ".curated"),
			)
		}
		Adapter.base = &skilladapter.BaseAdapter{
			ID:        id,
			Display:   "Codex",
			// IconEmoji 已废弃:项目规范禁止 emoji 作为图标。前端按 tool_id
			// 映射 mdi 图标渲染。这里留空串避免向前端输出乱码字节。
			IconEmoji: "",
			Tools: map[string][]string{
				skilladapter.ScopeGlobal:  global,
				skilladapter.ScopeProject: {".codex/skills"},
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
