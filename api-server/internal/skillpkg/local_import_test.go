package skillpkg

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skillstore"

	bzip2writer "github.com/dsnet/compress/bzip2"
	xz "github.com/ulikunitz/xz"
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

// TestImportFromFolder_BadFrontmatter 某个 skill 的 SKILL.md 缺 frontmatter 但有 H1 →
// 2026-07-10 起 ParseSkillMD 降级到 H1 取 name,所以仍能成功。改测"完全没 H1 也没
// frontmatter"的情况,确保这种真的 fail。
func TestImportFromFolder_BadFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeSkillMD(t, filepath.Join(root, "good", "SKILL.md"), "good", "good description ok")
	// 写一个坏 SKILL.md(没 frontmatter + 也没 H1 → 真的解析不出来)
	badDir := filepath.Join(root, "bad")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "SKILL.md"), []byte("body without frontmatter and no h1\nnot a markdown header\n"), 0o644); err != nil {
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

// 2026-07-11 增:tar 系列测试 + helper。
//
// 覆盖范围:.tar / .tar.gz / .tgz / .tar.bz2 / .tbz2 / .tar.xz / .txz。

// newTar 构造一个纯 tar 字节流。
func newTar(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for p, content := range entries {
		hdr := &tar.Header{
			Name: p,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

// newTarGz 构造一个 tar.gz 字节流。
func newTarGz(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	tarBuf := newTar(t, entries)
	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	if _, err := io.Copy(gw, tarBuf); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return &gzBuf
}

// newTarBz2 构造一个 tar.bz2 字节流。
//
// 2026-07-11:Go 标准库 compress/bzip2 只提供 NewReader(解压),不提供 writer。
// 这里用 dsnet/compress 的 bzip2 writer 做测试构造,production 端解压仍走
// 标准库的 compress/bzip2.Reader。
func newTarBz2(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	tarBuf := newTar(t, entries)
	var bzBuf bytes.Buffer
	bw, err := bzip2writer.NewWriter(&bzBuf, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(bw, tarBuf); err != nil {
		t.Fatal(err)
	}
	if err := bw.Close(); err != nil {
		t.Fatal(err)
	}
	return &bzBuf
}

// newTarXz 构造一个 tar.xz 字节流。
func newTarXz(t *testing.T, entries map[string]string) *bytes.Buffer {
	t.Helper()
	tarBuf := newTar(t, entries)
	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(xw, tarBuf); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return &xzBuf
}

// TestImportFromArchiveBytes_Tar 纯 tar 包。
func TestImportFromArchiveBytes_Tar(t *testing.T) {
	buf := newTar(t, map[string]string{
		"alpha/SKILL.md": "---\nname: alpha\ndescription: alpha description ok\n---\n# alpha\n",
	})

	store := makeStore(t)
	out, err := ImportFromArchiveBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromArchiveBytes: %v", err)
	}
	if out.Found != 1 || out.OK != 1 {
		t.Fatalf("counts: Found=%d OK=%d; want 1/1", out.Found, out.OK)
	}
}

// TestImportFromArchiveBytes_TarGz tar.gz / .tgz。
func TestImportFromArchiveBytes_TarGz(t *testing.T) {
	buf := newTarGz(t, map[string]string{
		"skills/foo/SKILL.md": "---\nname: foo\ndescription: foo description ok\n---\n# foo\n",
		"skills/bar/SKILL.md": "---\nname: bar\ndescription: bar description ok\n---\n# bar\n",
	})

	store := makeStore(t)
	out, err := ImportFromArchiveBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromArchiveBytes: %v", err)
	}
	if out.Found != 2 || out.OK != 2 {
		t.Fatalf("counts: Found=%d OK=%d; want 2/2", out.Found, out.OK)
	}
}

// TestImportFromArchiveBytes_TarBz2 tar.bz2 / .tbz2。
func TestImportFromArchiveBytes_TarBz2(t *testing.T) {
	buf := newTarBz2(t, map[string]string{
		"beta/SKILL.md": "---\nname: beta\ndescription: beta description ok\n---\n# beta\n",
	})

	store := makeStore(t)
	out, err := ImportFromArchiveBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromArchiveBytes: %v", err)
	}
	if out.OK != 1 {
		t.Fatalf("OK=%d, want 1", out.OK)
	}
}

// TestImportFromArchiveBytes_TarXz tar.xz / .txz。
func TestImportFromArchiveBytes_TarXz(t *testing.T) {
	buf := newTarXz(t, map[string]string{
		"gamma/SKILL.md":  "---\nname: gamma\ndescription: gamma description ok\n---\n# gamma\n",
		"gamma/extra.txt": "extra body",
	})

	store := makeStore(t)
	out, err := ImportFromArchiveBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromArchiveBytes: %v", err)
	}
	if out.OK != 1 {
		t.Fatalf("OK=%d, want 1", out.OK)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "gamma", "extra.txt")); err != nil {
		t.Fatalf("gamma/extra.txt missing: %v", err)
	}
}

