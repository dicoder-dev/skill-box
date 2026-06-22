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

// TestImport_WritesAuditLog 验证 import 成功后在 audit_log 落 action=import。
func TestImport_WritesAuditLog(t *testing.T) {
	svc, _, db := newTestSvc(t)
	// 自己造一个 zip:走 skillpkg.BuildBytes 反向闭环
	provider := &fakeCanonicalProvider{
		items: map[string]skilladapter.Canonical{
			"global:audit-import@0.1.0": {
				Manifest: skilladapter.Manifest{
					Name: "audit-import", Version: "0.1.0", Description: "this is a test skill for import audit", Triggers: []string{"a"},
				},
				Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
			},
		},
	}
	zipBytes, _, err := skillpkg.BuildBytes(skillpkg.ExportRequest{
		Skills: []string{"global:audit-import@0.1.0"},
	}, provider)
	if err != nil {
		t.Fatal(err)
	}
	inst := skillpkg.NewImporter(provider)
	_ = inst // 触发 _ = inst 包内引用避免 unused

	// 走真 service.Import — 让它装进 store 并写 audit
	installerProvider := &installOnlyProvider{canInstall: true}
	// 直接构造一个最小 zip,用 fakeInstaller
	zipBytes2, _, err := skillpkg.BuildBytes(skillpkg.ExportRequest{
		Skills: []string{"global:audit-import2@0.1.0"},
	}, &fakeCanonicalProvider{
		items: map[string]skilladapter.Canonical{
			"global:audit-import2@0.1.0": {
				Manifest: skilladapter.Manifest{
					Name: "audit-import2", Version: "0.1.0", Description: "this is a test skill for import audit", Triggers: []string{"a"},
				},
				Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = zipBytes

	// 走 ssvc.Create 装一个 skill 然后 Export → 再 Import
	ssvc, _ := skillStoreFromSvc(t)
	if _, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Source: "local",
		Manifest: skilladapter.Manifest{
			Name: "audit-import3", Version: "0.1.0", Description: "this is a test skill for import audit", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	exportedBytes, _, err := svc.BuildExport(skillpkg.ExportRequest{
		Skills: []string{"global:audit-import3@0.1.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Import(exportedBytes, skillpkg.ImportRequest{
		TargetScope: skilladapter.ScopeGlobal,
		// skills 不传 → 装全部
	}); err != nil {
		t.Fatal(err)
	}

	// export 也会写 1 条 + import 1 条
	var n int64
	if err := db.Model(&entity.AuditLog{}).Where("action = ?", "import").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log import count = %d, want 1", n)
	}
	if err := db.Model(&entity.AuditLog{}).Where("action = ?", "export").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log export count = %d, want 1", n)
	}
}

func skillStoreFromSvc(t *testing.T) (*sskill.Service, *skillstore.Store) {
	t.Helper()
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	return sskill.New(nil, nil, store), store
}

// fakeCanonicalProvider 给 skillpkg.BuildBytes 用。
type fakeCanonicalProvider struct {
	items map[string]skilladapter.Canonical
}

func (f *fakeCanonicalProvider) LoadCanonical(scope string, projectID uint, name, version string) (skilladapter.Canonical, bool, error) {
	k := scope + ":" + name + "@" + version
	if c, ok := f.items[k]; ok {
		return c, true, nil
	}
	return skilladapter.Canonical{}, false, nil
}

func (f *fakeCanonicalProvider) InstallCanonical(scope string, projectID uint, c skilladapter.Canonical, source string) (uint, error) {
	return 0, nil
}

type installOnlyProvider struct{ canInstall bool }
