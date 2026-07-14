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
	eng := sskilltest.NewEngineForTester(st)
	return sskilltest.New(db, db, store, st, eng), store
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
	_, err := svc.Run(&sskilltest.RunRequest{Scope: "global", Name: "nope-skill"})
	if !errors.Is(err, sskilltest.ErrStoreLoad) {
		t.Errorf("got %v, want ErrStoreLoad", err)
	}
}

func TestList_Default(t *testing.T) {
	svc, _ := newTestSvc(t)
	res, err := svc.List(&sskilltest.ListRequest{Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Size != 20 {
		t.Errorf("got %+v", res)
	}
}
