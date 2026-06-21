package skillstore

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"ginp-api/internal/skilladapter"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewAt(dir)
	if err != nil {
		t.Fatalf("NewAt: %v", err)
	}
	return s
}

func validManifest() skilladapter.Manifest {
	return skilladapter.Manifest{
		Name:        "code-review",
		Version:     "1.2.0",
		Description: "review code, 10-500 chars description requirement satisfied here",
		Triggers:    []string{"review", "code review"},
		Author:      "tester",
		License:     "MIT",
	}
}

func validCanonical() skilladapter.Canonical {
	return skilladapter.Canonical{
		Manifest: validManifest(),
		Files: []skilladapter.File{
			{Path: "SKILL.md", Content: "# Code Review\n"},
			{Path: "examples/review.sh", Content: "#!/usr/bin/env bash\necho review\n"},
		},
	}
}

func TestSaveAndLoad_Global(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	if err := s.Save(c, skilladapter.ScopeGlobal, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load(skilladapter.ScopeGlobal, "code-review", "1.2.0", 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Manifest.Name != c.Manifest.Name {
		t.Errorf("name drift: got %q want %q", got.Manifest.Name, c.Manifest.Name)
	}
	if len(got.Files) != 2 {
		t.Errorf("files: got %d want 2", len(got.Files))
	}
}

func TestSave_ProjectScope(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	if err := s.Save(c, skilladapter.ScopeProject, 42); err != nil {
		t.Fatalf("Save: %v", err)
	}
	dir := s.skillDir(skilladapter.ScopeProject, "code-review", "1.2.0", 42)
	if _, err := os.Stat(filepath.Join(dir, manifestFileName)); err != nil {
		t.Errorf("project manifest not at expected path: %v", err)
	}
}

func TestLoad_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Load(skilladapter.ScopeGlobal, "missing", "0.0.1", 0)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	_ = s.Save(c, skilladapter.ScopeGlobal, 0)

	if err := s.Delete(skilladapter.ScopeGlobal, "code-review", "1.2.0", 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// 第二次 delete 应当幂等成功(不返回错误)
	if err := s.Delete(skilladapter.ScopeGlobal, "code-review", "1.2.0", 0); err != nil {
		t.Fatalf("second Delete: %v", err)
	}
}

func TestListVersionsAndNames(t *testing.T) {
	s := newTestStore(t)
	m := validManifest()
	for _, v := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		m.Version = v
		if err := s.Save(skilladapter.Canonical{Manifest: m, Files: validCanonical().Files}, skilladapter.ScopeGlobal, 0); err != nil {
			t.Fatalf("Save %s: %v", v, err)
		}
	}
	// 再加一个别的 skill
	other := validManifest()
	other.Name = "debug"
	if err := s.Save(skilladapter.Canonical{Manifest: other, Files: nil}, skilladapter.ScopeGlobal, 0); err != nil {
		t.Fatalf("Save debug: %v", err)
	}

	vs, err := s.ListVersions(skilladapter.ScopeGlobal, "code-review", 0)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(vs) != 3 {
		t.Errorf("versions: got %d want 3 (%v)", len(vs), vs)
	}
	ns, err := s.ListNames(skilladapter.ScopeGlobal, 0)
	if err != nil {
		t.Fatalf("ListNames: %v", err)
	}
	if len(ns) != 2 {
		t.Errorf("names: got %d want 2 (%v)", len(ns), ns)
	}
}

func TestSave_Overwrite(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	if err := s.Save(c, skilladapter.ScopeGlobal, 0); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// 第二次保存,内容略有不同
	c.Files[0].Content = "# Code Review v2\n"
	if err := s.Save(c, skilladapter.ScopeGlobal, 0); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := s.Load(skilladapter.ScopeGlobal, "code-review", "1.2.0", 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Files[0].Content != "# Code Review v2\n" {
		t.Errorf("overwrite not applied: %q", got.Files[0].Content)
	}
}

func TestValidateManifest_Rejects(t *testing.T) {
	cases := []struct {
		name string
		m    skilladapter.Manifest
		want string
	}{
		{"bad name uppercase", func() skilladapter.Manifest { m := validManifest(); m.Name = "BadName"; return m }(), "name"},
		{"bad name starts with digit", func() skilladapter.Manifest {
			m := validManifest()
			m.Name = "1foo"
			return m
		}(), "name"},
		{"bad version", func() skilladapter.Manifest {
			m := validManifest()
			m.Version = "v1"
			return m
		}(), "version"},
		{"description too short", func() skilladapter.Manifest {
			m := validManifest()
			m.Description = "short"
			return m
		}(), "description"},
		{"no triggers", func() skilladapter.Manifest {
			m := validManifest()
			m.Triggers = nil
			return m
		}(), "triggers"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateManifest(c.m)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q missing keyword %q", err, c.want)
			}
		})
	}
}

func TestSafeRelPath_Rejects(t *testing.T) {
	for _, p := range []string{"../etc/passwd", "/abs/path", "foo/../../bar", "a\x00b"} {
		if _, err := safeRelPath(p); err == nil {
			t.Errorf("safeRelPath(%q) = nil; want error", p)
		}
	}
}

func TestSave_RejectsTraversal(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	c.Files = append(c.Files, skilladapter.File{Path: "../escape.txt", Content: "x"})
	if err := s.Save(c, skilladapter.ScopeGlobal, 0); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestHashFile_Stable(t *testing.T) {
	h1 := HashFile("hello")
	h2 := HashFile("hello")
	if h1 != h2 {
		t.Errorf("hash unstable: %s vs %s", h1, h2)
	}
	if h1 == HashFile("world") {
		t.Error("hash collision for different content")
	}
}

// 并发写测试:10 个 goroutine 同时写同一个 skill,最终目录内容应当一致。
func TestSave_Concurrent(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc := c
			cc.Files = []skilladapter.File{{Path: "SKILL.md", Content: "# v" + string(rune('0'+i)) + "\n"}}
			if err := s.Save(cc, skilladapter.ScopeGlobal, 0); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent save error: %v", err)
	}
}
