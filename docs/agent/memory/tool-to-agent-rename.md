# 「工具」概念统一改名为「Agent」(2026-10-30)

**背景:** 项目里代表「AI 编程宿主」(Claude Code / Cursor / Codex / Trae / …) 的概念
原名 "Tool / 工具",与 Anthropic 推行的 Agent Skills 开放标准(`~/.agents/skills/`)
术语撞车。2026-10-30 起统一改名为 **Agent / Agent 工具**。

## 改名范围(已确认)

- ✅ 用户可见文案:界面 + i18n(zh-CN / en-US)统一叫「Agent」
- ✅ 代码标识符:Go 实体 / 包 / 目录 / 函数 / 变量,前端组件 / store / API 层
- ✅ HTTP 路由:`/api/skillbox/tools/*` → `/api/skillbox/agents/*`
- ✅ JSON 字段:`tool_id` → `agent_id`、`tool` → `agent`、`applied_tools` → `applied_agents`

## ⚠️ 必须保持不变的外部契约(改动会丢用户数据)

| 契约 | 保持的值 | 原因 |
| --- | --- | --- |
| DB 表名 | `tools` / `tool_paths` | 用户已有 DB 快照,GORM 按表名建表,改名 = 丢数据 |
| DB 列名 | `tool_id` / `tool` | 同上;Go 字段名是 `AgentID`,靠 `gorm:"column:tool_id"` 映射 |
| 唯一索引名 | `uniq_tool_scope_category` | 改名会触发索引重建 |
| SKILL.md frontmatter | `target_tools` | 用户自己写的 skill 文件里的字段,改名破坏存量 skill |
| 配置 key | `skillbox.tool_paths` | 用户 `~/.skill-box` 配置里已有 |
| 物理目录 | `~/.skill-box/tool-icons/` | 用户已有自定义图标文件 |
| 共享池路径 | `~/.agents/skills/` | Anthropic Agent Skills 开放标准 |
| 无关 tool 词 | MCP `tools/`、GitHub `tools/<name>/`、UI「工具栏」、代码生成工具 | 与本概念无关 |

## 关键目录 / 包对照

| 改名前 | 改名后 |
| --- | --- |
| `internal/toolseed` | `internal/agentseed` |
| `internal/skilladapter/toolspecs` | `internal/skilladapter/agentspecs` |
| `internal/gapi/model/skillbox/mtool` | `internal/gapi/model/skillbox/magent` |
| `internal/gapi/service/tool/stool` | `internal/gapi/service/agent/sagent` |
| `internal/gapi/service/tool/toolicon` | `internal/gapi/service/agent/agenticon` |
| `internal/gapi/controller/skillbox/ctool` | `internal/gapi/controller/skillbox/cagent` |
| `entity.Tool` / `entity.ToolPath` | `entity.Agent` / `entity.AgentPath` |
| 前端 `views/ToolsView.vue` | `views/AgentsView.vue` |
| 前端 `components/ToolIcon.vue` | `components/AgentIcon.vue` |
| 前端 `core/store/tools.js` | `core/store/agents.js` |
| 前端 `api/skillbox/tools.js` | `api/skillbox/agents.js` |

## 教训(批量改名必读)

1. **不要用「一个正则 + 嵌套 s/// + /e」做批量改名。** 本次第一版脚本用
   `s{...}{ my $x = $1.$2.$3; $x =~ s{...}{...}g; }ge` + 占位符保护,
   结果占位符本身被后续规则二次改写(`@@P2_tooltip@@` → `@@P2_agenttip@@`),
   还原失败,连带把 `@click=` 改成 `role="@"`、`@apply` 指令吃掉 `@`、CSS `@apply` 编译失败。
2. **可靠做法:** ① 先枚举文件里所有含目标词的**完整 token**;② 用「保护占位符 →
   整串替换 → 还原」三步,占位符用**不会出现在正文的字符**(如 `\u0001`);③
   还原**放在最后一步之后**再跑一次全局校验。
4. **改完必须做「反向/正向等价校验」**:把 HEAD 版本按同样规则做一次替换,结果应与
   工作区**逐字节相等**。不等的地方就是误伤。本项目靠这个校验抓出了
   `navigator.userAgent` → `userTool`、`~/.agents/skills` → `~/.tools/skills`、
   `vercel-labs/agent-skills` → `tool-skills` 等事故。
5. **中文「工具」二义性**:"工具"既指 AI 编程工具,也指"手段/方式"(如"判定工具"、
   "时间格式化工具")。批量替换后必须逐条 review 语义。
