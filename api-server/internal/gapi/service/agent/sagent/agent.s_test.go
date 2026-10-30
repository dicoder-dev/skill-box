package stool_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/internal/skilladapter"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 起一个 sqlite 临时文件 DB,跑 e_tool + e_tool_path AutoMigrate,
// 写一条用户工具,验证 stool 业务规则。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=encoding=UTF-8"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&entity.Tool{}, &entity.ToolPath{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return db
}

// TestCreate_AndList 新建工具后能从 List 看到。
func TestCreate_AndList(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)

	_, err := svc.Create(&stool.CreateInput{
		ToolID:      "mytool",
		DisplayName: "My Tool",
		MdiIcon:     "mdi:tools",
		Maturity:    "stable",
		Enabled:     true,
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.mytool/skills"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(list))
	}
	if list[0].ToolID != "mytool" {
		t.Errorf("tool_id: got %q want mytool", list[0].ToolID)
	}
	if list[0].IsSystem {
		t.Error("user-created tool should have is_system=false")
	}
	if len(list[0].Paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(list[0].Paths))
	}
	if list[0].Paths[0].Path != "~/.mytool/skills" {
		t.Errorf("path: got %q", list[0].Paths[0].Path)
	}
}

// TestCreate_DuplicateToolID 重复 tool_id 应被拒。
func TestCreate_DuplicateToolID(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	in := &stool.CreateInput{ToolID: "dup", DisplayName: "D", MdiIcon: "mdi:d", Maturity: "stable"}
	if _, err := svc.Create(in); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(in)
	if err == nil {
		t.Error("expected duplicate tool_id error, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

// TestDelete_SystemFrozen 系统工具不可删。
func TestDelete_SystemFrozen(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	// 手工塞一条系统工具(模拟 seed 出来的)
	if err := db.Create(&entity.Tool{
		ToolID: "sys-tool", DisplayName: "S", MdiIcon: "mdi:s", IsSystem: true, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("create sys tool: %v", err)
	}
	err := svc.Delete("sys-tool")
	if err == nil {
		t.Error("expected ErrSystemToolFrozen, got nil")
	}
}

// TestUpdate_AndReload 修改 display_name 后,Reload 让 Registry 反映新值。
//
// 这条测试是 scope-status / skillimporter 走 DB adapter 的端到端验证:
// 改完 e_tool 行 → Reload → skilladapter.All() 返回新 name。
func TestUpdate_AndReload(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)

	// seed 一个工具
	created, err := svc.Create(&stool.CreateInput{
		ToolID: "user-tool", DisplayName: "Old Name", MdiIcon: "mdi:tools", Maturity: "stable",
		Enabled: true,
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.ut/skills"},
			{Scope: "project", Category: "user", Path: ".ut/skills"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = created

	// Reload 前 Registry 空(测试隔离)
	if n := len(skilladapter.All()); n != 0 {
		t.Logf("registry has %d adapters from previous test, expected 0 (test isolation leak)", n)
	}
	if err := svc.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if n := len(skilladapter.All()); n != 1 {
		t.Fatalf("expected 1 adapter after reload, got %d", n)
	}
	a := skilladapter.All()[0]
	if a.DisplayName() != "Old Name" {
		t.Errorf("display_name: got %q want Old Name", a.DisplayName())
	}

	// 改 display_name
	newName := "New Name"
	if _, err := svc.Update(&stool.UpdateInput{
		ToolID: "user-tool", DisplayName: &newName,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if a.DisplayName() == "New Name" {
		t.Error("in-memory adapter should still have old name before reload")
	}
	if err := svc.Reload(); err != nil {
		t.Fatalf("reload 2: %v", err)
	}
	a2 := skilladapter.All()[0]
	if a2.DisplayName() != "New Name" {
		t.Errorf("display_name after reload: got %q want New Name", a2.DisplayName())
	}
	if a2.Icon() != "mdi:tools" {
		t.Errorf("icon: got %q want mdi:tools", a2.Icon())
	}
}

// TestDelete_Cascade 删 user 工具应级联删 e_tool_path。
func TestDelete_Cascade(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	if _, err := svc.Create(&stool.CreateInput{
		ToolID: "casc", DisplayName: "C", MdiIcon: "mdi:c", Maturity: "stable",
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.casc"},
			{Scope: "global", Category: "system", Path: "~/.casc/.system"},
		},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete("casc"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// 验证 path 表已清
	var n int64
	if err := db.Model(&entity.ToolPath{}).Count(&n).Error; err != nil {
		t.Fatalf("count paths: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 paths after cascade delete, got %d", n)
	}
}

// TestCreate_DuplicateScopeCategoryPaths 2026-07-04 改:单 path 模型下,Create
// 入参中两条同 (scope, category) 的 path 必须被 Service 层拦截返 ErrPathExisted,
// 不依赖 DB uniqueIndex 兜底。
func TestCreate_DuplicateScopeCategoryPaths(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	_, err := svc.Create(&stool.CreateInput{
		ToolID:      "dup",
		DisplayName: "Dup",
		MdiIcon:     "mdi:robot-outline",
		Maturity:    "stable",
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.agents/skills", PathOrder: 0},
			{Scope: "global", Category: "user", Path: "~/.cline/skills", PathOrder: 1},
		},
	})
	if err == nil {
		t.Fatal("expected ErrPathExisted, got nil")
	}
	if !errors.Is(err, stool.ErrPathExisted) {
		t.Errorf("expected ErrPathExisted, got %v", err)
	}
}

// TestUpdate_DuplicateScopeCategoryPaths Update 路径同样要拦截同 (scope, category)
// 重复的入参。
func TestUpdate_DuplicateScopeCategoryPaths(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	if _, err := svc.Create(&stool.CreateInput{
		ToolID:      "updup",
		DisplayName: "UpDup",
		MdiIcon:     "mdi:robot-outline",
		Maturity:    "stable",
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.agents/skills", PathOrder: 0},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	paths := []stool.PathInput{
		{Scope: "global", Category: "user", Path: "~/.agents/skills", PathOrder: 0},
		{Scope: "global", Category: "user", Path: "~/.cline/skills", PathOrder: 1},
	}
	_, err := svc.Update(&stool.UpdateInput{ToolID: "updup", Paths: &paths})
	if err == nil {
		t.Fatal("expected ErrPathExisted on update, got nil")
	}
	if !errors.Is(err, stool.ErrPathExisted) {
		t.Errorf("expected ErrPathExisted, got %v", err)
	}
}

// TestAddOnePath_Conflict AddOnePath 第二次同 (scope, category) 必须返 ErrPathExisted。
func TestAddOnePath_Conflict(t *testing.T) {
	db := setupTestDB(t)
	svc := stool.New(db, db)
	if _, err := svc.Create(&stool.CreateInput{
		ToolID:      "addp",
		DisplayName: "AddP",
		MdiIcon:     "mdi:robot-outline",
		Maturity:    "stable",
		Paths: []stool.PathInput{
			{Scope: "global", Category: "user", Path: "~/.agents/skills", PathOrder: 0},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := svc.AddOnePath("addp", stool.PathInput{
		Scope: "global", Category: "user", Path: "~/.cline/skills", PathOrder: 0,
	})
	if err == nil {
		t.Fatal("expected ErrPathExisted on AddOnePath, got nil")
	}
	if !errors.Is(err, stool.ErrPathExisted) {
		t.Errorf("expected ErrPathExisted, got %v", err)
	}
	// 不同 (scope, category) 应该 OK
	if _, err := svc.AddOnePath("addp", stool.PathInput{
		Scope: "global", Category: "system", Path: "~/.cline/system", PathOrder: 0,
	}); err != nil {
		t.Errorf("expected ok on different (scope, category), got %v", err)
	}
}