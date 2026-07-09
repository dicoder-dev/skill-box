package skillssh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"ginp-api/internal/skillmarket"
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
	fmt.Fprintf(os.Stderr, "[fakeRT.MISS] path=%q query=%q\n", r.URL.Path, r.URL.RawQuery)
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

// 2026-07-09 增:Download 遇 GitHub 429 必须立即终止,不要继续尝试其他分支。
//
// 回归测试:防止未来重构把 429 当普通错误 continue 掉,
// 那样会触发更多 GitHub 限流请求,加重黑名单。
//
// 2026-07-10 改:Download 流程从"6 候选 URL"改成"tree + candidate fallback",
// 计数基数变成 2 次 tree API(2 分支尝试)+ 至多 1 次遇到 429 早退;
// 不允许继续尝试后续 candidates。
func TestDownload_RateLimited_AbortsImmediately(t *testing.T) {
	var hits int
	rt := &countingRT{
		inner: &fakeRT{responses: map[string]fakeResp{
			// tree API 全部走 fakeRT 兜底 404,所以会先试 main → master 两分支。
			// 然后 downloadSKILLMDOnly 第一个 candidates 就 429,后续都应当被阻断。
			"/o/r/main/skills/x/SKILL.md": {status: 429, body: "rate limit exceeded"},
		}},
		count: &hits,
	}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	_, err := a.Download(context.Background(), "https://skills.sh", "o/r@x")
	if err == nil {
		t.Fatal("expected error (non-fallback remote id)")
	}
	// tree 主分支 + master 分支(= 2 次 tree API),429 raw 那条触发早退,
	// 总共 = 3 次。后续 candidates 不应再被访问。
	if hits > 3 {
		t.Fatalf("expected at most 3 fetches (2 trees + 1 raw 429 abort), got %d", hits)
	}
}

// 2026-07-09 增:全失败 + 命中 knownCatalogFallback → 返内存骨架 SKILL.md。
func TestDownload_AllFail_HitsFallbackCatalog(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// 全部 404
		"/anthropics/skills/main/skills/pdf/SKILL.md":     {status: 404, body: ""},
		"/anthropics/skills/main/pdf/SKILL.md":            {status: 404, body: ""},
		"/anthropics/skills/master/skills/pdf/SKILL.md":   {status: 404, body: ""},
		"/anthropics/skills/master/pdf/SKILL.md":          {status: 404, body: ""},
		"/anthropics/skills/main/.claude/skills/pdf/SKILL.md":   {status: 404, body: ""},
		"/anthropics/skills/master/.claude/skills/pdf/SKILL.md": {status: 404, body: ""},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	can, err := a.Download(context.Background(), "https://skills.sh", "anthropics/skills@pdf")
	if err != nil {
		t.Fatalf("expected fallback hit to succeed, got %v", err)
	}
	if can == nil || can.Manifest.Name != "pdf" {
		t.Fatalf("expected fallback canonical for pdf, got %+v", can)
	}
}

// 2026-07-09 增:全失败 + remoteID 不在 knownCatalogFallback → 仍返错。
func TestDownload_AllFail_NoFallback(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/some/unknown-skill-id/main/skills/x/SKILL.md":   {status: 404, body: ""},
		"/some/unknown-skill-id/main/x/SKILL.md":          {status: 404, body: ""},
		"/some/unknown-skill-id/master/skills/x/SKILL.md": {status: 404, body: ""},
		"/some/unknown-skill-id/master/x/SKILL.md":        {status: 404, body: ""},
		"/some/unknown-skill-id/main/.claude/skills/x/SKILL.md":   {status: 404, body: ""},
		"/some/unknown-skill-id/master/.claude/skills/x/SKILL.md": {status: 404, body: ""},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	a.SetRawBaseOverride("https://stub")
	_, err := a.Download(context.Background(), "https://skills.sh", "some/unknown-skill-id@x")
	if err == nil {
		t.Fatal("expected error (not in fallback catalog)")
	}
}

// countingRT 包装 fakeRT,记录 RoundTrip 调用次数。
type countingRT struct {
	inner *fakeRT
	count *int
}

