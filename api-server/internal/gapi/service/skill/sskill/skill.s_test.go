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

// TestUpdate_PartialFilesPreservesOthers 2026-07-08 增,2026-07-12 改语义:
//
// 旧契约(2026-07-08):Save 是"原子全量覆盖",caller 必须传齐完整 files,
// 否则只 send 当前 dirty 文件,其他文件会因 tmp 目录没被写而消失。
//
// 新契约(2026-07-12):Save 会"保留前端不知道的文件" — caller 只传部分
// files 也行,其余从原 dir 复制回 tmp(对应 store.Save 的 WalkDir 复制逻辑,
// 见 store.go:198-239)。删除场景由 DeletedPaths 字段显式表达:
// **前端必须在 files 里剔除要删的路径**(否则 tmp 重建阶段会重新写出来,
// WalkDir 跳过也没用)。本测试固定新契约。
func TestUpdate_PartialFilesPreservesOthers(t *testing.T) {
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
	// 模拟前端只 send 当前 dirty 文件
	if _, err := svc.Update("partial", &sskill.WriteInput{
		Scope:    "global",
		Manifest: mk,
		Files: []skilladapter.File{
			{Path: "a.md", Content: "AAA-modified"},
		},
		// 不传 DeletedPaths → 原 dir 里 b.md/c.md 会被复制回 tmp,保留。
	}); err != nil {
		t.Fatal(err)
	}
	full, _ = svc.GetFull("partial")
	var gotPaths []string
	for _, f := range full.Files {
		gotPaths = append(gotPaths, f.Path)
	}
	t.Logf("after partial update: files = %v", gotPaths)
	// 新契约:partial update 必须保留 b.md / c.md(从原 dir 复制回 tmp)
	for _, kept := range []string{"b.md", "c.md"} {
		if _, err := os.Stat(filepath.Join(storeRoot, "partial", kept)); err != nil {
			t.Errorf("partial update SHOULD preserve %s; got err=%v", kept, err)
		}
	}
	// 验证保留后内容是原值(没被覆盖)
	for _, f := range full.Files {
		switch f.Path {
		case "b.md":
			if f.Content != "BBB" {
				t.Errorf("partial update preserved b.md but content drifted: %q", f.Content)
			}
		case "c.md":
			if f.Content != "CCC" {
				t.Errorf("partial update preserved c.md but content drifted: %q", f.Content)
			}
		}
	}
	// 显式删 b.md 时(契约:前端 files 里剔除 b.md + DeletedPaths=["b.md"]),
	// 必须真正物理删除。
	if _, err := svc.Update("partial", &sskill.WriteInput{
		Scope: "global",
		Manifest: mk,
		Files: []skilladapter.File{
			{Path: "a.md", Content: "AAA-modified"},
			{Path: "c.md", Content: "CCC"}, // 保留 c.md,b.md 剔除
		},
		DeletedPaths: []string{"b.md"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "partial", "b.md")); !os.IsNotExist(err) {
		t.Errorf("explicit DeletedPaths should remove b.md; got err=%v", err)
	}
	// 验证 c.md 还在(没被误删)
	if _, err := os.Stat(filepath.Join(storeRoot, "partial", "c.md")); err != nil {
		t.Errorf("c.md should still exist after explicit delete of b.md; got err=%v", err)
	}
}
