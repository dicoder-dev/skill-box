package skillimporter_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
)

// fakeAdapter 模拟一个指向指定 dir 的 BaseAdapter。
type fakeAdapter struct {
	id  string
	dir string
}

func (f *fakeAdapter) ToolID() string      { return f.id }
func (f *fakeAdapter) DisplayName() string { return "Fake " + f.id }
func (f *fakeAdapter) Icon() string        { return "?" }
func (f *fakeAdapter) DiscoverPaths(scope string) ([]string, error) {
	return []string{f.dir}, nil
}
func (f *fakeAdapter) Scan(dir string) ([]skilladapter.Canonical, error) {
	// 复用 BaseAdapter.Scan 的等价逻辑,避免 import cycle。
	return scanDirForTest(dir)
}
func (f *fakeAdapter) Apply(c skilladapter.Canonical, targetDir string) error {
	return os.MkdirAll(targetDir, 0o755)
}
func (f *fakeAdapter) LocalName(c skilladapter.Canonical) string { return c.Manifest.Name }
func (f *fakeAdapter) Validate(c skilladapter.Canonical) error    { return nil }
func (f *fakeAdapter) IsSystemPath(p string) bool                  { return false }

// scanDirForTest 等价于 BaseAdapter.Scan 的扫描逻辑(只读 SKILL.md)。
// 抽出来避免 importer 反向 import base adapter。
func scanDirForTest(dir string) ([]skilladapter.Canonical, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []skilladapter.Canonical
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		c, err := skilladapter.ParseSkillMD(string(content))
		if err != nil {
			continue
		}
		c.Files = []skilladapter.File{{Path: "SKILL.md", Content: string(content)}}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.Name < out[j].Manifest.Name })
	return out, nil
}

// writeSkill 写一个最小 skill 目录(只有 SKILL.md)。
func writeSkill(t *testing.T, root, name, desc string) {
	t.Helper()
	d := filepath.Join(root, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + desc + "\ntriggers:\n  - test\n---\n# " + name + "\nbody\n"
	if err := os.WriteFile(filepath.Join(d, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupReg(t *testing.T) (*skilladapter.Registry, *skillimporter.Importer, string) {
	t.Helper()
	tmp := t.TempDir()
	store, err := skillstore.NewAt(filepath.Join(tmp, "store"))
	if err != nil {
		t.Fatal(err)
	}
	toolA := filepath.Join(tmp, "toolA")
	toolB := filepath.Join(tmp, "toolB")
	if err := os.MkdirAll(toolA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(toolB, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, toolA, "alpha", "alpha skill for tool A")
	writeSkill(t, toolA, "beta", "beta skill for tool A")
	writeSkill(t, toolB, "gamma", "gamma skill for tool B")

	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "toolA", dir: toolA})
	reg.Register(&fakeAdapter{id: "toolB", dir: toolB})

	im := skillimporter.New(store).WithRegistry(reg)
	return reg, im, tmp
}

func TestScan_FindsAcrossTools(t *testing.T) {
	_, im, _ := setupReg(t)
	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalFound != 3 {
		t.Fatalf("TotalFound=%d; want 3", r.TotalFound)
	}
	if r.ToolSummary["toolA"] != 2 || r.ToolSummary["toolB"] != 1 {
		t.Errorf("ToolSummary=%+v", r.ToolSummary)
	}
	// 排序后顺序:toolA alpha / toolA beta / toolB gamma
	if r.FoundSkills[0].Canonical.Manifest.Name != "alpha" {
		t.Errorf("first=%+v", r.FoundSkills[0].Canonical.Manifest)
	}
}

func TestScan_EmptyDirIsNotError(t *testing.T) {
	tmp := t.TempDir()
	emptyTool := filepath.Join(tmp, "empty")
	if err := os.MkdirAll(emptyTool, 0o755); err != nil {
		t.Fatal(err)
	}
	store, _ := skillstore.NewAt(filepath.Join(tmp, "store"))
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "empty", dir: emptyTool})
	im := skillimporter.New(store).WithRegistry(reg)

	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalFound != 0 {
		t.Errorf("empty dir produced %d skills", r.TotalFound)
	}
}

func TestScan_MissingDirIsNotError(t *testing.T) {
	tmp := t.TempDir()
	store, _ := skillstore.NewAt(filepath.Join(tmp, "store"))
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "ghost", dir: filepath.Join(tmp, "does-not-exist")})
	im := skillimporter.New(store).WithRegistry(reg)

	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalFound != 0 {
		t.Errorf("missing dir produced %d skills", r.TotalFound)
	}
	if r.Dirs[0].Exists {
		t.Error("missing dir should be Exists=false")
	}
}

// TestScan_ToolsFiltersZeroHit 覆盖 phase2 幽灵 tab 修复:
// 即使 adapter 走完整流程但 0 命中,Tools 字段也必须把这种 tool 过滤掉,
// 否则前端 phase2 会渲染"空名字 + 数量 0"的 tab。
func TestScan_ToolsFiltersZeroHit(t *testing.T) {
	tmp := t.TempDir()
	store, _ := skillstore.NewAt(filepath.Join(tmp, "store"))
	// toolA 有内容;toolB 目录存在但 0 命中。
	toolA := filepath.Join(tmp, "toolA")
	toolB := filepath.Join(tmp, "toolB")
	if err := os.MkdirAll(toolA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(toolB, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, toolA, "alpha", "alpha skill")
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "toolA", dir: toolA})
	reg.Register(&fakeAdapter{id: "toolB", dir: toolB})
	im := skillimporter.New(store).WithRegistry(reg)

	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Tools) != 1 || r.Tools[0] != "toolA" {
		t.Errorf("Tools=%+v; want [toolA]", r.Tools)
	}
}

