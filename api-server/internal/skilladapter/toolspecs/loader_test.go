package toolspecs

import (
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
)

// TestSpecAdapter_PathExpansion 验证 ~/ 展开 + project 路径原样透传。
//
// 2026-06-30 二改:LoadAll 不再是"读 yaml",改为"读 DB";此处只测纯逻辑
// 转换 + ~/ 展开,与 DB 无关(不需要 sqlite 实例)。
func TestSpecAdapter_PathExpansion(t *testing.T) {
	spec := &ToolSpec{
		ToolID:      "test",
		DisplayName: "Test",
		MdiIcon:     "mdi:test",
		Paths: ToolPaths{
			Global: CategoryPaths{
				User:   []string{"~/test-global"},
				System: []string{"~/test-system"},
			},
			Project: CategoryPaths{
				User: []string{".test-project"},
			},
		},
	}
	a := NewSpecAdapter(spec)
	tools := a.Tools
	if got := tools[skilladapter.ScopeGlobal][0]; !strings.HasSuffix(got, "/test-global") {
		t.Errorf("expected global path to end with /test-global, got %q", got)
	}
	if got := tools[skilladapter.ScopeGlobal][0]; strings.HasPrefix(got, "~") {
		t.Errorf("~/ not expanded: %q", got)
	}
	if got := tools[skilladapter.ScopeProject][0]; got != ".test-project" {
		t.Errorf("project path should pass through unchanged, got %q", got)
	}
	if a.Icon() != "mdi:test" {
		t.Errorf("Icon: got %q want mdi:test", a.Icon())
	}
}

// TestSpecAdapter_IconPassThrough 验证 icon 字段透传,不再走 emoji。
func TestSpecAdapter_IconPassThrough(t *testing.T) {
	spec := &ToolSpec{
		ToolID: "x", DisplayName: "X", MdiIcon: "mdi:custom-icon",
		Paths: ToolPaths{
			Global:  CategoryPaths{User: []string{"~/x"}},
			Project: CategoryPaths{User: []string{".x"}},
		},
	}
	a := NewSpecAdapter(spec)
	if got := a.Icon(); got != "mdi:custom-icon" {
		t.Errorf("Icon: got %q want mdi:custom-icon", got)
	}
}

// TestToolSpec_Validate 全面的 schema 校验。
func TestToolSpec_Validate(t *testing.T) {
	good := &ToolSpec{ToolID: "ok", DisplayName: "OK", MdiIcon: "mdi:ok", Paths: ToolPaths{
		Global:  CategoryPaths{User: []string{"~/x"}},
		Project: CategoryPaths{User: []string{".x"}},
	}}
	if err := good.Validate(); err != nil {
		t.Errorf("good spec should pass, got %v", err)
	}

	bad := []struct {
		name string
		spec *ToolSpec
	}{
		{"empty tool_id", &ToolSpec{DisplayName: "X", MdiIcon: "mdi:x", Paths: good.Paths}},
		{"empty display", &ToolSpec{ToolID: "x", MdiIcon: "mdi:x", Paths: good.Paths}},
		{"empty mdi", &ToolSpec{ToolID: "x", DisplayName: "X", Paths: good.Paths}},
		{"bad maturity", &ToolSpec{ToolID: "x", DisplayName: "X", MdiIcon: "mdi:x", Maturity: "weird", Paths: good.Paths}},
		{"empty global", &ToolSpec{ToolID: "x", DisplayName: "X", MdiIcon: "mdi:x", Paths: ToolPaths{Project: good.Paths.Project}}},
		{"empty project", &ToolSpec{ToolID: "x", DisplayName: "X", MdiIcon: "mdi:x", Paths: ToolPaths{Global: good.Paths.Global}}},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			if err := b.spec.Validate(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}