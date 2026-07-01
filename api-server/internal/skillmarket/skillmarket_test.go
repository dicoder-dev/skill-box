package skillmarket_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/internal/skillmarket/skillhub"
	"ginp-api/internal/skillmarket/skillssh"
)

// stubAdapter 用于测试的最小适配器(覆盖 Discover/Detail/Download)。
type stubAdapter struct {
	id         string
	display    string
	items      []skillmarket.MarketItem
	can        *skilladapter.Canonical
	discoverOK int32
	detailOK   int32
	downloadOK int32
}

func (s *stubAdapter) SourceID() string    { return s.id }
func (s *stubAdapter) DisplayName() string { return s.display }
func (s *stubAdapter) BaseURL() string     { return "https://stub" }
func (s *stubAdapter) Discover(ctx context.Context, baseURL, keyword string) ([]skillmarket.MarketItem, error) {
	atomic.AddInt32(&s.discoverOK, 1)
	return s.items, nil
}
func (s *stubAdapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	atomic.AddInt32(&s.detailOK, 1)
	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{RemoteID: remoteID, Name: remoteID},
	}, nil
}
func (s *stubAdapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	atomic.AddInt32(&s.downloadOK, 1)
	if s.can != nil {
		return s.can, nil
	}
	return &skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: remoteID, Version: "0.1.0", Description: "stub skill for " + remoteID, Triggers: []string{remoteID}},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: " + remoteID + "\n---\nbody"}},
	}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := &skillmarket.Registry{}
	a := &stubAdapter{id: "stub", display: "Stub"}
	r.Register(a)
	got, ok := r.Get("stub")
	if !ok {
		t.Fatal("expected to find stub")
	}
	if got.SourceID() != "stub" {
		t.Errorf("got %s", got.SourceID())
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("expected miss for unknown id")
	}
}

func TestRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate registration")
		}
	}()
	r := &skillmarket.Registry{}
	r.Register(&stubAdapter{id: "x"})
	r.Register(&stubAdapter{id: "x"})
}

func TestRegistry_All_Sorted(t *testing.T) {
	r := &skillmarket.Registry{}
	r.Register(&stubAdapter{id: "b"})
	r.Register(&stubAdapter{id: "a"})
	r.Register(&stubAdapter{id: "c"})
	got := r.All()
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].SourceID() != "a" || got[1].SourceID() != "b" || got[2].SourceID() != "c" {
		t.Errorf("order wrong: %s %s %s", got[0].SourceID(), got[1].SourceID(), got[2].SourceID())
	}
}

func TestDefaultAdapters_Registered(t *testing.T) {
	for _, id := range []string{skillmarket.SourceSkillhub, skillmarket.SourceSkillsSH} {
		if _, ok := skillmarket.Get(id); !ok {
			t.Errorf("expected %s to be registered", id)
		}
	}
}

func TestDefaultAdapters_FactorySane(t *testing.T) {
	_ = skillhub.New()
	_ = skillssh.New()
	_ = skillhub.NewWithClient(nil)
	_ = skillssh.NewWithClient(nil)
}

func TestSanitizeSourceName(t *testing.T) {
	cases := map[string]string{
		"skillhub":   "skillhub",
		"skills.sh":  "skills-sh",
		"  Foo Bar ": "foo-bar",
		"":           "",
	}
	for in, want := range cases {
		if got := skillmarket.SanitizeSourceName(in); got != want {
			t.Errorf("SanitizeSourceName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestResolveBaseFromConfig(t *testing.T) {
	if got := skillmarket.ResolveBaseURL(&stubAdapter{}, ""); got != "https://stub" {
		t.Errorf("empty config should return default, got %q", got)
	}
	if got := skillmarket.ResolveBaseURL(&stubAdapter{}, "not json"); got != "https://stub" {
		t.Errorf("bad config should return default, got %q", got)
	}
}

func TestWrapErr(t *testing.T) {
	if err := skillmarket.WrapErr("verb", nil); err != nil {
		t.Errorf("nil should not be wrapped, got %v", err)
	}
	err := skillmarket.WrapErr("verb", context.DeadlineExceeded)
	if err == nil {
		t.Fatal("expected wrapped err")
	}
	if !strings.Contains(err.Error(), "verb") {
		t.Errorf("wrapped err should mention verb, got %v", err)
	}
}

// fakeRT 简单的 mock transport,用于测内置 adapter 的 E2E 路径。
type fakeRT struct {
	responses map[string]fakeResp
}

type fakeResp struct {
	status int
	body   string
}

func (f *fakeRT) RoundTrip(r *http.Request) (*http.Response, error) {
	for pattern, resp := range f.responses {
		if matchPathRT(r.URL.Path, pattern) {
			return &http.Response{
				StatusCode: resp.status,
				Body:       io.NopCloser(bytes.NewReader([]byte(resp.body))),
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Request:    r,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Request:    r,
	}, nil
}

func matchPathRT(path, pattern string) bool {
	if i := strings.Index(pattern, "*"); i >= 0 {
		return strings.HasPrefix(path, pattern[:i])
	}
	return path == pattern
}

// TestSkillHubAdapter_E2E_DiscoverDownload 走 mock transport 测一遍完整 Discover → Download 链路。
//
// 2026-07-01 改:对接新 API — Discover 走 /api/skills,Download 走单文件 fallback /api/v1/skills/{slug}/skill.md。
func TestSkillHubAdapter_E2E_DiscoverDownload(t *testing.T) {
	rt := &fakeRT{responses: map[string]fakeResp{
		"/api/skills": {
			status: 200,
			body: `{"code":0,"data":{"skills":[{"slug":"x","name":"X","version":"0.1.0","description":"x","ownerName":"alice","tags":[],"homepage":"https://x","updated_at":1782878868630}],"total":1}}`,
		},
		"/api/v1/skills/x/skill.md": {
			status: 200,
			body: `---
name: x
description: x 描述 x
version: 0.1.0
triggers: [x]
---

# X
`,
		},
	}}
	ad := skillhub.NewWithClient(&http.Client{Transport: rt})
	items, err := ad.Discover(context.Background(), "https://stub", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RemoteID != "x" {
		t.Fatalf("unexpected items: %+v", items)
	}
	can, err := ad.Download(context.Background(), "https://stub", "x")
	if err != nil {
		t.Fatal(err)
	}
	if can == nil || can.Manifest.Name != "x" {
		t.Errorf("expected parsed canonical, got %+v", can)
	}
}