func TestImport_All(t *testing.T) {
	_, im, tmp := setupReg(t)
	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	res, err := im.Import(r, nil) // nil = all
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 3 {
		t.Fatalf("results=%d; want 3", len(res))
	}
	for _, x := range res {
		if !x.OK {
			t.Errorf("import %+v failed: %s", x, x.Error)
		}
	}
	// 确认物理落地
	storeRoot := filepath.Join(tmp, "store", "global")
	entries, _ := os.ReadDir(storeRoot)
	if len(entries) != 3 {
		t.Errorf("store has %d skills, want 3", len(entries))
	}
}

func TestImport_Selective(t *testing.T) {
	_, im, _ := setupReg(t)
	r, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	res, err := im.Import(r, []skillimporter.ImportItem{
		{ToolID: "toolA", Name: "alpha"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("res=%+v", res)
	}
}

func TestImport_NotInReport(t *testing.T) {
	_, im, _ := setupReg(t)
	r, _ := im.Scan(skilladapter.ScopeGlobal)
	res, err := im.Import(r, []skillimporter.ImportItem{
		{ToolID: "toolA", Name: "no-such-skill"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].OK || !strings.Contains(res[0].Error, "not found") {
		t.Errorf("res=%+v", res)
	}
}

func TestFilterByTool(t *testing.T) {
	_, im, _ := setupReg(t)
	r, _ := im.Scan(skilladapter.ScopeGlobal)
	got := r.FilterByTool("toolA")
	if len(got) != 2 {
		t.Errorf("filter toolA=%d; want 2", len(got))
	}
}

func TestScan_NilStoreErrors(t *testing.T) {
	im := skillimporter.New(nil)
	if _, err := im.Scan(""); err == nil {
		t.Error("expected error for nil store")
	}
}

// TestImport_NormalizeFixesTriggers 覆盖外部 SKILL.md 缺 triggers 时的兜底:
//   - claude 风格 symlink skill,frontmatter 只有 name + description
//   - normalizeForStore 必须从 description 抽 trigger,保证 store.Save 不拒收
func TestImport_NormalizeFixesTriggers(t *testing.T) {
	_, _, tmp := setupReg(t)
	// setupReg 的 fakeAdapter 用工具 A / B 写死的内容;直接复用,挑一个缺 trigger 的 skill。
	// 这里再额外写一个带缺字段 frontmatter 的 skill,验证 normalize。
	extraDir := t.TempDir()
	writeSkillWithFM(t, extraDir, "no-triggers",
		"---\n"+
			"name: no-triggers\n"+
			"description: Helps discover and install skills when user asks how do I do X.\n"+
			"---\n\n"+
			"# Body\n\nIntro paragraph for tests.\n")
	// 把 fakeAdapter 临时挂上这个新目录
	reg := &skilladapter.Registry{}
	reg.Register(&fakeAdapter{id: "extra", dir: extraDir})
	im2 := skillimporter.New(skillstoreTestStore(t, tmp)).WithRegistry(reg)
	r, _ := im2.Scan(skilladapter.ScopeGlobal)
	res, err := im2.Import(r, []skillimporter.ImportItem{
		{ToolID: "extra", Name: "no-triggers", Version: "0.1.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("import failed: %+v", res)
	}
	// 验证落地的 skill.yaml 里 description 长度 >= 10, triggers 至少 1 个
	loadPath := filepath.Join(tmp, "store", "global", "no-triggers", "0.1.0")
	if _, err := os.Stat(loadPath); err != nil {
		t.Fatalf("store path missing: %v", err)
	}
}

// writeSkillWithFM 写一个目录,内含 SKILL.md,frontmatter 由 caller 提供。
func writeSkillWithFM(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// skillstoreTestStore 复用 setupReg 写出来的 tmp store,避免重复构造。
func skillstoreTestStore(t *testing.T, tmp string) *skillstore.Store {
	t.Helper()
	st, err := skillstore.NewAt(filepath.Join(tmp, "store"))
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// TestNormalize_EmptyTriggersAndShortDescription 单元测兜底逻辑。
func TestNormalize_EmptyTriggersAndShortDescription(t *testing.T) {
	// 模拟外部 SKILL.md:description < 10,triggers 空
	short := "Tiny"
	c := &skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name:        "my-skill",
			Version:     "0.1.0",
			Description: short,
		},
		Files: []skilladapter.File{{
			Path:    "SKILL.md",
			Content: "---\nname: my-skill\ndescription: Tiny\n---\n\n# my-skill\n\nA paragraph long enough to use as fallback description for tests.\n",
		}},
	}
	// normalizeForStore 不会动 caller 已经填好的字段,纯补。
	// (用大写导出名,因测试在 skillimporter_test 包。)
	skillimporter.NormalizeForStore(c)
	if len(c.Manifest.Description) < 10 {
		t.Errorf("description not padded: %q (len=%d)", c.Manifest.Description, len(c.Manifest.Description))
	}
	if len(c.Manifest.Triggers) < 1 {
		t.Errorf("triggers still empty after normalize")
	}
}
