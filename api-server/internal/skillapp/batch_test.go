package skillapp_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillapp"
)

// failAfterAdapter 第二次 Apply 时失败(用于测"第 N 步失败 → 整体回滚")。
type failAfterAdapter struct {
	fakeAdapter
	failOn map[string]bool // name → true = 这次 Apply 失败
}

func (f *failAfterAdapter) Apply(c skilladapter.Canonical, targetDir string) error {
	if f.failOn[c.Manifest.Name] {
		return errors.New("simulated apply fail for " + c.Manifest.Name)
	}
	return f.fakeAdapter.Apply(c, targetDir)
}

func TestBatchApplier_AllOK(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	ba := skillapp.NewBatchApplier(ap)
	items := []skillapp.BatchItem{
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("a"))},
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("b"))},
	}
	out := ba.Apply(items, false)
	if !out.AllOK {
		t.Errorf("AllOK = false, want true; errs = %+v", out.Items)
	}
	if out.RolledBack {
		t.Errorf("RolledBack = true, want false")
	}
}

func TestBatchApplier_AtomicRollback(t *testing.T) {
	root := t.TempDir()
	fa := &failAfterAdapter{
		fakeAdapter: fakeAdapter{id: "fake", root: root},
		failOn:      map[string]bool{"b": true},
	}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	ba := skillapp.NewBatchApplier(ap)
	items := []skillapp.BatchItem{
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("a"))},
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("b"))},
	}
	out := ba.Apply(items, true)
	if out.AllOK {
		t.Errorf("AllOK = true, want false (one failed)")
	}
	if !out.RolledBack {
		t.Errorf("RolledBack = false, want true (atomic)")
	}
	// 第一个 skill "a" 的目录应已被回滚删除(apply 前不存在 + atomic rollback 删 post_files)
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Errorf("'a' dir should be rolled back; stat err = %v", err)
	}
}

func TestBatchApplier_NonAtomicKeepsSucceeded(t *testing.T) {
	root := t.TempDir()
	fa := &failAfterAdapter{
		fakeAdapter: fakeAdapter{id: "fake", root: root},
		failOn:      map[string]bool{"b": true},
	}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	ba := skillapp.NewBatchApplier(ap)
	items := []skillapp.BatchItem{
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("a"))},
		{Tool: "fake", Scope: skilladapter.ScopeGlobal, Canonical: ptrCanon(sampleCanon("b"))},
	}
	out := ba.Apply(items, false)
	if out.AllOK {
		t.Errorf("AllOK = true, want false")
	}
	if out.RolledBack {
		t.Errorf("RolledBack = true, want false (non-atomic)")
	}
	// "a" 应保留
	if _, err := os.Stat(filepath.Join(root, "a")); err != nil {
		t.Errorf("'a' should remain: %v", err)
	}
}

func TestBatchApplier_Empty(t *testing.T) {
	reg := newReg(t, &fakeAdapter{id: "x", root: t.TempDir()})
	ap := skillapp.NewApplier(reg)
	ba := skillapp.NewBatchApplier(ap)
	out := ba.Apply(nil, true)
	if !out.AllOK {
		t.Errorf("empty → AllOK = false, want true")
	}
}
