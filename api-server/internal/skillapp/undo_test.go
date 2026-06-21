package skillapp_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skillapp"
)

func TestUndoRecord_NilErrors(t *testing.T) {
	if err := skillapp.Undo(nil); !errors.Is(err, skillapp.ErrApplyNotFound) {
		t.Errorf("nil → %v, want ErrApplyNotFound", err)
	}
}

func TestUndoRecord_AlreadyRolled(t *testing.T) {
	rec := &skillapp.UndoRecord{Status: skillapp.StatusRolledBack}
	if err := skillapp.Undo(rec); !errors.Is(err, skillapp.ErrAlreadyRolled) {
		t.Errorf("rolled → %v, want ErrAlreadyRolled", err)
	}
}

func TestUndoRecord_FailedCannotUndo(t *testing.T) {
	rec := &skillapp.UndoRecord{Status: skillapp.StatusFailed}
	err := skillapp.Undo(rec)
	if err == nil || !strings.Contains(err.Error(), "cannot undo") {
		t.Errorf("failed → %v, want cannot undo error", err)
	}
}

func TestUndoRecord_AppliedPasses(t *testing.T) {
	rec := &skillapp.UndoRecord{Status: skillapp.StatusApplied}
	if err := skillapp.Undo(rec); err != nil {
		t.Errorf("applied → %v, want nil", err)
	}
}

func TestUndoWithSnapshot_RemovesAddedFile(t *testing.T) {
	dir := t.TempDir()
	// 假设 pre 之前目录存在但为空 → apply 后写了文件 → undo 应删除它
	pre := &skillapp.PreSnapshot{
		TargetExisted: true,
		Files:         nil, // apply 前目录为空
		PostFiles:     []string{"SKILL.md"},
	}
	// 写一个文件
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	// undo
	if err := skillapp.UndoWithSnapshot(dir, pre.Marshal()); err != nil {
		t.Fatalf("undo: %v", err)
	}
	// 文件应被删
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); !os.IsNotExist(err) {
		t.Errorf("file should be removed; stat err = %v", err)
	}
}

func TestUndoWithSnapshot_RestoresContent(t *testing.T) {
	dir := t.TempDir()
	original := "ORIGINAL"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	pre := &skillapp.PreSnapshot{
		TargetExisted: true,
		Files: []skillapp.FileSnapshot{
			{Path: "SKILL.md", Existed: true, Content: original},
		},
		PostFiles: []string{"SKILL.md"},
	}
	// 模拟 apply 后内容被改
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("OVERWRITTEN"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := skillapp.UndoWithSnapshot(dir, pre.Marshal()); err != nil {
		t.Fatalf("undo: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Errorf("content = %q, want %q", got, original)
	}
}

func TestUndoWithSnapshot_EmptyPreSnapshot(t *testing.T) {
	dir := t.TempDir()
	// 空 JSON → 不报错
	if err := skillapp.UndoWithSnapshot(dir, ""); err != nil {
		t.Errorf("empty snap → %v, want nil", err)
	}
}
