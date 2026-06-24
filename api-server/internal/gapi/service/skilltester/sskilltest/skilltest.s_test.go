package sskilltest_test

import (
	"errors"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skilltester/sskilltest"
	"ginp-api/internal/settings"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestSvc(t *testing.T) (*sskilltest.Service, *skillstore.Store) {
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
	if err := db.AutoMigrate(
		&entity.SkillTestRun{},
		&entity.SkillTestResult{},
		&entity.AIProvider{},
		&entity.Setting{},
	); err != nil {
		t.Fatal(err)
	}
	st := settings.New(db, db)
	mgr := sskilltest.NewManagerForTester(st)
	return sskilltest.New(db, db, store, st, mgr), store
}


func TestRun_EmptyKey(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.Run(&sskilltest.RunRequest{Scope: "global", Name: ""})
	if !errors.Is(err, sskilltest.ErrEmptyKey) {
		t.Errorf("got %v, want ErrEmptyKey", err)
	}
}

func TestRun_BadScope(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.Run(&sskilltest.RunRequest{Scope: "weird", Name: "x"})
	if !errors.Is(err, sskilltest.ErrEmptyKey) {
		t.Errorf("got %v, want ErrEmptyKey", err)
	}
}

func TestRun_SkillNotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.Run(&sskilltest.RunRequest{Scope: "global", Name: "ghost", Version: "0.1.0"})
	if !errors.Is(err, sskilltest.ErrStoreLoad) {
		t.Errorf("got %v, want ErrStoreLoad", err)
	}
}


func TestList_Empty(t *testing.T) {
	svc, _ := newTestSvc(t)
	res, err := svc.List(&sskilltest.ListRequest{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("expected empty, got %+v", res)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.Get(99999)
	if !errors.Is(err, sskilltest.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}
