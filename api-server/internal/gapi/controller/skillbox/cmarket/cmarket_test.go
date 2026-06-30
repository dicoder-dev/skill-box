package cmarket

import (
	"testing"

	"ginp-api/pkg/ginp"
)

// 验证 4 + 4 个端点都注册到 router 表(2026-06-30 增 4 个新端点)。
func TestRoutesRegistered(t *testing.T) {
	want := map[string]string{
		// 旧 4 端点(2026-06-30 之前)
		"/api/skillbox/market/sources":            ginp.HttpGet,
		"/api/skillbox/market/skills":             ginp.HttpGet,
		"/api/skillbox/market/refresh":            ginp.HttpPost,
		"/api/skillbox/market/install":            ginp.HttpPost,
		// 新 4 端点(2026-06-30 增)
		"/api/skillbox/market/install-v2":                 ginp.HttpPost,
		"/api/skillbox/market/skills-with-installed":      ginp.HttpGet,
		"/api/skillbox/market/sources/aggregated":         ginp.HttpGet,
		"/api/skillbox/market/sources/:id/update":         ginp.HttpPost,
	}
	all := ginp.GetAllRouter()
	have := make(map[string]string, len(all))
	for _, r := range all {
		have[r.Path] = r.HttpType
	}
	for path, method := range want {
		got, ok := have[path]
		if !ok {
			t.Errorf("missing route %s", path)
			continue
		}
		if got != method {
			t.Errorf("route %s: method %s, want %s", path, got, method)
		}
	}
}
