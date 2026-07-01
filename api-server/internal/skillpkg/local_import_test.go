package skillpkg

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skillstore"
)

// makeStore 构造一个指向 t.TempDir/store 的 Store,测试结束自动清理。
func makeStore(t *testing.T) *skillstore.Store {
	t.Helper()
	s, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("NewAt: %v", err)
	}
	return s
}

// writeSkillMD 写一个最小 SKILL.md(含 frontmatter + H1)。
func writeSkillMD(t *testing.T, path, name, desc string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + desc + "\ntriggers:\n  - test\n---\n# " + name + "\nbody line\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestImportFromFolder_SingleSkill 单个 skill 目录(直接含 SKILL.md)。
func TestImportFromFolder_SingleSkill(t *testing.T) {
	root := t.TempDir()
	writeSkillMD(t, filepath.Join(root, "alpha", "SKILL.md"), "alpha", "alpha description ok")

	store := makeStore(t)
	out, err := ImportFromFolder(store, root)
	if err != nil {
		t.Fatalf("ImportFromFolder: %v", err)
	}
	if out.Found != 1 || out.OK != 1 || out.Failed != 0 {
		t.Fatalf("counts: Found=%d OK=%d Failed=%d; want 1/1/0", out.Found, out.OK, out.Failed)
	}
	if out.Results[0].Name != "alpha" || !out.Results[0].OK {
		t.Fatalf("result[0]: %+v", out.Results[0])
	}
	// 验证落盘
	if _, err := os.Stat(filepath.Join(store.Root(), "alpha", "SKILL.md")); err != nil {
		t.Fatalf("alpha not in store: %v", err)
	}
}

// TestImportFromFolder_NoSKILL 目录里没 SKILL.md → ErrNoSkillMD。
func TestImportFromFolder_NoSKILL(t *testing.T) {
	root := t.TempDir()
	// 写个无关文件,模拟空目录
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("no skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := makeStore(t)
	out, err := ImportFromFolder(store, root)
	if !errors.Is(err, ErrNoSkillMD) {
		t.Fatalf("err = %v, want ErrNoSkillMD", err)
	}
	if out == nil || out.Found != 0 {
		t.Fatalf("out.Found = %v, want 0", out)
	}
}

// TestImportFromFolder_MultiLevel 多级目录(Claude marketplaces 风格)。
// 验证自动下钻:skills/foo + skills/bar 都识别。
func TestImportFromFolder_MultiLevel(t *testing.T) {
	root := t.TempDir()
	writeSkillMD(t, filepath.Join(root, "skills", "foo", "SKILL.md"), "foo", "foo description ok")
	writeSkillMD(t, filepath.Join(root, "skills", "bar", "SKILL.md"), "bar", "bar description ok")

	store := makeStore(t)
	out, err := ImportFromFolder(store, root)
	if err != nil {
		t.Fatalf("ImportFromFolder: %v", err)
	}
	if out.Found != 2 || out.OK != 2 || out.Failed != 0 {
		t.Fatalf("counts: Found=%d OK=%d Failed=%d; want 2/2/0", out.Found, out.OK, out.Failed)
	}
	names := []string{out.Results[0].Name, out.Results[1].Name}
	if !(names[0] == "bar" && names[1] == "foo") {
		t.Fatalf("names = %v, want [bar foo] (sorted)", names)
	}
}

// TestImportFromFolder_BadFrontmatter 某个 skill 的 SKILL.md 缺 frontmatter → 该条 fail,其它 OK。
func TestImportFromFolder_BadFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeSkillMD(t, filepath.Join(root, "good", "SKILL.md"), "good", "good description ok")
	// 写一个坏 SKILL.md(没 frontmatter)
	badDir := filepath.Join(root, "bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte("# no frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := makeStore(t)
	out, err := ImportFromFolder(store, root)
	if err != nil {
		t.Fatalf("ImportFromFolder: %v", err)
	}
	if out.Found != 2 || out.OK != 1 || out.Failed != 1 {
		t.Fatalf("counts: Found=%d OK=%d Failed=%d; want 2/1/1", out.Found, out.OK, out.Failed)
	}
}

// TestImportFromFolder_NestedFiles 加载附属 file(非 SKILL.md)。
func TestImportFromFolder_NestedFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSkillMD(t, filepath.Join(dir, "SKILL.md"), "alpha", "alpha description ok")
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "more.txt"), []byte("more body"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := makeStore(t)
	out, err := ImportFromFolder(store, root)
	if err != nil {
		t.Fatalf("ImportFromFolder: %v", err)
	}
	if out.OK != 1 {
		t.Fatalf("OK = %d, want 1", out.OK)
	}
	// 落盘后 store 里 alpha 应有 SKILL.md + extra.txt + sub/more.txt
	for _, rel := range []string{"SKILL.md", "extra.txt", filepath.Join("sub", "more.txt")} {
		if _, err := os.Stat(filepath.Join(store.Root(), "alpha", rel)); err != nil {
			t.Fatalf("alpha/%s missing: %v", rel, err)
		}
	}
}

// TestImportFromZipBytes_SingleSkill zip 内单个 skill。
func TestImportFromZipBytes_SingleSkill(t *testing.T) {
	buf := newZip(t, map[string]string{
		"alpha/SKILL.md": "---\nname: alpha\ndescription: alpha description ok\n---\n# alpha\n",
	})

	store := makeStore(t)
	out, err := ImportFromZipBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromZipBytes: %v", err)
	}
	if out.Found != 1 || out.OK != 1 {
		t.Fatalf("counts: Found=%d OK=%d; want 1/1", out.Found, out.OK)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "alpha", "SKILL.md")); err != nil {
		t.Fatalf("alpha not in store: %v", err)
	}
}

