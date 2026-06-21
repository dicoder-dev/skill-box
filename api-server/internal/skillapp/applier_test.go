package skillapp_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillapp"
)

// fakeAdapter 一个最小可用的 adapter 实现(指向 tmp dir + 可控 Apply 行为)。
type fakeAdapter struct {
	id       string
	root     string // DiscoverPaths 唯一返回
	applyErr error  // nil = 成功
	touched  *[]string
}

func (f *fakeAdapter) ToolID() string      { return f.id }
func (f *fakeAdapter) DisplayName() string { return "Fake " + f.id }
func (f *fakeAdapter) Icon() string        { return "?"
}
func (f *fakeAdapter) DiscoverPaths(scope string) ([]string, error) {
	return []string{f.root}, nil
}
func (f *fakeAdapter) Scan(dir string) ([]skilladapter.Canonical, error) {
	return nil, nil
}
func (f *fakeAdapter) Apply(c skilladapter.Canonical, targetDir string) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	if f.touched != nil {
		*f.touched = append(*f.touched, targetDir)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	for _, fl := range c.Files {
		if err := os.MkdirAll(filepath.Join(targetDir, filepath.Dir(fl.Path)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(targetDir, fl.Path), []byte(fl.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeAdapter) LocalName(c skilladapter.Canonical) string { return c.Manifest.Name }
func (f *fakeAdapter) Validate(c skilladapter.Canonical) error    { return nil }

func newReg(t *testing.T, a skilladapter.Adapter) *skilladapter.Registry {
	t.Helper()
	r := &skilladapter.Registry{}
	r.Register(a)
	return r
}

func sampleCanon(name string) skilladapter.Canonical {
	return skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: name, Version: "0.1.0"},
		Files: []skilladapter.File{
			{Path: "SKILL.md", Content: "---\nname: " + name + "\n---\nbody for " + name},
		},
	}
}

func TestApplyOne_Success_PreSnapshot(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(sampleCanon("alpha")),
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Status != skillapp.StatusApplied {
		t.Errorf("status = %q, want applied", res.Status)
	}
	if res.PreSnapshot == nil {
		t.Fatal("pre snapshot nil")
	}
	// apply 之前目录不存在 → TargetExisted=false
	if res.PreSnapshot.TargetExisted {
		t.Errorf("TargetExisted = true, want false (fresh dir)")
	}
	// apply 之后目录存在
	if _, err := os.Stat(filepath.Join(root, "alpha")); err != nil {
		t.Errorf("target dir not created: %v", err)
	}
}

func TestApplyOne_RollsBack_OnApplyError(t *testing.T) {
	root := t.TempDir()
	// 先在目标位置塞一个文件,模拟"目标原本有内容"
	if err := os.MkdirAll(filepath.Join(root, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "ORIGINAL"
	if err := os.WriteFile(filepath.Join(root, "alpha", "SKILL.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	fa := &fakeAdapter{id: "fake", root: root, applyErr: errors.New("simulated failure")}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(sampleCanon("alpha")),
	})
	if err == nil {
		t.Fatal("expected apply error")
	}
	if res == nil {
		t.Fatal("expected result even on error (with pre-snapshot)")
	}
	if res.Status != skillapp.StatusFailed {
		t.Errorf("status = %q, want failed", res.Status)
	}
	// 验证原始内容还在(没被半成品污染)
	got, rerr := os.ReadFile(filepath.Join(root, "alpha", "SKILL.md"))
	if rerr != nil {
		t.Fatalf("read after rollback: %v", rerr)
	}
	if string(got) != original {
		t.Errorf("file content changed: got %q, want %q", got, original)
	}
}

func TestApplyOne_RejectsBadScope(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	_, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     "moon",
		Tools:     []string{"fake"},
		Canonical: ptrCanon(sampleCanon("alpha")),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid scope") {
		t.Errorf("err = %v, want invalid scope", err)
	}
}

func TestApplyOne_RejectsUnknownTool(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	_, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"ghost"},
		Canonical: ptrCanon(sampleCanon("alpha")),
	})
	if !errors.Is(err, skillapp.ErrToolNotFound) {
		t.Errorf("err = %v, want ErrToolNotFound", err)
	}
}

func TestApplyOne_RejectsEmptyFiles(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	_, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: &skilladapter.Canonical{Manifest: skilladapter.Manifest{Name: "x"}},
	})
	if !errors.Is(err, skillapp.ErrEmptyFiles) {
		t.Errorf("err = %v, want ErrEmptyFiles", err)
	}
}

func TestApplyOne_RejectsEmptyTools(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	_, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     nil,
		Canonical: ptrCanon(sampleCanon("alpha")),
	})
	if !errors.Is(err, skillapp.ErrEmptyTools) {
		t.Errorf("err = %v, want ErrEmptyTools", err)
	}
}

func TestSnapshotDir_NonExistent(t *testing.T) {
	// 私有 helper 通过 Apply 间接覆盖(目录不存在 → TargetExisted=false)。
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(sampleCanon("beta")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PreSnapshot == nil || res.PreSnapshot.TargetExisted {
		t.Errorf("expected target not existed; got %+v", res.PreSnapshot)
	}
}

func ptrCanon(c skilladapter.Canonical) *skilladapter.Canonical { return &c }
