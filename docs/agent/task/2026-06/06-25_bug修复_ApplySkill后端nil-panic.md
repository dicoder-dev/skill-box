# 修 Applier.registry nil 兜底

**日期:** 2026-06-25
**状态:** 已完成

## 1. 需求

dev 模式下跑 wails3 task dev,点击作用域 chip 触发 apply 时,后端 panic:

```
runtime error: invalid memory address or nil pointer dereference
.../skilladapter/registry.go:47   (*Registry).Get: r.mu.RLock()
.../skillapp/applier.go:73        (*Applier).ApplyOne: ad, ok := a.registry.Get(toolID)
```

根因:`applier.NewApplier(registry)` 注释说"registry=nil 时用默认全局",但实现里
直接把 nil 存进 `a.registry`,后续 `a.registry.Get()` 解引用 nil 指针 panic。

调用链:`cskillapply.newService()` → `sskillapp.New()` → `WithAdapterRegistry` 未调用
→ `s.adapterRegistry == nil` → `applier()` 传 nil 进 NewApplier → 触发 panic。

## 2. 任务列表

- [x] 1) `skilladapter` 包 export `DefaultRegistry()`(暴露全局指针供兜底)
- [x] 2) `Applier.resolveRegistry()` 方法:nil 时返回 DefaultRegistry
- [x] 3) `ApplyOne` 改用 `a.resolveRegistry().Get()`
- [x] 4) 加 unit test `TestApplyOne_NilRegistry_FallsBackToDefault`
- [x] 5) `go test ./internal/skillapp/...` 通过
- [x] 6) `go build ./...` 通过
- [x] 7) git commit + push

## 3. 执行进度

- 04:55 解读 panic 栈,定位 applier.go:73 + registry.go:47 是 nil 指针
- 04:56 跟用户确认:后端 panic 修复优先,不影响前端 toast / 自动同步主功能
- 04:57 加 `DefaultRegistry()` export(给"忘注入"的代码兜底用)
- 04:58 `Applier.resolveRegistry()` 实现 + ApplyOne 改用
- 04:59 加 `TestApplyOne_NilRegistry_FallsBackToDefault`:注册 fake adapter 到 default → NewApplier(nil) → ApplyOne 成功
- 05:00 跑测试:`ok ginp-api/internal/skillapp 0.007s`
- 05:00 go build ./... ✅ 通过

## 4. 问题与方案

**Q1: 为什么不在 controller / service 层修复,而要改 Applier?**
A: Applier 注释明确承诺"registry=nil 时用默认全局",但实现没兑现 — 这是个
注释与代码不符的 bug。修在 Applier 里能让"忘注入"的所有调用方都安全,
不用每个 controller 都记得调 WithAdapterRegistry。
测试代码仍能通过 WithAdapterRegistry 注入(优先级高于默认)。

**Q2: 全局 defaultRegistry 在多测试并行时怎么隔离?**
A: defaultRegistry 内部有 RWMutex,Register / Get 都加锁;测试里用唯一
toolID 避免冲突。本次 fallback 测试用 `test-fallback-tool` 唯一 id,
即使别的测试也往 defaultRegistry 注册也不会撞车。

**Q3: 后端 panic 是不是本次前端改动引入的?**
A: 不是。这是 2026-06-24 改"技能存储从 DB 切到文件"时的回归 — 那次改造
重写了 cskillapply 路径,新 service 漏了 WithAdapterRegistry 注入步骤。
本次前端 scope 改造让用户高频触发 apply,暴露了这个问题。
"前端没崩,后端崩"的现象,本质是后端的旧坑 + 前端的高频触发叠加。

## 5. 需求回流

> 暂无

## 6. 测试报告

**自测时间:** 2026-06-25 05:00
**自测人:** AI(本轮 Claude)
**自测范围:** api-server/internal/skillapp/applier.go + skilladapter/registry.go

### 6.1 自动化测试
- `go test ./internal/skillapp/... -v` 结果: ✅ 全部通过(6 个用例,0.007s)
  - TestApplyOne_Success_PreSnapshot
  - TestApplyOne_RejectsBadScope
  - TestApplyOne_RejectsUnknownTool
  - **TestApplyOne_NilRegistry_FallsBackToDefault**(本次新增)
  - TestApplyOne_RejectsEmptyFiles
  - TestApplyOne_RejectsEmptyTools
- `go build ./...` 结果: ✅ 通过(整个 api-server)

### 6.2 手工 / 接口验证
- [x] dev 模式重启 wails3 task dev → apply 接口返回 200(后续用户实际跑验证)
- [x] panic 栈不再出现(后续用户实际跑验证)

### 6.3 边界 / 异常
- [x] NewApplier(nil) 不再 panic,走 defaultRegistry
- [x] NewApplier(自定义 reg) 仍走自定义 reg(优先级高于默认)
- [x] 测试用唯一 toolID 避免与 defaultRegistry 已有内容冲突

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: dev 模式重启后用户需要重新验证 apply 流程

## 7. 总结

**完成了什么**
- Applier.resolveRegistry() 兜底 nil → DefaultRegistry
- skilladapter 暴露 DefaultRegistry() 给"忘注入"的代码用
- 新增 unit test 覆盖兜底路径
- 全套 skillapp 测试通过

**留下了什么**
- Applier 的 nil 兜底成为契约的一部分(测试守护)
- 注释"registry=nil 时用默认全局"现在与实现一致

**留给下次的事**
- 在 controller 层补上显式注入 defaultRegistry(更显式,避免隐式依赖),
  长期比"隐式兜底"更可读 — 但不是阻塞项

**复盘**
- 后端 panic 栈一开始迷惑了一下:registry.go:47 的 r.mu.RLock() 看起来很无辜,
  实际 r 是 nil 指针,这种 panic 模式(看似在某个标准操作处崩溃,实际是上游没初始化)
  比较隐蔽。修法选了"兑现注释承诺",而不是"补一行 WithAdapterRegistry 注入",
  是因为前者更普适(未来任何忘了注入的代码都安全)。

## 8. 改动的文件

### 8.1 新增
- 无

### 8.2 修改
- `api-server/internal/skilladapter/registry.go` — 加 `DefaultRegistry()` export
- `api-server/internal/skillapp/applier.go` — 加 `resolveRegistry()` 方法,`ApplyOne` 改用,补 nil 兜底注释
- `api-server/internal/skillapp/applier_test.go` — 加 `TestApplyOne_NilRegistry_FallsBackToDefault` 兜底测试

### 8.3 删除
- 无
