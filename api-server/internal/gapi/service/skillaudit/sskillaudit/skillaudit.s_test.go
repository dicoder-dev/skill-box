package sskillaudit_test

import (
	"errors"
	"path/filepath"
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestSvc(t *testing.T) (*sskillaudit.Service, *sskill.Service) {
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
		&entity.SkillTag{},
		&entity.SkillFileSnapshot{},
	); err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(db, db, store)
	return sskillaudit.New(db, db, store), ssvc
}

func sample(name, body string) skilladapter.Canonical {
	return skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name: name, Version: "0.1.0",
			Description: "this is a test skill for " + name,
			Triggers:    []string{"test"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: body}},
	}
}

func TestCreateTag_OK(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, err := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "v1 body").Files,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.CreateTag(&sskillaudit.CreateTagInput{
		SkillID: row.ID,
		Tag:     "v1.0.0",
		Message: "release",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TagID == 0 || out.Files != 1 {
		t.Errorf("unexpected: %+v", out)
	}
}

func TestCreateTag_SkillNotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: 999, Tag: "v1"})
	if !errors.Is(err, sskillaudit.ErrSkillNotFound) {
		t.Errorf("err = %v, want ErrSkillNotFound", err)
	}
}

func TestCreateTag_InvalidTag(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, _ := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "x").Files,
	})
	_, err := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: ""})
	if !errors.Is(err, sskillaudit.ErrInvalidTag) {
		t.Errorf("err = %v, want ErrInvalidTag", err)
	}
}

func TestListTags_Empty(t *testing.T) {
	svc, _ := newTestSvc(t)
	tags, err := svc.ListTags(999)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Errorf("empty: got %d", len(tags))
	}
}

func TestListTags_AfterCreate(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, _ := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "x").Files,
	})
	if _, err := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: "v1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: "v2"}); err != nil {
		t.Fatal(err)
	}
	tags, err := svc.ListTags(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Errorf("len = %d, want 2", len(tags))
	}
}

func TestDeleteTag_OK(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, _ := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "x").Files,
	})
	out, _ := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: "v1"})
	if err := svc.DeleteTag(out.TagID); err != nil {
		t.Errorf("delete: %v", err)
	}
	// 再列应空
	tags, _ := svc.ListTags(row.ID)
	if len(tags) != 0 {
		t.Errorf("after delete: %d", len(tags))
	}
}

func TestDeleteTag_NotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	if err := svc.DeleteTag(999); !errors.Is(err, sskillaudit.ErrTagNotFound) {
		t.Errorf("err = %v, want ErrTagNotFound", err)
	}
}

func TestDiff_CurrentVsTag(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, _ := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "first body").Files,
	})
	// 打 tag = 当前
	t1, _ := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: "v1"})
	// 改 skill
	if _, err := ssvc.Update(row.Scope, row.Name, row.Version, row.ProjectID, &sskill.WriteInput{
		Scope:    row.Scope,
		Manifest: skilladapter.Manifest{Name: row.Name, Version: row.Version, Description: "this is a test skill for alpha", Triggers: []string{"a"}},
		Files:    sample("alpha", "second body").Files,
	}); err != nil {
		t.Fatal(err)
	}
	// diff: left=tag(0=v1 first body), right=current (second body)
	out, err := svc.Diff(&sskillaudit.DiffInput{
		SkillID:    row.ID,
		LeftTagID:  t1.TagID,
		RightTagID: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Modified == 0 {
		t.Errorf("expected modified > 0; got %+v", out)
	}
}

func TestRollback_OK(t *testing.T) {
	svc, ssvc := newTestSvc(t)
	row, _ := ssvc.Create(&sskill.WriteInput{
		Scope: skilladapter.ScopeGlobal,
		Manifest: skilladapter.Manifest{
			Name: "alpha", Version: "0.1.0", Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "v1 body").Files,
	})
	// 打 tag
	t1, _ := svc.CreateTag(&sskillaudit.CreateTagInput{SkillID: row.ID, Tag: "v1"})
	// 改 skill
	if _, err := ssvc.Update(row.Scope, row.Name, row.Version, row.ProjectID, &sskill.WriteInput{
		Scope: row.Scope,
		Manifest: skilladapter.Manifest{
			Name: row.Name, Version: row.Version, Description: "this is a test skill for alpha", Triggers: []string{"a"},
		},
		Files: sample("alpha", "v2 body").Files,
	}); err != nil {
		t.Fatal(err)
	}
	// 回滚
	out, err := svc.Rollback(&sskillaudit.RollbackInput{TagID: t1.TagID})
	if err != nil {
		t.Fatal(err)
	}
	if out.PreRollbackTagID == 0 {
		t.Errorf("PreRollbackTagID = 0")
	}
	// 验证 pre-rollback tag 是隐式的
	tags, _ := svc.ListTags(row.ID)
	if len(tags) != 2 {
		t.Errorf("after rollback: tag count = %d, want 2 (v1 + pre-rollback)", len(tags))
	}
	// 验证文件内容已恢复
	full, _ := ssvc.GetFull(row.Scope, row.Name, row.Version, row.ProjectID)
	if len(full.Canonical.Files) == 0 || full.Canonical.Files[0].Content != "v1 body" {
		t.Errorf("file content not restored: %+v", full.Canonical.Files)
	}
}

func TestRollback_NotFound(t *testing.T) {
	svc, _ := newTestSvc(t)
	_, err := svc.Rollback(&sskillaudit.RollbackInput{TagID: 999})
	if !errors.Is(err, sskillaudit.ErrTagNotFound) {
		t.Errorf("err = %v, want ErrTagNotFound", err)
	}
}
