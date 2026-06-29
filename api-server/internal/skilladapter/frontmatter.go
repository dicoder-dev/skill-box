package skilladapter

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatterBlock 解析 SKILL.md 顶部的 YAML frontmatter。
//
// 格式约定(与 Anthropic / Codex / Trae / Claude Code 一致):
//
//	---
//	name: foo
//	description: bar
//	version: 0.1.0
//	---
//	# Body...
//
// 返回 (frontmatter YAML, body, error)。无 frontmatter 时 frontmatter 为空字符串,error == nil。
func splitFrontmatter(content string) (fm string, body string, err error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---") {
		return "", normalized, nil
	}
	rest := normalized[3:]
	rest = strings.TrimLeft(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", normalized, errors.New("frontmatter: missing closing ---")
	}
	fm = rest[:end]
	body = strings.TrimLeft(rest[end+4:], "\n")
	return fm, body, nil
}

// ParseSkillMD 解析一个 SKILL.md 内容,产出 Canonical{Manifest, Files}。
//
// 硬错误:
//   - 没有 frontmatter(必须以 "---" 开头)
//   - frontmatter YAML 解析失败
//   - name 既不在 frontmatter 也不在 H1
//
// name 兜底:frontmatter 缺 name 但有 H1 时,从 H1 提取。
func ParseSkillMD(content string) (*Canonical, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("skilladapter: %w", err)
	}
	if fm == "" {
		return nil, errors.New("skilladapter: missing frontmatter (must start with ---)")
	}

	var m Manifest
	if err := yaml.Unmarshal([]byte(fm), &m); err != nil {
		return nil, fmt.Errorf("skilladapter: frontmatter yaml: %w", err)
	}

	// name 兜底:从 H1 提取(仅当 frontmatter 没有时)
	if m.Name == "" {
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				m.Name = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				break
			}
		}
	}
	if m.Name == "" {
		return nil, errors.New("skilladapter: no name in frontmatter or H1")
	}
	m.Name = NormalizeName(m.Name)
	if m.Version == "" {
		m.Version = "0.1.0"
	}

	return &Canonical{
		Manifest: m,
		Files:    []File{{Path: "SKILL.md", Content: content}},
	}, nil
}

// NormalizeName 把任意 name 折叠成 ^[a-z][a-z0-9-]{1,63}$。
// 规则:
//   - 已有 '-' 保留;多个连续 '-' 折叠为单个
//   - ' ' / '_' / '/' / '\\' / '.' 折叠为单个 '-'
//   - 其它非 [a-z0-9-] 字符(包括中文)直接丢弃
//   - 数字开头补 's-' 前缀
//   - 长度超过 64 截断后再 trim 末尾 '-'
//   - 输入为空时返回 ""
func NormalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-', r == ' ', r == '_', r == '/', r == '\\', r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return ""
	}
	if len(out) > 64 {
		out = out[:64]
		out = strings.TrimRight(out, "-")
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "s-" + out
		if len(out) > 64 {
			out = out[:64]
		}
	}
	return out
}

// NormalizeGroupName 把任意分组名折叠成 "a/b/c" 形式的相对路径。
//
// 2026-06-29 增:为支持"分组即子目录"的多级分组,skill 名仍走 NormalizeName
// (不允许 '/'),分组名独立规约,允许 '/',但仍拒绝 .. / 绝对路径 / 空段。
//
// 规则:
//   - '/': 路径分隔符保留,折叠连续 '/' 为单个
//   - 段内允许 [a-z0-9-_](其它字符与 '.' / ' ' 等折叠为 '-')
//   - 拒绝以 '/' 开头或结尾
//   - 拒绝出现 '..' 段
//   - 拒绝空字符串 / 仅含分隔符
//
// 注意:返回的 path 仍需走 skillstore.safeRelPath 二次校验(防非法字符漏过)。
func NormalizeGroupName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "/")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var b strings.Builder
		lastDash := false
		for _, r := range p {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
				b.WriteRune(r)
				lastDash = false
			case r == '-', r == '_':
				if !lastDash && b.Len() > 0 {
					b.WriteByte('-')
					lastDash = true
				}
			default:
				// ' ' / '.' / 其它字符 → 折叠为 '-'
				if !lastDash && b.Len() > 0 {
					b.WriteByte('-')
					lastDash = true
				}
			}
		}
		seg := strings.TrimRight(b.String(), "-")
		if seg == "" {
			continue
		}
		if len(seg) > 64 {
			seg = seg[:64]
			seg = strings.TrimRight(seg, "-")
			if seg == "" {
				continue
			}
		}
		// 数字开头段:补 'g-' 前缀(与 NormalizeName 的 's-' 对齐)
		if seg[0] >= '0' && seg[0] <= '9' {
			seg = "g-" + seg
			if len(seg) > 64 {
				seg = seg[:64]
			}
		}
		// 显式拒绝 '..' 段(虽然 '.' 不在白名单已被折叠成 '-',但留一道防线)
		if seg == "." || seg == ".." {
			continue
		}
		out = append(out, seg)
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "/")
}

