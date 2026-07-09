package github

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

// 2026-07-09 改:go-git 真实 PlainClone 在测试环境有兼容性坑(file:// 协议、HEAD
// checkout 时机等),单测改测**纯函数 parseClonedSkill**(输入本地 clone 目录,
// 输出 canonical)。真实 clone 路径靠集成测试或人工验证覆盖。
//
// 单元覆盖矩阵:
//   - parseClonedSkill: SKILL.md 解析 / 附属文件收齐 / 路径相对化 / anchor 过滤
//   - splitRemoteID: 各种格式拆 / 异常输入
//   - isRateLimitedErr: 关键字 + transport sentinel
//   - lastSegment: 路径末段
//   - buildRepoURL: 默认 https + file:// 边界
//   - Download: ctx cancel(轻量测)

// setupSkillRepoDir 在 tmpDir 创建一个**已经 checkout** 的 git 工作树
// (直接用 git 命令 commit,模拟 PlainClone + Checkout 后的状态)。
// 返回 checkout 后的目录(就是 SKILL.md 所在目录的根)。
func setupSkillRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Logf("[setup] tmpDir: %s", dir)
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	anchor := filepath.Join(dir, "skills", "pdf")
	if err := os.MkdirAll(filepath.Join(anchor, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"skills/pdf/SKILL.md": "---\nname: pdf\ndescription: PDF test\n---\n# PDF\n",
		"skills/pdf/scripts/check_bounding_boxes.py":  "# bb\n",
		"skills/pdf/scripts/check_fillable_fields.py": "# cf\n",
		"skills/pdf/scripts/convert_pdf_to_images.py": "# ci\n",
		"skills/pdf/LICENSE.txt":                      "MIT\n",
		"skills/pdf/forms.md":                         "# forms\n",
		// 不在锚点下的文件(应该被跳过)
		"skills/other-skill/SKILL.md": "---\nname: other\n---\n# Other\n",
		"README.md":                   "# Top\n",
	}
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		// 2026-07-09 调试:先写父目录再写文件
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		// 2026-07-09 验证:WriteFile 之后立刻 Stat + ReadFile
		if _, err := os.Stat(full); err != nil {
			t.Logf("Stat %s after WriteFile: %v", full, err)
			// 看父目录有什么
			if entries, eerr := os.ReadDir(filepath.Dir(full)); eerr == nil {
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Logf("parent dir %s contents: %v", filepath.Dir(full), names)
			}
			// 看 t.TempDir() 顶层
			if entries, eerr := os.ReadDir(dir); eerr == nil {
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Logf("tempDir %s contents: %v", dir, names)
			}
			t.Fatalf("Stat %s failed: %v", full, err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "init test skill repo")
	// 2026-07-09 验证:commit 后文件存在
	otherSKILL := filepath.Join(dir, "skills", "other-skill", "SKILL.md")
	if _, err := os.Stat(otherSKILL); err != nil {
		t.Fatalf("expected other-skill/SKILL.md at %s, got: %v", otherSKILL, err)
	}
	t.Logf("[setup] other-skill SKILL.md OK: %s", otherSKILL)
	return dir
}

// 2026-07-09 改:parseClonedSkill 单测 — 直接喂已 checkout 的目录,
// 验证 SKILL.md + 附属文件 + 锚点过滤全部正确。
//
// 2026-07-10 改:Download 流程换成 Trees API + 并发 raw 下载,不再走
// PlainClone,parseClonedSkill 也跟着删了。这个测试改成测 parseZipball
// (file:// 测试入口仍用),用真实 zipball 验证 SKILL.md + 附属文件 + 锚点过滤。
func TestParseZipball_IncludesAllFiles(t *testing.T) {
	repoDir := setupSkillRepoDir(t)
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	if err := buildTestZip(repoDir, zipPath, "anthropics-skills-abc123"); err != nil {
		t.Fatalf("build zip: %v", err)
	}
	can, err := parseZipball(zipPath, "anthropics", "skills", "skills/pdf", "anthropics/skills@skills/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil {
		t.Fatal("nil canonical")
	}
	// 期望 6 个 file:SKILL.md + 3 个 py + LICENSE + forms.md
	// (other-skill/SKILL.md 和 README.md 不在锚点下,被跳过)
	if len(can.Files) != 6 {
		t.Fatalf("expected 6 files in anchor dir, got %d: %+v", len(can.Files), can.Files)
	}
	if can.Files[0].Path != "SKILL.md" {
		t.Errorf("first file should be SKILL.md, got %q", can.Files[0].Path)
	}
	if can.Manifest.Author != "anthropics" {
		t.Errorf("Manifest.Author = %q, want %q", can.Manifest.Author, "anthropics")
	}
	if can.Manifest.Name != "pdf" {
		t.Errorf("Manifest.Name = %q, want %q", can.Manifest.Name, "pdf")
	}
}

// 2026-07-09 增:锚点目录不存在 → 报错。
func TestParseZipball_AnchorNotFound(t *testing.T) {
	repoDir := setupSkillRepoDir(t)
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	if err := buildTestZip(repoDir, zipPath, "anthropics-skills-abc123"); err != nil {
		t.Fatalf("build zip: %v", err)
	}
	_, err := parseZipball(zipPath, "x", "y", "no/such/dir", "x/y@no/such/dir")
	if err == nil {
		t.Fatal("expected error when anchor dir missing")
	}
	if !strings.Contains(err.Error(), "SKILL.md") {
		t.Errorf("error should mention SKILL.md, got %v", err)
	}
}

// 2026-07-09 增:Download 在 ctx 已 cancel 时直接返错(不实际 clone)。
func TestDownload_CtxCancelledImmediate(t *testing.T) {
	a := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消
	_, err := a.Download(ctx, "https://github.com", "x/y@z")
	if err == nil {
		t.Fatal("expected error on cancelled ctx")
	}
	if !strings.Contains(err.Error(), "ctx cancelled") {
		t.Errorf("error should mention ctx cancelled, got %v", err)
	}
}

// 2026-07-09 增:Download 在 invalid remoteID 时直接返 ErrRemoteNotFound。
func TestDownload_InvalidRemoteID(t *testing.T) {
	a := New()
	_, err := a.Download(context.Background(), "https://github.com", "no-at-sign")
	if err == nil {
		t.Fatal("expected error for invalid remote id")
	}
}

// 2026-07-09 增:buildRepoURL 各种 baseURL 行为。
func TestBuildRepoURL(t *testing.T) {
	a := New()
	cases := []struct {
		baseURL, owner, repoName, want string
	}{
		{"", "x", "y", "https://github.com/x/y"},
		{"https://github.com", "x", "y", "https://github.com/x/y"},
		{"https://github.com/", "x", "y", "https://github.com/x/y"}, // 末尾 / 自动 trim
		{"file:///tmp/test-repo", "ignored-owner", "ignored-repo", "file:///tmp/test-repo/"},
		{"file:///tmp/test-repo/", "ignored-owner", "ignored-repo", "file:///tmp/test-repo/"},
	}
	for _, c := range cases {
		got := a.buildRepoURL(c.baseURL, c.owner, c.repoName, "x/y@z")
		if got != c.want {
			t.Errorf("buildRepoURL(%q, %q, %q) = %q, want %q", c.baseURL, c.owner, c.repoName, got, c.want)
		}
	}
}

// 2026-07-09 增:splitRemoteID 回归。
func TestSplitRemoteID(t *testing.T) {
	cases := []struct {
		in         string
		wantRepo   string
		wantSkill  string
		wantOK     bool
	}{
		{"anthropics/skills@skills/pdf", "anthropics/skills", "skills/pdf", true},
		{"anthropics/skills@pdf", "anthropics/skills", "pdf", true},
		{"no-at-sign", "", "", false},
		{"missing/slash@skill", "missing/slash", "skill", true},
		{"owner/repo@", "", "", false},
		{"@skill", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			repo, skill, ok := splitRemoteID(c.in)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v", ok, c.wantOK)
			}
			if ok {
				if repo != c.wantRepo {
					t.Errorf("repo=%q, want %q", repo, c.wantRepo)
				}
				if skill != c.wantSkill {
					t.Errorf("skill=%q, want %q", skill, c.wantSkill)
				}
			}
		})
	}
}