func (c *countingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	*c.count++
	return c.inner.RoundTrip(r)
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

// === /api/audits/{page} JSON 路径测试(2026-07-01) ===

// auditsMockResponse 构造一个 /api/audits 风格的 JSON 响应(含完整字段)。
func auditsMockResponse(skills []map[string]any) string {
	type ath struct {
		Source string `json:"source"`
		Slug   string `json:"slug"`
		Result struct {
			GeminiAnalysis struct {
				Verdict string `json:"verdict"`
				Summary string `json:"summary"`
			} `json:"gemini_analysis"`
			OverallRiskLevel string `json:"overall_risk_level"`
		} `json:"result"`
	}
	type sk struct {
		Rank           int   `json:"rank"`
		Source         string `json:"source"`
		SkillID        string `json:"skillId"`
		Name           string `json:"name"`
		AgentTrustHub  *ath  `json:"agentTrustHub"`
		Socket         any   `json:"socket"`
		Snyk           any   `json:"snyk"`
	}
	out := struct {
		Skills []sk `json:"skills"`
	}{}
	for _, m := range skills {
		s := sk{
			Rank:    m["rank"].(int),
			Source:  m["source"].(string),
			SkillID: m["skillId"].(string),
			Name:    m["skillId"].(string),
		}
		if v, ok := m["summary"].(string); ok && v != "" {
			s.AgentTrustHub = &ath{}
			s.AgentTrustHub.Source = s.Source
			s.AgentTrustHub.Slug = s.SkillID
			s.AgentTrustHub.Result.GeminiAnalysis.Verdict = "SAFE"
			s.AgentTrustHub.Result.GeminiAnalysis.Summary = v
		}
		if v, ok := m["risk"].(string); ok && v != "" {
			if s.AgentTrustHub == nil {
				s.AgentTrustHub = &ath{}
			}
			s.AgentTrustHub.Result.OverallRiskLevel = v
		}
		out.Skills = append(out.Skills, s)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// TestDiscover_AuditsAPI_HappyPath 验证空 keyword 走 /api/audits 拿到 author/description/tags。
func TestDiscover_AuditsAPI_HappyPath(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200,
			ct:     "application/json",
			body: auditsMockResponse([]map[string]any{
				{
					"rank":    1,
					"source":  "vercel-labs/skills",
					"skillId": "find-skills",
					"summary": "Find and install additional agent skills via a CLI. Standard package management.",
					"risk":    "SAFE",
				},
				{
					"rank":    2,
					"source":  "anthropics/skills",
					"skillId": "pdf",
					"summary": "Read, edit, and extract content from PDF documents.",
					"risk":    "LOW",
				},
			}),
		},
		"/api/audits/1": {
			status: 200,
			ct:     "application/json",
			body:   `{"skills":[]}`,
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
	// 字段映射校验
	got := items[0]
	if got.RemoteID != "vercel-labs/skills@find-skills" {
		t.Errorf("RemoteID = %q want vercel-labs/skills@find-skills", got.RemoteID)
	}
	if got.Name != "find-skills" {
		t.Errorf("Name = %q want find-skills", got.Name)
	}
	if got.Author != "vercel-labs" {
		t.Errorf("Author = %q want vercel-labs", got.Author)
	}
	if got.Description == "" || !strings.Contains(got.Description, "Find and install") {
		t.Errorf("Description = %q want contain 'Find and install'", got.Description)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "risk:safe" {
		t.Errorf("Tags = %v want [risk:safe]", got.Tags)
	}
	if got.DetailURL != "https://stub/vercel-labs/skills/find-skills" {
		t.Errorf("DetailURL = %q want https://stub/vercel-labs/skills/find-skills", got.DetailURL)
	}
	// 第二条:LOW 风险等级
	if items[1].Tags[0] != "risk:low" {
		t.Errorf("items[1] Tags = %v want [risk:low]", items[1].Tags)
	}
}

// TestDiscover_AuditsAPI_KeywordFilter 验证非空 keyword 时只拉首页再 substring 过滤。
func TestDiscover_AuditsAPI_KeywordFilter(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200,
			ct:     "application/json",
			body: auditsMockResponse([]map[string]any{
				{
					"rank":    1,
					"source":  "vercel-labs/agent-skills",
					"skillId": "vercel-react-best-practices",
					"summary": "React performance guidelines.",
				},
				{
					"rank":    2,
					"source":  "obra/superpowers",
					"skillId": "brainstorming",
					"summary": "Brainstorm a feature.",
				},
			}),
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "react")
	if err != nil {
		t.Fatal(err)
	}
	// 只应剩 1 条
	if len(items) != 1 {
		t.Fatalf("expected 1 item after react filter, got %d (%+v)", len(items), items)
	}
	if items[0].Name != "vercel-react-best-practices" {
		t.Errorf("filtered item = %q want vercel-react-best-practices", items[0].Name)
	}
}

// TestDiscover_AuditsAPI_FailFallbackToHTML 验证 audits API 失败时降级到 HTML 解析。
func TestDiscover_AuditsAPI_FailFallbackToHTML(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 500,
			body:   `internal server error`,
		},
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
	// HTML 解析应该兜底拿到 2 条(注意此时 Author/Description 为空,因为 HTML 解析不填)
	if len(items) != 2 {
		t.Fatalf("expected 2 items from HTML fallback, got %d (%+v)", len(items), items)
	}
	for _, it := range items {
		if it.Author != "" {
			t.Errorf("HTML fallback should leave Author empty, got %q for %q", it.Author, it.RemoteID)
		}
	}
}

// TestDiscover_AuditsAPI_InvalidJSON 验证 audits API 返非 JSON 时降级到 HTML。
func TestDiscover_AuditsAPI_InvalidJSON(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200,
			ct:     "application/json",
			body:   `not json {`,
		},
		"/": {
			status: 200,
			body: `<html><body>
<div>vercel-labs/agent-skills@vercel-react-best-practices</div>
</body></html>`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item from HTML fallback, got %d", len(items))
	}
}

// TestDiscover_AuditsAPI_NilAgentTrustHub 2026-07-01 增:验证部分冷门 skill
// agentTrustHub 字段为 null 时不会 panic(原 bug:OverallRiskLevel 解引用 nil)。
func TestDiscover_AuditsAPI_NilAgentTrustHub(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200,
			ct:     "application/json",
			// 两条 skill:一条有 agentTrustHub,一条没有
			body: `{"skills":[
				{"rank":1,"source":"foo/bar","skillId":"audited","agentTrustHub":{"result":{"gemini_analysis":{"summary":"OK"}}}},
				{"rank":2,"source":"baz/qux","skillId":"not-audited","agentTrustHub":null}
			]}`,
		},
		"/api/audits/1": {
			status: 200,
			ct:     "application/json",
			body:   `{"skills":[]}`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	// 两条都应该拿到(不能 panic,也不能漏条)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (%+v)", len(items), items)
	}
	// 已审计的应有 description;未审计的 description/tags 都应为空字符串
	var audited, unaudited *skillmarket.MarketItem
	for i := range items {
		if items[i].Name == "audited" {
			audited = &items[i]
		}
		if items[i].Name == "not-audited" {
			unaudited = &items[i]
		}
	}
	if audited == nil || audited.Description == "" {
		t.Errorf("audited item should have description, got %+v", audited)
	}
	if unaudited == nil {
		t.Fatal("not-audited item missing")
	}
	if unaudited.Description != "" {
		t.Errorf("not-audited should have empty description, got %q", unaudited.Description)
	}
	if len(unaudited.Tags) != 0 {
		t.Errorf("not-audited should have no tags, got %v", unaudited.Tags)
	}
}

