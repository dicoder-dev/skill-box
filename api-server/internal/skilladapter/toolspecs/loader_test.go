package toolspecs

import (
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
)

// TestLoadAll 全部内嵌 spec 都能加载 + Validate 通过。
// 启动期硬约束,任何不合法的 spec 都让服务起不来。
func TestLoadAll(t *testing.T) {
	specs, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(specs) < 9 {
		t.Errorf("expected at least 9 specs, got %d", len(specs))
	}
	for _, s := range specs {
		t.Run(s.ToolID, func(t *testing.T) {
			if s.ToolID == "" {
				t.Error("empty tool_id")
			}
			if s.DisplayName == "" {
				t.Error("empty display_name")
			}
			if !strings.HasPrefix(s.MdiIcon, "mdi:") {
				t.Errorf("mdi_icon %q must start with mdi:", s.MdiIcon)
			}
			if s.Maturity != "" {
				switch s.Maturity {
				case "stable", "experimental", "deprecated":
				default:
					t.Errorf("invalid maturity %q", s.Maturity)
				}
			}
		})
	}
}

// TestSpecAdapter_PathExpansion 验证 ~/ 展开 + project 路径原样透传。
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

// TestSpecificSpecs 抽 1-2 个工具的关键断言,避免 YAML 改错时漏检。
func TestSpecificSpecs(t *testing.T) {
	specs, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	byID := make(map[string]*ToolSpec, len(specs))
	for _, s := range specs {
		byID[s.ToolID] = s
	}

	// claude:global.user 必须含 .agents/skills(Agent Skills 标准)
	if c, ok := byID["claude"]; !ok {
		t.Error("missing claude spec")
	} else {
		found := false
		for _, p := range c.Paths.Global.User {
			if filepath.Base(p) == "skills" && strings.Contains(p, ".agents") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("claude.global.user should contain ~/.agents/skills, got %v", c.Paths.Global.User)
		}
	}

	// codex:global.system 必须含 .system
	if c, ok := byID["codex"]; !ok {
		t.Error("missing codex spec")
	} else {
		found := false
		for _, p := range c.Paths.Global.System {
			if strings.Contains(p, ".system") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("codex.global.system should contain .system path, got %v", c.Paths.Global.System)
		}
	}

	// 新增 4 个工具必须都是 stable 或 experimental
	for _, id := range []string{"antigravity", "cline", "codebuddy", "jetbrains"} {
		s, ok := byID[id]
		if !ok {
			t.Errorf("missing new spec: %s", id)
			continue
		}
		if s.Maturity != "stable" && s.Maturity != "experimental" {
			t.Errorf("%s: maturity should be stable or experimental, got %q", id, s.Maturity)
		}
	}
}