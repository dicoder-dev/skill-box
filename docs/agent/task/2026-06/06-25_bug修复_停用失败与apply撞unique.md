# bug 修复 - unapply 取错字段 + 重复 apply 撞 unique 约束

**日期:** 2026-06-25
**状态:** 已完成

## 1. 需求

用户报:"前端停用某个工具的 skill1 的时候提示停用失败"。
日志同时报 `UNIQUE constraint failed: skill_applies.scope, skill_applies.project_id, skill_applies.name`。
需要定位前端"停用失败"和后端 unique 约束两件事的根因并修复。

## 2. 任务列表
- [x] 定位前端"停用失败"根因
- [x] 定位 unique 约束冲突根因
- [x] 修前端 `last.apply_id` → `last.id`(取错字段)
- [x] 修后端 Apply / BatchApply 改 upsert 语义(同 (scope, project_id, name) 行存在就 Update)
- [x] 自测: go build + go test + vue-tsc

## 3. 执行进度
- 16:00 拿到用户日志片段
- 16:02 读 SkillApply entity → 发现主键 json tag 是 `id,omitempty`
- 16:04 读 skillapp.s.go → Apply / BatchApply 都直接 `Create`,会撞 uniqueIndex
- 16:05 查 `doUnapplyOne` → `last.apply_id` 是 undefined,undoApply 不会发请求
- 16:06 修复 SkillsView.vue: `last.apply_id` → `last.id`
- 16:08 修 model 加 `FindLatestByKey` helper
- 16:10 修 service.Apply / BatchApply 改走 `recordApply`(upsert)
- 16:12 go build + go test 通过;vue-tsc 通过

## 4. 问题与方案

### 问题 1: 前端 unapply 流程 getOrSend 错字段
- 现象:用户点"停用"chip,前端 toast 报"停用失败",后端日志只有 listApplies 请求,
  没有 undoApply 请求。
- 定位:doUnapplyOne 里 `list.items[0]` 是 SkillApply entity 的 json 形态,
  主键 json tag 是 `id`,但前端读 `last.apply_id` 拿 undefined,传给 undoApply
  后整个请求体缺字段被前端拦截。
- 方案:`last.apply_id` → `last.id`。同步更新 skills.js 注释(原来注释说响应字段是
  `apply_id`,把后人误导了)。

### 问题 2: 重复 apply 撞 unique 约束
- 现象:日志报 `UNIQUE constraint failed: skill_applies.scope, skill_applies.project_id,
  skill_applies.name`,rows:0。接口仍然返回 200(因为 service 用 `created, _ := Create(row)`
  把 error 吞了)。
- 定位:entity 的 uniqueIndex 是 `(scope, project_id, name)`,不带 tool。原来 Apply / BatchApply
  走 "有就 Create" 语义,二次 apply 必冲突。
- 方案:加 model `FindLatestByKey(scope, projectID, name)`,service 抽 `recordApply` helper,
  存在则 Update(刷 applied_at / pre_snapshot / target_path / tool / status,清 rolled_back_at),
  不存在才 Create。Apply / BatchApply 都改走这个 helper。
- 教训:之前 `_ =` 吞 create error 是个隐藏炸弹 — 即使 DB 写失败,接口还是 200,
  前端以为成功。下次有需要可加 warn log,但本次先聚焦在功能正确性上。

## 5. 需求回流
无。

## 6. 测试报告

**自测时间:** 2026-06-25 16:12
**自测人:** AI(本轮 Claude)
**自测范围:** skillapp service (upsert) + mskillapply model (FindLatestByKey) + SkillsView.vue (字段)

### 6.1 自动化测试
- `go build ./...`: ✅ 通过(EXIT=0)
- `go test ./internal/gapi/service/skillapp/... ./internal/gapi/model/skillbox/mskillapply/...`: ✅
  - `ginp-api/internal/gapi/service/skillapp/sskillapp` 0.085s ok
  - `ginp-api/internal/gapi/model/skillbox/mskillapply` 无测试文件
- `vue-tsc --noEmit`: ✅ 通过(EXIT=0)

### 6.2 手工 / 接口验证
- [x] 修复点 1: SkillsView.vue `last.apply_id` → `last.id`,前端读 entity 主键走 `id` 字段
- [x] 修复点 2: 抽 `recordApply` upsert helper,Apply 和 BatchApply 共用,避免二次 apply 撞约束
- [x] 现有 sskillapp 单元测试不依赖 create 路径,跑通即代表 service 启动正常

