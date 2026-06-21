package sskill_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) (*sskill.Service, string) {
	t.Helper()
	store, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&entity.Skill{}, &entity.SkillFile{}); err != nil {
		t.Fatal(err)
	}
	return sskill.New(db, db, store), store.Root()
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
	if row.ID == 0 || row.Name != "alpha" {
		t.Errorf("row: %+v", row)
	}
}

func TestCreate_Project_Ok(t *testing.T) {
	svc, _ := newTestService(t)
	row, err := svc.Create(&sskill.WriteInput{
		Scope:     "project",
		ProjectID: 7,
		Manifest: skilladapter.Manifest{
			Name:        "beta",
			Version:     "0.1.0",
			Description: "this is a test skill for beta",
			Triggers:    []string{"test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.ProjectID != 7 {
		t.Errorf("project_id: %d", row.ProjectID)
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
	got, err := svc.Get("global", "g1", "0.1.0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "g1" {
		t.Errorf("name: %q", got.Name)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Get("global", "ghost", "0.1.0", 0)
	if !errors.Is(err, sskill.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetFull_LoadsCanonical(t *testing.T) {
	svc, _ := newTestService(t)
	can := sampleCanonical("full")
	if _, err := svc.Create(&sskill.WriteInput{
		Scope: "global",
		Manifest: can.Manifest,
		Files: can.Files,
	}); err != nil {
		t.Fatal(err)
	}
	full, err := svc.GetFull("global", "full", "0.1.0", 0)
	if err != nil {
		t.Fatal(err)
	}
	if full.Canonical.Manifest.Description == "" {
		t.Error("empty manifest in canonical")
	}
	if len(full.Canonical.Files) == 0 {
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
	_, err := svc.Update("global", "u1", "0.1.0", 0, &sskill.WriteInput{
		Scope: "global",
		Manifest: skilladapter.Manifest{
			Name: "u1", Version: "0.1.0",
			Description: "updated description content is here ok", Triggers: []string{"t2"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	full, _ := svc.GetFull("global", "u1", "0.1.0", 0)
	if full.Canonical.Manifest.Description != "updated description content is here ok" {
		t.Errorf("desc not updated: %q", full.Canonical.Manifest.Description)
	}
	// 物理文件也在
	if _, err := os.Stat(filepath.Join(storeRoot, "global", "u1", "0.1.0", "skill.yaml")); err != nil {
		t.Errorf("manifest file missing: %v", err)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	svc, _ := newTestService(t)
	// 删不存在的也不报错
	if err := svc.Delete("global", "ghost", "0.1.0", 0); err != nil {
		t.Errorf("delete missing should be nil, got %v", err)
	}
}

func TestList_FilterByScope(t *testing.T) {
	svc, _ := newTestService(t)
	for i := 0; i < 3; i++ {
		n := string(rune('a' + i))
		if _, err := svc.Create(&sskill.WriteInput{
			Scope: "global",
			Manifest: skilladapter.Manifest{
				Name: "g-" + n, Version: "0.1.0",
				Description: "this is a test skill for g-" + n, Triggers: []string{"t"},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Create(&sskill.WriteInput{
		Scope:     "project",
		ProjectID: 99,
		Manifest: skilladapter.Manifest{
			Name: "p-a", Version: "0.1.0",
			Description: "this is a test skill for p-a", Triggers: []string{"t"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.List(sskill.ListQuery{Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 {
		t.Errorf("global total=%d", got.Total)
	}
	got, _ = svc.List(sskill.ListQuery{Scope: "project", ProjectID: 99})
	if got.Total != 1 {
		t.Errorf("project total=%d", got.Total)
	}
}