// TestImportFromArchiveBytes_Unsupported 不支持的格式(.rar / .7z)→ ErrUnsupportedArchive。
func TestImportFromArchiveBytes_Unsupported(t *testing.T) {
	fake7z := append([]byte{0x37, 0x7a, 0xbc, 0xaf, 0x27, 0x1c}, []byte("7z fake content")...)

	store := makeStore(t)
	_, err := ImportFromArchiveBytes(store, fake7z)
	if !errors.Is(err, ErrUnsupportedArchive) {
		t.Fatalf("err = %v, want ErrUnsupportedArchive", err)
	}
}

// TestDetectArchiveKind 单元测试格式检测。
func TestDetectArchiveKind(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want archiveKind
	}{
		{"zip magic", zipMagic, archiveZip},
		{"gzip magic", gzipMagic, archiveTarGz},
		{"bzip2 magic", bzip2Magic, archiveTarBz2},
		{"xz magic", xzMagic, archiveTarXz},
		{"unknown", []byte("random data without any magic"), archiveUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectArchiveKind(tt.data, "unknown.bin")
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestImportFromArchiveBytes_WindowsBackslashZip 2026-07-11 增:
//
// Windows 上(尤其右键压缩)生成的 zip 用反斜杠做路径分隔符,如果不解码,
// path.Base/Dir 会把整个路径当 basename,导致 SKILL.md 检测失败。
// 真实用户案例:deepwhite-screenwriting-v1-complete-*.zip 导入报"未找到 SKILL.md"。
func TestImportFromArchiveBytes_WindowsBackslashZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// 用反斜杠模拟 Windows 风格 zip
	w, _ := zw.Create(`deepwhite-screenwriting-v1\SKILL.md`)
	_, _ = w.Write([]byte("---\nname: screenwriting\ndescription: screenwriting skill\n---\n# body\n"))
	w2, _ := zw.Create(`deepwhite-screenwriting-v1\references\engine.md`)
	_, _ = w2.Write([]byte("engine body"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	store := makeStore(t)
	out, err := ImportFromArchiveBytes(store, buf.Bytes())
	if err != nil {
		t.Fatalf("ImportFromArchiveBytes: %v", err)
	}
	if out.Found != 1 || out.OK != 1 {
		t.Fatalf("counts: Found=%d OK=%d; want 1/1", out.Found, out.OK)
	}
	// 落盘后 store 里 screenwriting 应有 SKILL.md + references/engine.md
	if _, err := os.Stat(filepath.Join(store.Root(), "screenwriting", "SKILL.md")); err != nil {
		t.Fatalf("screenwriting/SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "screenwriting", "references", "engine.md")); err != nil {
		t.Fatalf("screenwriting/references/engine.md missing: %v", err)
	}
}

// TestNormalizeArchivePath 2026-07-11 增:单元测试路径归一化。
func TestNormalizeArchivePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`a\b\c.md`, "a/b/c.md"},
		{`a\\b.md`, "a/b.md"},
		{`a//b///c.md`, "a/b/c.md"},
		{`a/b/c.md`, "a/b/c.md"},
	}
	for _, tt := range tests {
		got := normalizeArchivePath(tt.in)
		if got != tt.want {
			t.Errorf("normalizeArchivePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}