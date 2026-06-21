package skillhub

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeRT 自定义 http.RoundTripper,不监听端口(沙盒限制)。
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
		if matchPath(r.URL.Path, pattern) {
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

func matchPath(path, pattern string) bool {
	// pattern 用 * 任意后缀
	if i := strings.Index(pattern, "*"); i >= 0 {
		return strings.HasPrefix(path, pattern[:i])
	}
	return path == pattern
}

func newFakeClient(responses map[string]fakeResp) *http.Client {
	return &http.Client{Transport: &fakeRT{responses: responses}}
}

func TestDiscover_ParseJSON(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills": {
			status: 200,
			body: `[
				{"id":"code-review","name":"Code Review","version":"1.0.0","author":"alice","tags":["review"]},
				{"id":"debug-helper","name":"Debug Helper","version":"0.2.0","author":"bob","tags":["debug","diagnostic"]}
			]`,
		},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://stub")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (%+v)", len(items), items)
	}
	if items[0].RemoteID != "code-review" || items[0].Name != "Code Review" {
		t.Errorf("first item: %+v", items[0])
	}
	if items[1].Tags[0] != "debug" {
		t.Errorf("tags: %+v", items[1].Tags)
	}
}

func TestDiscover_FallbackOnError(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills": {status: 500, body: "internal"},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://stub")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("fallback should have >=3, got %d", len(items))
	}
}

func TestDiscover_FallbackOnNonJSON(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills": {status: 200, body: "<html>not json</html>", ct: "text/html"},
	})
	a := NewWithClient(rt)
	items, err := a.Discover(context.Background(), "https://stub")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Errorf("non-JSON should fallback, got %d", len(items))
	}
}

func TestDownload_Fallback(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/code-review/skill.md": {status: 404, body: "no"},
	})
	a := NewWithClient(rt)
	can, err := a.Download(context.Background(), "https://stub", "code-review")
	if err != nil {
		t.Fatalf("fallback should not error: %v", err)
	}
	if can == nil || can.Manifest.Name != "code-review" {
		t.Errorf("expected code-review canonical, got %+v", can)
	}
	if len(can.Files) == 0 || !strings.Contains(can.Files[0].Content, "code-review") {
		t.Errorf("fallback canonical body should mention code-review, got %+v", can.Files)
	}
}

func TestDownload_ParsesSkillMD(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/remote-1234/skill.md": {
			status: 200,
			body: `---
name: code-review
description: 对当前 diff 做静态代码审查,聚焦可读性与潜在 bug
version: 1.0.0
triggers:
  - review
---

# Code Review
`,
			ct: "text/markdown",
		},
	})
	a := NewWithClient(rt)
	can, err := a.Download(context.Background(), "https://stub", "remote-1234")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil || can.Manifest.Name != "code-review" || can.Manifest.Version != "1.0.0" {
		t.Errorf("expected parsed canonical, got %+v", can)
	}
}

func TestDetail_NotFound(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/code-review": {status: 404, body: "no"},
		"/api/v1/skills/no-such-id":  {status: 404, body: "no"},
	})
	a := NewWithClient(rt)
	// 走 fallback 里的 id 命中
	d, err := a.Detail(context.Background(), "https://stub", "code-review")
	if err != nil {
		t.Fatalf("fallback detail should not error: %v", err)
	}
	if d == nil || d.Name != "code-review" {
		t.Errorf("expected fallback detail, got %+v", d)
	}
	// 不在 fallback 里的 id 报 ErrRemoteNotFound
	_, err = a.Detail(context.Background(), "https://stub", "no-such-id")
	if err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestDetail_Parsed(t *testing.T) {
	rt := newFakeClient(map[string]fakeResp{
		"/api/v1/skills/x": {
			status: 200,
			body: `{"id":"x","name":"X","version":"0.1.0","description":"x desc","author":"a","license":"MIT","tags":["t1","t2"]}`,
		},
	})
	a := NewWithClient(rt)
	d, err := a.Detail(context.Background(), "https://stub", "x")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || d.Name != "X" || d.License != "MIT" {
		t.Errorf("unexpected detail: %+v", d)
	}
}

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
