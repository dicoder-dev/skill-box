# 工具概念统一改名为 Agent

**日期:** 2026-10-30
**状态:** 已完成

## 1. 需求

用户原话：「项目上的定义"工具"统一改名为 "Agent"」。

澄清后确认的范围（用户选择）：

- **改名范围 = 文案 + 代码标识符**（不含数据库表名 / 列名）
- **对外称呼 = 统一叫 "Agent"**（不用「AI Agent」）

即：项目里代表「AI 编程宿主（Claude Code / Cursor / Codex 等）」的 "Tool / 工具" 概念，
统一改名为 "Agent / Agent 工具"。DB 表名与列名保持不变以免动用户数据。

## 2. 任务列表

- [x] 后端：目录 / 包改名（mtool→magent、stool→sagent、ctool→cagent、toolseed→agentseed、toolspecs→agentspecs）
- [x] 后端：entity / model / service / controller 标识符改名
- [x] 后端：HTTP 路由 `/api/skillbox/tools/*` → `/api/skillbox/agents/*`
- [x] 后端：下游消费方（skillapp / skillimporter / skillpkg / skillstore / market / scope_status 等）跟随改名
- [x] 前端：ToolsView→AgentsView、ToolIcon→AgentIcon、store/tools→agents、api 文件改名
- [x] 前端：i18n 中文文案「工具」→「Agent」，英文同步
- [x] 文档：memory 新增改名决策记录 + 同步索引
- [x] 自测：`go build` / `go vet` / `go test` + `npm run build` + 前后端等价性校验
- [x] 提交并推送

## 3. 执行进度

- 10:30 调研完成：后端 / 前端「工具」概念分布摸清
- 10:32 与用户确认改名范围（文案 + 代码标识符，DB 表名不变，统一叫 Agent）
- 11:00 后端完成：目录 / 包 / 实体 / 路由 / JSON tag 全量改名，DB 契约保持
- 11:40 前端第一次批量替换 → **出现多处误伤**（详见 §4.1），回退重来
- 12:10 前端用「保护占位符 → 整串替换 → 还原」重做，改为 Node 脚本，避免 Perl 嵌套替换陷阱
- 12:30 前后端等价性校验通过（工作区 == HEAD 按同样规则替换的结果）
- 12:50 全量测试通过，文档同步完成

## 4. 问题与方案

### 4.1 第一版批量改名脚本翻车（最重要教训）

**现象**：Perl 脚本 `s{...}{ my $x = $1.$2.$3; $x =~ s{...}{...}g; $x }ge` + 占位符保护，
结果占位符自身被后续规则二次改写（`@@P2_tooltip@@` → `@@P2_agenttip@@`），还原失败。

**连带事故**（靠等价性校验 + 构建报错抓到）：

| 事故 | 现象 |
| --- | --- |
| `@click="toggleTheme"` → `role="@"` | Vue 模板属性被破坏 |
| `@apply ...` CSS 指令丢 `@` | PostCSS 报 `Unknown word apply`，构建失败 |
| `navigator.userAgent` → `userTool` | 浏览器 API 被改坏 |
| `~/.agents/skills` → `~/.tools/skills` | Agent Skills 标准目录被改坏 |
| `/api/skillbox/skills/global-agent/toggle` → `global-tool/toggle` | 路由与后端不一致 |
| `vercel-labs/agent-skills` → `tool-skills` | 外部仓库名被改坏 |
| `tooltipTest` → `.tooltipTest` | i18n key 失效 |

**方案**：回退前端到 HEAD，改用 Node 脚本重做 ——
① 枚举全部含 tool 的完整 token；② 用 `\u0001` 控制字符做保护占位符（不会出现在正文）；
③ 整串替换（非正则猜测）；④ **还原放在最后**；⑤ 还原后再跑一次全局等价性校验。

**沉淀**：`docs/agent/memory/tool-to-agent-rename.md`。

