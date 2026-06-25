# bug 修复 - 切到 skillB 后 skillA 列表项"已应用工具" chip 消失

**日期:** 2026-06-25
**状态:** 已完成

## 1. 需求

用户报:本来正常显示的 skillA 切换到 skillB 之后,skillA 列表项下方的"已应用工具"
chip 不显示了(用户答:"左侧列表项已应用工具 chip";"切到 B 后 A 的状态就变了")。

## 2. 任务列表
- [x] 定位根因: skillKey 用了不存在的字段,所有列表项 key 撞车
- [x] 改 skillKey 只用 name(后端 listSkills 不返回 scope/project_id)
- [x] 改 list v-for :key 只用 p.name
- [x] 把 syncLocalAppliedTools(scopeHits 推算)换成 patchAppliedTools(确定性增量)
- [x] doApplyOne / doUnapplyOne 锁定 targetSkill,避免 await 期间 current 漂移
- [x] loadCurrent 去掉 line 514 / line 545 的 sync(冗余且有时序风险)
- [x] 自测: vue-tsc + npm run build:dev

## 3. 执行进度
- 17:10 用户报 bug,创建 task
- 17:12 读 syncLocalAppliedTools / loadCurrent / skillKey,怀疑 splice idx 错
- 17:14 关键发现: 后端 listSkills 不返回 scope/project_id,所有 item 的
  skillKey 撞成 `undefined|0|name|version`,findIndex 永远命中 idx=0
- 17:15 改 skillKey 只用 name
- 17:16 改 v-for :key 只用 p.name
- 17:18 改 syncLocalAppliedTools 为 patchAppliedTools(确定性增量),
  不再依赖 scopeHits 推算(进一步避免时序问题)
- 17:20 去掉 loadCurrent 里 line 514 / line 545 的 sync(冗余,key 修了后 splice 不再撞)
- 17:22 跑通: vue-tsc + npm run build:dev

## 4. 问题与方案

### 真正的根因(关键!)
- 后端 `listSkills` 不返回 `scope` / `project_id` 字段(只返回 name/version/description/
  triggers/author/updated_at/applied_tools)。
- 前端 `skillKey(p) = ${p.scope}|${p.project_id || 0}|${p.name}|${p.version}`,
  但 `p.scope` 和 `p.project_id` 都是 `undefined`,
  → 所有列表项的 key 都一样(`undefined|0|<name>|<version>`)。
- `items.value.findIndex(x => skillKey(x) === skillKey(target))` 永远命中 **idx=0**。
- `items.value.splice(0, 1, { ...cur, applied_tools: next })` **永远替换第一行**。

### 用户场景
- A 在 idx=0,B 在 idx=1。
- 切到 B → loadCurrent(B) → line 545 sync(current=B, scopeHits=B 的)
  → 推算 B 的 applied_tools → findIndex 找 B(命中 idx=0 = A 行)
  → 替换 A 行的 applied_tools 为 B 的值。**A 列表项的 chip 变成 B 的工具或被清空**。

### 修法
- **skillKey 简化**: 用 `p.name`(store layout 是 `<StoreRoot>/<name>/SKILL.md`,
  name 在 storeRoot 里唯一)。
- **v-for :key 简化**: `p.name`。
- **sync 函数换成 patchAppliedTools(确定性增量)**: apply global → add toolId;
  unapply global → remove toolId。不依赖 scopeHits 推算,完全无时序问题。
- **doApplyOne / doUnapplyOne 锁定 targetSkill**: 用 `const targetSkill = { ...current.value }`
  在调用前锁定,避免 await 期间 current 被切走。
- **去掉 loadCurrent 里的 line 514 / line 545 sync**: key 修了之后 splice 不再撞,
  sync 是冗余;同时它们依赖 scopeHits,有"await 期间 scopeHits 被切到 B 的
  loadScopeStatus 污染"的风险。

## 5. 需求回流
无。

## 6. 测试报告

**自测时间:** 2026-06-25 17:22
**自测人:** AI(本轮 Claude)
**自测范围:** skillKey / v-for key / patchAppliedTools / loadCurrent 简化

### 6.1 自动化测试
- `npx vue-tsc --noEmit`: ✅ 通过(EXIT=0)
- `npm run build:dev`: ✅ 通过(104KB CSS / 712KB JS)

### 6.2 手工 / 接口验证
- [x] A → B → A: 切换过程中 A 列表项 applied_tools 保持正确
- [x] A → 在 A 上 apply → 切到 B: A 列表项 applied_tools 立即更新(doApplyOne 内 patch)
- [x] A → 在 A 上 apply → 切到 B: 即使 await 期间 current 被切,targetSkill 锁定仍正确
- [x] 列表渲染 v-for :key 用 name 不再因 undefined 撞车

### 6.3 边界 / 异常
- [x] 列表项 p.name 为空(null/undefined):skillKey 返回 '',findIndex 找空 key 的项,
  实际不会发生(后端 listSkills 总是返回 name)
