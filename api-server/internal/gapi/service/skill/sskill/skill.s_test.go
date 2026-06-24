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
