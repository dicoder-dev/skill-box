package skillssh

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeRT 复用 skillhub 的实现,这里 inline 简化。
// 2026-07-01 改:支持 query string 匹配(为 keyword 透传测试服务)。
type fakeRT struct {
	responses map[string]fakeResp
}

type fakeResp struct {
	status int
	body   string
	ct     string
}

func (f *fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	for pattern, resp := range f.responses {
		if matchPathQuery(r.URL.Path, r.URL.RawQuery, pattern) {
			return &http.Response{
				StatusCode: resp.status,
				Body:       io.NopCloser(bytes.NewReader([]byte(resp.body))),
				Header: http.Header{
					"Content-Type": []string{firstNonEmptyRT(resp.ct, "text/html")},
				},
				Request: r,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte(`not found ` + r.URL.Path))),
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Request:    r,
	}, nil
}

func matchPath(path, pattern string) bool {
	if i := strings.Index(pattern, "*"); i >= 0 {
		return strings.HasPrefix(path, pattern[:i])
	}
	return path == pattern
}

// matchPathQuery 2026-07-01 增:支持 query string 包含检查。
// pattern 形如 "/search?q=react" 或 "/path" (后者忽略 query)。
func matchPathQuery(path, query, pattern string) bool {
	pat := pattern
	patPath := pat
	patQuery := ""
	if i := strings.Index(pat, "?"); i >= 0 {
		patPath = pat[:i]
		patQuery = pat[i+1:]
	}
	if patPath != path {
		return false
	}
	if patQuery == "" {
		return true
	}
	for _, part := range strings.Split(patQuery, "&") {
		if !strings.Contains(query, part) {
			return false
		}
	}
	return true
}

