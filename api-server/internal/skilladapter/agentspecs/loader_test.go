package agentspecs

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
	spec := &AgentSpec{
		AgentID:     "test",
		DisplayName: "Test",
		MdiIcon:     "mdi:test",
		Paths: AgentPaths{
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
	agents := a.Agents
	if got := agents[skilladapter.ScopeGlobal][0]; !strings.HasSuffix(got, "/test-global") {
		t.Errorf("expected global path to end with /test-global, got %q", got)
	}
	if got := agents[skilladapter.ScopeGlobal][0]; strings.HasPrefix(got, "~") {
		t.Errorf("~/ not expanded: %q", got)
	}
	if got := agents[skilladapter.ScopeProject][0]; got != ".test-project" {
		t.Errorf("project path should pass through unchanged, got %q", got)
	}
	if a.Icon() != "mdi:test" {
		t.Errorf("Icon: got %q want mdi:test", a.Icon())
	}
}

// TestSpecAdapter_IconPassThrough 验证 icon 字段透传,不再走 emoji。
func TestSpecAdapter_IconPassThrough(t *testing.T) {
	spec := &AgentSpec{
		AgentID: "x", DisplayName: "X", MdiIcon: "mdi:custom-icon",
		Paths: AgentPaths{
			Global:  CategoryPaths{User: []string{"~/x"}},
			Project: CategoryPaths{User: []string{".x"}},
		},
	}
	a := NewSpecAdapter(spec)
	if got := a.Icon(); got != "mdi:custom-icon" {
		t.Errorf("Icon: got %q want mdi:custom-icon", got)
	}
}

// TestAgentSpec_Validate 全面的 schema 校验。
func TestAgentSpec_Validate(t *testing.T) {
	good := &AgentSpec{AgentID: "ok", DisplayName: "OK", MdiIcon: "mdi:ok", Paths: AgentPaths{
		Global:  CategoryPaths{User: []string{"~/x"}},
		Project: CategoryPaths{User: []string{".x"}},
	}}
	if err := good.Validate(); err != nil {
		t.Errorf("good spec should pass, got %v", err)
	}

	bad := []struct {
		name string
		spec *AgentSpec
	}{
		{"empty tool_id", &AgentSpec{DisplayName: "X", MdiIcon: "mdi:x", Paths: good.Paths}},
		{"empty display", &AgentSpec{AgentID: "x", MdiIcon: "mdi:x", Paths: good.Paths}},
		{"empty mdi", &AgentSpec{AgentID: "x", DisplayName: "X", Paths: good.Paths}},
		{"bad maturity", &AgentSpec{AgentID: "x", DisplayName: "X", MdiIcon: "mdi:x", Maturity: "weird", Paths: good.Paths}},
		{"empty global", &AgentSpec{AgentID: "x", DisplayName: "X", MdiIcon: "mdi:x", Paths: AgentPaths{Project: good.Paths.Project}}},
		{"empty project", &AgentSpec{AgentID: "x", DisplayName: "X", MdiIcon: "mdi:x", Paths: AgentPaths{Global: good.Paths.Global}}},
	}
	for _, b := range bad {
		t.Run(b.name, func(t *testing.T) {
			if err := b.spec.Validate(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
