// Package trae 是 Trae 的 Adapter 实现。
//
// Trae 本机(2026-06 探测)skill 实际放在 ~/.agents/skills(Agent Skills 标准个人级),
// ~/.trae/skills 是 symlink 入口(用户日常用)。skill-box 写盘根改为 ~/.agents/skills
// 与 Claude / Codex 对齐,避免 MkdirAll 破坏用户 symlink 布局,同时让三个工具能共享
// 同一份 skill。
//
// 项目级则用 <project>/.trae/skills/(跟全局入口 ~/.trae/skills 命名一致)——
// 项目级不像全局有 symlink 兜底,必须直接落到工具自身目录,Trae 才能读到。
//
// 全部按 BaseAdapter 通用逻辑处理(目录 + SKILL.md + YAML frontmatter)。
package trae

import (
	"os"
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
			// 写盘 + 扫描根 = ~/.agents/skills(Agent Skills 标准)。
			global = append(global, filepath.Join(home, ".agents", "skills"))
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
