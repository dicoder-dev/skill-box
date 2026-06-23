---
name: 高风险操作必须确认
description: 涉及删除 / 推送 / 强推 / 强 reset 等不可逆动作前必须先确认
type: feedback
---

## 规则

只有 **删除文件 / 强推 / 强 reset / 跳过 hooks / 强 --force 等不可逆操作**
需要先跟用户确认。其他任何操作(写文件、跑测试、跑构建、调 MCP、git add/commit)
**直接做,无需确认**。

**Why:** 用户的全局 CLAUDE.md 明确写了这条 —— 删除类操作的代价高(丢失工作),
其他操作的代价低(可读 git diff / 可重跑)。

**How to apply:** 见 `Bash` 命令的危险性表:

| 命令 | 处置 |
| --- | --- |
| `rm <文件>` / `rm -rf <目录>` | **先确认** |
| `git reset --hard` / `git push --force` / `git checkout --` | **先确认** |
| `git commit --no-verify` / `--no-gpg-sign` | **先确认** |
| `git push`(普通) | 直接做 |
| `git add` / `git commit` / `git status` / `git diff` / `git log` | 直接做 |
| `go test` / `npm install` / `wails3 task ...` | 直接做 |
| 调 MCP 工具(understand_image / web_search 等) | 直接做 |

边界情况:批量删除多个文件也要确认,不要"用 `replace_all` 一键删"绕过。