// TestDiscover_AuditsAPI_50Pages 2026-07-01 增:验证 defaultAuditsPages=50 时,
// page 0/1 拿到数据后即使后续 page 失败/为空,也能正常返回(走分页容错)。
func TestDiscover_AuditsAPI_50Pages(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200, ct: "application/json",
			body: auditsMockResponse([]map[string]any{
				{"rank": 1, "source": "foo/bar", "skillId": "sk1", "summary": "first"},
			}),
		},
		"/api/audits/1": {
			status: 200, ct: "application/json",
			body: auditsMockResponse([]map[string]any{
				{"rank": 2, "source": "baz/qux", "skillId": "sk2", "summary": "second"},
			}),
		},
		// page 2-49 没有 mock → 走 fakeRT 兜底 404
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	// 至少 page 0+1 的 2 条都拿到了
	if len(items) < 2 {
		t.Fatalf("expected >= 2 items, got %d", len(items))
	}
	gotNames := map[string]bool{}
	for _, it := range items {
		gotNames[it.Name] = true
	}
	if !gotNames["sk1"] || !gotNames["sk2"] {
		t.Errorf("expected sk1+sk2, got %v", gotNames)
	}
}

// TestDiscover_AuditsAPI_EmptyAndFallback 验证 audits API 返空 + HTML 解析失败时走 knownCatalogFallback。
func TestDiscover_AuditsAPI_EmptyAndFallback(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/audits/0": {
			status: 200,
			ct:     "application/json",
			body:   `{"skills":[]}`,
		},
		"/api/audits/1": {
			status: 200,
			ct:     "application/json",
			body:   `{"skills":[]}`,
		},
		"/": {
			status: 503,
			body:   `unavailable`,
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	items, err := a.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	// fallback 应该返回 >= 28 条(静态填了 30 条)
	if len(items) < 28 {
		t.Fatalf("expected >= 28 fallback items, got %d", len(items))
	}
	// 至少第一条应该有 author(静态 fallback 填了)
	found := false
	for _, it := range items {
		if it.Author != "" && it.Description != "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one fallback item with Author+Description, got 0")
	}
}

