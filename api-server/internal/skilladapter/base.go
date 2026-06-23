package skilladapter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BaseAdapter 提供 Adapter 的通用实现,具体工具只需填充 DiscoverPaths + LocalName。
//
// 假设:目标工具的 skill 布局是 "<skills_dir>/<name>/SKILL.md" 单文件目录结构
// (Trae / Codex / OpenCode / Claude 都是这样,只有 frontmatter 字段裁剪不同)。
type BaseAdapter struct {
	ID      string
	Display string
	// IconEmoji 用于前端展示;不需要图标可留空。
	IconEmoji string
	// LocalNameFn canonical name → 目标工具的最终目录名。nil 时用 manifest.Name。
	LocalNameFn func(c Canonical) string
	// Tools 提供每个 scope 下的候选 skills 根目录;DiscoverPaths 直接返回。
	Tools map[string][]string
	// SystemPaths 标记哪些扫描根属于"系统级"(plugin 自带 / vendor curated 等),
	// 前端 phase2 会把扫出来的 skill 列为不可勾选的"只读参考",避免把工具自带
	// 的 skill 当作 user skill 误导入覆盖。空 map 表示该工具没有 system 路径。
	SystemPaths map[string][]string
}

func (b *BaseAdapter) ToolID() string      { return b.ID }
func (b *BaseAdapter) DisplayName() string { return b.Display }
func (b *BaseAdapter) Icon() string        { return b.IconEmoji }

