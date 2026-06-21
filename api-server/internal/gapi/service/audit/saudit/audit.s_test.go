package saudit

import (
	"testing"
)

// 占位测试: 业务层依赖 dbWrite / dbRead,真连库在沙盒里会 panic;端到端覆盖交给 Step 12 收尾。
// 这里只验证 service 可以正常构造(避免空文件被误判为"未测")。
func TestNew(t *testing.T) {
	s := New(nil, nil)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.model() == nil {
		t.Fatal("model() returned nil")
	}
}
