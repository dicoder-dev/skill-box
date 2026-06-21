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
}

func (b *BaseAdapter) ToolID() string      { return b.ID }
func (b *BaseAdapter) DisplayName() string { return b.Display }
func (b *BaseAdapter) Icon() string        { return b.IconEmoji }

func (b *BaseAdapter) DiscoverPaths(scope string) ([]string, error) {
	return b.Tools[scope], nil
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

// Scan 扫描单个目录,识别所有 "<name>/SKILL.md" 形式的 skill。
// 损坏 / 缺 frontmatter 的子目录跳过,不中断整体扫描。
func (b *BaseAdapter) Scan(dir string) ([]Canonical, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Canonical
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// 跳过隐藏目录(.system / .curated 等)
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 跟随 symlink(本机 codex/skills/find-skills 就是 symlink 到 .agents/skills)
		skillDir := filepath.Join(dir, e.Name())
		if _, err := filepath.EvalSymlinks(skillDir); err == nil {
			// symlink 解析成功就用解析后的;失败回退原路径
		}
		c, err := readSkillDir(skillDir)
		if err != nil {
			// 单个 skill 损坏不影响整体,跳过
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out, nil
}

// readSkillDir 读取一个 skill 目录,产出 Canonical(只填 SKILL.md 一个文件;
// 其它附属文件 v1 不导入,可在 P1 加)。
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
