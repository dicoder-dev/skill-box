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

// 2026-07-03 增:多级分组(2026-06-29)后,旧 API(Load/Exists/Delete)只查根下
// 直接子目录,漏掉 aa/debug-helper 这种嵌套 skill → apply 链路返 404。
// 新增 LoadByName / ExistsByName / DeleteByName 全树按 name 找,这里回归验证。
func TestLoadByName_FindsNestedSkill(t *testing.T) {
	s := newTestStore(t)

	// 1) 先建一个"根下" skill(对照组)
	rootC := validCanonical()
	rootC.Manifest.Name = "root-skill"
	if err := s.Save(rootC); err != nil {
		t.Fatalf("Save root: %v", err)
	}

	// 2) 建一个"嵌套在分组 aa 下"的 skill(回归目标)
	// 注:Save 自身不自动建分组父目录(需要先 CreateGroupDir),这里走公共
	// API 建好父目录,顺带验证嵌套 save 路径可用。
	if err := s.CreateGroupDir("aa"); err != nil {
		t.Fatalf("CreateGroupDir: %v", err)
	}
	nestedC := validCanonical()
	nestedC.Manifest.Name = "debug-helper"
	nestedC.Manifest.GroupPath = "aa"
	if err := s.Save(nestedC); err != nil {
		t.Fatalf("Save nested: %v", err)
	}

	// 3) 老 Load(name) 只能查根下 — 验证它确实漏掉嵌套(这就是 bug 现场)
	if _, err := s.Load("debug-helper"); err == nil {
		t.Errorf("old Load unexpectedly succeeded for nested skill (bug should reproduce here)")
	}

	// 4) 新 LoadByName 必须能找到嵌套 skill
	got, err := s.LoadByName("debug-helper")
	if err != nil {
		t.Fatalf("LoadByName: %v", err)
	}
	if got.Manifest.Name != "debug-helper" {
		t.Errorf("name = %q want debug-helper", got.Manifest.Name)
	}
	if got.Manifest.GroupPath != "aa" {
		t.Errorf("groupPath = %q want aa", got.Manifest.GroupPath)
	}

	// 5) ExistsByName / DeleteByName 同样工作
	if !s.ExistsByName("debug-helper") {
		t.Errorf("ExistsByName should find nested skill")
	}
	if err := s.DeleteByName("debug-helper"); err != nil {
		t.Fatalf("DeleteByName: %v", err)
	}
	if s.ExistsByName("debug-helper") {
		t.Errorf("DeleteByName did not remove the skill")
	}
	// 幂等:再删一次返 nil
	if err := s.DeleteByName("debug-helper"); err != nil {
		t.Errorf("DeleteByName should be idempotent, got %v", err)
	}

	// 6) 根下的 skill 仍然能被 LoadByName 找到(浅层优先)
	if _, err := s.LoadByName("root-skill"); err != nil {
		t.Errorf("LoadByName should find root skill: %v", err)
	}
}

// 浅层优先:同名 skill 同时存在根下和分组里时,LoadByName 返根下的那条。
func TestLoadByName_ShallowFirst(t *testing.T) {
	s := newTestStore(t)

	rootC := validCanonical()
	rootC.Manifest.Name = "dup-skill"
	rootC.Manifest.Description = "root one"
	if err := s.Save(rootC); err != nil {
		t.Fatalf("Save root: %v", err)
	}

	if err := s.CreateGroupDir("aa"); err != nil {
		t.Fatalf("CreateGroupDir: %v", err)
	}
	nestedC := validCanonical()
	nestedC.Manifest.Name = "dup-skill"
	nestedC.Manifest.Description = "nested one"
	nestedC.Manifest.GroupPath = "aa"
	if err := s.Save(nestedC); err != nil {
		t.Fatalf("Save nested: %v", err)
	}

	got, err := s.LoadByName("dup-skill")
	if err != nil {
		t.Fatalf("LoadByName: %v", err)
	}
	if got.Manifest.Description != "root one" {
		t.Errorf("shallow-first failed: got description %q, want %q", got.Manifest.Description, "root one")
	}
	if got.Manifest.GroupPath != "" {
		t.Errorf("shallow-first: expected root groupPath, got %q", got.Manifest.GroupPath)
	}
}

// 找不到时返 ErrNotFound(与 Load 行为一致,便于上层 service.Is 判断)。
func TestLoadByName_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.LoadByName("nope")
	if err != ErrNotFound {
		t.Errorf("err = %v want ErrNotFound", err)
	}
}
