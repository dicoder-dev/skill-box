package sskillapp_test

import (
	"errors"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/settings"
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
// 2026-07-02 增:为 MigrateMode 测试支持 symlink 落盘。
func (f *fakeAdapter) ApplyLink(c skilladapter.Canonical, targetDir string) error {
	return nil
}
func (f *fakeAdapter) LocalName(c skilladapter.Canonical) string { return c.Manifest.Name }
func (f *fakeAdapter) Validate(c skilladapter.Canonical) error    { return nil }
func (f *fakeAdapter) IsSystemPath(p string) bool                  { return false }

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
		&entity.SkillFile{},
		&entity.SkillApply{},
		&entity.MarketSkill{},
		&entity.AuditLog{},
		&entity.Setting{},
	); err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(store)
	appSvc := sskillapp.New(db, db, func() (*sskill.Service, error) { return ssvc, nil })
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "fake", root: t.TempDir()})
	appSvc.WithAdapterRegistry(reg)
	// 2026-07-02 增:注入 settings,让 Apply / MigrateMode 走 settings.apply_mode。
	appSvc.WithSettings(settings.New(db, db))
	return appSvc, ssvc, store, reg
}

func TestApply_NoTools_ErrEmptyTools(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 1) 建一个 skill
	_, err := ssvc.Create(&sskill.WriteInput{
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
		Name:  "alpha",
		Tools: nil,
	})
	if !errors.Is(err, sskillapp.ErrEmptyTools) {
		t.Errorf("err = %v, want ErrEmptyTools", err)
	}
}

func TestApply_SkillNotFound(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.Apply(&sskillapp.ApplyInput{
		Name:  "no-such-skill",
		Tools: []string{"fake"},
	})
	if !errors.Is(err, sskillapp.ErrSkillNotFound) {
		t.Errorf("err = %v, want ErrSkillNotFound", err)
	}
}

func TestApply_OneSkill_OneTool(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	_, err := ssvc.Create(&sskill.WriteInput{
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
		Name:  "alpha",
		Tools: []string{"fake"},
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
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "s1", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ssvc.Create(&sskill.WriteInput{
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
			{Name: "s1", Tools: []string{"fake"}},
			{Name: "s2", Tools: []string{"fake"}},
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
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Apply(&sskillapp.ApplyInput{Name: "alpha", Tools: []string{"fake"}}); err != nil {
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
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill", Triggers: []string{"a"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.CheckUpdates(sskillapp.CheckUpdatesInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %+v", items)
	}
	if len(items) > 0 && items[0].UpdateAvailable {
		t.Errorf("no market data → no update available")
	}
}

func TestCheckUpdates_WithMarket(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 建一个本地 skill,source=market, source_ref="skillhub:alpha"
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name:        "alpha",
			Version:     "0.1.0",
			Description: "this is a test skill",
			Triggers:    []string{"a"},
			Source:      "market",
			SourceRef:   "skillhub:alpha",
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
	items, err := svc.CheckUpdates(sskillapp.CheckUpdatesInput{})
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

// 2026-07-02 增:验证 MigrateMode 把 settings.apply_mode 切换 + 给每条
// status=applied 行重新落盘的能力。这里 fakeAdapter.Apply/ApplyLink 都返 nil,
// 所以 Entries 里都是 OK,但 settings 已被切到新模式 + SkillApply.ApplyMode 字段
// 也同步更新。
func TestMigrateMode_SwitchCopyToSymlink(t *testing.T) {
	svc, ssvc, _, _ := newTestSvc(t)
	// 1) 建一个 skill
	_, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "mig-skill", Version: "0.1.0", Description: "d", Triggers: []string{"x"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 2) apply 一次(默认 copy 模式)
	_, err = svc.Apply(&sskillapp.ApplyInput{
		Scope: skilladapter.ScopeGlobal,
		Name:  "mig-skill",
		Tools: []string{"fake"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 3) 切到 symlink
	res, err := svc.MigrateMode(skillapp.ModeSymlink)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if res.FromMode != skillapp.ModeCopy || res.ToMode != skillapp.ModeSymlink {
		t.Errorf("from/to = %q/%q, want copy/symlink", res.FromMode, res.ToMode)
	}
	if res.Total != 1 || res.OK != 1 || res.Failed != 0 {
		t.Errorf("entries = %+v, want total=1 ok=1 failed=0", res)
	}
	// 4) 二次切(已 symlink → symlink),Total=0
	res2, err := svc.MigrateMode(skillapp.ModeSymlink)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Total != 0 {
		t.Errorf("idempotent re-migrate: total = %d, want 0", res2.Total)
	}
	// 5) 切回 copy
	res3, err := svc.MigrateMode(skillapp.ModeCopy)
	if err != nil {
		t.Fatal(err)
	}
	if res3.FromMode != skillapp.ModeSymlink || res3.ToMode != skillapp.ModeCopy {
		t.Errorf("back to copy: from/to = %q/%q", res3.FromMode, res3.ToMode)
	}
	if res3.Total != 1 || res3.OK != 1 {
		t.Errorf("back to copy: entries = %+v", res3)
	}
}

func TestMigrateMode_InvalidMode(t *testing.T) {
	svc, _, _, _ := newTestSvc(t)
	_, err := svc.MigrateMode("nonsense")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}
