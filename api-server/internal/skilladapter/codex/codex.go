// Package codex 是 Codex 的 Adapter 实现。
//
// Codex 在本机(2026-06 探测)的 skill 目录:
//
//	~/.agents/skills/<name>/SKILL.md                  ← Agent Skills 标准个人级路径(Anthropic 推行)
//	~/.codex/skills/<name>/SKILL.md                  ← 用户日常 symlink 入口(指向 ~/.agents/skills)
//	~/.codex/skills/.system/<name>/SKILL.md          ← system 级 skill
//	~/.codex/vendor_imports/skills/skills/.curated/<name>/SKILL.md ← system 级(vendor curated)
//	<project>/.agents/skills/<name>/SKILL.md         ← 项目级(Agent Skills 标准)
//
// 分档:
//   - user   : ~/.agents/skills(默认勾选,可取消)
//   - system : .system / vendor_imports/.curated(只读参考,不可勾选)
//
// 写盘根 = ~/.agents/skills(个人级)/ <project>/.agents/skills(项目级):
// 与 Claude / Trae 共享同一标准目录,通过 symlink 互相可见。Codex 读取这些 skill 时
// 走 ~/.agents/skills,所以写入这个根后工具才能真正加载。
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
		var system []string
		if home != "" {
			// 写盘 + 扫描根 = ~/.agents/skills(Agent Skills 标准个人级)。
			global = append(global, filepath.Join(home, ".agents", "skills"))
			// system 根:.system 是 Codex 自带;vendor_imports/.curated 是 vendor curated
			system = append(system,
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
				skilladapter.ScopeProject: {".agents/skills"},
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
