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
	// 真实 BaseAdapter 行为:覆盖式,先清旧 target(symlink / 目录 / 文件)
	// —— 这样 copy / symlink 模式切换时能正常替换。
	if linfo, lerr := os.Lstat(targetDir); lerr == nil && linfo != nil {
		_ = os.RemoveAll(targetDir)
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

// ApplyLink 软链接实现(2026-07-02 增):用 Canonical.SourceDir 作 symlink 目标。
// 真实 BaseAdapter 行为一致;这里测的是 Applier 在 symlink 模式下选 ApplyLink。
func (f *fakeAdapter) ApplyLink(c skilladapter.Canonical, targetDir string) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	if c.SourceDir == "" {
		return errors.New("fake: ApplyLink requires non-empty SourceDir")
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	// 清理旧 target
	if linfo, lerr := os.Lstat(targetDir); lerr == nil && linfo != nil {
		_ = os.RemoveAll(targetDir)
	}
	return os.Symlink(c.SourceDir, targetDir)
}
func (f *fakeAdapter) LocalName(c skilladapter.Canonical) string { return c.Manifest.Name }
func (f *fakeAdapter) Validate(c skilladapter.Canonical) error    { return nil }
func (f *fakeAdapter) IsSystemPath(p string) bool                  { return false }

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

// TestApplyOne_NilRegistry_FallsBackToDefault 验证 2026-06-25 修复:
// NewApplier(nil) 不能 panic,要在 resolveRegistry() 处退化到 skilladapter.DefaultRegistry。
//
// 修复前:传 nil 后 a.registry 是 nil,a.registry.Get() 直接 nil pointer panic。
// 修复后:resolveRegistry 内部判 nil,返回 defaultRegistry,Get 正常工作。
func TestApplyOne_NilRegistry_FallsBackToDefault(t *testing.T) {
	// 注册到 defaultRegistry(全进程共享,可能有别的测试残留,清理时加锁)
	toolID := "test-fallback-tool"
	t.Cleanup(func() {
		// 没暴露 unregister,所以让 defaultRegistry 留着这个 id — 不影响其他测试。
		// 只要 toolID 唯一,不会跟其他 fake 冲突。
	})
	// 注册一个 stub adapter 到默认 registry
	defaultReg := skilladapter.DefaultRegistry()
	// 防御:如果 toolID 已经被注册(比如其他测试用同一个 id),跳过
	if _, exists := defaultReg.Get(toolID); !exists {
		root := t.TempDir()
		fa := &fakeAdapter{id: toolID, root: root}
		defaultReg.Register(fa)
	}

	// 关键:NewApplier(nil) — 旧版会 panic
	ap := skillapp.NewApplier(nil)
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{toolID},
		Canonical: ptrCanon(sampleCanon("fallback-skill")),
	})
	if err != nil {
		t.Fatalf("apply with nil registry: %v (want success via default fallback)", err)
	}
	if res == nil || res.Status != skillapp.StatusApplied {
		t.Errorf("res = %+v, want applied", res)
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

// TestApplyOne_SymlinkMode_CreatesSymlink(2026-07-02):
// Applier.Mode=symlink 时,target 应该是软链接指向 canonical.SourceDir,
// 而非普通目录。同时 PreSnapshot 应该识别 target 之前是 symlink(用于撤销判断)。
func TestApplyOne_SymlinkMode_CreatesSymlink(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	ap.Mode = skillapp.ModeSymlink

	canon := sampleCanon("delta")
	canon.SourceDir = t.TempDir() // 模拟 skillstore 源端
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(canon),
	})
	if err != nil {
		t.Fatalf("apply symlink: %v", err)
	}
	// 1) target 是 symlink
	linfo, lerr := os.Lstat(res.TargetPath)
	if lerr != nil {
		t.Fatalf("lstat target: %v", lerr)
	}
	if linfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target %s is not a symlink (mode=%v)", res.TargetPath, linfo.Mode())
	}
	// 2) 链到的真实目录就是 canonical.SourceDir
	got, _ := os.Readlink(res.TargetPath)
	if got != canon.SourceDir {
		t.Errorf("symlink dst = %q, want %q", got, canon.SourceDir)
	}
	// 3) 第一次 apply,snapshot 应标 target 不存在
	if res.PreSnapshot == nil || res.PreSnapshot.TargetExisted {
		t.Errorf("expected fresh symlink snapshot, got %+v", res.PreSnapshot)
	}
	// 4) PostFiles 应只有 target 自身
	if len(res.PreSnapshot.PostFiles) != 1 || res.PreSnapshot.PostFiles[0] != res.TargetPath {
		t.Errorf("post_files = %v, want only target", res.PreSnapshot.PostFiles)
	}
}

