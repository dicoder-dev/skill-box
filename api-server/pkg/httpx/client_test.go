package httpx

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetJSONWithUA_Basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 UA / Accept / gzip header 是否下发
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("expected UA %q, got %q", UserAgent, r.Header.Get("User-Agent"))
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			t.Errorf("expected gzip in Accept-Encoding, got %q", r.Header.Get("Accept-Encoding"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, err := GetJSONWithUA(context.Background(), NewClient(5*time.Second), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if body != `{"ok":true}` {
		t.Errorf("body: %s", body)
	}
}

func TestGetJSONWithUA_Gzip(t *testing.T) {
	want := strings.Repeat("x", 1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(want))
		_ = gz.Close()
	}))
	defer srv.Close()

	body, err := GetJSONWithUA(context.Background(), NewClient(5*time.Second), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if body != want {
		t.Errorf("body length: got %d, want %d", len(body), len(want))
	}
}

func TestGetJSONWithUA_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	_, err := GetJSONWithUA(context.Background(), NewClient(5*time.Second), srv.URL)
	if err == nil {
		t.Fatal("expected 404 error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404: %v", err)
	}
}

func TestNewClient_KeepAliveReuse(t *testing.T) {
	// 验证 NewClient 返回的 Transport 是我们调优过的(KeepAlive 配置存在)
	c := NewClient(10 * time.Second)
	if c.Timeout != 10*time.Second {
		t.Errorf("timeout not set")
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost: got %d, want 10", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout: got %v, want 90s", tr.IdleConnTimeout)
	}
	if tr.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("TLSHandshakeTimeout: got %v, want 10s", tr.TLSHandshakeTimeout)
	}
}

func TestNewNoRedirectClient(t *testing.T) {
	c := NewNoRedirectClient(5 * time.Second)
	if c.CheckRedirect == nil {
		t.Fatal("expected CheckRedirect to be set")
	}
	// 模拟一次 redirect 行为
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/from" {
			http.Redirect(w, r, "/to", 302)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp, err := c.Get(srv.URL + "/from")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Errorf("expected 302 (no follow), got %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}

func TestGetJSONWithUA_ContextCancel(t *testing.T) {
	// 测试 ctx 取消能立刻终止(返回 error)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 模拟慢响应
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := GetJSONWithUA(ctx, NewClient(10*time.Second), srv.URL)
	if err == nil {
		t.Fatal("expected error on cancel")
	}
}