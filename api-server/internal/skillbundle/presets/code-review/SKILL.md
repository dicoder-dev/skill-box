---
name: code-review
version: 0.1.0
description: 对当前 diff 做静态代码审查,聚焦可读性、潜在 bug 与命名一致性,给出可执行的修复建议。
triggers:
  - review
  - code review
  - /review
  - 审查
  - 代码审查
author: Skill Box
license: MIT
target_tools:
  - codex
  - claude
  - opencode
  - cursor
  - trae
---

# Code Review

针对当前 git diff(或指定文件)做静态代码审查。

## 何时触发

- 提交前自检:`/review` 或 "review this diff"
- 同事 PR review:粘一段 diff + "请审查"
- 重构后回归:粘改动范围 + "再 review 一下"

## 行为

1. 读 diff(优先 `git diff` 范围;无则读全文件)
2. 按 4 个维度评估:
   - **可读性**(0-5):命名 / 函数长度 / 嵌套深度 / 注释合理性
   - **正确性**(0-5):边界条件 / 错误处理 / 并发安全
   - **一致性**(0-5):与项目已有风格的偏离
   - **可维护性**(0-5):耦合度 / 单元测试覆盖 / 文档完整性
3. 给出每条问题:
   - 行号 / 文件
   - 维度分类
   - 严重程度(blocker / major / minor / nit)
   - 修复建议(可粘贴的代码片段)
4. 末尾给 1 段总结(整体评分 + 关键风险)

## 输出格式

```text
## Code Review — <file or scope>

### 评分
- 可读性:4 / 5
- 正确性:3 / 5
- 一致性:5 / 5
- 可维护性:4 / 5

### 问题清单
- [major] path/to/file.go:42 — 错误处理吞掉了具体错误信息
  ```go
  // before
  if err != nil { return err }
  // after
  if err != nil { return fmt.Errorf("load config: %w", err) }
  ```

### 总结
整体质量良好,主要风险是错误处理粒度不足。
```

## 限制

- 不执行代码,只静态分析
- 大文件(> 2000 行)只 review 改动的范围
- 不修改文件,只给建议