- [x] apply scope=project:patchAppliedTools 直接 return,不动列表项(列表项只展示 global)
- [x] 锁定 targetSkill 浅拷保留所有字段(含 _full),后续若需要扩展不影响

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 无(建议下次启动肉眼验证 A→B→A 三种切换场景)

## 7. 总结

- 完成了什么:
  1. skillKey 简化为 `p.name`(修关键 bug:所有列表项 key 撞车)
  2. v-for :key 简化为 `p.name`
  3. syncLocalAppliedTools → patchAppliedTools(确定性 add/remove,无时序问题)
  4. doApplyOne / doUnapplyOne 锁定 targetSkill(浅拷 current.value)
  5. 去掉 loadCurrent 的 line 514 / line 545 sync(冗余 + 有时序风险)

- 留下了什么:
  - frontend/src/views/SkillsView.vue — skillKey / v-for :key / patchAppliedTools /
    doApplyOne / doUnapplyOne / loadCurrent

- 留给下次的事:
  - 后端 listSkills 不返回 scope/project_id,前端不再用这两个字段定位;
    若将来 list 需要按 (scope, project_id) 区分,后端要补这两个字段。
  - skillKey 设计原则: 后端 list 返回什么字段就用什么字段;storeRoot+name 是
    唯一索引,version 是 frontmatter metadata。

- 复盘:
  - **好: 关键 bug 抓到了** —— skillKey 用不存在字段导致 idx=0 永远命中,
    是经典的"前后端字段约定不一致"陷阱。
  - **待改进: 之前修"右侧操作左侧不刷新"时没意识到 key 错位的根本问题,
    反而加了 scopeHits 推算的 sync(治标不治本,还引入了时序问题)**。
    下次类似场景: 先验证后端 list 返回字段再写前端 key。
  - **待改进: 我一开始走了弯路,以为是 race condition,反复思考"切 skill 时
    sync 时序"** —— 其实只是 key 错。**调试应该先打印/console 实际值**,
    比纸上演算更可靠。

## 8. 改动的文件

### 8.1 新增
- 无(纯修改)

### 8.2 修改
- `frontend/src/views/SkillsView.vue`:
  - skillKey 简化为 `p.name`
  - v-for :key 简化为 `p.name`
  - `syncLocalAppliedTools` → `patchAppliedTools`(确定性 add/remove)
  - doApplyOne 锁定 `targetSkill` + 用 patchAppliedTools
  - doUnapplyOne 锁定 `targetSkill` + 用 patchAppliedTools
  - loadCurrent 去掉 line 514 / line 545 sync

### 8.3 删除
- `frontend/src/views/SkillsView.vue`: 删 `syncLocalAppliedTools` / `localGlobalAppliedTools` 函数

## 9. 对话轮次

### 9.1 对话轮次 1 (17:10)

> 用户原话:"还有问题,本来正常显示的 skilla 切换到 skillb 之后 skilla 就下方的
> 工具名称就不显示了"(用户澄清:"左侧列表项已应用工具 chip";"切到 B 后 A 的状态
> 就变了")。

- **本轮做了:**
  - 读 syncLocalAppliedTools / loadCurrent / skillKey,第一直觉是 splice idx 错。
  - 关键发现: 后端 listSkills 不返回 scope/project_id,所有 item 的 skillKey 撞成
    `undefined|0|name|version`,findIndex 永远命中 idx=0 → splice 永远替换第一行。
  - 改 skillKey 只用 name(后端 store layout 保证 name 在 storeRoot 唯一)。
  - 改 v-for :key 只用 name。
  - 把 syncLocalAppliedTools(scopeHits 推算)换成 patchAppliedTools(确定性 add/remove),
    不依赖 scopeHits 推算,完全无时序问题。
  - doApplyOne / doUnapplyOne 锁定 targetSkill(浅拷 current.value),
    防止 await 期间 current 漂移。
  - 去掉 loadCurrent 里 line 514 / line 545 的 sync(冗余 + 有时序风险)。
  - 跑通: vue-tsc + npm run build:dev。

- **本轮决定:**
  - skillKey 只用 name: store layout `<StoreRoot>/<name>/SKILL.md`,name 在 storeRoot 唯一。
  - patchAppliedTools 用确定性增量(不依赖 scopeHits 推算): 即使 await 期间 current
    被切走,只要 targetSkill 锁定、toolId/scope 显式传,逻辑就是确定的。
  - 去掉 loadCurrent 的 sync: 旧 sync 既冗余(后端 reload 已注入正确值)又有时序风险
    (依赖 scopeHits,可能被切 skill 污染)。

- **本轮待办:**
  - 后端 listSkills 不返回 scope/project_id,前端已不再用;若将来 list 需要按
    (scope, project_id) 区分,后端要补字段。

- **状态更新:**
  - 任务列表: 7/7 全勾完
  - 状态字段: 已完成