// RenderSkillMD 把 Canonical{Manifest, Files} 序列化成 SKILL.md 文本。
// 替换现有 frontmatter(若有);body 优先取第一个 file(通常是 SKILL.md)
// 的内容去掉 frontmatter 后的部分;无 file 时给一个最小 body 兜底
// (避免 caller 传只 Manifest 没 Files 时写出空 SKILL.md)。
func RenderSkillMD(c Canonical) string {
	body := ""
	if len(c.Files) > 0 {
		body = c.Files[0].Content
		if _, existing, err := splitFrontmatter(body); err == nil && existing != "" {
			body = existing
		}
	}
	if body == "" {
		body = "# " + c.Manifest.Name + "\n"
	}
	yml, err := yaml.Marshal(c.Manifest)
	if err != nil {
		yml = []byte("# manifest marshal error\n")
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yml)
	buf.WriteString("---\n")
	buf.WriteString(body)
	return buf.String()
}

// safeRelPath 拒绝 ..、绝对路径、含 \0 等可疑 path。包内复用。
func safeRelPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("absolute path not allowed")
	}
	if strings.Contains(p, "\x00") {
		return "", fmt.Errorf("path contains NUL")
	}
	cleaned := filepath.Clean(p)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal not allowed")
	}
	return cleaned, nil
}

// WriteSkillDir 把 Canonical 物化到 dir(<name>/SKILL.md + 附属文件)。
// 单一来源:SKILL.md 包含 frontmatter 元数据 + 全部正文,其它 File 按 Path 铺平。
// 与 ReadSkillDir 配对使用,保证读写一致。
//
// 行为:
//   - 目录不存在会创建;目标目录已存在时先清空再写(覆盖式,跟旧 skillstore 语义一致)
//   - SKILL.md 是必填(由 caller 构造 Canonical 时保证;本函数兜底:Files 为空
//     或没有 SKILL.md 时,自动用 RenderSkillMD 渲染一份)
//   - 其它 File.Path 必须是相对路径且不含 ..(防穿越)
func WriteSkillDir(dir string, c Canonical) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("skilladapter: mkdir %s: %w", dir, err)
	}
	// 清空目标目录下的文件(. 和 .. 之外的),避免上一次写残留
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("skilladapter: readdir %s: %w", dir, err)
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}

	// 确保 SKILL.md 一定存在
	hasMain := false
	for _, f := range c.Files {
		if f.Path == "SKILL.md" {
			hasMain = true
			break
		}
	}
	if !hasMain {
		c.Files = append([]File{{Path: "SKILL.md", Content: RenderSkillMD(c)}}, c.Files...)
	}

	for _, f := range c.Files {
		if f.Path == "" {
			continue
		}
		rel, err := safeRelPath(f.Path)
		if err != nil {
			return fmt.Errorf("skilladapter: invalid path %q: %w", f.Path, err)
		}
		dst := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("skilladapter: mkdir %s: %w", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("skilladapter: write %s: %w", dst, err)
		}
	}
	return nil
}

// ReadSkillDir 从 dir 读 Canonical(等同 base.go 的 readSkillDir,提到本包对外可见)。
// 约定:目录里必须存在 SKILL.md(作为元数据唯一源);其它附属文件一并加载。
func ReadSkillDir(dir string) (Canonical, error) {
	skillMD := filepath.Join(dir, "SKILL.md")
	content, err := os.ReadFile(skillMD)
	if err != nil {
		return Canonical{}, fmt.Errorf("skilladapter: read %s: %w", skillMD, err)
	}
	c, err := ParseSkillMD(string(content))
	if err != nil {
		return Canonical{}, err
	}
	// 装齐所有文件
	var files []File
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, File{Path: filepath.ToSlash(rel), Content: string(b)})
		return nil
	})
	if err != nil {
		return Canonical{}, fmt.Errorf("skilladapter: walk %s: %w", dir, err)
	}
	// SKILL.md 一定存在(已 ReadFile 过一次),不需要兜底补
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	c.Files = files
	return *c, nil
}
