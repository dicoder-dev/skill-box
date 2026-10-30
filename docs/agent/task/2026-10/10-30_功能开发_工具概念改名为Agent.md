# 工具概念统一改名为 Agent

**日期:** 2026-10-30
**状态:** 进行中

## 1. 需求

用户原话：「项目上的定义"工具"统一改名为 "Agent"」。

澄清后确认的范围（用户选择）：

- **改名范围 = 文案 + 代码标识符**（不含数据库表名 / 列名）
- **对外称呼 = 统一叫 "Agent"**（不用「AI Agent」）

即：项目里代表「AI 编程宿主（Claude Code / Cursor / Codex 等）」的 "Tool / 工具" 概念，
统一改名为 "Agent / Agent 工具"。DB 表名与列名保持不变以免动用户数据。

## 2. 任务列表

- [ ] 后端：目录 / 包改名（mtool→magent、stool→sagent、ctool→cagent、toolseed→agentseed、toolspecs→agentspecs）
- [ ] 后端：entity / model / service / controller 标识符改名
- [ ] 后端：HTTP 路由 `/api/skillbox/tools/*` → `/api/skillbox/agents/*`
- [ ] 后端：下游消费方（skillapp / skillimporter / skillpkg / skillstore / market / scope_status 等）跟随改名
- [ ] 前端：ToolsView→AgentsView、ToolIcon→AgentIcon、store/tools→agents、api 文件改名
- [ ] 前端：i18n 中文文案「工具」→「Agent」，英文同步
- [ ] 文档：docs/agent/memory、docs/project、代码注释同步
- [ ] 自测：`go test ./...` + `npm run build`
- [ ] 提交并推送

## 3. 执行进度

- 10:30 调研完成：后端 / 前端「工具」概念分布已摸清（见 §8）
- 10:32 与用户确认改名范围（文案 + 代码标识符，DB 表名不变）

## 4. 问题与方案

### 4.1 哪些东西必须保持不变（外部契约）

| 契约 | 值 | 原因 |
| --- | --- | --- |
| DB 表名 | `tools` / `tool_paths` | 用户已有 DB 快照，改名要写迁移，风险高 |
| DB 列名 | `tool_id` / `tool` | 同上；Go 字段名改 Agent 但 gorm column tag 保留旧列名 |
| SKILL.md frontmatter | `target_tools` | 用户自己写的 skill 文件里的字段，改名会破坏存量 skill |
| 配置 key | `skillbox.tool_paths` | 用户 ~/.skill-box 配置里已有 |
| 物理目录 | `~/.skill-box/tool-icons/` | 用户已有自定义图标文件 |
| 无关 tool 词 | MCP `tools/` 目录、GitHub `react-toolkit`、`工具栏`、代码生成工具 | 与本概念无关 |

## 5. 需求回流

无。

## 6. 测试报告

> 待填。

## 7. 总结

> 待填。

## 8. 改动的文件

> 待填。

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash grep` — 摸清「工具」概念在前后端的分布
