package skillhub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// 模拟慢响应(每页 200ms) — 验证并行翻页比串行快。
//
// 串行预期:50 页 × 200ms = 10s
// 并发 4 预期:~50/4 × 200ms ≈ 2.5s
// 留余量:测试断言 < 6s(略大于 2.5s,因为 httptest 也耗时间)
//
// 2026-07-02 注:此测试只验证**耗时**,不验证 item 数(并发 + stub page=1 fallback
// 让计数难以严格断言)。
func TestDiscover_Pagination_ParallelSpeedup(t *testing.T) {
	const pages = 50
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		time.Sleep(200 * time.Millisecond) // 模拟三方源慢响应
		w.Header().Set("Content-Type", "application/json")
		// 返 1 条 (并发下 pagesLoaded*pageSize=50*100=5000 ≥ total=5000 → stop)
		fmt.Fprintf(w, `{"code":0,"data":{"skills":[{"slug":"s%d","name":"S","description":"","version":"0.1.0","ownerName":"o","tags":[],"homepage":"","updated_at":0}],"total":5000}}`,
			atomic.LoadInt32(&hits))
	}))
	defer srv.Close()

	a := NewWithClient(&http.Client{Transport: &mockRT{base: srv.URL}})
	start := time.Now()
	_, err := a.Discover(context.Background(), srv.URL, "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	finalHits := atomic.LoadInt32(&hits)
	t.Logf("discover fetched %d pages in %v (avg %.0fms/page)", finalHits, elapsed, float64(elapsed.Milliseconds())/float64(finalHits))
	// 串行需 ~50*200ms=10s, 并发 4 需 ~13*200ms=2.5s; 留 6s 阈值(覆盖 httptest 开销 + CI 抖动)
	if elapsed > 6*time.Second {
		t.Errorf("discover too slow: %v (expected < 6s with concurrency=4)", elapsed)
	}
	// 至少拉了 13 页(13*200ms/4 ≈ 650ms,远小于 6s)
	if finalHits < 13 {
		t.Errorf("expected at least 13 pages fetched, got %d", finalHits)
	}
}

// mockRT 让 fetcher 用 srv.URL 当 host(httptest 默认就是)
type mockRT struct {
	base string
}

func (m *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	// 转发给真正的 httptest server,模拟我们的 http.Client 用 fakeRT 行为
	// 但这里直接用 default transport 转发即可
	return http.DefaultTransport.RoundTrip(req)
}