// TestImportFromZipBytes_NoSKILL zip 里没 SKILL.md → ErrNoSkillMD。
func TestImportFromZipBytes_NoSKILL(t *testing.T) {
	buf := newZip(t, map[string]string{
		"readme.txt": "nothing here",
	})

	store := makeStore(t)
	out, err := ImportFromZipBytes(store, buf.Bytes())
	if !errors.Is(err, ErrNoSkillMD) {
		t.Fatalf("err = %v, want ErrNoSkillMD", err)
	}
	if out == nil || out.Found != 0 {
		t.Fatalf("out.Found = %v, want 0", out)
	}
}

// TestImportFromZipBytes_MultiSkill zip 含多个 skill,全部导入。
func TestImportFromZipBytes_MultiSkill(t *testing.T) {
	buf := newZip(t, map[string]string{
		"skills/foo/SKILL.md": "---\nname: foo\ndescription: foo description ok\n---\n# foo\n",
		"skills/bar/SKILL.md": "---\nname: bar\ndescription: bar description ok\n---\n# bar\n",
		"skills/foo/extra.txt": "foo extra",
	})

	store := makeStore(t)
	out, err := ImportFromZipBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromZipBytes: %v", err)
	}
	if out.Found != 2 || out.OK != 2 {
		t.Fatalf("counts: Found=%d OK=%d; want 2/2", out.Found, out.OK)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "foo", "SKILL.md")); err != nil {
		t.Fatalf("foo missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "bar", "SKILL.md")); err != nil {
		t.Fatalf("bar missing: %v", err)
	}
}

// TestImportFromZipPath zip 从磁盘读。
func TestImportFromZipPath(t *testing.T) {
	buf := newZip(t, map[string]string{
		"alpha/SKILL.md": "---\nname: alpha\ndescription: alpha description ok\n---\n# alpha\n",
	})
	zipPath := filepath.Join(t.TempDir(), "skills.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	store := makeStore(t)
	out, err := ImportFromZipPath(store, zipPath)
	if err != nil {
		t.Fatalf("ImportFromZipPath: %v", err)
	}
	if out.OK != 1 || out.SourceKind != SourceZipPath {
		t.Fatalf("OK=%d Kind=%v; want 1/%v", out.OK, out.SourceKind, SourceZipPath)
	}
	if out.Source != zipPath {
		t.Fatalf("Source = %q, want %q", out.Source, zipPath)
	}
}

// TestImportFromZipBytes_ZipSlip 路径里有 .. 的 entry 被跳过。
func TestImportFromZipBytes_ZipSlip(t *testing.T) {
	// 手工构造一个含 ../ 的 entry + 正常 SKILL.md
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// 正常 SKILL.md
	w, _ := zw.Create("alpha/SKILL.md")
	_, _ = w.Write([]byte("---\nname: alpha\ndescription: alpha description ok\n---\n# alpha\n"))
	// 攻击性 entry:试图写到 ../../../etc/passwd(不合法,应该被 groupZipBySkillDir 过滤)
	w2, _ := zw.Create("../../../etc/passwd")
	_, _ = w2.Write([]byte("evil"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	store := makeStore(t)
	out, err := ImportFromZipBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromZipBytes: %v", err)
	}
	if out.Found != 1 || out.OK != 1 {
		t.Fatalf("counts: Found=%d OK=%d; want 1/1 (zip slip entry must be skipped)", out.Found, out.OK)
	}
}

// TestImportFromFolder_NotDir 给一个文件路径,返 error。
func TestImportFromFolder_NotDir(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := makeStore(t)
	_, err := ImportFromFolder(store, f)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("err = %v, want 'not a directory'", err)
	}
}

// newZip 构造一个 zip 字节流,entries 是 path→content。
func newZip(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for p, content := range entries {
		w, err := zw.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}