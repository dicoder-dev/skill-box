package github

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// 2026-07-09 增:Download 走 zipball,验证 SKILL.md + 锚点目录所有文件都收。
//
// 2026-07-09 改(关键 bug):早期 raw URL 实现只下一个 SKILL.md,用户截图显示
// pdf 仓库 scripts/ 下有 9 个 .py + LICENSE + reference.md 全部丢失。
// 现在改走 codeload.github.com/{owner}/{repo}/zipball/{branch},解压后
// 收 SKILL.md 所在目录所有 file。
func TestDownload_Zipball_IncludesAllFiles(t *testing.T) {
	zipBytes := buildZipballZip(t, "anthropics-skills-abc1234", map[string]string{
		"skills/pdf/SKILL.md":                          "---\nname: pdf\ndescription: PDF\n---\n# PDF\n",
		"skills/pdf/scripts/check_bounding_boxes.py":   "# bb\n",
		"skills/pdf/scripts/check_fillable_fields.py":  "# cf\n",
		"skills/pdf/scripts/convert_pdf_to_images.py":  "# ci\n",
		"skills/pdf/LICENSE.txt":                       "MIT\n",
		"skills/pdf/forms.md":                          "# forms\n",
		"skills/pdf/reference.md":                      "# ref\n",
		// 不在锚点目录下的文件应被跳过(避免把整仓库都装进来)
		"skills/other-skill/SKILL.md":                  "---\nname: other\n---\n# Other\n",
		"README.md":                                    "# Top-level\n",
	})

	zipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))
		w.WriteHeader(http.StatusOK)
		w.Write(zipBytes)
	}))
	defer zipServer.Close()

	oldBase := defaultZipballBase
	defaultZipballBase = zipServer.URL
	t.Cleanup(func() { defaultZipballBase = oldBase })

	a := New()
	can, err := a.Download(context.Background(), "https://github.com", "anthropics/skills@skills/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil {
		t.Fatal("nil canonical")
	}
	// 期望 7 个 file:SKILL.md + 5 个 scripts/LICENSE + 2 个 md
	// (other-skill/SKILL.md 和 README.md 不在锚点下,被跳过)
	if len(can.Files) != 7 {
		t.Fatalf("expected 7 files in anchor dir, got %d: %+v", len(can.Files), can.Files)
	}
	if can.Files[0].Path != "SKILL.md" {
		t.Errorf("first file should be SKILL.md, got %q", can.Files[0].Path)
	}
	// author 应为 owner(anthropics),不是写死 "GitHub"
	if can.Manifest.Author != "anthropics" {
		t.Errorf("Manifest.Author = %q, want %q", can.Manifest.Author, "anthropics")
	}
	// name 兜底取 lastSegment("skills/pdf") = "pdf"
	if can.Manifest.Name != "pdf" {
		t.Errorf("Manifest.Name = %q, want %q", can.Manifest.Name, "pdf")
	}
	// 不能含其它 skill / 顶层 README 的内容
	for _, f := range can.Files {
		if f.Path == "README.md" {
			t.Errorf("leaked top-level file: %q", f.Path)
		}
	}
}

// TestDownload_Zipball_MainNotFound_FallbackMaster 2026-07-09 增:main 404 → 试 master。
func TestDownload_Zipball_MainNotFound_FallbackMaster(t *testing.T) {
	var mainHit, masterHit int
	zipBytes := buildZipballZip(t, "anthropics-skills-abc", map[string]string{
		"skills/pdf/SKILL.md": "---\nname: pdf\n---\n# PDF\n",
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/anthropics/skills/zipball/main", func(w http.ResponseWriter, r *http.Request) {
		mainHit++
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/anthropics/skills/zipball/master", func(w http.ResponseWriter, r *http.Request) {
		masterHit++
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipBytes)
	})
	router := httptest.NewServer(mux)
	defer router.Close()

	oldBase := defaultZipballBase
	defaultZipballBase = router.URL
	t.Cleanup(func() { defaultZipballBase = oldBase })

	a := New()
	can, err := a.Download(context.Background(), "https://github.com", "anthropics/skills@skills/pdf")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil || can.Manifest.Name != "pdf" {
		t.Fatalf("expected pdf canonical from master branch, got %+v", can)
	}
	if mainHit != 1 {
		t.Errorf("main branch should be tried once, got %d", mainHit)
	}
	if masterHit != 1 {
		t.Errorf("master branch should be tried once, got %d", masterHit)
	}
}

// TestSplitRemoteID 2026-07-09 增:基本 split 回归。
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
		{"missing/slash@skill", "missing/slash", "skill", true}, // 实际是合法的 owner/skill,不是缺 slash
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

// buildZipballZip 构造模拟 codeload.github.com 的 zipball 响应。
// wrapperDir 是顶层包裹目录(真实 zipball 形如 "{owner}-{repo}-{commit_sha}")。
func buildZipballZip(t *testing.T, wrapperDir string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for relPath, content := range files {
		fullPath := wrapperDir + "/" + relPath
		f, err := w.Create(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(f, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}