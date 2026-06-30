package skilladapter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skilladapter/toolspecs"
	"ginp-api/internal/toolseed"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupRegistryFromSQLite 用 sqlite 内存 DB 模拟"启动期 seed + reload"全流程。
//
// 2026-06-30 二改:adapter 不再 init() 自动注册;改成在测试里手起 sqlite
// 内存 DB,跑 EnsureSeeded + ReloadAllFromDB,验证 Registry 内容。完整还原
// 启动期链路,避免 dbs 包全局状态依赖。
func setupRegistryFromSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	// in-memory sqlite + 多连接共享 cache
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=encoding=UTF-8"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// AutoMigrate e_tool + e_tool_path(测试用最小集)
	if err := db.AutoMigrate(&entity.Tool{}, &entity.ToolPath{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := toolseed.EnsureSeeded(db, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := toolspecs.ReloadAllFromDB(db); err != nil {
		t.Fatalf("reload: %v", err)
	}
	return db
}

// TestAllAdaptersRegistered 验证默认 Registry 在 DB 加载后包含全部默认工具。
//
// 2026-06-30 二改:adapter 不再 init() 自动注册,而是依赖启动期
// toolspecs.ReloadAllFromDB(db);本测试用 sqlite 内存 DB 模拟"seed + reload"
// 全流程,验证 9 个默认工具 + Icon() 字段透传。
func TestAllAdaptersRegistered(t *testing.T) {
	_ = setupRegistryFromSQLite(t)

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
		icon := a.Icon()
		if icon == "" {
			t.Errorf("adapter %s has empty Icon (mdi_icon in spec missing?)", a.ToolID())
		}
		if !strings.HasPrefix(icon, "mdi:") {
			t.Errorf("adapter %s icon %q must start with mdi:", a.ToolID(), icon)
		}
	}
	// 5 个老工具必须存在(向后兼容);新工具(antigravity/cline/codebuddy/jetbrains)
	// 不在此断言,数量由 seed 9 个默认决定。
	for _, want := range skilladapter.AllTools {
		if !ids[want] {
			t.Errorf("missing legacy adapter for %q", want)
		}
	}
	// 至少得有 9 个(5 老 + 4 新),避免 seed 漏工具
	if got := len(all); got < 9 {
		t.Errorf("expected at least 9 adapters (5 legacy + 4 new), got %d", got)
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
