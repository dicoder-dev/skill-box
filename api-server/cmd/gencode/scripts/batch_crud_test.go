package script

import (
	"ginp-api/cmd/gencode/desc"
	"testing"
)

func TestBatchCrud(t *testing.T) {
	// 测试实体列表
	entities := []string{
		"TestEntity1",
		"TestEntity2",
		"TestEntity3",
	}

	// 调用批量生成函数
	desc.GenBatchCrud(entities)
}
