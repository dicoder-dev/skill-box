# 修 ScopePanel window listener 漏清理导致幽灵请求

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

前一轮修了 SkillsView.onScopeChange 自递归派发后,接口爆发收敛,但日志里仍可见
"幽灵请求" —— 用户 apply code-review,UI 当前选中 code-review,日志却出现
canvas-design 的 scope-status 请求。原因不明,需要深挖。

## 2. 任务列表

- [x] 读 SkillScopePanel.vue onMounted/onUnmounted 逻辑
- [x] 定位幽灵请求根因:addEventListener 漏 removeEventListener
- [x] 补 onUnmounted + import
- [x] npm run build 验证
- [x] memory 沉淀
- [x] git commit + push

## 3. 执行进度

- 21:50 读日志样本、查 scope 调用方,定位幽灵请求根因
- 21:52 编辑 SkillScopePanel.vue 加 onUnmounted,build 通过
- 21:53 commit + push

## 4. 问题与方案

### 根因

`SkillScopePanel.vue:209-212` `onMounted` 调:
```js
window.addEventListener('skillbox:scope-refresh', onScopeRefresh)
```

**没有配套 onUnmounted 清理**。

`currentIdentity` 绑到 InlinePanel `:key`,切 skill 时 key 变 → InlinePanel 重 mount
→ ScopePanel 重 mount。但 `window.addEventListener` 是**全局**挂在 window 上,
不受 Vue 生命周期管控 —— 旧实例卸载时 listener 仍留在 window。

后续 `dispatchEvent('skillbox:scope-refresh')`(doApplyOne 触发)会让**所有
滞留 instance** 各自走 `loadScope()`。N 个 instance → N 个并发 GET。
实例有几个,看用户最近切过几次 skill — 例子里的 2 个 name(用户当前选 code-review
但实例里残留 canvas-design 的)证明上一刻 ScopePanel 实例没被清理。

### 修复

`SkillScopePanel.vue:209-212` 上下文:
- import 加 `onUnmounted`
- 末尾补 `onUnmounted(() => window.removeEventListener('skillbox:scope-refresh', onScopeRefresh))`

## 5. 需求回流

(无)

## 6. 测试报告

**自测时间:** 2026-07-08 21:53
**自测人:** AI(本轮 Claude)
**自测范围:** SkillScopePanel.vue 的 mount/unmount 生命周期

### 6.1 自动化测试

- `npm run build` 结果: ✅ 通过(10.67s)

### 6.2 手工 / 接口验证

- [x] 用例 1:加 onUnmounted,build 通过
- 建议生产验证(用户执行):切 skill A → apply → 切 skill B → apply → 回切 A → 查 log
  - 改前:看到 2 个 skill 都发起 scope-status
  - 改后:每次只看到当前选中那 1 个

### 6.3 边界 / 异常

- [x] 复用 listener 引用(onScopeRefresh 是具名函数,不是匿名),removeEventListener 能正确匹配
- [x] import 顺序:onUnmounted 与 onMounted 一起在顶部 import,符合 Composition API 风格

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无

## 7. 总结

**完成了什么:** SkillScopePanel.vue 补 `onUnmounted` 清理 `window.addEventListener('skillbox:scope-refresh', ...)`。

**留下了什么:** memory `~/.claude/.../memory/scope-panel-listener-cleanup.md`

**留给下次的事:**
- (建议顺手)扫其它 Vue 组件是否还有同样"addEventListener 漏 removeEventListener"
  模式。目前只查到 SkillScopePanel 这一处。

**复盘:** 教训 — Vue 组件里 `window.addEventListener` 必须配 `onUnmounted`,
Vue 不会自动接管 window 监听,需显式 cancel。判断标准:任何 `onMounted` 里挂全局
副作用(addEventListener / setInterval / WebSocket / 第三方订阅)必须有匹配清理。

## 8. 改动的文件

### 8.1 新增
- (无)

### 8.2 修改
- `frontend/src/components/skill/SkillScopePanel.vue` — import 增 `onUnmounted`,
  onMounted 块后面追加 `onUnmounted` 清理 window listener

### 8.3 删除
- (无)

## 9. 工具与用途

### 9.1 MCP 工具
- (无)

### 9.2 Skill
- (无)

### 9.3 CLI
- `Bash npm run build` — 前端 build(10.67s 通过)
- `Bash git diff` — 校验本轮改动范围

## 1.1 对话轮次 (21:50)

> 用户原话:贴日志说现在应该好了(高频爆发的修好了),但还有 canvas-design 这种
> 幽灵请求

- **本轮做了:** 读 ScopePanel + InlinePanel,定位 addEventListener 漏清理
- **本轮决定:** 补 onUnmounted
- **本轮待办:** build 验证 + commit
- **本轮工具:** `Read SkillScopePanel.vue` / `Bash grep -n removeEventListener`
- **状态更新:** 进行中
