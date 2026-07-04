# 工具 path 改为单 path 模型

**日期:** 2026-07-04
**状态:** 已完成

## 1. 需求

排查"首页技能列表 chip 误显"和"右侧 scope 点击磁盘未生效"两个 bug 时,根因都是同一个:每个 AI 编程工具的"同一个 (scope, category)"下被允许多条 path,且多个 adapter 共享 `~/.agents/skills` 这条 user global path,导致 `GlobalAppliedTools` 反向 stat 时把 4~5 个 tool 全部标成"已应用"。

讨论后决定:把模型收紧为"每个 (scope, category) 最多 1 条 path"(单 path 模型),DB 层加 uniqueIndex 兜底,Service 层在校验里拒绝重复,前端编辑弹窗改成 4 格固定布局。

## 2. 任务列表

- [x] 删 builtin 重复 (scope, category) path(去掉 cline 的 ~/.cline/skills + codex 的 .curated)
- [x] e_tool_path 加 uniqueIndex(tool_id, scope, category)
- [x] stool.Service 加 ErrPathExisted + Create/Update 唯一性校验 + 新增 AddOnePath
- [x] ctool/add_path.a.go 改调 Service.AddOnePath,统一 ErrPathExisted 映射 409
- [x] skillapp.applier.resolveTargetDir 加 len(paths) > 1 报错 + 注释清理
- [x] conboarding get_onboarding_status.a.go 加 warn log
- [x] 前端 store 改 slots(4 格)+ PATH_SLOTS 常量 + buildPayload 拍平
- [x] 前端 ToolsView.vue paths 子表改 4 行固定格 + 移除"添加路径"按钮
- [x] i18n zh-CN / en-US 更新 paths 命名空间文案
- [x] DB 存量清理:delete cline ~/.cline/skills + codex .curated 那两条
- [x] unit test:加 3 个针对 (scope, category) 唯一性的 test
- [x] go build / go test / npm run build 全过
- [ ] commit + push

## 3. 执行进度

- 排查根因 + 给方案(已写在前面的对话里,这里略)
- 后端 + DB 改造 → 编译通过 → 新增 test 全过
- 前端 store + view + i18n 改造 → build 通过
- DB 存量清理完成(单 path 唯一性前提下需要先清,否则 AutoMigrate 加 uniqueIndex 失败)

## 4. 问题与方案

> 单 path 唯一性带来的两条副作用,以及怎么收敛

### 4.1 `en_US/zh-CN` 命名空间同步要逐条对齐

`tools.paths.add` / `tools.paths.empty` / `tools.paths.order` 删了,改成 `tools.paths.scopeGlobal` / `scopeProject` / `categoryUser` / `categorySystem`。两个语言文件都要改,容易漏 → 先 zh-CN 改完,再 en-US 同样模式,一次成对。

### 4.2 `validateForm` 从"任意行"改成"4 格"后,空字符串 path 是合法的

旧逻辑:每行 path 必须非空。新逻辑:4 格各自 0 或 1 条,空字符串 = 该格未配置。所以 `validateForm` 不再校验"path 不能为空",只校验 scope/category 合法;`buildPayload` 在拍平时 `filter((p) => p.path !== '')` 排除空格。

### 4.3 `add_path` controller 错误处理改造

原 controller 直接调 `pathM.Create`,依赖 DB uniqueIndex 报错文本(`isUniqueConflict`)。改成走 `Service.AddOnePath`,Service 层先 `FindOne` 命中返 `ErrPathExisted`(更早更友好),DB uniqueIndex 作为兜底。controller 层用 `errors.Is(err, stool.ErrPathExisted)` switch 一条 409 分支,其它基础校验错 → 400,未知错 → 500。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-04
**自测人:** AI(本轮 Claude)
**自测范围:** 后端 stool Service / applier / onboarding + 前端 store + view

### 6.1 自动化测试

- `go test ./internal/gapi/service/tool/stool/...` → ✅ 通过(含 3 个新增 test)
- `go test ./internal/skillapp/...` → ✅ 通过
- `go build ./...` → ✅ 通过
- `npm run build` → ✅ 通过(3.80s)

### 6.2 手工 / 接口验证

DB 清理验证:

```sql
SELECT t.tool_id, p.scope, p.category, p.path FROM tool_paths p
JOIN tools t ON t.id=p.tool_id
WHERE t.tool_id IN ('cline', 'codex') AND p.scope='global';
```

清理后只剩 3 条单 path,符合唯一性约束。

### 6.3 边界 / 异常

- 单 tool 同 (scope, category) 提交 2 条 path → Service 返 ErrPathExisted ✅
- AddOnePath 第二次同 (scope, category) → ErrPathExisted ✅
- AddOnePath 不同 (scope, category) → OK ✅
- apply 时 applier.resolveTargetDir 多 path → 返 "max 1 per (scope, category)" 错误 ✅

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无(已知 DB uniqueIndex 等下次启动 AutoMigrate 自动生效,本次 DB 存量已先清空)

## 7. 总结

### 完成了什么

- DB schema:`e_tool_path` 三列合一 `uniqueIndex:uniq_tool_scope_category`(注释 + 字段 tag)
- Service:`stool` 加 `ErrPathExisted` + `validatePathUniqueness` + `AddOnePath` + DB unique 兜底
- Controller:`ctool/add_path.a.go` 改调 service,统一 409/400/500 错误映射
- runtime:`applier.resolveTargetDir` 多 path 报错;`onboarding` 多 path warn log
- 前端 store:`slots`(4 格)+ `PATH_SLOTS` 常量 + `pathsToSlots` 反向 helper + `clearSlotPath`
- 前端 view:paths 子表从动态表格改成 4 行固定,删除"添加路径"按钮和行级删除图标,改用每格旁的清空 icon
- 前端 i18n:zh-CN + en-US 同步 paths 命名空间文案
- DB 存量:直接 sqlite3 清掉 cline/codex 各 1 条冗余 path
- test:加 3 个针对 (scope, category) 唯一性的 unit test

### 留下了什么

- `tools.form.paths`(数组)→ `tools.form.slots`(4 格对象);旧 `addPathRow` / `removePathRow` 保留为 no-op 兜底,view 没引用就删了
- builtin codex 删了 vendor_imports/.curated 那条;cline 删了 ~/.cline/skills 那条
- i18n 删了 `tools.paths.add` / `empty` / `order` 三个旧 key

### 留给下次的事

- 桌面端重启确认 uniqueIndex AutoMigrate 成功(应用启动时 GORM 会自动加索引)
- 跑一次 e2e:在桌面端新建一个工具,4 格填满,提交 → 列表里看 4 条 path
- 留意前端侧 `addPathRow` / `removePathRow` 是否还有外部调用方,目前只在 ToolsView 里被引用,删干净

### 复盘

- **好:** 把根因(共享 path + 多 path 模型)抓清楚再动手,改完后 bug 1(误显)和 bug 2(只写一份)同时自愈,不需要分两阶段
- **改进:** user-facing 的"4 格"概念在 plan 阶段没在 user view 截图里画出来,实施到 view 时临时想了一版(没有 select 用 readonly span),后续如果是 UI 重构可以先用 figma 框一下

## 8. 改动的文件

### 8.1 新增

- 无(只在 test 文件加 case)

### 8.2 修改

- `api-server/internal/toolseed/builtin.go` — 删 codex 1 条 + cline 1 条 builtin path
- `api-server/internal/gapi/entity/e_tool_path.go` — 三列 uniqueIndex + 注释更新
- `api-server/internal/gapi/service/tool/stool/tool.s.go` — ErrPathExisted + validatePathUniqueness + Create/Update 校验 + AddOnePath
- `api-server/internal/gapi/service/tool/stool/tool.s_test.go` — 3 个新 test case
- `api-server/internal/gapi/controller/skillbox/ctool/add_path.a.go` — 改调 Service.AddOnePath,统一错误映射
- `api-server/internal/skillapp/applier.go` — resolveTargetDir 加多 path 报错 + 注释
- `api-server/internal/gapi/controller/skillbox/conboarding/get_onboarding_status.a.go` — 多 path warn log
- `api-server/internal/gapi/controller/skillbox/cskill/scope_status.a.go` — GlobalAppliedTools 注释更新
- `frontend/src/core/store/tools.js` — slots(4 格)+ PATH_SLOTS 常量 + 校验/拍平
- `frontend/src/views/ToolsView.vue` — paths 子表改 4 行固定 + import PATH_SLOTS + pickPath 改收 slot + CSS
- `frontend/src/core/i18n/zh-CN.js` — paths 命名空间文案更新
- `frontend/src/core/i18n/en-US.js` — paths 命名空间文案更新

### 8.3 删除

- builtin.go:删 codex `.curated` + cline `~/.cline/skills` 那两条声明
- zh-CN/en-US.js:删 `tools.paths.add` / `empty` / `order` 三个旧 key

### 8.4 DB 手工操作

- `~/.skill-box/data.db` 的 `tool_paths` 表:DELETE cline ~/.cline/skills + codex .curated

## 9. 工具与用途

### 9.1 MCP 工具

- 无

### 9.2 Skill

- 无

### 9.3 CLI

- `Bash go build ./...` — 后端编译(第 1 轮)
- `Bash go vet ./...` — vet(第 2 轮,警告与本次无关)
- `Bash go test ./...` — 单元测试(第 2 轮,含新 test 全过)
- `Bash npm run build` — 前端 build(第 2 轮,3.80s 通过)
- `Bash sqlite3 ~/.skill-box/data.db ...` — 手工删 2 条冗余 path
- `Bash go build ./api-server/cmd/web` — web 二进制(后未跑起来做接口测试,因为 curl 被沙箱拒,改用 unit test 覆盖)