### 4.2 必须保持不变的外部契约

| 契约 | 值 | 原因 |
| --- | --- | --- |
| DB 表名 | `tools` / `tool_paths` | 用户已有 DB 快照，改名等于丢数据 |
| DB 列名 | `tool_id` / `tool` | 同上；Go 字段名改 AgentID，靠 `gorm:"column:tool_id"` 映射 |
| 唯一索引名 | `uniq_tool_scope_category` | 改名会触发索引重建 |
| SKILL.md frontmatter | `target_tools` | 用户自己写的 skill 文件里的字段 |
| 配置 key | `skillbox.tool_paths` | 用户 `~/.skill-box` 配置里已有 |
| 物理目录 | `~/.skill-box/tool-icons/` | 用户已有自定义图标文件 |
| 共享池路径 | `~/.agents/skills/` | Anthropic Agent Skills 开放标准 |
| 无关 tool 词 | MCP `tools/`、GitHub `tools/<name>/`、UI「工具栏」、代码生成工具 | 与本概念无关 |

### 4.3 中文「工具」二义性

「工具」既指 AI 编程工具，也指「手段 / 方式」（"判定工具"、"时间格式化工具"、"工具方法"）。
批量替换后逐条 review，手段语义改写成「方式 / 能力」或保留原文。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-10-30 12:30-13:00
**自测人:** AI（本轮 Claude）

### 6.1 自动化测试

- `go build ./...` → ✅ 通过
- `go vet ./...` → ✅ 无新增问题（仅既有 `pkg/cfg` / `pkg/httpclient` 告警，与本次无关）
- `go test ./internal/...` → ✅ 相关包全通过（agentseed / skilladapter / agentspecs / sagent / skillimporter /
  skillapp / sskill / smarket / skillbundle …）
  ⚠️ `internal/db/pgsql` + `internal/gen/db/pgsql` 的 `TestConnect` 失败：需要本地 PostgreSQL 实例，
  环境依赖，**与本次改名无关**
- `npm run build` → ✅ 通过（vite 5.4.21，9.9s）
- 前后端等价性校验脚本（工作区 == HEAD 按同一规则替换的产物）→ ✅ 逐字节相等，无额外破坏

### 6.2 手工 / 接口验证

- [x] 前后端路由一致：后端 10 条 `/api/skillbox/agents/*` + `/api/files/agent-icons/:filename`
      ↔ 前端请求路径逐条比对一致 → ✅
- [x] DB 契约保持：`tableNameAgent = "tools"` / `tableNameAgentPath = "tool_paths"`，
      `gorm:"column:tool_id"` 不变 → ✅（AutoMigrate 不会新建表）
- [x] JSON tag 改名后端 / 前端同步：`agent_id` / `agent` / `applied_agents` / `agent_paths` → ✅
- [x] 无关语义未被误伤：`target_tools`、`~/.agents/skills`、`global-agent/toggle`、
      `navigator.userAgent`、tooltip / toolbar → ✅ 等价性校验逐条确认

### 6.3 边界 / 异常

- [x] 用户已有 DB 快照：表名列名不变 → 无需迁移脚本，无数据丢失风险 ✅
- [x] 用户已有自定义图标：`~/.skill-box/tool-icons/` 目录名不变 → 图标仍可读 ✅
- [x] 用户已有 skill 文件 frontmatter `target_tools` 不变 → apply 链路不受影响 ✅

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无（pgsql 连接测试失败为环境依赖）

## 7. 总结

- **完成了什么**：后端（实体 / 包 / 目录 / 路由 / service / 下游消费方）+ 前端（视图 / 组件 /
  store / API / i18n 文案）全量把 "工具" 概念改名为 "Agent"；DB 契约与外部数据格式保持不变。
