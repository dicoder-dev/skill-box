// Package skillbundle 把 6 个预置 skill 用 go:embed 打入二进制,启动时自动 seed 到 store + DB。
//
// 设计要点(见 docs/project/需求规划.md 第 13 节):
//   - 内置 skill 与用户自建 skill 同形:走同一条 sskill.Create 路径(落盘 + 落库)
//   - source = "bundle",source_ref = "skillbox/<version>" 标识来源
//   - 已存在同 (scope=global, project_id=0, name, version) 时跳过(seed 是幂等的)
//   - 启动失败不阻塞服务(DB 写入错误只 log,不 panic);skill 缺失可下次启动再补
//   - LoadAll 在测试里复用(测试不写库,只验证 parse 是否过)
package skillbundle

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"ginp-api/internal/skilladapter"
)

//go:embed presets
var presetsFS embed.FS

// PresetMeta 描述一个预置 skill 的最小元信息。
type PresetMeta struct {
	Key     string // 目录名,如 "code-review"
	Subpath string // 在 embed.FS 里的路径,如 "presets/code-review"
}

// All 内置 skill 列表(固定顺序,启动时按这个顺序 seed)。
// 注意:不依赖 fs.ReadDir 的顺序(embed 内部按字典序)。
var All = []PresetMeta{
	{Key: "code-review", Subpath: "presets/code-review"},
	{Key: "commit-msg", Subpath: "presets/commit-msg"},
	{Key: "debug-helper", Subpath: "presets/debug-helper"},
	{Key: "doc-generator", Subpath: "presets/doc-generator"},
	{Key: "unit-test-gen", Subpath: "presets/unit-test-gen"},
	{Key: "perf-opt", Subpath: "presets/perf-opt"},
}

// LoadAll 解析所有预置 skill,产出 Canonical 列表。
// 任何单个 skill 解析失败都立即返回 error(整批不进 store)。
// 顺序与 All 一致。
func LoadAll() ([]skilladapter.Canonical, error) {
	out := make([]skilladapter.Canonical, 0, len(All))
	for _, p := range All {
		c, err := LoadOne(p.Subpath)
		if err != nil {
			return nil, fmt.Errorf("skillbundle: load %s: %w", p.Key, err)
		}
		out = append(out, *c)
	}
	return out, nil
}

// LoadOne 从 embed.FS 解析单个预置 skill。
//
// 目录布局(与磁盘上的 skill 一致):
//
//	<Subpath>/SKILL.md          — frontmatter + body,parse 后得 Manifest
//	<Subpath>/examples/*.sh     — 可选,按 Path 注入 Files(相对 Subpath)
//
// SKILL.md 必须存在;examples/ 缺失是 OK 的。
func LoadOne(subpath string) (*skilladapter.Canonical, error) {
	skillMDBytes, err := presetsFS.ReadFile(subpath + "/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}
	c, err := skilladapter.ParseSkillMD(string(skillMDBytes))
	if err != nil {
		return nil, fmt.Errorf("parse SKILL.md: %w", err)
	}

	// 收集 examples/ 下所有非目录文件
	files := append([]skilladapter.File{}, c.Files...)
	entries, err := presetsFS.ReadDir(subpath + "/examples")
	if err == nil {
		// 有 examples/ 才追加;目录缺失不算错
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := "examples/" + e.Name()
			data, err := presetsFS.ReadFile(subpath + "/" + path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", path, err)
			}
			files = append(files, skilladapter.File{
				Path:    path,
				Content: string(data),
			})
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	} else if !errorsIsNotExist(err) {
		return nil, fmt.Errorf("read examples dir: %w", err)
	}

	c.Files = files
	return c, nil
}

// Keys 返回所有预置 skill 的 key(用于 UI 列表 / 文档生成)。
func Keys() []string {
	keys := make([]string, len(All))
	for i, p := range All {
		keys[i] = p.Key
	}
	return keys
}

// errorsIsNotExist embed.FS 在路径不存在时返回 *PathError;不依赖 syscall.ENOENT
// 这种系统细节,直接判断子串(embed 错误格式固定)。
func errorsIsNotExist(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "file does not exist") || strings.HasSuffix(msg, ": no such file or directory")
}

// PresetsFS 暴露 embed.FS(只读)给需要列目录的调用方(目前没有,保留)。
func PresetsFS() fs.FS {
	return presetsFS
}