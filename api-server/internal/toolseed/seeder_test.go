package toolseed_test

import (
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/toolseed"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")+"?_pragma=encoding=UTF-8"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&entity.Tool{}, &entity.ToolPath{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestEnsureSeeded_Empty seed 9 个默认工具到空 DB。
func TestEnsureSeeded_Empty(t *testing.T) {
	db := setupTestDB(t)
	if err := toolseed.EnsureSeeded(db, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var n int64
	db.Model(&entity.Tool{}).Count(&n)
	if n != 9 {
		t.Errorf("expected 9 tools, got %d", n)
	}
	var paths int64
	db.Model(&entity.ToolPath{}).Count(&paths)
	if paths < 14 {
		// 9 个工具中 Codex 4 条,部分工具 2-3 条,合计 ≥ 14
		t.Errorf("expected at least 14 paths, got %d", paths)
	}
}

// TestEnsureSeeded_AlreadySeeded 已有数据时不再 seed。
func TestEnsureSeeded_AlreadySeeded(t *testing.T) {
	db := setupTestDB(t)
	// 第一次:seed
	if err := toolseed.EnsureSeeded(db, db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	// 第二次:应跳过
	if err := toolseed.EnsureSeeded(db, db); err != nil {
		t.Fatalf("second seed (should be no-op): %v", err)
	}
	var n int64
	db.Model(&entity.Tool{}).Count(&n)
	if n != 9 {
		t.Errorf("expected still 9 tools after no-op seed, got %d", n)
	}
}

// TestEnsureSeeded_SkipIfUserAdded 已有用户加的工具时也跳过(因为 Count>0)。
func TestEnsureSeeded_SkipIfUserAdded(t *testing.T) {
	db := setupTestDB(t)
	// 模拟"用户先加了 1 个工具"的场景
	if err := db.Create(&entity.Tool{
		ToolID: "user-1", DisplayName: "U1", MdiIcon: "mdi:u", IsSystem: false, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("create user tool: %v", err)
	}
	if err := toolseed.EnsureSeeded(db, db); err != nil {
		t.Fatalf("seed (should be no-op): %v", err)
	}
	var n int64
	db.Model(&entity.Tool{}).Count(&n)
	if n != 1 {
		t.Errorf("seed should not add when user already added; expected 1, got %d", n)
	}
}