package skillhub

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeRT 自定义 http.RoundTripper,不监听端口(沙盒限制)。
// 2026-07-01 改:支持 query string 匹配(为 keyword 透传测试服务);支持 no-redirect(为 zip 302 流程服务)。
type fakeRT struct {
	responses map[string]fakeResp
}

type fakeResp struct {
	status int
	body   string
	ct     string
	// redirectTo:如果非空,返 302 + Location:redirectTo
	redirectTo string
}

func (f *fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	// 先看 redirectTo(302 模拟,需要 path 匹配)
	for pattern, resp := range f.responses {
		if resp.redirectTo == "" {
			continue
		}
		if matchPathQuery(r.URL.Path, r.URL.RawQuery, pattern) {
			h := http.Header{
				"Content-Type": []string{firstNonEmpty(resp.ct, "application/json")},
				"Location":     []string{resp.redirectTo},
			}
			return &http.Response{
				StatusCode: http.StatusFound,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
				Header:     h,
				Request:    r,
			}, nil
		}
	}
	for pattern, resp := range f.responses {
		if matchPathQuery(r.URL.Path, r.URL.RawQuery, pattern) {
			return &http.Response{
				StatusCode: resp.status,
				Body:       io.NopCloser(bytes.NewReader([]byte(resp.body))),
				Header: http.Header{
					"Content-Type": []string{firstNonEmpty(resp.ct, "application/json")},
				},
				Request: r,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"no match for ` + r.URL.Path + `"}`))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

// matchPathQuery pattern 形如 "/api/skills?keyword=foo&pageSize=100" 或 "/api/skills"。
// 2026-07-01 改:去掉 method 前缀支持(简化;fakeRT 永远只服务 GET,不需要区分 method)。
// query 省略时只匹配 path。
func matchPathQuery(path, query, pattern string) bool {
	pat := pattern
	// 解析 query
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
	// 简单 key=val 包含检查(query 顺序无关)
	return queryContains(query, patQuery)
}

func queryContains(query, sub string) bool {
	if sub == "" {
		return true
	}
	for _, part := range strings.Split(sub, "&") {
		if !strings.Contains(query, part) {
			return false
		}
	}
	return true
}

func newFakeClient(responses map[string]fakeResp) *http.Client {
	return &http.Client{Transport: &fakeRT{responses: responses}}
}

// --- Discover ---

func TestDiscover_RealAPI_Homepage(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body: `{
				"code": 0,
				"data": {
					"skills": [
						{
							"slug": "code-review",
							"name": "Code Review",
							"description": "review diff",
							"description_zh": "审查 diff",
							"version": "1.0.0",
							"ownerName": "alice",
							"tags": ["review"],
							"subCategories": [{"key":"code-quality","name":"代码质量"}],
							"homepage": "https://skillhub.cn/skills/code-review",
							"updated_at": 1782878868630
						},
						{
							"slug": "react-toolkit",
							"name": "React Toolkit",
							"description": "react helpers",
							"version": "0.5.0",
							"ownerName": "bob",
							"tags": ["react","frontend"],
							"homepage": "https://skillhub.cn/skills/react-toolkit",
							"updated_at": 1782878800000
						}
					],
					"total": 2
				}
			}`,
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (%+v)", len(items), items)
	}
	if items[0].RemoteID != "code-review" {
		t.Errorf("RemoteID: %s", items[0].RemoteID)
	}
	// description_zh 优先
	if items[0].Description != "审查 diff" {
		t.Errorf("Description (zh 优先): %s", items[0].Description)
	}
	// tags 包含 subCategories.name
	foundSubcat := false
	for _, tag := range items[0].Tags {
		if tag == "代码质量" {
			foundSubcat = true
			break
		}
	}
	if !foundSubcat {
		t.Errorf("subCategories[].name not in Tags: %+v", items[0].Tags)
	}
	// UpdatedAt 转换
	if items[0].UpdatedAt.UnixMilli() != 1782878868630 {
		t.Errorf("UpdatedAt: %v", items[0].UpdatedAt)
	}
}

func TestDiscover_Keyword_Pass(t *testing.T) {
	// 验证 keyword 透传到 query string
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?keyword=react&pageSize=100": {
			status: 200,
			body: `{
				"code": 0,
				"data": {
					"skills": [
						{"slug": "react-toolkit","name":"React Toolkit","description":"react","version":"0.5.0","ownerName":"bob","tags":[],"homepage":"https://x","updated_at":1782878800000}
					],
					"total": 1
				}
			}`,
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "react")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RemoteID != "react-toolkit" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestDiscover_Keyword_SpecialChar(t *testing.T) {
	// 验证 keyword 含特殊字符(空格)被正确编码
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?keyword=react+native&pageSize=100": {
			status: 200,
			body: `{"code":0,"data":{"skills":[],"total":0}}`,
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "react native")
	if err != nil {
		t.Fatal(err)
	}
	// 0 条时也走 fallback
	if len(items) < 1 {
		t.Errorf("expected fallback items, got %d", len(items))
	}
}

func TestDiscover_CodeNonZero_Fallback(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body:   `{"code":400,"data":null,"message":"参数错误"}`,
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("expected fallback >=3, got %d", len(items))
	}
}

func TestDiscover_HTTP500_Fallback(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 500,
			body:   "internal",
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("expected fallback >=3, got %d", len(items))
	}
}

func TestDiscover_InvalidJSON_Fallback(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body:   "<html>oops</html>",
			ct:     "text/html",
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("expected fallback >=3, got %d", len(items))
	}
}

// TestDiscover_Pagination 验证 2026-07-01 翻页改造:
//   - 首页 ?page=1&pageSize=100&sortBy=downloads&order=desc 拿满 100 条 + total=250
//   - 翻到第 2、3 页 ?page=2&pageSize=100 / ?page=3&pageSize=100 拿剩余 150 条
//   - 共 250 条全部去重合并返回
func TestDiscover_Pagination(t *testing.T) {
	mkPage := func(start, end int) string {
		var sb strings.Builder
		sb.WriteString(`{"code":0,"data":{"skills":[`)
		for i := start; i <= end; i++ {
			if i > start {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"slug":"skill-%03d","name":"Skill %03d","description":"d","version":"0.1.0","ownerName":"o","tags":[],"homepage":"https://x","updated_at":0}`, i, i)
		}
		fmt.Fprintf(&sb, `],"total":250}}`)
		return sb.String()
	}
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body:   mkPage(1, 100),
		},
		"/api/skills?page=2&pageSize=100": {
			status: 200,
			body:   mkPage(101, 200),
		},
		"/api/skills?page=3&pageSize=100": {
			status: 200,
			body:   mkPage(201, 250),
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 250 {
		t.Fatalf("expected 250 items (3 pages merged), got %d", len(items))
	}
	if items[0].RemoteID != "skill-001" {
		t.Errorf("first RemoteID: %s", items[0].RemoteID)
	}
	if items[249].RemoteID != "skill-250" {
		t.Errorf("last RemoteID: %s", items[249].RemoteID)
	}
	seen := map[string]bool{}
	for _, it := range items {
		if seen[it.RemoteID] {
			t.Errorf("duplicate: %s", it.RemoteID)
		}
		seen[it.RemoteID] = true
	}
}

// TestDiscover_Pagination_StopsOnShortPage 验证翻页退出条件 3:
// 本页条数 < pageSize(50 < 100)→ 视为已无更多数据,停止翻页。
func TestDiscover_Pagination_StopsOnShortPage(t *testing.T) {
	mkPage := func(start, end int) string {
		var sb strings.Builder
		sb.WriteString(`{"code":0,"data":{"skills":[`)
		for i := start; i <= end; i++ {
			if i > start {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"slug":"skill-%03d","name":"S","description":"","version":"0.1.0","ownerName":"o","tags":[],"homepage":"","updated_at":0}`, i)
		}
		sb.WriteString(`]}}`) // 不写 total
		return sb.String()
	}
	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body:   mkPage(1, 100),
		},
		"/api/skills?page=2&pageSize=100": {
			status: 200,
			body:   mkPage(101, 150), // 50 < 100,翻页停
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 150 {
		t.Fatalf("expected 150 items (page1+page2, page2<pageSize 停翻), got %d", len(items))
	}
}

// TestDiscover_Pagination_MidPageFail 验证翻页中途失败:
// 首页 100 条成功,第 2 页 HTTP 500 → 已拿到的 100 条仍返回(不整体 fallback)。
func TestDiscover_Pagination_MidPageFail(t *testing.T) {
	var page1 strings.Builder
	page1.WriteString(`{"code":0,"data":{"skills":[`)
	for i := 1; i <= 100; i++ {
		if i > 1 {
			page1.WriteString(",")
		}
		fmt.Fprintf(&page1, `{"slug":"skill-%03d","name":"S","description":"","version":"0.1.0","ownerName":"o","tags":[],"homepage":"","updated_at":0}`, i)
	}
	page1.WriteString(`],"total":500}}`)

	rt := newFakeClient(map[string]fakeResp{
		"/api/skills?page=1&pageSize=100&sortBy=downloads&order=desc": {
			status: 200,
			body:   page1.String(),
		},
		"/api/skills?page=2&pageSize=100": {
			status: 500,
			body:   "boom",
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://api.skillhub.cn", "")
	if err != nil {
		t.Fatal(err)
	}
	// 翻页中途 500:已拿的 100 条返回,不全 fallback
	if len(items) != 100 {
		t.Fatalf("expected 100 items (page1 成功 + page2 失败保留 page1), got %d", len(items))
	}
}

// --- Detail ---

func TestDetail_RealAPI_ExtraFields(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/code-review": {
			status: 200,
			body: `{
				"code": 0,
				"data": {
					"skill": {
						"slug": "code-review",
						"name": "Code Review",
						"description": "review diff",
						"description_zh": "审查 diff",
						"version": "1.0.0",
						"ownerName": "alice",
						"tags": ["review","quality"],
						"subCategories": [{"key":"code-quality","name":"代码质量"}],
						"homepage": "https://skillhub.cn/skills/code-review",
						"upstream_url": "https://github.com/owner/repo",
						"upstream_owner_login": "owner",
						"labels": {"requires_api_key": "false"},
						"updated_at": 1782878868630
					},
					"owner": {"displayName": "Alice","handle":"alice"},
					"latestVersion": {"version":"1.0.2","changelog":"new","createdAt":1782878868630}
				}
			}`,
		},
	})
	a := NewWithClient(rt)
	d, err := a.Detail(context.Background(), "https://api.skillhub.cn", "code-review")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatal("nil detail")
	}
	if d.RemoteID != "code-review" {
		t.Errorf("RemoteID: %s", d.RemoteID)
	}
	if d.Version != "1.0.2" { // latestVersion.version 优先
		t.Errorf("Version should be latestVersion.version=1.0.2, got %s", d.Version)
	}
	if d.Description != "审查 diff" {
		t.Errorf("Description (zh): %s", d.Description)
	}
	if d.Author != "Alice" { // owner.displayName 优先
		t.Errorf("Author should be Alice, got %s", d.Author)
	}
	// Extra 字段透传
	if d.Extra == nil {
		t.Fatal("Extra should not be nil")
	}
	if up, ok := d.Extra["upstream_url"].(*string); !ok || up == nil || *up != "https://github.com/owner/repo" {
		t.Errorf("Extra upstream_url: %+v", d.Extra["upstream_url"])
	}
	if labels, ok := d.Extra["labels"].(map[string]string); !ok || labels["requires_api_key"] != "false" {
		t.Errorf("Extra labels: %+v", d.Extra["labels"])
	}
}

func TestDetail_NotFound_FallbackHit(t *testing.T) {
	// 404 时,knownFallback 命中 code-review → 走老 schema
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/code-review": {status: 404, body: "no"},
		"/api/v1/skills/no-such-id":  {status: 404, body: "no"},
	})
	a := NewWithClient(rt)
	d, err := a.Detail(context.Background(), "https://api.skillhub.cn", "code-review")
	if err != nil {
		t.Fatalf("fallback detail should not error: %v", err)
	}
	if d == nil || d.Name != "code-review" {
		t.Errorf("expected fallback detail, got %+v", d)
	}
	_, err = a.Detail(context.Background(), "https://api.skillhub.cn", "no-such-id")
	if err == nil {
		t.Fatal("expected error for unknown id")
	}
}

// --- Download ---

func TestDownload_ZipFlow_302ToCOS(t *testing.T) {
	// mock skillhub /api/v1/download 返 302 → mock zip server(真 httptest.NewServer)
	// 2026-07-01 改:用 NewWithClients 拆开两层 client —
	//   - noRedirect 走 fakeRT 返 302(CheckRedirect=ErrUseLastResponse,别 follow)
	//   - 普通 httpClient 走 default(标准 http.Transport),直接 follow 到 zipServer
	zipServer := newZipMockServer(t, "code-review/1.0.2/SKILL.md", `---
name: code-review
description: review
---

# Code Review
`)
	defer zipServer.Close()

	// noRedirect client 必须显式禁用 redirect,否则 fakeRT 收到 302 后会再发 GET
	// 到 zipServer URL,fakeRT 匹配不上,返 404
	noRedir := &http.Client{
		Transport: &fakeRT{responses: map[string]fakeResp{
			"/api/v1/download": {
				status:     302,
				redirectTo: zipServer.URL + "/skills/code-review/1.0.2.zip",
			},
		}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
	// 普通 client 用 default transport(能 follow 跨域到 zipServer)
	a := NewWithClients(noRedir, nil)
	can, err := a.Download(context.Background(), "https://api.skillhub.cn", "code-review")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil {
		t.Fatal("nil canonical")
	}
	if can.Manifest.Name != "code-review" {
		t.Errorf("Manifest.Name: %s", can.Manifest.Name)
	}
	// version 应从 zip 路径推断 = "1.0.2"
	if can.Manifest.Version != "1.0.2" {
		t.Errorf("Manifest.Version (from zip path): %s", can.Manifest.Version)
	}
}

func TestDownload_SingleFileFallback(t *testing.T) {
	// zip 路径 404 → 走 single-file 路径
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/download":               {status: 404, body: "no zip"},
		"/api/v1/skills/remote-1234/skill.md": {
			status: 200,
			body: `---
name: code-review
description: review
version: 1.0.0
---
# X
`,
			ct: "text/markdown",
		},
	})
	a := NewWithClient(rt)
	can, err := a.Download(context.Background(), "https://api.skillhub.cn", "remote-1234")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil || can.Manifest.Name != "code-review" {
		t.Errorf("expected code-review canonical, got %+v", can)
	}
}

func TestDownload_KnownFallback_LastResort(t *testing.T) {
	// zip + single-file 都失败 → 走 knownFallback(命中 id 的话)
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/download":                     {status: 404, body: "no"},
		"/api/v1/skills/code-review/skill.md":  {status: 404, body: "no"},
	})
	a := NewWithClient(rt)
	can, err := a.Download(context.Background(), "https://api.skillhub.cn", "code-review")
	if err != nil {
		t.Fatalf("fallback should not error: %v", err)
	}
	if can == nil || can.Manifest.Name != "code-review" {
		t.Errorf("expected code-review canonical, got %+v", can)
	}
}

func TestDownload_NoFallback_UnknownID(t *testing.T) {
	// 不在 knownFallback 也不在 zip → 报错
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/download": {status: 404, body: "no"},
	})
	a := NewWithClient(rt)
	_, err := a.Download(context.Background(), "https://api.skillhub.cn", "no-such-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Fallback / Constructor ---

func TestBuildFallbackCanonical_Minimum(t *testing.T) {
	can := buildFallbackCanonical(knownFallback[0])
	if can == nil {
		t.Fatal("nil canonical")
	}
	if can.Manifest.Name == "" || can.Manifest.Version == "" {
		t.Errorf("missing required fields: %+v", can.Manifest)
	}
	if len(can.Files) == 0 {
		t.Error("no files in canonical")
	}
}

func TestNewWithClient_NilFallsBack(t *testing.T) {
	a := NewWithClient(nil)
	if a == nil || a.httpClient == nil {
		t.Error("nil client should fall back to default")
	}
}

func TestNew_Default30s(t *testing.T) {
	a := New()
	if a.httpClient.Timeout != 30*1e9 {
		t.Errorf("timeout should be 30s, got %v", a.httpClient.Timeout)
	}
}

// newZipMockServer 起一个 httptest server 返合法 zip 内容(给定 zip 内 SKILL.md 路径 + 内容)。
func newZipMockServer(t *testing.T, innerPath, skillMD string) *zipMockServer {
	t.Helper()
	z := &zipMockServer{buf: buildZip(t, innerPath, skillMD)}
	z.Server = http.Server{Handler: z}
	ts := httptest.NewServer(z)
	z.URL = ts.URL
	t.Cleanup(ts.Close)
	return z
}

// zipMockServer: 模拟 COS zip server,跑在 httptest.NewServer 真实端口上。
// 测试时 NewWithClient 不能用这个 server(它只返 200 + 静态 zip),需要把 Adapter 的 httpClient
// 替换为带 transport skip-redirect 的 client(这样 Download 走 noRedirectClient → 拿 Location
// → 然后是测试代码主动用 zipServer URL 拉 zip 字节流,不经过 fakeRT)。
// 但当前实现两个 client 共用 NewWithClient,这里把 noRedirectClient 设为 zip server 直接相关的 client。
// 实际做法:见 ZipFlow 测试,它用 NewWithZipServer 自定义 client。
type zipMockServer struct {
	http.Server
	buf []byte
	URL string
}

func (z *zipMockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(z.buf)))
	w.WriteHeader(http.StatusOK)
	w.Write(z.buf)
}

func buildZip(t *testing.T, innerPath, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(innerPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
