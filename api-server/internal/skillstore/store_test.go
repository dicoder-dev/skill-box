package skillstore

import (
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

func TestSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load("code-review")
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

func TestLoad_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Load("missing")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	_ = s.Save(c)
	if err := s.Delete("code-review"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// 第二次 delete 应当幂等成功(不返回错误)
	if err := s.Delete("code-review"); err != nil {
		t.Fatalf("second Delete: %v", err)
	}
}

func TestList_FiltersKeyword(t *testing.T) {
	s := newTestStore(t)
	// 2026-06-24:无 version 层,Save 同名会覆盖;用 3 个不同 name 验证 list / keyword
	names := []string{"code-review", "code-format", "debug"}
	for _, n := range names {
		m := validManifest()
		m.Name = n
		if err := s.Save(skilladapter.Canonical{Manifest: m, Files: validCanonical().Files}); err != nil {
			t.Fatalf("Save %s: %v", n, err)
		}
	}
	all, err := s.List("")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("names: got %d want 3 (%v)", len(all), all)
	}
	filtered, err := s.List("code")
	if err != nil {
		t.Fatalf("List code: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("keyword filter: got %d want 2 (%v)", len(filtered), filtered)
	}
}

func TestSave_Overwrite(t *testing.T) {
	s := newTestStore(t)
	c := validCanonical()
	if err := s.Save(c); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// 第二次保存,内容略有不同
	c.Files[0].Content = "# Code Review v2\n"
	if err := s.Save(c); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, err := s.Load("code-review")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Save 会用 RenderSkillMD 重新拼 frontmatter + body;body 部分应是 v2
	if !strings.Contains(got.Files[0].Content, "# Code Review v2") {
		t.Errorf("overwrite not applied: %q", got.Files[0].Content)
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
			if err := s.Save(cc); err != nil {
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

// 避免 unused import 警告
var _ = filepath.Join