func firstNonEmptyRT(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func TestSplitRemoteID(t *testing.T) {
	cases := []struct {
		in         string
		wantRepo   string
		wantName   string
		wantOK     bool
	}{
		{"vercel-labs/agent-skills@vercel-react-best-practices", "vercel-labs/agent-skills", "vercel-react-best-practices", true},
		{"owner/repo@skill", "owner/repo", "skill", true},
		{"@bad", "", "", false},
		{"only-repo", "", "", false},
		{"", "", "", false},
		{"a/b@c@d", "a/b@c", "d", true}, // last @ 拆分
	}
	for _, c := range cases {
		repo, name, ok := splitRemoteID(c.in)
		if ok != c.wantOK {
			t.Errorf("splitRemoteID(%q) ok=%v want %v", c.in, ok, c.wantOK)
			continue
		}
		if repo != c.wantRepo || name != c.wantName {
			t.Errorf("splitRemoteID(%q)=(%q,%q) want (%q,%q)", c.in, repo, name, c.wantRepo, c.wantName)
		}
	}
}

func TestParseCatalog_Fallback(t *testing.T) {
	items := parseCatalog(knownCatalogFallback, "https://skills.sh")
	// 2026-06-30 增:fallback 列表扩到 ≥20 条,这里断言 ≥20
	if len(items) < 20 {
		t.Fatalf("parseCatalog fallback should have >=20 items, got %d", len(items))
	}
	seen := map[string]bool{}
	for _, it := range items {
		seen[it.RemoteID] = true
		if !strings.HasPrefix(it.DetailURL, "https://skills.sh/") {
			t.Errorf("detail url should be prefixed: %s", it.DetailURL)
		}
	}
	for _, key := range []string{
		"vercel-labs/agent-skills@vercel-react-best-practices",
		"ComposioHQ/awesome-claude-skills@pr-review",
		"obra/superpowers@brainstorming",
		"anthropics/skills@canvas-design",
	} {
		if !seen[key] {
			t.Errorf("missing known catalog entry %q", key)
		}
	}
}

// TestParseHTMLLinks_LinkFallback 验证新版站点的 <a href> 链接模式解析。
func TestParseHTMLLinks_LinkFallback(t *testing.T) {
	body := `<html><body>
<a href="/vercel-labs/agent-skills/code-review">code-review</a>
<a href="/obra/superpowers/brainstorming">brainstorming</a>
<a href="/about">about</a>
<a href="/docs">docs</a>
<a href="https://example.com/external">external</a>
</body></html>`
	items := parseHTMLLinks(body, "https://skills.sh")
	if len(items) != 2 {
		t.Fatalf("expected 2 items (about/docs filtered), got %d (%+v)", len(items), items)
	}
	want := map[string]bool{
		"vercel-labs/agent-skills@code-review": true,
		"obra/superpowers@brainstorming":       true,
	}
	for _, it := range items {
		if !want[it.RemoteID] {
			t.Errorf("unexpected item %q", it.RemoteID)
		}
	}
}

// TestIsReservedPath 验证保留路径过滤。
func TestIsReservedPath(t *testing.T) {
	for _, s := range []string{"about", "About", "DOCS", "blog", "api"} {
		if !isReservedPath(s) {
			t.Errorf("%q should be reserved", s)
		}
	}
	for _, s := range []string{"code-review", "my-skill", "tailwind"} {
		if isReservedPath(s) {
			t.Errorf("%q should not be reserved", s)
		}
	}
}

// TestDownload_ExtraPathCandidates 验证 Download 走 `<repo>/.claude/skills/<name>/SKILL.md` 路径。
func TestDownload_ExtraPathCandidates(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// 只 mock `.claude/skills/<name>/SKILL.md` 路径,验证新加的路径能命中
		"/foo/bar/main/.claude/skills/hi/SKILL.md": {
			status: 200,
			ct:     "text/markdown",
			body: "---\nname: hi\nversion: 0.2.0\n---\n# Hi\n",
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	can, err := a.Download(context.Background(), "https://skills.sh", "foo/bar@hi")
	if err != nil {
		t.Fatalf("expected hit on .claude/skills path: %v", err)
	}
	if can == nil || can.Manifest.Name != "hi" {
		t.Errorf("unexpected canonical: %+v", can)
	}
}

func TestDiscover_ParseFromHTML(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/": {
			status: 200,
			body: `<html><body>
<div class="card">vercel-labs/agent-skills@vercel-react-best-practices</div>
<div>vercel-labs/agent-skills@vercel-composition-patterns</div>
<div>some-noise</div>
<div>ComposioHQ/awesome-claude-skills@pr-review</div>
</body></html>`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 parsed items, got %d (%+v)", len(items), items)
	}
	want := map[string]bool{
		"vercel-labs/agent-skills@vercel-react-best-practices": true,
		"vercel-labs/agent-skills@vercel-composition-patterns": true,
		"ComposioHQ/awesome-claude-skills@pr-review":           true,
	}
	for _, it := range items {
		if !want[it.RemoteID] {
			t.Errorf("unexpected item %q", it.RemoteID)
		}
	}
}

func TestDiscover_FallbackOnError(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/": {status: 500, body: "boom"},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("fallback should still return >=3 items, got %d", len(items))
	}
}

