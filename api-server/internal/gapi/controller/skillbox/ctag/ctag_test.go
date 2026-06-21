package ctag

import (
	"testing"

	"ginp-api/pkg/ginp"
)

// TestRoutesRegistered 验证 5 个 tag 端点都注册到 router 表。
func TestRoutesRegistered(t *testing.T) {
	want := map[string]string{
		"/api/skillbox/skills/tags/create":  ginp.HttpPost,
		"/api/skillbox/skills/tags/list":    ginp.HttpGet,
		"/api/skillbox/skills/tags/delete":  ginp.HttpPost,
		"/api/skillbox/skills/tags/diff":    ginp.HttpGet,
		"/api/skillbox/skills/tags/rollback": ginp.HttpPost,
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
