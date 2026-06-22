---
name: unit-test-gen
version: 0.1.0
description: 为函数 / 方法自动生成单元测试(table-driven + 表外分支),明确覆盖 happy path / 边界 / 异常。
triggers:
  - test
  - unit test
  - /test
  - 测试
  - 单元测试
  - 加测试
author: Skill Box
license: MIT
target_tools:
  - codex
  - claude
  - opencode
  - cursor
  - trae
---

# Unit Test Generator

自动为函数 / 方法生成单元测试,而不是只贴一个 `TestXxx`。

## 何时触发

- 新写完一个函数,想加测试:`test <func>` 或 "给这个函数加单元测试"
- 看一段旧代码没测试:"补单测"
- 重构前想锁定行为:粘源码 + "加测试保持现有行为"

## 行为

按"分层覆盖"原则组织用例:

1. **happy path**:正常入参 + 预期出参(至少 1 条)
2. **边界**:零值 / 空串 / 空切片 / 最大值 / 最小值
3. **异常**:错误入参 / nil 引用 / 越界 / 不存在的 key
4. **依赖隔离**:外部调用用 fake / mock / stub,不打真实 IO

## 输出格式

```text
// <file>: <func> 测试
package <pkg>

import (
  "testing"
)

func Test<Func>_<Scenario>(t *testing.T) {
  tests := []struct {
    name    string
    args    args
    want    <ReturnType>
    wantErr bool
  }{
    {"happy path", args{...}, ..., false},
    {"zero value", args{...}, ..., false},
    {"invalid input", args{...}, ..., true},
  }
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      got, err := <Func>(tt.args...)
      if (err != nil) != tt.wantErr {
        t.Fatalf("...")
      }
      if !reflect.DeepEqual(got, tt.want) {
        t.Errorf("...")
      }
    })
  }
}
```

## 限制

- 不调用被测函数的真实依赖(DB / 网络);用 fake / mock
- 不生成并发测试用例(`t.Parallel` / `go test -race`),除非显式要求
- 不自动运行 `go test`;只产出代码
- 输出前必须看一遍源码,不要"凭函数名猜语义"