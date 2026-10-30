package ssupdater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newManifestServer 起一个返回指定 manifest 的测试服务器。
func newManifestServer(t *testing.T, version string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"channel":"stable","version":"` + version + `","min_supported":"1.0.0","assets":[]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchManifestPrefersFirstURL(t *testing.T) {
	first := newManifestServer(t, "1.2.3", http.StatusOK)
	second := newManifestServer(t, "9.9.9", http.StatusOK)

	cli := NewClient()
	m, source, err := cli.FetchManifestWithSource(context.Background(), []string{first.URL, second.URL}, "stable")
	if err != nil {
		t.Fatalf("拉取失败: %v", err)
	}
	if m.Version != "1.2.3" {
		t.Errorf("version = %q, 期望 1.2.3(应命中第一个源)", m.Version)
	}
	if source != first.URL {
		t.Errorf("source = %q, 期望 %q", source, first.URL)
	}
}

func TestFetchManifestFallsBackToSecondURL(t *testing.T) {
	broken := newManifestServer(t, "", http.StatusInternalServerError)
	backup := newManifestServer(t, "1.2.3", http.StatusOK)

	cli := NewClient()
	m, source, err := cli.FetchManifestWithSource(context.Background(), []string{broken.URL, backup.URL}, "stable")
	if err != nil {
		t.Fatalf("拉取失败(应降级到第二源): %v", err)
	}
	if m.Version != "1.2.3" {
		t.Errorf("version = %q, 期望 1.2.3", m.Version)
	}
	if source != backup.URL {
		t.Errorf("source = %q, 期望降级到 %q", source, backup.URL)
	}
}

func TestFetchManifestAllSourcesFailed(t *testing.T) {
	broken := newManifestServer(t, "", http.StatusNotFound)
	cli := NewClient()
	if _, _, err := cli.FetchManifestWithSource(context.Background(), []string{broken.URL}, "stable"); err == nil {
		t.Error("所有源失败时应返 error")
	}
}

func TestCacheKeyDiffersByURLSet(t *testing.T) {
	// 两个源的 manifest 内容不同(urls 指向各自站点),缓存键必须区分开,
	// 否则切语言后会拿到上一个源的 manifest。
	k1 := cacheKey("stable", []string{"https://gitee.com/x/manifest.gitee.json"})
	k2 := cacheKey("stable", []string{"https://raw.githubusercontent.com/x/manifest.github.json"})
	if k1 == k2 {
		t.Error("不同源列表应生成不同缓存键")
	}
	same1 := cacheKey("stable", []string{"https://a", "https://b"})
	same2 := cacheKey("stable", []string{"https://a", "https://b"})
	if same1 != same2 {
		t.Error("相同源列表应生成相同缓存键")
	}
}

func TestCompareStillWorks(t *testing.T) {
	if got := Compare("1.0.0", "1.2.0", "1.0.0"); got != StatusAvailable {
		t.Errorf("Compare = %q, 期望 %q", got, StatusAvailable)
	}
	if got := Compare("1.2.0", "1.2.0", "1.0.0"); got != StatusUpToDate {
		t.Errorf("Compare = %q, 期望 %q", got, StatusUpToDate)
	}
	if got := Compare("0.9.0", "1.2.0", "1.0.0"); got != StatusMustUpdate {
		t.Errorf("Compare = %q, 期望 %q", got, StatusMustUpdate)
	}
}
