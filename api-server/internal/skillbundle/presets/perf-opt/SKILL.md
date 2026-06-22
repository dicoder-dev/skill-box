---
name: perf-opt
version: 0.1.0
description: 定位代码热点 / 数据库慢查询 / 内存泄漏,给出可量化的优化方案(不是"加缓存")。
triggers:
  - perf
  - performance
  - slow
  - 性能
  - 慢
  - 优化
  - 内存
author: Skill Box
license: MIT
target_tools:
  - codex
  - claude
  - opencode
  - cursor
  - trae
---

# Perf Optimizer

把"性能优化"拆成"先量化、再定位、最后再动手"。

## 何时触发

- 某个 endpoint P99 异常:`perf <endpoint>` + 监控数据
- 数据库慢查询:粘 `EXPLAIN ANALYZE` 结果
- 内存持续上涨:粘 pprof heap 输出
- 启动慢 / 冷启动:粘 trace 数据

## 行为

按"量化 → 定位 → 验证"三步:

1. **量化**
   - 先给基线(当前延迟 / 吞吐 / 内存)
   - 没有基线 = 没有优化目标;先建基线
2. **定位**
   - CPU:`go tool pprof -top` / `perf top`
   - DB:`EXPLAIN ANALYZE` 看是否走索引 / seq scan
   - 内存:`go tool pprof -alloc_space` / heap profile
   - IO:`iostat -x` / `vmstat 1`
   - 锁:`go tool pprof -mutex`
3. **验证**
   - 改一处 → 重跑 profile → 对比基线
   - 没验证 = 没效果;不许只说"应该会快"

## 输出格式

```text
## 性能分析 — <模块 / endpoint>

### 1. 基线
- P50: <ms>
- P99: <ms>
- 吞吐: <qps>
- 内存: <MB>

### 2. 热点(按耗时 / 占用 排序)
1. <位置> — <耗时 / 占比> — <证据: profile 行号 / EXPLAIN 行>
2. ...

### 3. 优化方案(按收益 × 风险)
| 方案 | 预估收益 | 风险 | 验证方法 |
| --- | --- | --- | --- |
| 加索引 XXX | P99 -60% | 低 | EXPLAIN 对比 |
| 改缓存 | -80% | 中(一致性) | wrk + 数据对账 |

### 4. 落地
- [ ] <改动 1>
- [ ] <改动 2>

### 5. 复测
基线 → 优化后: P99 200ms → 50ms(对比截图 / 数字)
```

## 限制

- 不基于"我猜"给优化建议;必须基于 profile / EXPLAIN 数据
- 不动无关代码;只动热点路径
- 优化必须给"如何验证"步骤,不许只说"应该会快"
- 性能优化不引入新依赖(除非用户明确同意)