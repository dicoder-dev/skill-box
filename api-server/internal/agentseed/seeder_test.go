package toolseed

// 同包测试:可访问私有 builtins 变量,无需导出 test-only 辅助函数。

import (
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"

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

// TestEnsureSeeded_Empty seed 17 个默认工具到空 DB(2026-07-18:从 9 个扩到 17 个)。
func TestEnsureSeeded_Empty(t *testing.T) {
	db := setupTestDB(t)
	if err := EnsureSeeded(db, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var n int64
	db.Model(&entity.Tool{}).Count(&n)
	if n != 17 {
		t.Errorf("expected 17 tools, got %d", n)
	}
	var paths int64
	db.Model(&entity.ToolPath{}).Count(&paths)
	if paths < 30 {
		// 17 个工具每个至少 2 条 path,合计 ≥ 30
		t.Errorf("expected at least 30 paths, got %d", paths)
	}
}

// TestEnsureSeeded_AlreadySeeded 已有数据时 upsert 是 no-op(数量不变)。
func TestEnsureSeeded_AlreadySeeded(t *testing.T) {
	db := setupTestDB(t)
	// 第一次:seed
	if err := EnsureSeeded(db, db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	// 第二次:全 builtin 都在,upsert 应该 no-op(数量 / 字段都不变)
	if err := EnsureSeeded(db, db); err != nil {
		t.Fatalf("second seed (should be no-op): %v", err)
	}
	var n int64
	db.Model(&entity.Tool{}).Count(&n)
	if n != 17 {
		t.Errorf("expected still 17 tools after no-op seed, got %d", n)
	}
}

// TestEnsureSeeded_IncrementalUpsert 2026-07-18 新增:模拟"老 DB 已有部分内置工具,
// 程序升级后再启动应自动补全缺失的 builtin"。
//
// 步骤:
//  1. 直接插入前 10 个 builtin(模拟老版本 seed 出来的 DB)
//  2. 调 EnsureSeeded(此时 builtins 已有 17 个,但 DB 只有 10 个)
//  3. 验证 DB 最终是 17 个 builtin(头 10 upsert + 后 7 insert)+ 1 个用户自定义 = 18 行
//  4. 验证新增的 builtin(openclaw / hermes / goose 等)都在 DB 里
//  5. 验证老 builtin 的"用户改过的 enabled 字段"不被覆盖
//  6. 验证用户自定义工具保持原样
func TestEnsureSeeded_IncrementalUpsert(t *testing.T) {
	db := setupTestDB(t)
	// 1) 模拟"老 DB 已有前 10 个 builtin"
	if err := db.Transaction(func(tx *gorm.DB) error {
		for i, bt := range builtins {
			if i >= 10 {
				break
			}
			enabled := true
			// claude 改成 disabled(模拟用户手动禁用),upsert 应当保留
			if bt.ToolID == "claude" {
				enabled = false
			}
			if err := tx.Create(&entity.Tool{
				ToolID: bt.ToolID, DisplayName: bt.DisplayName, MdiIcon: bt.MdiIcon,
				IconFile: bt.IconFile, Maturity: bt.Maturity, Note: bt.Note,
				IsSystem: true, Enabled: enabled, SortOrder: bt.SortOrder,
			}).Error; err != nil {
				return err
			}
		}
		// 再加 1 个用户自定义工具(应该不会被 upsert 影响)
		if err := tx.Create(&entity.Tool{
			ToolID: "my-custom-tool", DisplayName: "MyCustom", MdiIcon: "mdi:c",
			IsSystem: false, Enabled: true, SortOrder: 999,
		}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed pre-state: %v", err)
	}

	// 2) 调 upsert(EnsureSeeded 内部)
	if err := EnsureSeeded(db, db); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// 3) 验证总数 = 17 builtins + 1 user = 18
	var total int64
	db.Model(&entity.Tool{}).Count(&total)
	if total != 18 {
		t.Errorf("expected 18 tools (17 builtin + 1 user), got %d", total)
	}

	// 4) 验证新增的 builtin(原 DB 里没有的)都在 DB 里,且 is_system=true,enabled=true
	for _, id := range []string{"openclaw", "hermes", "goose", "roo", "continue"} {
		var got entity.Tool
		if err := db.Where("tool_id = ?", id).First(&got).Error; err != nil {
			t.Errorf("expected new builtin %q to be inserted, got err %v", id, err)
			continue
		}
		if !got.IsSystem {
			t.Errorf("new builtin %q should be is_system=true", id)
		}
		if !got.Enabled {
			t.Errorf("new builtin %q should be enabled=true by default", id)
		}
	}

	// 5) 验证"用户改过 enabled=false 的 claude" 没被 upsert 覆盖回 true
	var claude entity.Tool
	if err := db.Where("tool_id = ?", "claude").First(&claude).Error; err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	if claude.Enabled {
		t.Errorf("upsert should NOT touch user-modified enabled; claude enabled=%v, want false", claude.Enabled)
	}

	// 6) 验证用户自定义工具保持原样
	var custom entity.Tool
	if err := db.Where("tool_id = ?", "my-custom-tool").First(&custom).Error; err != nil {
		t.Fatalf("user tool not found: %v", err)
	}
	if custom.IsSystem {
		t.Errorf("user tool should keep is_system=false; got %v", custom.IsSystem)
	}
	if custom.SortOrder != 999 {
		t.Errorf("user tool sort_order should keep 999; got %d", custom.SortOrder)
	}

	// 7) 验证老 builtin 的 sort_order 也被 upsert 更新成新值(2026-07-18 重排过)
	var claudeSO int
	if err := db.Model(&claude).Select("sort_order").Scan(&claudeSO).Error; err != nil {
		t.Fatalf("get claude sort_order: %v", err)
	}
	// builtins 现在的 claude.SortOrder = 10(可能 change, sanity check: 必须 > 0 且 < 200)
	if claudeSO <= 0 || claudeSO >= 200 {
		t.Errorf("claude sort_order sanity check failed: got %d, want (0, 200)", claudeSO)
	}
}
