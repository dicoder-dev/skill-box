---
name: commit-msg
version: 0.1.0
description: 根据当前 staged diff 生成符合 Conventional Commits 规范的提交信息,支持中英文。
triggers:
  - commit
  - commit message
  - /commit
  - 提交
  - 提交信息
author: Skill Box
license: MIT
target_tools:
  - codex
  - claude
  - opencode
  - cursor
  - trae
---

# Commit Message Generator

按 Conventional Commits 规范自动生成 commit message。

## 何时触发

- 写完一段改动准备 commit:`/commit` 或 "生成 commit 信息"
- 已有 `git diff --staged` 输出,粘进来直接出结果
- 多人协作统一 commit 风格

## 行为

1. 读 `git diff --staged`(优先);或粘进来的 diff 文本
2. 推断 type:
   - `feat`:新功能
   - `fix`:bug 修复
   - `refactor`:重构(无功能变化)
   - `docs`:仅文档
   - `test`:仅测试
   - `chore`:构建 / 工具 / 依赖
   - `perf`:性能
   - `style`:格式(无逻辑变化)
3. scope:从改动路径推断(目录名 / 模块名)
4. 标题 ≤ 72 字符,祈使语气,首字母不大写(英文)
5. body:多段空行分隔的 bullet,解释"为什么"而不是"是什么"
6. footer:有关联 issue / breaking change 时填 `Refs:` / `BREAKING CHANGE:`

## 输出格式

```text
<type>(<scope>): <subject>

<body line 1>
<body line 2>

<footer>
```

例:

```text
feat(auth): 接入 OAuth 2.0 第三方登录

- 新增 Google / GitHub provider
- 复用现有 session 存储
- refresh token 走独立路由

Refs: #123
```

## 限制

- 不实际执行 `git commit`,只生成文本
- 不读 git config;默认走英文 subject,可在 prompt 指定 "用中文"
- breaking change 必须在 footer 显式声明
