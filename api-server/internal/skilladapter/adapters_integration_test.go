package skilladapter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
	_ "ginp-api/internal/skilladapter/claude"
	_ "ginp-api/internal/skilladapter/codex"
	_ "ginp-api/internal/skilladapter/cursor"
	_ "ginp-api/internal/skilladapter/opencode"
	_ "ginp-api/internal/skilladapter/trae"
)

func TestAllAdaptersRegistered(t *testing.T) {
	all := skilladapter.All()
	ids := make(map[string]bool, len(all))
	for _, a := range all {
		if a.ToolID() == "" {
			t.Errorf("adapter with empty ToolID: %+v", a)
		}
		ids[a.ToolID()] = true
		if a.DisplayName() == "" {
			t.Errorf("adapter %s has empty DisplayName", a.ToolID())
		}
	}
	for _, want := range skilladapter.AllTools {
		if !ids[want] {
			t.Errorf("missing adapter for %q", want)
		}
	}
}

func TestParseSkillMD_RealTraeFile(t *testing.T) {
	// 用 trae 的真实文件做集成测试(若本机 trae 不存在则 skip)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(home, ".trae", "skills", "find-skills", "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("trae find-skills not present: %v", err)
	}
	c, err := skilladapter.ParseSkillMD(string(content))
	if err != nil {
		t.Fatalf("ParseSkillMD: %v", err)
	}
	if c.Manifest.Name != "find-skills" {
		t.Errorf("name: got %q want find-skills", c.Manifest.Name)
	}
	if c.Manifest.Description == "" {
		t.Error("empty description")
	}
}

func TestParseSkillMD_NoFrontmatter(t *testing.T) {
	_, err := skilladapter.ParseSkillMD("# Just a body\nNo frontmatter here.\n")
	if err == nil {
		t.Error("expected error for content without frontmatter")
	}
}

func TestParseSkillMD_BadYAML(t *testing.T) {
	_, err := skilladapter.ParseSkillMD("---\nname: [bad\n---\n# body\n")
	if err == nil {
		t.Error("expected error for bad yaml")
	}
}

func TestParseSkillMD_FallbackNameFromH1(t *testing.T) {
	content := `---
description: some skill
---
# My Skill
body
`
	c, err := skilladapter.ParseSkillMD(content)
	if err != nil {
		t.Fatalf("ParseSkillMD: %v", err)
	}
	if c.Manifest.Name != "my-skill" {
		t.Errorf("fallback name: got %q want my-skill", c.Manifest.Name)
	}
}

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Code Review":       "code-review",
		"foo_bar":           "foo-bar",
		"Already-Snake":     "already-snake",
		"中文名 abc":         "abc",
		"1abc":              "s-1abc",
		"   trim me  ":      "trim-me",
		"a":                 "a",
		"":                  "",
	}
	for in, want := range cases {
		if got := skilladapter.NormalizeName(in); got != want {
			t.Errorf("NormalizeName(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestRenderSkillMD_RoundTrip(t *testing.T) {
	c, err := skilladapter.ParseSkillMD("---\nname: foo\ndescription: bar\n---\n# foo\nbody\n")
	if err != nil {
		t.Fatalf("ParseSkillMD: %v", err)
	}
	rendered := skilladapter.RenderSkillMD(*c)
	// 重新解析
	c2, err := skilladapter.ParseSkillMD(rendered)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if c2.Manifest.Name != "foo" || c2.Manifest.Description != "bar" {
		t.Errorf("round trip drift: %+v", c2.Manifest)
	}
}

func TestAdapterApply_PopulatesSkillDir(t *testing.T) {
	a, ok := skilladapter.Get(skilladapter.ToolTrae)
	if !ok {
		t.Skip("trae adapter not registered")
	}
	tmp := t.TempDir()
	canonical := skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name:        "demo",
			Version:     "0.1.0",
			Description: "demo skill for adapter apply test, satisfies length",
			Triggers:    []string{"demo"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: demo\n---\n# demo\n"}},
	}
	target := filepath.Join(tmp, a.LocalName(canonical))
	if err := a.Apply(canonical, target); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(content), "name: demo") {
		t.Errorf("applied content lost frontmatter: %q", content)
	}
}
