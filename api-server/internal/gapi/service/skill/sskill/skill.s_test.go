package sskill_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
)

func newTestService(t *testing.T) (*sskill.Service, string) {
	t.Helper()
	store, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	return sskill.New(store), store.Root()
}

func sampleCanonical(name string) skilladapter.Canonical {
	return skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name:        name,
			Version:     "0.1.0",
			Description: "this is a test skill for " + name,
			Triggers:    []string{"test", name},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: " + name + "\n---\nbody"}},
	}
}

func TestCreate_Global_Ok(t *testing.T) {
	svc, _ := newTestService(t)
	row, err := svc.Create(&sskill.WriteInput{
		Scope: "global",
		Files: sampleCanonical("alpha").Files,
		Manifest: skilladapter.Manifest{
			Name:        "alpha",
			Version:     "0.1.0",
			Description: "this is a test skill for alpha",
			Triggers:    []string{"test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if row == nil || row.Manifest.Name != "alpha" {
		t.Errorf("row: %+v", row)
	}
}

func TestCreate_InvalidScope(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Create(&sskill.WriteInput{Scope: "weird"})
	if !errors.Is(err, sskill.ErrInvalidScope) {
		t.Errorf("got %v, want ErrInvalidScope", err)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Create(&sskill.WriteInput{Scope: "global"})
	if !errors.Is(err, sskill.ErrEmptyName) {
		t.Errorf("got %v, want ErrEmptyName", err)
	}
}

func TestGet_Found(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.Create(&sskill.WriteInput{
		Scope: "global",
		Manifest: skilladapter.Manifest{
			Name: "g1", Version: "0.1.0",
			Description: "this is a test skill for g1", Triggers: []string{"t"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get("g1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.Name != "g1" {
		t.Errorf("name: %q", got.Manifest.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Get("ghost")
	if !errors.Is(err, sskill.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetFull_LoadsCanonical(t *testing.T) {
	svc, _ := newTestService(t)
	can := sampleCanonical("full")
	if _, err := svc.Create(&sskill.WriteInput{
		Scope:    "global",
		Manifest: can.Manifest,
		Files:    can.Files,
	}); err != nil {
		t.Fatal(err)
	}
	full, err := svc.GetFull("full")
	if err != nil {
		t.Fatal(err)
	}
	if full.Manifest.Description == "" {
		t.Error("empty manifest in canonical")
	}
	if len(full.Files) == 0 {
		t.Error("no files in canonical")
	}
}

func TestUpdate_OverwritesStore(t *testing.T) {
	svc, storeRoot := newTestService(t)
	if _, err := svc.Create(&sskill.WriteInput{
		Scope: "global",
		Manifest: skilladapter.Manifest{
			Name: "u1", Version: "0.1.0",
			Description: "this is a test skill for u1", Triggers: []string{"t"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Update("u1", &sskill.WriteInput{
		Scope: "global",
		Manifest: skilladapter.Manifest{
			Name: "u1", Version: "0.1.0",
			Description: "updated description content is here ok", Triggers: []string{"t2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	full, _ := svc.GetFull("u1")
	if full.Manifest.Description != "updated description content is here ok" {
		t.Errorf("desc not updated: %q", full.Manifest.Description)
	}
	// 物理文件也在
	if _, err := os.Stat(filepath.Join(storeRoot, "u1", "SKILL.md")); err != nil {
		t.Errorf("manifest file missing: %v", err)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	svc, _ := newTestService(t)
	// 删不存在的也不报错
	if err := svc.Delete("ghost"); err != nil {
		t.Errorf("delete missing should be nil, got %v", err)
	}
}

func TestList_FilterByName(t *testing.T) {
	svc, _ := newTestService(t)
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if _, err := svc.Create(&sskill.WriteInput{
			Scope: "global",
			Manifest: skilladapter.Manifest{
				Name: n, Version: "0.1.0",
				Description: "this is a test skill for " + n, Triggers: []string{"t"},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.List("al")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Errorf("keyword 'al' = %+v", got)
	}
}

// TestUpdate_PartialFilesDropsOthers 2026-07-08 增:Save 接口是"原子全量覆盖"语义,
// 必须 caller 拼齐完整的 files 数组才能不丢文件;否则只 send 当前 dirty 文件,其他
// 文件会因 tmp 目录没被写而消失。本测试复现根因,作为契约固定下来:
// 任何调用方如果遇到 "传 files 但丢文件",责任在 caller,而非 store。
// 详见 SkillFileInlinePanel.saveCurrent (frontend) 为修复点的对应改动。
func TestUpdate_PartialFilesDropsOthers(t *testing.T) {
	svc, storeRoot := newTestService(t)
	mk := skilladapter.Manifest{
		Name: "partial", Version: "0.1.0",
		Description: "this is a partial update repro for partial save", Triggers: []string{"t"},
	}
	if _, err := svc.Create(&sskill.WriteInput{
		Scope:    "global",
		Manifest: mk,
		Files: []skilladapter.File{
			{Path: "a.md", Content: "AAA"},
			{Path: "b.md", Content: "BBB"},
			{Path: "c.md", Content: "CCC"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	full, _ := svc.GetFull("partial")
	if len(full.Files) < 4 { // SKILL.md + a + b + c
		t.Fatalf("precondition: expected 4 files, got %d", len(full.Files))
	}
	// 模拟前端只 send 当前 dirty 文件(契约未对齐)
	if _, err := svc.Update("partial", &sskill.WriteInput{
		Scope:    "global",
		Manifest: mk,
		Files: []skilladapter.File{
			{Path: "a.md", Content: "AAA-modified"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	full, _ = svc.GetFull("partial")
	// SKILL.md + a.md(b/c 应该丢失) — 确认根因可复现
	var gotPaths []string
	for _, f := range full.Files {
		gotPaths = append(gotPaths, f.Path)
	}
	t.Logf("after partial update: files = %v", gotPaths)
	for _, lost := range []string{"b.md", "c.md"} {
		// 磁盘侧再二次校验
		if _, err := os.Stat(filepath.Join(storeRoot, "partial", lost)); err == nil {
			t.Errorf("BUG NOT REPRODUCED: %s unexpectedly still exists on disk", lost)
		}
	}
	// 兜底断言:确认这两个文件已不在 full.Files 里(契约硬性)
	for _, f := range full.Files {
		if f.Path == "b.md" || f.Path == "c.md" {
			t.Errorf("partial update MUST drop %s per current Save contract; got %+v", f.Path, gotPaths)
		}
	}
}