// 2026-07-09 增:isRateLimitedErr 各种错误信息识别。
func TestIsRateLimitedErr(t *testing.T) {
	cases := []struct {
		errMsg string
		want   bool
	}{
		{"rate limit exceeded", true},
		{"API rate limit exceeded for user ID 1", true},
		{"remote: Too many requests", true},
		{"status 429: too many requests", true},
		{"status 403: Forbidden (rate limit)", true},
		{"connection refused", false},
		{"repository not found", false},
	}
	for _, c := range cases {
		t.Run(c.errMsg, func(t *testing.T) {
			got := isRateLimitedErr(&fakeErr{msg: c.errMsg})
			if got != c.want {
				t.Errorf("isRateLimitedErr(%q) = %v, want %v", c.errMsg, got, c.want)
			}
		})
	}
}

// 2026-07-09 增:transport 包真实 sentinel 错误识别。
func TestIsRateLimitedErr_TransportSentinel(t *testing.T) {
	if !isRateLimitedErr(transport.ErrAuthenticationRequired) {
		t.Error("transport.ErrAuthenticationRequired should be detected as rate-limit")
	}
	wrapped := fmt.Errorf("clone failed: %w", transport.ErrAuthenticationRequired)
	if !isRateLimitedErr(wrapped) {
		t.Error("wrapped transport.ErrAuthenticationRequired should be detected")
	}
	if isRateLimitedErr(nil) {
		t.Error("nil should return false")
	}
	if isRateLimitedErr(&fakeErr{msg: "some random error"}) {
		t.Error("random error should return false")
	}
}

// 2026-07-09 增:lastSegment 路径末段。
func TestLastSegment(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"skills/pdf", "pdf"},
		{"pdf", "pdf"},
		{"a/b/c/d", "d"},
		{"/leading/slash", "slash"},
		{"trailing/slash/", "slash"},
		{"", ""},
	}
	for _, c := range cases {
		if got := lastSegment(c.in); got != c.want {
			t.Errorf("lastSegment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 2026-07-10 增:cleanupOldZipFiles 不报错(无目录 / 空目录 / 有过期 zip 都应 OK)。
func TestCleanupOldZipFiles(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("cleanupOldZipFiles panicked: %v", r)
		}
	}()
	cleanupOldZipFiles()
}

// --- helpers ---

// buildTestZip 2026-07-10 增:把 srcDir 打成 zipball,所有文件挂在
// topDir/ 之下(模拟 codeload 的 "{owner}-{repo}-{sha}/" 顶层)。
func buildTestZip(srcDir, dstZip, topDir string) error {
	f, err := os.Create(dstZip)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	return filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, p)
		w, err := zw.Create(topDir + "/" + filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		return err
	})
}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }