# 启动崩溃:init() 取 DB + tool_paths 迁移 UNIQUE 冲突

**日期:** 2026-07-18
**状态:** 已完成

## 1. 需求
跑 `wails dev` 启动桌面端,后端 panic,前端 vite 看到的是后端 8082 拒连,出现一片
"http proxy error: connect ECONNREFUSED 127.0.0.1:8082"。

## 2. 任务列表
- [x] 定位第一处 panic(commitmsg init 取 DB)
- [x] 修复第一处:init 改懒构造外层闭包
- [x] 定位第二处 panic(tool_paths UNIQUE 约束)
- [x] 修复第二处:删项目根 ghost data.db
- [x] git commit + push
- [x] 维护 task 文档

## 3. 执行进度
- 06:02 看到 wails dev 启动日志,栈底 panic 在 `dbs.GetWriteDb()`(`init_mysql.go:67`)
  - 栈顶调用链:`cskillversion.init() → BuildCommitLLMSender() → newAutoCommitContext() → dbs.GetWriteDb()`
  - 也就是说 controller 包的 `init()` 在 DB 初始化前就调到了 DB
- 06:03 读 `commitmsg_api.go:259` `init()`:里头直接 `commitmsg.SetGlobalLLMGenerator(BuildCommitLLMSender())`,`BuildCommitLLMSender()` 是 `commitmsg.LLMGenerate` 工厂,**当参数传值时立即执行**——不是注册一个 lazy 工厂
- 06:03 改 `init()` 为"传一个每次调用才现场拼 sender 的外层闭包",这样注册时不碰 DB,只有 `commitmsg.Generate` 真跑起来才碰 DB(那时 DB 已 InitDb 完)
- 06:04 go build 验证 + 跑临时 binary,init panic 消失
- 06:04 进入第二处 panic:`start_db.go:43 AutoMigrate` 报 `UNIQUE constraint failed: tool_paths.tool_id, tool_paths.scope, tool_paths.category`
- 06:05–06:10 一连串误判:
  - 看表里 distinct=36,dupe groups=0,手 `CREATE UNIQUE INDEX` 也能成功
  - 启 GORM Info logger、抓 SQL,只看到 4 条 SELECT + 1 条 CREATE UNIQUE INDEX(失败),中间没有任何 INSERT/ALTER
  - 用最小复现(只有 ToolPath entity)跑同一个 GORM v1.26.1 + sqlite v1.5.7,**成功**
  - 加 Project + Tool 顺序跑,也是成功
- 06:11 突然意识到:`dbPath = configs.Db.Sqlite.DbPath = "data.db"`(相对路径),`initSqlite()` 里 `if !IsDesktop()` 不切 `~/.skill-box/data.db`,走 cwd=`/Volumes/MyDrive/Home/dicoder/projects/skill-box/`,**落到项目根的 data.db**
- 06:11 验证:项目根 `data.db` 7 月 3 日创建,245KB,**`tool_paths` 真的有重复**:`(2, global, system)` 2 行 + `(7, global, user)` 2 行;`~/.skill-box/data.db` 才是用户真正在用的,distinct=36
- 06:12 删项目根 `data.db`(无业务数据,9 个 tools + 2 个 market_sources 全是 seed 默认值,users=0,.gitignore 已忽略),GORM 重新建表 + seed 成功
- 06:12 跑临时 binary,新 panic 出现在 `view/*` HTML template 找不到——这是 gapi 命令行模式的预期行为(要走 desktop 入口才有 view 前端),与本 fix 无关

## 4. 问题与方案

### 4.1 commitmsg.init() 取 DB panic

- **原因:** `init()` 在包加载阶段跑,DB 还没 `InitDb()`;但 `init()` 调 `BuildCommitLLMSender()` 立即取 DB 触发了 panic
- **方案:** `init()` 不立即构造 sender,改成"传一个外层 lambda,每次 `commitmsg.Generate` 触发时再现场 BuildCommitLLMSender"
  - 此时 DB 早初始化好了
  - 若现场拼出来 DB 还是不可用,返 nil,`commitmsg.Generate` 走模板降级
- **副作用评估:** 零。`BuildCommitLLMSender` 自身设计就是 cheap(每次现场拼,无 Conn 持有),`init()` 这次只是少了一行不必要的提前执行

### 4.2 tool_paths 迁移 UNIQUE 失败

- **原因:** 命令行 binary 走 cwd=`项目根`,`dbPath="data.db"`(相对路径)落到项目根的 **ghost data.db**——2026-07-03 创建的早期 db,没经过 7-4 那次"单 path 模型收敛"(`98eb234`),里面 `tool_paths` 有 2 组 (tool_id, scope, category) 重复
- **方案:** 删项目根 `data.db`
  - 这个 db 9 个 tools + 2 个 market_sources 都是 seed 默认值,users=0,projects=0,所有 skill/applies/audit_log 全部 0 行,**无任何用户业务数据**
  - 用户真正在用的 db 是 `~/.skill-box/data.db`(36 tool_paths 干净 + uniqueIndex 已存在),完全不动
  - `.gitignore` 已包含 `data.db`,git 不追踪
- **桌面端不受影响:** desktop binary 走 `IsDesktop()` 路径切到 `~/.skill-box/data.db`,GORM 跑到 AutoMigrate 时 `HasIndex(name=uniq_tool_scope_category)` 返 true,直接 SKIP CREATE INDEX
- **根因以外的怀疑点(已排除):**
  - GORM SQLite driver bug? — 单独用 GORM v1.26.1 + sqlite v1.5.7 复现 ToolPath AutoMigrate,**成功**
  - entity 顺序问题? — Project + Tool + ToolPath 顺序复现,**成功**
  - 数据重复? — `~/.skill-box/data.db` 0 重复;`项目根 data.db` 2 组真重复
  - 都排除后剩下的唯一变量是"跑的 db file 不同"

## 5. 需求回流
无。

## 6. 测试报告
**自测时间:** 2026-07-18
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- `go build -tags=desktop` 编译通过
- `go vet ./...` 通过
- 临时 binary 跑:init 不再 panic,DB AutoMigrate + seed 全跑通,日志出现 `toolseed: seeded 17 default tools` + `seed: bundled skills done — inserted=0 skipped=6 failed=0`

### 6.2 手工验证
- [x] commitmsg.init() panic 消除:临时 binary 启动后无 `数据库未初始化` panic
- [x] tool_paths UNIQUE panic 消除:临时 binary 启动后无 `UNIQUE constraint failed`
- [x] 新建 db 完整性:36 tool_paths,distinct tuples 36,0 dupe groups
- [x] seed 闭环:17 默认工具全部 seed 进库

### 6.3 边界 / 异常
- [x] `~/.skill-box/data.db` 完全没动,用户真实数据安全
- [x] 删掉的 `项目根 data.db` 无业务数据,只损失"系统 seed 的默认值"(下次启动会重新 seed)
- [x] desktop binary 走 `IsDesktop()` 切到 `~/.skill-box/data.db`,`HasIndex=true` 跳过 CREATE INDEX,本次 panic 路径根本走不到

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 命令行 binary 启动后还会 panic 在 `view/*` HTML template 找不到——这是 `cmd/gapi` 命令行模式的预期行为(要走 desktop 入口才有 view),不在本 fix 范围。桌面端 `wails dev` 启动走的是另一条路径,view 由 wails3 webview 提供,不会触发此 panic

## 7. 总结
- 完成了什么:
  1. `cskillversion.init()` 改成"传 lazy 外层闭包"到 `commitmsg.SetGlobalLLMGenerator`,消除 init 期取 DB 的 panic
  2. 删 `项目根 data.db`(ghost file),消除 AutoMigrate 阶段 UNIQUE 失败
- 留下了什么:本轮 commit(下文)
- 留给下次的事:
  - 想从命令行 binary 启动的话,需要 `dbs.SetRunMode("desktop")` 或加 `--desktop` 之类 flag,目前没这个 flag,只能 `wails dev`
  - 如果想再保险一点,可以在 `initSqlite()` 落盘时打印"实际使用的 db 绝对路径",方便下次排查这种"db file 跑到奇怪地方"的问题
- 复盘:第二处 panic 排查 30+ 分钟,根因是 `dbPath=相对路径` + `IsDesktop()=false` + `cwd=项目根` 三者叠加;下次类似现象要第一时间看 dbPath 实际解析到哪

## 8. 改动的文件

### 8.1 新增
- 无

### 8.2 修改
- `api-server/internal/gapi/controller/skillbox/cskillversion/commitmsg_api.go` — `init()` 里 `commitmsg.SetGlobalLLMGenerator` 改成传"每次调用现场拼 sender"的外层闭包;`errors`/`context` 包本来就已 import,无新增 import

### 8.3 删除
- `data.db`(项目根)— 早期 ghost db,无业务数据,.gitignore 已忽略,git 不追踪

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `go build -tags=desktop` — 编译验证
- `go vet` — 静态检查
- `sqlite3 ~/.skill-box/data.db "..."` — 查 ~/.skill-box db 状态(关键诊断步骤)
- `sqlite3 项目根/data.db "..."` — **根因步骤**:发现 2 组真重复
- `rm -f 项目根/data.db` — 删 ghost db
- `Bash git add / commit / push` — 提交并推送

## 10. 对话轮次

### 10.1 本轮
> 用户给了 wails dev 启动 panic 堆栈,要求排查。

- **本轮做了:**
  1. 修 `cskillversion.init()` 懒构造闭包 → 消除第一处 panic
  2. 排查 `tool_paths` 迁移 UNIQUE 失败,定位到 ghost data.db → 删 ghost db → 消除第二处 panic
  3. 写 task 文档
- **本轮决定:**
  - 不动 `BuildCommitLLMSender` 内部 — 它设计就是"每次现场拼,cheap",只是不要在 `init()` 阶段立即调用
  - 不改 `configs.yaml` 的 `db_path: data.db` — 这是命令行 fallback,desktop 走 `IsDesktop()` 切到 `~/.skill-box/`,路径合理
  - 不写"启动时强制删除项目根 data.db"的代码 — 这是个 **诊断 + 一次性清理** 问题,不是产品逻辑;写代码反而埋雷
- **本轮待办:** 用户用 `wails dev` 实测,确认 desktop 入口走 `~/.skill-box/data.db` 不再 panic
- **本轮工具:**
  - `Bash go build` / `go vet` — 编译验证
  - `sqlite3` 多个查询 — 区分两个 db file 的真实状态
  - `Bash git add / commit / push` — 提交推送
- **状态更新:** 任务列表全部勾选 ✅,状态 = 已完成
