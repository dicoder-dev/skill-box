package settings

import (
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dsn := filepath.ToSlash(t.TempDir()) + "/test.db"
	db, err := gorm.Open(sqlite.Open(dsn+"?_pragma=foreign_keys(0)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.Setting{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return New(db, db)
}

func TestSetGet_RoundTrip(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Set("theme", "dark"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, ok, err := svc.Get("theme")
	if err != nil || !ok || v != "dark" {
		t.Fatalf("Get: ok=%v v=%q err=%v", ok, v, err)
	}
}

func TestSet_Overwrite(t *testing.T) {
	svc := newTestService(t)
	_ = svc.Set("theme", "dark")
	if err := svc.Set("theme", "light"); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}
	v, _, _ := svc.Get("theme")
	if v != "light" {
		t.Errorf("overwrite: got %q want light", v)
	}
}

func TestGet_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, ok, err := svc.Get("missing")
	if err != nil || ok {
		t.Fatalf("Get missing: ok=%v err=%v", ok, err)
	}
}

func TestSetGetJSON(t *testing.T) {
	svc := newTestService(t)
	type prefs struct {
		Lang string `json:"lang"`
		Font int    `json:"font"`
	}
	want := prefs{Lang: "zh-CN", Font: 14}
	if err := svc.SetJSON("ui", want); err != nil {
		t.Fatalf("SetJSON: %v", err)
	}
	var got prefs
	ok, err := svc.GetJSON("ui", &got)
	if err != nil || !ok {
		t.Fatalf("GetJSON: ok=%v err=%v", ok, err)
	}
	if got != want {
		t.Errorf("round trip: got %+v want %+v", got, want)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	svc := newTestService(t)
	_ = svc.Set("x", "1")
	if err := svc.Delete("x"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := svc.Delete("x"); err != nil {
		t.Fatalf("second Delete: %v", err)
	}
}

func TestGetAll(t *testing.T) {
	svc := newTestService(t)
	_ = svc.Set("a", "1")
	_ = svc.Set("b", "2")
	snap, err := svc.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if snap.Items["a"] != "1" || snap.Items["b"] != "2" || len(snap.Items) != 2 {
		t.Errorf("snapshot mismatch: %+v", snap.Items)
	}
}
