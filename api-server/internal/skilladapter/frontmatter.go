package skilladapter

import (
	"bytes"
	"errors"
	"fmt"
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

// RenderSkillMD 把 Canonical{Manifest, Files} 序列化成 SKILL.md 文本。
// 替换现有 frontmatter(若有);body 保留原 SKILL.md 第一个文件。
func RenderSkillMD(c Canonical) string {
	if len(c.Files) == 0 {
		return ""
	}
	body := c.Files[0].Content
	_, existing, err := splitFrontmatter(body)
	if err != nil || existing == "" {
		existing = body
	}
	yml, err := yaml.Marshal(c.Manifest)
	if err != nil {
		yml = []byte("# manifest marshal error\n")
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(yml)
	buf.WriteString("---\n")
	buf.WriteString(existing)
	return buf.String()
}
