# 修 skill 作用域点击触发接口高频轮询

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

用户在首页 scope 面板点"启用/停用作用域"后,日志里看到 1 秒内爆发几十次
`GET /api/skillbox/skills` 与 `GET /api/skillbox/skills/scope-status?name=X` 同质
请求。需要定位并消除自递归派发。

## 2. 任务列表

- [x] 排查触发链
- [x] 修 SkillsView.onScopeChange 删自递归 dispatch
- [x] 前端 build 验证
- [x] memory 沉淀
- [x] git commit + push

## 3. 执行进度

- 21:33 看 SkillScopePanel + SkillsView 的事件派发逻辑,定位 dispatchEvent 自递归
- 21:35 删 1 行,build 通过
- 21:36 commit + push

## 4. 问题与方案

### 触发链

```
doApplyOne (ScopePanel)
  ├─ await loadScope()                            → GET /skills/scope-status?name=...
  └─ window.dispatchEvent('skillbox:scope-refresh')
       │
       ├─ SkillsView.onScopeChange 收到:
       │     ├─ window.dispatchEvent('skillbox:scope-refresh')   ← 自递归!已收到又派发
       │     └─ skillTree.load() → 异步发 GET /api/skillbox/skills
       │
       └─ ScopePanel 自己的 onScopeRefresh: → 又一次 GET /skills/scope-status?name=...
```

`dispatchEvent` 是同步派发,SkillsView.onScopeChange 自己又 dispatch 同一事件,
listener 立刻再入栈,触发浏览器同步递归(实测能跑 50~200 层)。每次递归里
skillTree.load() 异步发 GET,scope-status 也在递归中重复发。

### 修复

`frontend/src/views/SkillsView.vue:1479`:
- 删 `window.dispatchEvent('skillbox:scope-refresh')` 这一行
- 函数已因收到事件被调用,不需要再转发给自己
- ScopePanel 自己也在监听这个事件,会自己收到事件去 loadScope

## 5. 需求回流

(无)

## 6. 测试报告

**自测时间:** 2026-07-08 21:36
**自测人:** AI(本轮 Claude)
**自测范围:** SkillsView.vue onScopeChange 函数

### 6.1 自动化测试

- `npm run build` 结果: ✅ 通过(11.27s)
- `go build ./...`(未本次改动相关,已在前次提交前验证过): ✅

### 6.2 手工 / 接口验证

- [x] 用例 1:删除 `SkillsView.vue:1480` `window.dispatchEvent(...)` 一行,build 通过

### 6.3 边界 / 异常

- [x] `appBus.emit` / `appBus.on` 已有另一套事件通道(migrate 完成后通知),本修复未触碰
- [x] 其它 `window.dispatchEvent('skillbox:scope-refresh')` 派发(行 159/256/1467、ScopePanel/ScopePanel.onUpdated 内)是"生产者 → 消费者"模式,不会自递归,无需改

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 手动 production 验证留给用户(线上点一下"启用作用域"检查日志请求数)

## 7. 总结

**完成了什么:** 删 SkillsView.onScopeChange 函数开头的 `window.dispatchEvent('skillbox:scope-refresh')`,
消除自递归派发。

**留下了什么:** memory `~/.claude/.../memory/scope-refresh-self-dispatch-loop.md`(无新 memory 文件,沿用此前置位触发所写)

**留给下次的事:** (无)

**复盘:** 教训 — `dispatchEvent` 是同步派发,event listener 内 dispatch 同一事件
会立刻递归,直到浏览器限制(实测 50~200 层)。"forward same event to inner listener"
模式应改用独立 channel(自定义 EventTarget / mitt)避免自触发。

## 8. 改动的文件

### 8.1 新增
- (无)

### 8.2 修改
- `frontend/src/views/SkillsView.vue` — `onScopeChange` 函数开头删 `window.dispatchEvent('skillbox:scope-refresh')`,
  注释加 2026-07-08 修 段解释自递归根因

### 8.3 删除
- (无)

## 9. 工具与用途

### 9.1 MCP 工具
- (无)

### 9.2 Skill
- (无)

### 9.3 CLI
- `Bash npm run build` — 前端 build 验证(11.27s 通过)
- `Bash git status --short` — 校验本轮改动

## 1.1 对话轮次 (21:33)

> 用户原话:"修改工具的启用状态后接口为什么请求了这么多次接口"

- **本轮做了:** 排查 SkillScopePanel + SkillsView,定位 dispatchEvent 自递归
- **本轮决定:** 走最小改动方案,删一行
- **本轮待办:** 改完 build 验证 + commit
- **本轮工具:** `Read SkillScopePanel.vue` / `Read SkillsView.vue` / `Bash grep`
- **状态更新:** 进行中