func (b *BaseAdapter) DiscoverPaths(scope string) ([]string, error) {
	// 合并 user 根(Tools)与 system 根(SystemPaths),统一返回;
	// caller 通过 IsSystemPath 判断每条根的档位。
	seen := make(map[string]bool)
	var out []string
	for _, p := range b.Tools[scope] {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, p := range b.SystemPaths[scope] {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out, nil
}

// IsSystemPath 判定给定的扫描根路径是否属于该 adapter 的 system 级别。
//
// 路径比较前先 EvalSymlinks 拿真实路径,再用字符串前缀匹配 ——
// 这样 symlink (如 ~/.claude/skills/<name> → ~/.agents/skills/xxx) 不影响分类,
// 但仍然要避免一个 system 路径恰好是 user 路径前缀的乌龙,所以同时要求
// "完全相等 或 后跟分隔符"。
func (b *BaseAdapter) IsSystemPath(p string) bool {
	if len(b.SystemPaths) == 0 {
		return false
	}
	realP, err := filepath.EvalSymlinks(p)
	if err != nil {
		realP = p
	}
	for _, list := range b.SystemPaths {
		for _, sp := range list {
			realSP, err := filepath.EvalSymlinks(sp)
			if err != nil {
				realSP = sp
			}
			if realP == realSP {
				return true
			}
			if strings.HasPrefix(realP, realSP+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}

func (b *BaseAdapter) LocalName(c Canonical) string {
	if b.LocalNameFn != nil {
		return b.LocalNameFn(c)
	}
	return c.Manifest.Name
}

func (b *BaseAdapter) Validate(c Canonical) error {
	if c.Manifest.Name == "" {
		return fmt.Errorf("%s: skill name is empty", b.ID)
	}
	return nil
}

// maxScanDepth 递归扫描的最大深度,防止 symlink 环 / 异常嵌套死循环。
// 8 层足够覆盖 Claude marketplaces(marketplaces/<m>/plugins/<p>/skills/<n> = 5 层)
const maxScanDepth = 8

// Scan 递归扫描 root 下的所有子目录,识别"自身包含 SKILL.md 文件"的目录作为 skill 根。
//
// 实现要点(2026-06-23 重写):
//   - 递归:Claude marketplaces 路径深 4~5 层(marketplaces/<m>/plugins/<p>/skills/<n>),
//     1 层 ReadDir 不够。
//   - 不主动跳 . 开头的目录:.system / .curated / .agents 等都是合法 skill 容器
//     (Codex 的 .system / .curated 即是例子)。
//   - 跟随 symlink:Trae 的 skill 全部以 symlink 形式存在(../../.agents/skills/xxx);
//     os.ReadDir 的 entry.IsDir() 对 symlink 指向目录时返回 false,需要 os.Stat 二次确认。
//   - 用 EvalSymlinks 真实路径做去重,防止 symlink 环导致死循环 / 重复发现。
//   - 限最大深度 maxScanDepth 兜底。
//   - 跳过 system 子树:本 adapter 声明的 SystemPaths(无论是哪个 scope)
//     都会被作为"跳过集合",在 user 根扫描时不会下钻到对应子路径,避免
//     system skill 被当作 user skill 重复发现。
//
// root 自身若是 system 路径仍允许扫描 —— caller 显式以 system 根调用时,
// 我们要把这个根下的 skill 全部识别为 system 类别。
func (b *BaseAdapter) Scan(root string) ([]Canonical, error) {
	// 入口不存在当 0 个,不是 error(与原行为一致)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Canonical
	seen := make(map[string]bool)
	skip := b.systemPathSet()
	// root 自身从 skip 集合里移除:它要么不是 system 路径,要么是 caller
	// 主动以 system 根身份调进来(由 importer 打 category=system),都该扫。
	if real, err := filepath.EvalSymlinks(root); err == nil {
		delete(skip, real)
	} else {
		delete(skip, root)
	}
	walkSkills(root, 0, seen, skip, &out)
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out, nil
}

// systemPathSet 把 SystemPaths(跨 scope)摊平 + 解析 symlink 真实路径,
// 给 walkSkills 作为"子树跳过表"用。
func (b *BaseAdapter) systemPathSet() map[string]bool {
	if len(b.SystemPaths) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, list := range b.SystemPaths {
		for _, sp := range list {
			real, err := filepath.EvalSymlinks(sp)
			if err != nil {
				real = sp
			}
			set[real] = true
		}
	}
	return set
}

// walkSkills 递归向下找"自身有 SKILL.md 的目录"。
// 找到后该目录视为一个 skill,不再下钻;否则继续向下递归。
//
// skip 是 system 路径真实路径集合 —— 当前路径(以及它的祖先)命中集合时,不再下钻。
// 这样 BaseAdapter.Scan(user_root) 会自动绕开 SystemPaths 下的子树,避免重复。
func walkSkills(dir string, depth int, seen map[string]bool, skip map[string]bool, out *[]Canonical) {
	if depth > maxScanDepth {
		return
	}
	// 当前目录是 system 根 → 直接跳过,不视为 skill 也不下钻
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		if skip[real] {
			return
		}
		if seen[real] {
			return
		}
		seen[real] = true
	} else {
		// EvalSymlinks 失败(如 symlink 损坏)也用原始路径做一次去重
		if seen[dir] {
			return
		}
		seen[dir] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		// 跳过显式元数据文件(.DS_Store / .codex-system-skills.marker 等),它们不是 skill
		if !looksLikeSkillContainer(name) {
			continue
		}
		path := filepath.Join(dir, name)
		// 子目录命中 system 集合 → 整个子树跳过,不视为 skill 不下钻
		if real, err := filepath.EvalSymlinks(path); err == nil && skip[real] {
			continue
		}
		// os.Stat 自动跟随 symlink;对 symlink → 目录的情况也能正确判定为目录
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		// 自身有 SKILL.md → 视为 skill 根
		if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err == nil {
			c, err := readSkillDir(path)
			if err != nil {
				continue // 损坏的 skill 跳过,不影响整体
			}
			*out = append(*out, c)
			continue
		}
		// 没有 SKILL.md → 继续下钻(Claude marketplaces 的中间层)
		walkSkills(path, depth+1, seen, skip, out)
	}
}

// looksLikeSkillContainer 过滤掉明显不是 skill 容器的条目。
//
// 当前规则(保守):
//   - 真实文件(.DS_Store / *.md / *.json ...)直接 false;
//     symlink → 文件 / symlink → 目录会让 os.Stat 决定,这里只做名字初筛。
//   - 名字以 . 开头的目录(.system / .curated / .agents)允许进入;
//     readSkillDir 会进一步校验 SKILL.md 是否存在。
//   - 名字以 . 开头且长度很短(<=2)且包含在已知元数据集合内 → 跳过;
//     但目前没收集到这种场景,先按"不主动跳"实现,后续按需细化。
func looksLikeSkillContainer(name string) bool {
	if name == "" {
		return false
	}
	// 已知元数据文件(精确名)
	switch name {
	case ".DS_Store", "Thumbs.db":
		return false
	}
	// 已知元数据文件(扩展名)
	if strings.HasSuffix(name, ".marker") {
		return false
	}
	// 其它一律放行,包括 .system / .curated / .agents 这类隐藏目录
	return true
}

// readSkillDir 读取一个 skill 目录,产出 Canonical(只填 SKILL.md 一个文件;
// 其它附属文件 v1 不导入,可在 P1 加)。
// 真实目录绝对路径同时写入 c.SourceDir,供 importer 产出 FoundSkill.SourcePath。
func readSkillDir(dir string) (Canonical, error) {
	skillMD := filepath.Join(dir, "SKILL.md")
	content, err := os.ReadFile(skillMD)
	if err != nil {
		return Canonical{}, err
	}
	c, err := ParseSkillMD(string(content))
	if err != nil {
		return Canonical{}, err
	}
	c.Files = []File{{Path: "SKILL.md", Content: string(content)}}
	// 用 EvalSymlinks 解析真实路径,避免 symlink 链上 source_path 一会儿是
	// ~/.claude/skills 一会儿是 ~/.agents/skills/xxx,便于前端稳定展示。
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		c.SourceDir = real
	} else {
		c.SourceDir = dir
	}
	return *c, nil
}

// Apply 把 canonical 写到 targetDir。
// targetDir 必须已存在(adapter 不创建顶层);本函数负责写 SKILL.md + 附属文件。
func (b *BaseAdapter) Apply(c Canonical, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("%s: mkdir %s: %w", b.ID, targetDir, err)
	}
	for _, f := range c.Files {
		if f.Path == "" {
			continue
		}
		dst := filepath.Join(targetDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
