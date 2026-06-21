package cskillpkg

import (
	"testing"

	"ginp-api/pkg/ginp"
)

// 验证 3 个端点都注册到 router 表。
func TestRoutesRegistered(t *testing.T) {
	want := map[string]string{
		"/api/skillbox/pkg/export":  ginp.HttpPost,
		"/api/skillbox/pkg/import":  ginp.HttpPost,
		"/api/skillbox/pkg/preview": ginp.HttpPost,
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
