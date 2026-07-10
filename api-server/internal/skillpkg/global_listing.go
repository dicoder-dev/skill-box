// Package skillpkg - global_listing.go
//
// 2026-07-10 增:首页"导入技能"弹窗新增 Tab「全局目录」,直接列出 ~/.agents/skills
// 下所有候选 skill(无依赖缓存,不区分 tool/user)。用户可勾选后批量导入 skillbox 库。
//
// 为什么单独一个文件,不复用 skilladapter.BaseAdapter.Scan:
//   - BaseAdapter.Scan 是"按某个 adapter 的 Tools[scope] 路径去扫",面向的是
//     claude/codex/trae 等工具各自的语义(允许跳过 SystemPaths 之类)。
//   - 这里的诉求是"看 ~/.agents/skills 里到底有哪些 skill",跟具体工具无关,
//     也不该被任何 SystemPaths 跳过(全局目录本身就是 user 根,不是 system)。
//   - 复用 ImportFromFolder 已有的 collectSkillDirs 来做磁盘遍历 + skip-symlink
//     + 深度限制,避免重写一份 walk 逻辑。
//
// 两个公开函数:
//   - ListGlobalSkills(root):扫描 root 下所有 skill 根,产出轻量候选列表(name/version/source_path/description/exists)。
//   - ImportFromPaths(store, paths):按磁盘绝对路径列表逐个导入,聚合出 LocalImportResult。
//
// SourceGlobalPaths 常量定义在 local_import.go(SourceKind 四元组的第 4 项)。
package skillpkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
)

// GlobalSkillCandidate 全局目录下识别出的一个候选 skill。
// 前端用这个渲染候选列表(name/version/source_path + description 辅助搜索)。
type GlobalSkillCandidate struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	SourcePath  string `json:"source_path"` // skill 根目录绝对路径(EvalSymlinks 已规范化)
	Exists      bool   `json:"exists"`     // 磁盘是否仍然存在(扫描后被删时为 false)
}

// ListGlobalSkills 递归扫描 root 下所有"自身含 SKILL.md"的子目录,产出候选列表。
//
// 行为:
//   - root 不存在(用户没建 ~/.agents/skills)→ 返 [] + nil,不是 error(前端展示空列表)。
//   - 单个 SKILL.md 解析失败(无 frontmatter / 无 name)→ 该条跳过,不影响整体。
//   - symlink 目录:不跟随,跟 ImportFromFolder 行为一致(避免越界)。
//   - 复用 collectSkillDirs 拿 skill 根列表,再对每个根 readSkillMeta 读 frontmatter 轻量字段。
func ListGlobalSkills(root string) ([]GlobalSkillCandidate, error) {
	cleaned := filepath.Clean(root)
	if cleaned == "" {
		return nil, errors.New("skillpkg: empty global root")
	}
	if _, err := os.Stat(cleaned); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("skillpkg: stat %s: %w", cleaned, err)
	}

	// 真实路径:避免 symlink 链造成重复扫描同一根。
	realRoot := cleaned
	if r, err := filepath.EvalSymlinks(cleaned); err == nil {
		realRoot = r
	}

	roots, err := collectSkillDirs(realRoot, maxWalkDepth)
	if err != nil {
		return nil, fmt.Errorf("skillpkg: walk %s: %w", cleaned, err)
	}

	out := make([]GlobalSkillCandidate, 0, len(roots))
	for _, dir := range roots {
		c, err := readSkillMeta(dir)
		if err != nil {
			// 单条损坏不阻断整体。
			continue
		}
		out = append(out, c)
	}
	// 按 name 排序,前端展示顺序稳定。
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ImportFromPaths 按磁盘绝对路径列表逐个导入,聚合出 LocalImportResult。
//
// 与 ImportFromFolder 的区别:
//   - ImportFromFolder 是"一个根目录递归",会自动用 collectSkillDirs 找所有 skill 根。
//   - 这里 caller 已经知道要哪些 skill(source_path 已选定),直接对每条调 importOneFromDir
//     落地,跳过 walk 步骤 — 性能更好,且支持"只导入选中的部分"。
//
// 行为:
//   - paths 为空 → 返 ErrNoSkillMD(跟 ImportFromFolder 一致,前端好统一提示)。
//   - 单条解析 / 落盘失败 → 该条 OK=false,其它继续。
func ImportFromPaths(store *skillstore.Store, paths []string) (*LocalImportResult, error) {
	if store == nil {
		return nil, errors.New("skillpkg: nil store")
	}
	if len(paths) == 0 {
		return nil, ErrNoSkillMD
	}

	out := &LocalImportResult{
		Source:     "<global-list>",
		SourceKind: SourceGlobalPaths,
	}

	for _, p := range paths {
		cleaned := filepath.Clean(p)
		// source_path 是 EvalSymlinks 后的真实路径;落盘时允许"目标已不存在"(用户先选了又删)
		// — 单条 fail,不影响其它。
		results := importOneFromDir(store, cleaned)
		out.Results = append(out.Results, results...)
	}
	out.Found = len(paths)
	tallyResults(out)
	return out, nil
}

// readSkillMeta 轻量读一个 skill 根的 frontmatter,只取 name/version/description。
// 不读附属 files(候选列表场景不需要,导入时再走完整 readCanonicalFromDir)。
func readSkillMeta(dir string) (GlobalSkillCandidate, error) {
	skillMDPath := filepath.Join(dir, skillMDName)
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return GlobalSkillCandidate{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	manifest, err := skilladapter.ParseSkillMD(string(content))
	if err != nil {
		return GlobalSkillCandidate{}, fmt.Errorf("parse SKILL.md: %w", err)
	}
	// SourcePath 用 EvalSymlinks 真实路径,跟其它 SourcePath 字段保持一致风格。
	src := dir
	if r, e := filepath.EvalSymlinks(dir); e == nil {
		src = r
	}
	return GlobalSkillCandidate{
		Name:        manifest.Manifest.Name,
		Version:     manifest.Manifest.Version,
		Description: manifest.Manifest.Description,
		SourcePath:  src,
		Exists:      true,
	}, nil
}