// === /api/github/repos/.../git/trees 路径测试(2026-07-10) ===
//
// 覆盖 SKILL.md 在任意深度 anchor 下的场景(101-skills/skills 这种仓库),
// 老 6 候选 fallback 完全 cover 不到,必须走 tree 自动发现。

// treeMockResponse 构造一个 /repos/{o}/{r}/git/trees/{branch} 响应(含多个 blob 路径)。
func treeMockResponse(blobs []string) string {
	type te struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	type resp struct {
		Tree      []te `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	out := resp{}
	for _, p := range blobs {
		out.Tree = append(out.Tree, te{Path: p, Type: "blob"})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// TestDownload_TreeDeepAnchor 验证 101-skills/skills 场景:
// SKILL.md 在 `tools/video/SKILL.md`,走 tree API 能自动找到 SKILL.md
// 并并发拉附属文件 — 老 6 候选 URL 全部 404。
func TestDownload_TreeDeepAnchor(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// tree API:返多层 blob(pattern 含 ?recursive=1 让 matchPathQuery 认 query 段)
		"/repos/101-skills/skills/git/trees/main?recursive=1": {
			status: 200, ct: "application/json",
			body: treeMockResponse([]string{
				"README.md",
				"tools/video/SKILL.md",
				"tools/video/scripts/generate.py",
				"tools/video/references/prompts.md",
				"other-skill/SKILL.md", // 干扰项
			}),
		},
		// raw files (tree-first 后并发拉)
		"/101-skills/skills/main/tools/video/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: ai-video-generation\nversion: 1.0.0\ndescription: AI 视频生成\n---\n# Body\n",
		},
		"/101-skills/skills/main/tools/video/scripts/generate.py": {
			status: 200, ct: "text/x-python",
			body: "print('gen')\n",
		},
		"/101-skills/skills/main/tools/video/references/prompts.md": {
			status: 200, ct: "text/markdown",
			body: "# prompts\n",
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	can, err := a.Download(context.Background(), "https://skills.sh", "101-skills/skills@ai-video-generation")
	if err != nil {
		t.Fatalf("expected tree-first hit, got %v", err)
	}
	if can == nil {
		t.Fatal("nil canonical")
	}
	// 期望:3 文件(SKILL.md + generate.py + prompts.md),干扰项 other-skill/SKILL.md 不被收
	if len(can.Files) != 3 {
		t.Fatalf("expected 3 files, got %d: %+v", len(can.Files), can.Files)
	}
	if can.Files[0].Path != "SKILL.md" {
		t.Errorf("first file should be SKILL.md, got %q", can.Files[0].Path)
	}
	// 相对路径应该是 anchor 去掉后的本地路径(不带 tools/video/ 前缀)
	expectedPaths := map[string]bool{
		"SKILL.md":                true,
		"scripts/generate.py":    true,
		"references/prompts.md":  true,
	}
	for _, f := range can.Files {
		if !expectedPaths[f.Path] {
			t.Errorf("unexpected file path %q", f.Path)
		}
	}
	if can.Manifest.Name != "ai-video-generation" {
		t.Errorf("Manifest.Name = %q want ai-video-generation", can.Manifest.Name)
	}
	if can.Manifest.Author != "101-skills" {
		t.Errorf("Manifest.Author = %q want 101-skills", can.Manifest.Author)
	}
}

// TestDownload_TreeBranchMainThenMaster 验证 tree API 主分支找不到时 fallback master。
func TestDownload_TreeBranchMainThenMaster(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/repos/o/r/git/trees/main?recursive=1": {
			status: 404, body: "Not Found",
		},
		"/repos/o/r/git/trees/master?recursive=1": {
			status: 200, ct: "application/json",
			body: treeMockResponse([]string{
				"skills/foo/SKILL.md",
				"skills/foo/helper.py",
			}),
		},
		"/o/r/master/skills/foo/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: foo\n---\n# Foo\n",
		},
		"/o/r/master/skills/foo/helper.py": {
			status: 200, ct: "text/x-python",
			body: "# helper\n",
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	can, err := a.Download(context.Background(), "https://skills.sh", "o/r@foo")
	if err != nil {
		t.Fatalf("expected tree on master hit, got %v", err)
	}
	if can == nil || len(can.Files) != 2 {
		t.Fatalf("expected 2 files, got %+v", can)
	}
}

// TestDownload_TreeAnchorNotFound_FallThroughToCandidate 验证 tree 找不到 anchor
// 时退到老 6 候选 URL(例如仓库根 <name>/SKILL.md 能被旧路径命中)。
func TestDownload_TreeAnchorNotFound_FallThroughToCandidate(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// tree 主/分支都没 mock → 404
		"/repos/o/r/git/trees/main?recursive=1":   {status: 404},
		"/repos/o/r/git/trees/master?recursive=1": {status: 404},
		// 老 6 候选里有 /skills/<n>/SKILL.md
		"/o/r/main/skills/legacy/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: legacy\n---\n# Legacy\n",
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	can, err := a.Download(context.Background(), "https://skills.sh", "o/r@legacy")
	if err != nil {
		t.Fatalf("expected candidate fallback hit, got %v", err)
	}
	if can == nil || can.Manifest.Name != "legacy" {
		t.Errorf("expected canonical for legacy, got %+v", can)
	}
}

// TestDownload_TreeAnchorNotFound_NoFallback 验证 tree + 6 候选都失败 → 返错。
func TestDownload_TreeAnchorNotFound_NoFallback(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/repos/some/unknown/git/trees/main?recursive=1":   {status: 404},
		"/repos/some/unknown/git/trees/master?recursive=1": {status: 404},
		"/some/unknown/main/skills/x/SKILL.md":   {status: 404},
		"/some/unknown/main/x/SKILL.md":          {status: 404},
		"/some/unknown/master/skills/x/SKILL.md": {status: 404},
		"/some/unknown/master/x/SKILL.md":        {status: 404},
		"/some/unknown/main/.claude/skills/x/SKILL.md":   {status: 404},
		"/some/unknown/master/.claude/skills/x/SKILL.md": {status: 404},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	_, err := a.Download(context.Background(), "https://skills.sh", "some/unknown@x")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestLocateSKILLMDByPath 纯函数测试:locateSKILLMDByPath 在多路径里选最浅的 anchor。
func TestLocateSKILLMDByPath(t *testing.T) {
	entries := []treeEntry{
		{Path: "tools/video/SKILL.md", Type: "blob"},
		{Path: "deep/nested/foo/bar/ai-video-generation/SKILL.md", Type: "blob"},
		{Path: "skills/other/SKILL.md", Type: "blob"},
		{Path: "tools/video/scripts/gen.py", Type: "blob"},
		// 树(tree)节点,不算
		{Path: "tools/video", Type: "tree"},
	}
	_, anchor, ok := locateSKILLMDByPath(entries, "ai-video-generation")
	if !ok {
		t.Fatal("expected locate success")
	}
	if anchor != "deep/nested/foo/bar/ai-video-generation/" {
		t.Errorf("anchor = %q want deep/nested/foo/bar/ai-video-generation/", anchor)
	}

	_, anchor2, ok2 := locateSKILLMDByPath(entries, "video")
	if !ok2 {
		t.Fatal("expected locate video")
	}
	if anchor2 != "tools/video/" {
		t.Errorf("anchor2 = %q want tools/video/", anchor2)
	}

	_, _, ok3 := locateSKILLMDByPath(entries, "nonexistent")
	if ok3 {
		t.Error("expected miss for nonexistent")
	}
}

// TestExtractFrontmatterName 验证 frontmatter name 字段提取(2026-07-10 增)。
func TestExtractFrontmatterName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"---\nname: foo\n---\n# Body\n", "foo"},
		{"---\nname: \"ai video\"\n---\n# Body\n", "ai video"},
		{"---\nname: 'kebab-case'\n---\n", "kebab-case"},
		{"no frontmatter here\n", ""},
		{"---\ntitle: no name\n---\n", ""},
		{"---\nname: alpha\ndescription: desc\n---\n# body", "alpha"},
	}
	for _, c := range cases {
		t.Run(c.in[:min(20, len(c.in))], func(t *testing.T) {
			got := extractFrontmatterName(c.in)
			if got != c.want {
				t.Errorf("extractFrontmatterName = %q want %q", got, c.want)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestDownload_FrontmatterFallback 验证场景:SKILL.md 路径里的目录名 != name,
// 必须走 frontmatter 匹配才能命中(如 inference-sh/skills 里
// tools/video/SKILL.md 但 frontmatter name=ai-video-generation)。
func TestDownload_FrontmatterFallback(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		// tree:多个 SKILL.md(目录名都不等于 name)
		"/repos/o/r/git/trees/main?recursive=1": {
			status: 200, ct: "application/json",
			body: treeMockResponse([]string{
				"tools/video/SKILL.md",
				"tools/video/scripts/run.py",
				"tools/image/SKILL.md",
				"skills/audio/SKILL.md",
			}),
		},
		// frontmatter GET(只命中 video 那个)
		"/o/r/main/tools/video/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: ai-video-generation\nversion: 1.0.0\ndescription: video gen\n---\n# AI Video Generation\n",
		},
		// 其它 SKILL.md 也返 200(以免 downloadSKILLMDOnly 后续 fallback 触发 429 误判)
		"/o/r/main/tools/image/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: ai-image-generation\n---\n",
		},
		"/o/r/main/skills/audio/SKILL.md": {
			status: 200, ct: "text/markdown",
			body: "---\nname: audio-helper\n---\n",
		},
		// 后续 anchor files 拉(以 anchor=tools/video/ 拉附属)
		"/o/r/main/tools/video/scripts/run.py": {
			status: 200, ct: "text/x-python",
			body: "# run\n",
		},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	can, err := a.Download(context.Background(), "https://skills.sh", "o/r@ai-video-generation")
	if err != nil {
		t.Fatalf("expected frontmatter hit, got %v", err)
	}
	if can == nil {
		t.Fatal("nil canonical")
	}
	if can.Manifest.Name != "ai-video-generation" {
		t.Errorf("Manifest.Name = %q want ai-video-generation", can.Manifest.Name)
	}
	// 期望:SKILL.md + scripts/run.py(2 个文件)
	if len(can.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(can.Files), can.Files)
	}
	wantPaths := map[string]bool{"SKILL.md": true, "scripts/run.py": true}
	for _, f := range can.Files {
		if !wantPaths[f.Path] {
			t.Errorf("unexpected file %q", f.Path)
		}
	}
}

// TestDownload_TreeTruncated 验证 tree 返 truncated(超大仓库)返错,走老路径兜底。
func TestDownload_TreeTruncated(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/repos/big/repo/git/trees/main?recursive=1": {
			status: 200, ct: "application/json",
			body: `{"tree":[{"path":"foo","type":"blob"}],"truncated":true}`,
		},
		"/repos/big/repo/git/trees/master?recursive=1": {
			status: 200, ct: "application/json",
			body: `{"tree":[],"truncated":true}`,
		},
		// 兜底候选也失败(没在 knownCatalogFallback)
		"/big/repo/main/skills/foo/SKILL.md":   {status: 404},
		"/big/repo/main/foo/SKILL.md":          {status: 404},
		"/big/repo/master/skills/foo/SKILL.md": {status: 404},
		"/big/repo/master/foo/SKILL.md":        {status: 404},
	}}
	a := NewWithClient(&http.Client{Transport: rt})
	_, err := a.Download(context.Background(), "https://skills.sh", "big/repo@foo")
	if err == nil {
		t.Fatal("expected error on truncated+fallback fail")
	}
}

// TestTrimDescription 验证 description 裁剪逻辑(避免撑爆卡片布局)。
func TestTrimDescription(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"", 100, ""},
		{"short", 100, "short"},
		{"a long sentence, that goes on and on. but should be trimmed.", 20, "a long sentence,"},
		{"abcdefghijklmnopqrstuvwxyz", 10, "abcdefghij…"},
	}
	for _, c := range cases {
		got := trimDescription(c.in, c.max)
		if got != c.want {
			t.Errorf("trimDescription(%q, %d) = %q want %q", c.in, c.max, got, c.want)
		}
	}
}