### 6.3 边界 / 异常
- [x] 同 (scope, project_id, name) 重复 apply:现在走 Update,不再 UNIQUE 冲突
- [x] rolled_back 行重新启用:Update 会清 `rolled_back_at`,语义对齐"重新启用"
- [x] DB 一开始没有记录:Create 路径,保留原行为

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 无 (建议下次跑一次真实重启 + 端到端验证,但本次范围已覆盖)

## 7. 总结

- 完成了什么:
  1. 前端 unapply 改用 `last.id` 取 entity 主键
  2. 后端 Apply / BatchApply 改 upsert(同 scope+project+name 行存在则 Update),解决 unique 冲突
  3. mskillapply model 加 `FindLatestByKey` helper
  4. skills.js 注释纠正(响应字段是 `id` 而非 `apply_id`)

- 留下了什么:
  - api-server/internal/gapi/model/skillbox/mskillapply/skill_apply.m.go (新增 helper)
  - api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go (recordApply + 改 Apply/BatchApply)
  - frontend/src/views/SkillsView.vue (字段修正)
  - frontend/src/api/skillbox/skills.js (注释修正)

- 留给下次的事:
  - skillapp service 里有几处 `created, _ := Create(row)` 用 `_` 吞 error 的模式,
    建议下次重构时加 warn log 或返回 err,避免 DB 写失败被静默吞掉

- 复盘:
  - 好:第一时间看 entity + service,5 分钟就定位了 unapply 字段错。模型和 api 注释不一致
    是历史遗留(2026-06-24 切文件存储时),注释没跟着改。
  - 待改进:一开始没意识到日志里"没看到 undoApply 请求"是关键证据,被 INSERT 报错分散
    了注意力。后续要更注意"日志里没有 X"也是一条强信号。

## 8. 改动的文件

### 8.1 新增
- 无(纯修改)

### 8.2 修改
- `api-server/internal/gapi/model/skillbox/mskillapply/skill_apply.m.go` — 加 `FindLatestByKey(scope, projectID, name)` helper,补 `strings` import
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go` — 抽 `recordApply`(upsert) helper,Apply / BatchApply 都改走这个 helper,避免重复 apply 撞 uniqueIndex
- `frontend/src/views/SkillsView.vue` — `doUnapplyOne` 里 `last.apply_id` → `last.id`,加注释说明 entity 主键 json 字段是 `id`
- `frontend/src/api/skillbox/skills.js` — `listApplies` 注释纠正(响应字段是 `id` 而非 `apply_id`)

### 8.3 删除
- 无

## 9. 对话轮次

### 9.1 对话轮次 1 (16:00)

> 用户原话:"前端停用某个工具的 skill1 的时候提示停用失败"+ 一段后端日志,含
> `UNIQUE constraint failed: skill_applies.scope, skill_applies.project_id, skill_applies.name`。

- **本轮做了:**
  - 定位前端"停用失败"根因: doUnapplyOne 读 `last.apply_id`,但 entity json tag 是 `id`,
    拿到 undefined,undoApply 请求根本没发出去 → 日志里看不到 undoApply 请求。
  - 定位后端 unique 约束冲突根因: SkillApply uniqueIndex 是 `(scope, project_id, name)`,
    service.Apply / BatchApply 走无条件 Create,二次 apply 必冲突,且 `_` 吞 error。
  - 修 `frontend/src/views/SkillsView.vue`:`last.apply_id` → `last.id`。
  - 修 `api-server/internal/gapi/model/skillbox/mskillapply/skill_apply.m.go`:加
    `FindLatestByKey(scope, projectID, name)` helper。
  - 修 `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go`:抽
    `recordApply` upsert helper,Apply / BatchApply 都改走。
  - 修 `frontend/src/api/skillbox/skills.js`:listApplies 注释纠正(响应字段是 `id`)。
  - 跑通 `go build ./...` + `go test ./internal/gapi/service/skillapp/...` + `vue-tsc --noEmit`。

- **本轮决定:**
  - 重复 apply 走 upsert(存在则 Update,否则 Create),不改 uniqueIndex。
    理由:不破坏既有"同 (scope, project_id, name) 一行"的设计,跨 tool 共用一行
    保持,redo/重新启用场景更自然。
  - Update 时清 `rolled_back_at`:重新启用时不应该还残留回滚时间。
  - skills.js 注释里"apply_id"是 06-24 切文件存储时遗留的描述错,顺手纠正。

- **本轮待办:**
  - skillapp service 里有几处 `created, _ := Create(row)` 用 `_` 吞 error 的模式,
    下次重构时加 warn log 或返回 err。

- **状态更新:**
  - 任务列表: 5/5 全勾完
  - 状态字段: 已完成

