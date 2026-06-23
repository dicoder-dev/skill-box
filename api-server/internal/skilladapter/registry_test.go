package skilladapter

import "testing"

// stubAdapter 测试用最小实现。
type stubAdapter struct {
	id string
}

func (s *stubAdapter) ToolID() string             { return s.id }
func (s *stubAdapter) DisplayName() string        { return "Stub " + s.id }
func (s *stubAdapter) Icon() string               { return ":ghost:" }
func (s *stubAdapter) DiscoverPaths(string) ([]string, error) { return nil, nil }
func (s *stubAdapter) Scan(string) ([]Canonical, error)        { return nil, nil }
func (s *stubAdapter) Apply(Canonical, string) error           { return nil }
func (s *stubAdapter) LocalName(c Canonical) string            { return c.Manifest.Name }
func (s *stubAdapter) Validate(Canonical) error                { return nil }
func (s *stubAdapter) IsSystemPath(string) bool               { return false }

func TestRegistry_RegisterGetAll(t *testing.T) {
	// 用独立 registry 避免污染 package-level 默认表。
	r := &Registry{m: make(map[string]Adapter)}
	a := &stubAdapter{id: "stub-a"}
	r.mu.Lock()
	r.m[a.ToolID()] = a
	r.mu.Unlock()

	got, ok := r.Get("stub-a")
	if !ok || got != a {
		t.Fatalf("Get(stub-a) = (%v, %v); want (%v, true)", got, ok, a)
	}

	if _, ok := r.Get("missing"); ok {
		t.Fatal("Get(missing) returned ok=true")
	}

	all := r.All()
	if len(all) != 1 || all[0].ToolID() != "stub-a" {
		t.Fatalf("All() = %v; want 1 entry stub-a", all)
	}
}

func TestAllTools_Stable(t *testing.T) {
	if len(AllTools) != 5 {
		t.Fatalf("AllTools has %d entries; want 5", len(AllTools))
	}
	seen := make(map[string]bool, 5)
	for _, id := range AllTools {
		if seen[id] {
			t.Fatalf("AllTools contains duplicate %q", id)
		}
		seen[id] = true
	}
}

func TestScopeConstants(t *testing.T) {
	if ScopeGlobal != "global" || ScopeProject != "project" {
		t.Fatalf("scope constants drifted: %q %q", ScopeGlobal, ScopeProject)
	}
}
