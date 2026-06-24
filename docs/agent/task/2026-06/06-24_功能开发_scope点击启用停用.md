# scope chip 点击:启用 / 停用 skill

**日期:** 2026-06-24
**状态:** 已完成(改用现有 cskillapply 接口)

## 1. 需求

把 scope chip 从只读展示改成可点按钮,点击 = 启用/停用 skill 到对应 (tool, scope, project) 位置。

## 2. 任务列表

- [x] 后端:复用现有 `cskillapply.ApplySkill` (POST /api/skillbox/skills/apply) — 入参 `{scope, project_id, name, tools: [toolID]}`
- [x] 后端:复用现有 `cskillapply.UndoSkill` (POST /api/skillbox/skills/apply/undo) — 按 apply_id 撤销
- [x] 后端:复用现有 `cskillapply.ListApplies` (GET /api/skillbox/skills/apply/list) — 找最近一条未撤销的 apply_id
- [x] 删我误加的 `cskill/apply_skill.a.go` — 跟 cskillapply 路由冲突导致 wails 启动 panic
- [x] 前端 API client:调整 `applySkill` 入参;新增 `listApplies` / `undoApply`;删 `unapplySkill`
- [x] 前端 scope chip:`<span>` → `<button>`,加 @click 绑定
  - [x] `handleToolChipClick` — 工具行批量启用/停用
  - [x] `handleScopeChipClick` — 作用域行单点启用/停用
  - [x] `doApplyOne` — 调 applySkill
  - [x] `doUnapplyOne` — listApplies 找 apply_id → undoApply
- [x] i18n 调整 unapplyConfirm 文案(改成"走 apply/undo 还原 PreSnapshot",不是物理删)
- [x] 删 applyOverwrite 相关 i18n key(走后端内部覆盖)
- [x] `go build ./...` 通过
- [x] `npm run build` 通过
- [x] commit + push

## 3. 执行进度

- 03:10 跟用户确认三点(复制 / 同名弹确认 / 已生效删除)后,先写自定义 apply/unapply controller
- 03:15 写完 `apply_skill.a.go`、build 通过
- 03:20 写前端 handler 全部就位、build 通过、commit + push
- 03:25 用户跑 `wails3 task dev` 报 panic `handlers are already registered for path '/api/skillbox/skills/apply'`
- 03:30 排查:`cskillapply` 包已有同名路由;`apply/batch` 是 `apply` 子路径,触发 gin 路由树 panic
- 03:35 删我加的 `apply_skill.a.go`,改用 cskillapply 系列
- 03:40 调整前端 applySkill 入参(name+scope+project_id+tools[])
- 03:45 重写 unapply 流程:listApplies 找 apply_id → undoApply
- 03:50 build 双通过

## 4. 问题与方案

**问题 1:panic `handlers are already registered for path '/api/skillbox/skills/apply'`**

我加 controller 前没 `grep` 查重,跟 cskillapply 已有的同名路由冲突。

**方案:** 删我的 `cskill/apply_skill.a.go`,改用 `cskillapply` 系列(apply_skill/apply_batch/undo_skill/list_applies/check_updates)。这其实更对:skillapp 已有完整 service 层(PreSnapshot + ApplyID 回填 + audit log),比我从零写的覆盖式 + 物理 rm -rf 强多了。

**问题 2:applySkill 入参格式变了**

我之前用 `{name, tool_id, scope, project_id, force}`,cskillapply 实际用 `{scope, project_id, name, tools: [toolID]}`(tools 是数组,支持一次多工具)。

**方案:** 前端 `doApplyOne` 改入参为 `{name, scope, project_id, tools: [h.tool_id]}`。

**问题 3:unapply 怎么知道 apply_id**

scope-status 只 stat 磁盘,不知道 SkillApply 行的 apply_id。

**方案:** `doUnapplyOne` 走两步:先 `listApplies({scope, name, tool, status:'applied', size:1})` 找最近一条未撤销的 apply,再 `undoApply({apply_id})`。失败兜底:找不到 active apply 记录 → 报错。

**问题 4:覆盖确认流程不再需要**

cskillapply 走 skillapp 内部的 PreSnapshot + 原子写,同名存在时直接覆盖,前端不用弹单独的"覆盖"确认框。

**方案:** 删 `applyOverwriteTitle/Message` i18n key。

## 5. 需求回流

(暂无)

## 6. 测试报告

**自测时间:** 2026-06-24
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 后端 `go build ./...` 结果: ✅ 通过
- 前端 `npm run build` 结果: ✅ 通过(289.99 kB JS / 81.45 kB CSS,gzip 后 97.91 kB / 12.66 kB)

### 6.2 手工验证(代码 review)
- [x] 我的 `cskill/apply_skill.a.go` 已删除,路由冲突解决
- [x] `applySkill` 入参对齐 cskillapply 实际签名
- [x] `doUnapplyOne` 走 listApplies → undoApply 两步
- [x] `busyKey` 仍用 `${tool}|${scope}|${projectID}`
- [x] i18n 移除 `applyOverwrite*` 两个 key
- [x] `unapplyConfirmMessage` 文案改成"走 apply/undo 还原 PreSnapshot"

### 6.3 边界 / 异常
- [x] 找不到 active apply 记录:`scopeError` 显示"no active apply record found"
- [x] listApplies / undoApply 失败:catch 后 scopeError 显示后端错误
- [x] 跨设备 / 权限不足:走 skillapp 内部原子写,失败时 PreSnapshot 还原

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题:dev server click-through 验收待跑(用户已起 `wails3 task dev` 确认无 panic)
- 复盘: 重做 controller 前应该先 `grep -rn` 全仓搜一遍,这次没搜导致 panic 被用户报错才发现。下次开新功能先 `grep` 排查冲突再动手

## 7. 总结

- 完成了什么: scope chip 改成可点按钮,点击 = 启用/停用对应 (tool, scope, project) 位置,带删除二次确认
- 留下了什么:
  - `frontend/src/api/skillbox/skills.js` — 调 `applySkill` 入参,加 `listApplies` / `undoApply`
  - `frontend/src/views/SkillsView.vue` — span→button + 三个 click handler + busyKey + chip-busy 样式
  - `frontend/src/core/i18n/{zh-CN,en-US}.js` — 调 unapplyConfirm 文案,删 applyOverwrite
- 没留什么(已删):
  - `api-server/internal/gapi/controller/skillbox/cskill/apply_skill.a.go` — 跟 cskillapply 重复,panic 源,已删
- 留给下次的事:
  - dev server click-through
  - "已生效但无 apply 记录" 兜底:用户从外部 cp 出来的目录走 skillbox 强制接管(让用户能"补登记"apply),后续可加
- 复盘: 写新 controller 前应该 grep 查重,这次没查导致路由 panic。功能是好事,但走的是已有 service,本来直接用就行,没必要造轮子。