- **留下了什么**：`docs/agent/memory/tool-to-agent-rename.md`（改名决策 + 契约清单 + 批量改名翻车教训）。
- **留给下次的事**：新增内置 Agent 时，注意 DB 表名仍是 `tools`，代码里用 `entity.Agent`。
- **复盘**：批量改名必须配「等价性校验脚本」，光靠 build / grep 抓不全语义级误伤。

## 8. 改动的文件

### 8.1 新增
- `docs/agent/memory/tool-to-agent-rename.md` — 改名决策、外部契约清单、批量改名教训
- `api-server/internal/agentseed/`（原 `internal/toolseed/`）— 内置 Agent seed
- `api-server/internal/skilladapter/agentspecs/`（原 `toolspecs/`）— Agent 元数据加载层
- `api-server/internal/gapi/model/skillbox/magent/`（原 `mtool/`）— Agent 模型层
- `api-server/internal/gapi/service/agent/sagent/`（原 `service/tool/stool/`）— Agent 业务层
- `api-server/internal/gapi/service/agent/agenticon/`（原 `service/tool/toolicon/`）— 图标存储
- `api-server/internal/gapi/controller/skillbox/cagent/`（原 `ctool/`）— Agent HTTP 接口
- `api-server/internal/gapi/entity/e_agent.go` / `e_agent_path.go`（原 `e_tool*.go`）— Agent 实体
- `frontend/src/views/AgentsView.vue`（原 `ToolsView.vue`）— Agent 元数据管理视图
- `frontend/src/components/AgentIcon.vue`（原 `ToolIcon.vue`）— Agent 图标组件
- `frontend/src/core/store/agents.js`（原 `store/tools.js`）— Agent store
- `frontend/src/api/skillbox/agents.js`（原 `api/skillbox/tools.js`）— Agent 接口客户端

### 8.2 修改
- `api-server/internal/gapi/entity/e_skill_apply.go` — `Tool` → `Agent` 字段
- `api-server/internal/skilladapter/{types,base,registry}.go` — Adapter 契约 `ToolID()` → `AgentID()`
- `api-server/internal/skillapp/{types,applier,batch,undo}.go` — apply 链路字段改名
- `api-server/internal/skillimporter/importer.go`、`internal/skillpkg/*.go`、`internal/skillstore/store.go` — 跟随改名
- `api-server/internal/gapi/service/{skill,skillapp,market}/*` — 业务层跟随改名
- `api-server/internal/gapi/controller/skillbox/{cskill,cskillapply,cproject,conboarding,cmarket}/*` — 控制器跟随改名
- `api-server/cmd/bootstrap/*`、`cmd/oneoff/import_trae_skills/main.go` — 启动编排跟随改名
- `frontend/src/core/i18n/zh-CN.js`、`en-US.js` — 文案「工具」→「Agent」，key 同步改名
- `frontend/src/App.vue`、`views/{SkillsView,OnboardingView,ProjectsView,SettingsView}.vue` — 引用跟随改名
- `frontend/src/components/{TreeNode,AIDialog,GlobalImportPanel,LocalImportPanel,MarketImportPanel,OnboardingImportDialog,RichTextEditor}.vue`
- `frontend/src/components/skill/{SkillScopePanel,SkillFileInlinePanel,CodeViewer,CsvViewer}.vue`
- `frontend/src/core/{icons/iconparkMap,utils/iconUploadHandler,utils/runtime,composables/useRightPanelMode}.js`
- `docs/agent/memory/MEMORY.md`、`docs/agent/memory/project.md` — 索引与项目记忆同步

### 8.3 删除
- 无（均为重命名）

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash grep` — 摸清「工具」概念在前后端的分布
- `Bash perl` — 后端标识符 / 中文文案批量改名
- `Bash node` — 前端安全改名脚本 + 等价性校验脚本
- `Bash go build / go vet / go test` — 后端验证（12:30 全部通过，pgsql 连接测试除外）
- `Bash npm run build` — 前端构建验证（12:35 通过）
- `Bash git status / diff / log` — 核对改动与提交状态
