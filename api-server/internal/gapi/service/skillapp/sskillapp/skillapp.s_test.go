package sskillapp_test

import (
	"errors"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillapp"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeAdapter 服务测试用 - 走 sskill.Service.Create 写到 store,
// 然后 skillapp.Applier 落到 fakeAdapter 的 root。
type fakeAdapter struct {
	id   string
	root string
}

func (f *fakeAdapter) ToolID() string      { return f.id }
func (f *fakeAdapter) DisplayName() string { return "Fake " + f.id }
func (f *fakeAdapter) Icon() string        { return "?" }
func (f *fakeAdapter) DiscoverPaths(scope string) ([]string, error) {
	return []string{f.root}, nil
}
func (f *fakeAdapter) Scan(dir string) ([]skilladapter.Canonical, error) {
	return nil, nil
}
func (f *fakeAdapter) Apply(c skilladapter.Canonical, targetDir string) error {
	return nil // 不必真正写,Service 层只验证链路
}
func (f *fakeAdapter) LocalName(c skilladapter.Canonical) string { return c.Manifest.Name }
func (f *fakeAdapter) Validate(c skilladapter.Canonical) error    { return nil }

func newTestSvc(t *testing.T) (*sskillapp.Service, *sskill.Service, *skillstore.Store, *skilladapter.Registry) {
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
		&entity.SkillApply{},
		&entity.MarketSkill{},
		&entity.AuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(db, db, store)
	appSvc := sskillapp.New(db, db, func() (*sskill.Service, error) { return ssvc, nil })
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "fake", root: t.TempDir()})
	appSvc.WithAdapterRegistry(reg)
	return appSvc, ssvc, store, reg
}

func TestApply_NoTools_ErrEmptyTools(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 1) 建一个 skill
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name:        "alpha",
			Version:     "0.1.0",
			Description: "this is a test skill",
			Triggers:    []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Apply(&sskillapp.ApplyInput{
		SkillID: row.ID,
		Tools:   nil,
	})
	if !errors.Is(err, sskillapp.ErrEmptyTools) {
		t.Errorf("err = %v, want ErrEmptyTools", err)
	}
}

func TestApply_SkillNotFound(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.Apply(&sskillapp.ApplyInput{
		SkillID: 999,
		Tools:   []string{"fake"},
	})
	if !errors.Is(err, sskillapp.ErrSkillNotFound) {
		t.Errorf("err = %v, want ErrSkillNotFound", err)
	}
}

func TestApply_OneSkill_OneTool(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name:        "alpha",
			Version:     "0.1.0",
			Description: "this is a test skill",
			Triggers:    []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Apply(&sskillapp.ApplyInput{
		SkillID: row.ID,
		Tools:   []string{"fake"},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !out.AllOK {
		t.Errorf("AllOK = false; applies=%+v", out.Applies)
	}
	if len(out.Applies) != 1 {
		t.Fatalf("len(applies) = %d, want 1", len(out.Applies))
	}
}

func TestBatchApply_Atomic(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 建两个 skill
	r1, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "s1", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "s2", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.BatchApply(&sskillapp.BatchApplyInput{
		Items: []sskillapp.ApplyInput{
			{SkillID: r1.ID, Tools: []string{"fake"}},
			{SkillID: r2.ID, Tools: []string{"fake"}},
		},
		Atomic: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.AllOK {
		t.Errorf("AllOK = false (fake apply 不应失败): %+v", out)
	}
}

func TestUndo_NotFound(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.Undo(999)
	if !errors.Is(err, skillapp.ErrApplyNotFound) {
		t.Errorf("err = %v, want ErrApplyNotFound", err)
	}
}

func TestUndo_ZeroID(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.Undo(0)
	if !errors.Is(err, skillapp.ErrApplyNotFound) {
		t.Errorf("err = %v, want ErrApplyNotFound", err)
	}
}

func TestList_Empty(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	res, err := svc.List(sskillapp.ListInput{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("expected empty, got %+v", res)
	}
}

func TestList_InvalidStatus(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.List(sskillapp.ListInput{Status: "weird"})
	if err == nil {
		t.Errorf("expected error for invalid status")
	}
}

func TestList_AfterApply(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Apply(&sskillapp.ApplyInput{SkillID: row.ID, Tools: []string{"fake"}}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.List(sskillapp.ListInput{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total < 1 {
		t.Errorf("total = %d, want >=1", res.Total)
	}
}

func TestCheckUpdates_NoMarket(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.CheckUpdates("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SkillID != row.ID {
		t.Errorf("expected 1 item for skill %d, got %+v", row.ID, items)
	}
	if items[0].UpdateAvailable {
		t.Errorf("no market data → no update available")
	}
}

func TestCheckUpdates_WithMarket(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 建一个本地 skill,source=market, source_ref="skillhub:alpha"
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope:     skilladapter.ScopeGlobal,
		Source:    "market",
		SourceRef: "skillhub:alpha",
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 走 svc 的 db 写一条 market_skill
	svc.WriteMarketSkillForTest(&entity.MarketSkill{
		SourceName: "skillhub", RemoteID: "alpha", Name: "alpha", Version: "0.2.0",
	})
	items, err := svc.CheckUpdates("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if !items[0].UpdateAvailable {
		t.Errorf("expected update available; got %+v", items[0])
	}
}

// TestApply_WritesAuditLog 验证成功 apply 后 audit_log 进了 1 条 action=apply。
func TestApply_WritesAuditLog(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope:    skilladapter.ScopeGlobal,
		Source:   "local",
		Manifest: skilladapter.Manifest{Name: "audit-target", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"}},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Apply(&sskillapp.ApplyInput{
		SkillID: row.ID,
		Tools:   []string{"fake"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.AllOK {
		t.Fatalf("expected AllOK; got %+v", out)
	}
	// 直接查 audit_log 表
	var n int64
	if err := svc.GetDBForTest().Model(&entity.AuditLog{}).Where("action = ? AND target_id = ?", "apply", row.ID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log apply count = %d, want 1", n)
	}
}

// TestUndo_WritesAuditLog 验证成功 undo 后 audit_log 进了 1 条 action=undo。
func TestUndo_WritesAuditLog(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope:    skilladapter.ScopeGlobal,
		Source:   "local",
		Manifest: skilladapter.Manifest{Name: "undo-audit", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"}},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Apply(&sskillapp.ApplyInput{
		SkillID: row.ID,
		Tools:   []string{"fake"},
	})
	if err != nil || !out.AllOK || len(out.Applies) == 0 {
		t.Fatalf("apply prep failed: %v %+v", err, out)
	}
	applyID := out.Applies[0].ApplyID
	if _, err := svc.Undo(applyID); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := svc.GetDBForTest().Model(&entity.AuditLog{}).Where("action = ? AND target_id = ?", "undo", row.ID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit_log undo count = %d, want 1", n)
	}
}