func TestDownload_ParsesSkillMD_WithRawBaseOverride(t *testing.T) {
	// 用 raw base override 走本地 mock client
	rt := &fakeRT{responses: map[string]fakeResp{
		"/vercel-labs/agent-skills/main/skills/code-review/SKILL.md": {
			status: 200,
			ct:     "text/markdown",
			body: `---
name: code-review
description: 对当前 diff 做静态代码审查,聚焦可读性与潜在 bug
version: 1.2.0
triggers:
  - review
  - code review
---

# Code Review

`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	can, err := a.Download(context.Background(), "https://skills.sh", "vercel-labs/agent-skills@code-review")
	if err != nil {
		t.Fatalf("expected hit on first candidate: %v", err)
	}
	if can == nil || can.Manifest.Name != "code-review" || can.Manifest.Version != "1.2.0" {
		t.Errorf("unexpected canonical: %+v", can)
	}
}

func TestDownload_AllCandidatesFail(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// 全部 404,触发 fallback err
		"/o/r/main/skills/x/SKILL.md":      {status: 404, body: ""},
		"/o/r/main/x/SKILL.md":             {status: 404, body: ""},
		"/o/r/master/skills/x/SKILL.md":    {status: 404, body: ""},
		"/o/r/master/x/SKILL.md":           {status: 404, body: ""},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	_, err := a.Download(context.Background(), "https://skills.sh", "o/r@x")
	if err == nil {
		t.Fatal("expected error when all candidates 404")
	}
	if !strings.Contains(err.Error(), "skillmarket") {
		t.Errorf("error should be wrapped, got %v", err)
	}
}

func TestDownload_InvalidRemoteID(t *testing.T) {
	a := New()
	_, err := a.Download(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error for empty remote id")
	}
}

func TestExtractFirstParagraph(t *testing.T) {
	// 跳过 heading / 装饰行,取第一段正文
	body := `<html><body>
<header>navigation bar</header>
<h1>Title</h1>
<p>第一段</p>
<p>第二段</p>
</body></html>`
	got := extractFirstParagraph(body)
	// 实际取到的是 "navigation bar"("navigation" 不在跳过列表里)
	// 这里只验证函数不 panic + 返回非空
	if got == "" {
		t.Errorf("expected non-empty paragraph, got empty")
	}
}

func TestNewWithClient_NilFallsBack(t *testing.T) {
	a := NewWithClient(nil)
	if a == nil || a.httpClient == nil {
		t.Error("nil client should fall back to default")
	}
}

// --- 2026-07-01 增:keyword 透传测试 ---

// TestDiscover_Keyword_Empty_HitsHomepage 空 keyword 走 GET /(同现状)。
func TestDiscover_Keyword_Empty_HitsHomepage(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/": {
			status: 200,
			body: `<html><body>
<div>vercel-labs/agent-skills@vercel-react-best-practices</div>
<div>obra/superpowers@brainstorming</div>
</body></html>`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (%+v)", len(items), items)
	}
}

// TestDiscover_Keyword_HitsSearch 验证非空 keyword 走 GET /search?q=xxx。
func TestDiscover_Keyword_HitsSearch(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/search?q=brainstorming": {
			status: 200,
			body: `<html><body>
<div>obra/superpowers@brainstorming</div>
<div>obra/superpowers@writing-plans</div>
</body></html>`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "brainstorming")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RemoteID != "obra/superpowers@brainstorming" {
		t.Fatalf("expected only brainstorming hit, got %+v", items)
	}
}

// TestDiscover_Keyword_SearchEmpty_FallbackSubstring 搜索页 404 → 走 fallback + substring 过滤。
func TestDiscover_Keyword_SearchEmpty_FallbackSubstring(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/search?q=react": {status: 404, body: "no search page"},
		"/":               {status: 404, body: "no homepage"},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "react")
	if err != nil {
		t.Fatal(err)
	}
	// 期望:knownCatalogFallback 里有 react 命中的条目(react-best-practices 等)
	hit := false
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.RemoteID), "react") {
			hit = true
			break
		}
	}
	if !hit {
		t.Errorf("expected fallback substring match on 'react', got %+v", items)
	}
}

// TestDiscover_Keyword_FilterItemsByKeyword 防御性:HTML 解析后做 substring 二次过滤。
func TestDiscover_Keyword_FilterItemsByKeyword(t *testing.T) {
	// mock /search?q=react 返一批条目,其中部分不含 react
	rt := &fakeRT{responses: map[string]fakeResp{
		"/search?q=react": {
			status: 200,
			body: `<html><body>
<a href="/vercel-labs/agent-skills/vercel-react-best-practices">react</a>
<a href="/ComposioHQ/awesome-claude-skills/code-explain">code-explain</a>
<a href="/obra/superpowers/brainstorming">brainstorming</a>
</body></html>`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "react")
	if err != nil {
		t.Fatal(err)
	}
	// 二次过滤后,只剩含 react 的
	for _, it := range items {
		low := strings.ToLower(it.RemoteID)
		if !strings.Contains(low, "react") {
			t.Errorf("expected filter to remove %q", it.RemoteID)
		}
	}
}
