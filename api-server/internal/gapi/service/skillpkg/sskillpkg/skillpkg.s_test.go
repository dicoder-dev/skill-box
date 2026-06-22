package sskillpkg_test

import (
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillpkg/sskillpkg"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillpkg"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestSvc(t *testing.T) (*sskillpkg.Service, *sskill.Service, *gorm.DB) {
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
		&entity.Skill{},
		&entity.SkillFile{},
		&entity.AuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(db, db, store)
	svc := sskillpkg.New(db, db, func() (*sskill.Service, error) { return ssvc, nil })
	return svc, ssvc, db
}

// TestExportImport_WritesAuditLog 验证 export + import 成功路径上 audit_log 各落 1 条。
func TestExportImport_WritesAuditLog(t *testing.T) {
	svc, ssvc, db := newTestSvc(t)
	if _, err := ssvc.Create(&sskill.WriteInput{
		Scope:  skilladapter.ScopeGlobal,
		Source: "local",
		Manifest: skilladapter.Manifest{
			Name: "audit-pkg", Version: "0.1.0", Description: "this is a test skill for pkg audit", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	exportedBytes, _, err := svc.BuildExport(skillpkg.ExportRequest{
		Skills: []skillpkg.SkillRef{{Scope: skilladapter.ScopeGlobal, Name: "audit-pkg", Version: "0.1.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Import(exportedBytes, skillpkg.ImportRequest{
		TargetScope: skilladapter.ScopeGlobal,
	}); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Model(&entity.AuditLog{}).Where("action = ?", "export").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log export count = %d, want 1", n)
	}
	if err := db.Model(&entity.AuditLog{}).Where("action = ?", "import").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log import count = %d, want 1", n)
	}
}