// TestApplyOne_SymlinkMode_UndoRemovesLink(2026-07-02):
// 第二次 apply 同样 source,target 已是 symlink;snapshot 应该标
// TargetWasSymlink=true;UndoWithSnapshot 应该只 Remove 链接,不会把源端文件删了。
func TestApplyOne_SymlinkMode_UndoRemovesLink(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg)
	ap.Mode = skillapp.ModeSymlink

	canon := sampleCanon("epsilon")
	canon.SourceDir = t.TempDir()
	target := filepath.Join(root, canon.Manifest.Name)

	// 预置一个 symlink(target → 源端),模拟"已 apply 过一次"
	if err := os.Symlink(canon.SourceDir, target); err != nil {
		t.Fatal(err)
	}

	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(canon),
	})
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if res.PreSnapshot == nil || !res.PreSnapshot.TargetWasSymlink {
		t.Errorf("expected TargetWasSymlink=true, got %+v", res.PreSnapshot)
	}
	// 验证撤销:只 Remove 链接,源端目录还在
	if err := skillapp.UndoWithSnapshot(res.TargetPath, res.PreSnapshot.Marshal()); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if _, err := os.Lstat(res.TargetPath); !os.IsNotExist(err) {
		t.Errorf("target should be gone, lstat err=%v", err)
	}
	if _, err := os.Stat(canon.SourceDir); err != nil {
		t.Errorf("source dir should still exist: %v", err)
	}
}

// TestApplyOne_CopyMode_Default(2026-07-02):
// 不设 Mode 时,默认走 copy(老行为,兼容性兜底)。
func TestApplyOne_CopyMode_Default(t *testing.T) {
	root := t.TempDir()
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)
	ap := skillapp.NewApplier(reg) // Mode 空 → 走 copy
	canon := sampleCanon("zeta")
	res, err := ap.ApplyOne(skillapp.ApplyInput{
		Scope:     skilladapter.ScopeGlobal,
		Tools:     []string{"fake"},
		Canonical: ptrCanon(canon),
	})
	if err != nil {
		t.Fatal(err)
	}
	linfo, _ := os.Lstat(res.TargetPath)
	if linfo.Mode()&os.ModeSymlink != 0 {
		t.Errorf("default mode should NOT be symlink")
	}
	// 文件应在
	if _, err := os.Stat(filepath.Join(res.TargetPath, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md missing in copy mode: %v", err)
	}
}

func ptrCanon(c skilladapter.Canonical) *skilladapter.Canonical { return &c }

// TestApplyOne_SymlinkMode_RealDisk(2026-07-02):
// 端到端落盘验证:在 tmp dir 准备"源 skill 目录" + 目标 root,跑 symlink 模式
// apply,验证 target 真的是 symlink,且 readlink 指向源端;再跑 copy 模式
// apply 同名 skill,验证 target 被替换成普通目录(且 SKILL.md 存在)。
func TestApplyOne_SymlinkMode_RealDisk(t *testing.T) {
	root := t.TempDir()
	// 准备源 skill(SKILL.md + 一个子文件)
	src := filepath.Join(t.TempDir(), "src-skill")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	fa := &fakeAdapter{id: "fake", root: root}
	reg := newReg(t, fa)

	canon := sampleCanon("omega")
	canon.SourceDir = src

	// 1) symlink 模式 apply
	ap1 := skillapp.NewApplier(reg)
	ap1.Mode = skillapp.ModeSymlink
	res1, err := ap1.ApplyOne(skillapp.ApplyInput{
		Scope: skilladapter.ScopeGlobal, Tools: []string{"fake"}, Canonical: ptrCanon(canon),
	})
	if err != nil {
		t.Fatalf("symlink apply: %v", err)
	}
	linfo, _ := os.Lstat(res1.TargetPath)
	if linfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("phase1: target is not a symlink")
	}
	// 2) copy 模式 apply(切换 mode 后再 ApplyOne)。fakeAdapter 在 Apply 内部
	//    先 RemoveAll 旧 target,模拟真实 BaseAdapter 的"覆盖式"行为。
	ap2 := skillapp.NewApplier(reg)
	ap2.Mode = skillapp.ModeCopy
	// 先把 fakeAdapter 的 Apply 升级:复制标准 BaseAdapter 的"清理旧 target"
	// 行为,这样从 symlink 切到 copy 时,旧 symlink 会被先删,新 copy 才生效。
	res2, err := ap2.ApplyOne(skillapp.ApplyInput{
		Scope: skilladapter.ScopeGlobal, Tools: []string{"fake"}, Canonical: ptrCanon(canon),
	})
	if err != nil {
		t.Fatalf("copy apply: %v", err)
	}
	linfo2, _ := os.Lstat(res2.TargetPath)
	if linfo2.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("phase2: target should be a regular dir, but is symlink")
	}
	// 3) 文件应在(走的是 Apply 的 fakeAdapter 写文件逻辑)
	if _, err := os.Stat(filepath.Join(res2.TargetPath, "SKILL.md")); err != nil {
		t.Errorf("phase2: SKILL.md should exist in copy mode: %v", err)
	}
	// 4) 源端物理文件没被破坏
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
		t.Errorf("source SKILL.md should be untouched: %v", err)
	}
}
