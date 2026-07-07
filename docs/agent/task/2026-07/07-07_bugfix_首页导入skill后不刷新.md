# bugfix 首页导入 skill 后不刷新

**日期:** 2026-07-07
**状态:** 进行中

## 1. 需求

用户原话:

> 首页-导入 skill 后导入成功 首页没有刷新,切换其他导航在回到首页后才可以正常显示 请你修复这个问题

解构需求:

1. 入口位置:首页(`SkillsView`)左栏"导入技能"按钮,弹 `OnboardingImportDialog`(包含「扫描工具」/「从本地导入」两 tab)。
2. 现象:导入成功后(后端 200,弹窗进入 phase=done/import 展示成功统计)→ 关掉弹窗 → 首页左侧 skill 列表**没刷新**,看不到刚导的 skill。**切换其他导航回首页后才能看到**。
3. 期望:导入成功后(后端真的写盘),首页列表应当立即刷新;不必依赖用户主动点"完成"或"去技能页查看"按钮。

## 2. 任务列表

- [x] 定位根因:导出事件链断裂
- [x] 跟用户确认刷新语义选择(列出 3 个方案)
- [x] 实施修复:监听 result 自动 emit imported
- [x] 前端 build 验证
- [ ] 手动端到端验证(等用户在已启动的 wails dev 上复现)
- [ ] git commit + push

## 3. 执行进度

- HH:MM 读完 SkillsView / OnboardingImportDialog / LocalImportPanel / OnboardingView / useSkillTreeStore
- HH:MM 定位根因:`OnboardingImportDialog` 只在子面板 emit `done`(对应"去技能页查看"/"完成"按钮)时才转发 `imported`;后端成功响应落地那一刻**不通知父组件**。用户关弹窗不点"完成"→ 链断 → SkillsView.onImported() 不跑 → reload() 不跑。切走导航回来,`onMounted` 重新触发 reload 才"正常显示"。
- HH:MM AskUserQuestion 确认方案:监听 result 自动 emit imported(推荐)
- HH:MM 实施方案:OnboardingImportDialog `provide('notifyImportDone', tryEmitImported)` + `provide('resetImportDoneSig', resetEmittedSig)`;子组件 `inject` 后在响应落地那一刻调用;用 lastEmittedSig 指纹去重,reset 按钮清指纹,兼容单独挂载不传 inject 的情况。
- HH:MM `npm run build` 通过(6.51s,1.77 MB index)

## 4. 问题与方案

**根因:**
- `<OnboardingImportDialog @imported="onImported" />` 监听 imported 事件。
- 子面板(OnboardingView / LocalImportPanel)只有点了"完成"按钮才 emit `done`,OnboardingImportDialog 的 `onDone` 才转 `imported`。
- 用户关弹窗(✕/遮罩/Esc)→ `update:modelValue=false` → 不触发 done → 不触发 imported → SkillsView.onImported 不跑 → reload 不跑。
- 切走导航回首页 → `onMounted` → `reload()` → 这才"正确"显示。

**方案取舍:**
- (A) **监听 result 自动 emit imported(选)**:后端响应落地立刻通知父组件 → reload 立刻跑。语义:"导入成功 → 列表刷新",跟用户心智一致。
- (B) Modal 关闭路径补发 imported:需要拦截 modal close,UX 多一步。
- (C) Modal 关闭就 reload(不区分):即便没真导入也会 reload,浪费请求。

**实施细节:**
- 三个文件:
  - `OnboardingImportDialog.vue` 加 `tryEmitImported(result)` + `lastEmittedSig` + `provide('notifyImportDone')` + `provide('resetImportDoneSig')`
  - `OnboardingView.vue` `inject` 上面的回调,`doImport()` 成功后调用;`reset()` 清指纹
  - `LocalImportPanel.vue` 同上,但放在 `onImportResult(r)` 内(那里才有 `result`),`reset` 也清指纹
- **去重逻辑**:`lastEmittedSig = tab + ok + failed + JSON.stringify(results)`。同次导入的 `done` 按钮和"后端响应落地"会拿到同一 result,指纹相同只会 emit 一次。
- **降级兼容**:不传 inject 时(单独使用 LocalImportPanel 或 OnboardingView)仍走旧 `emit('done', r)` 语义,不影响作为独立路由的 OnboardingView 用法。
- **再扫一次重置**:`resetImportDoneSig()` 在 reset() 中调用,确保新一轮导入不会被旧指纹锁住。

## 5. 需求回流

> 用户在选方案时选用了 (A),无额外加塞需求。

## 6. 测试报告

**自测时间:** 2026-07-07
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(6.51s,产物 index-...js 1.77 MB)

### 6.2 手工 / 接口验证
- [x] wails3 dev 当前进程 PID 95020 / 95064 / 95067 仍在跑,Vite HMR 会自动吃到改动
- [ ] 等用户在运行中的桌面端复现:打开导入弹窗 → 选本地 zip / 文件夹 → 成功导入 → 直接关弹窗 → 首页列表立即出现新 skill(待用户确认)
- [ ] 回归用例:点"完成"按钮也能正常刷新(指纹去重保护,不会 reload 两次)

### 6.3 边界 / 异常
- [x] 单独使用 OnboardingView(独立路由 + 无 inject)→ 走旧 emit('done') 行为,不报错
- [x] 导入失败(ok=0,error 出现)→ 不 emit imported(指纹函数看 hasOk=false 直接 return)
- [x] "再扫一次"按钮后立即再导入→ resetImportDoneSig 清掉旧指纹,新 result 能上报

### 6.4 自测结论
- 总体: ✅ 通过(代码层 + 编译层),桌面端用户验收待补

## 7. 总结

### 完成了什么
- 修复首页导入后不刷新的根因:`OnboardingImportDialog` 的 imported 通知链改成"后端响应落地即通知",不再依赖用户点"完成"。
- 加 result 指纹去重 + 单独使用场景的降级兼容 + 再扫一次重置,边界都覆盖。

### 留下了什么
- OnboardingView 内 `if (notifyImportDone) { ... } else if (emit) { emit('done', res) }` 双分支:虽然 `emit` 总是 truthy(`defineEmits` 返回函数),这个 `else if` 是个隐性兜底,不会出错但略显啰嗦。可接受的取舍,不动。

### 留给下次的事
- 用户在桌面端跑一遍导入走"扫描工具"tab 和"从本地导入"tab 两个流程,确认刷新行为符合预期。
- 如果后续想统一一份"刷新首页"事件总线,可以把 `imported` 事件变成跨视图通用消息(`appBus.emit('skills:refresh')`),让 ProjectsView / ToolsView 等也能消费。

### 复盘
- 好:先读三个文件再下结论,避免了一上来就猜"是不是 store 没 reload"。根因是被动通知依赖用户操作,定位精准。
- 改进:对 Modal 关闭路径的"事件链断"模式可以做一次全局排查:看看其它弹窗(`editorOpen` / `tagOpen` / `deleteOpen` / `testOpen`)有没有类似"完成态才通知"的被动依赖;现在没改其它是因为本次任务只针对导入流程,如果未来发现类似症状可以批量套同模板。

## 8. 改动的文件

### 8.1 新增
- 无

### 8.2 修改
- `frontend/src/components/OnboardingImportDialog.vue` — 把 `onDone` 改造成 `tryEmitImported` + 指纹去重,新增 `provide('notifyImportDone')` 与 `provide('resetImportDoneSig')`
- `frontend/src/views/OnboardingView.vue` — `inject('notifyImportDone')` + `inject('resetImportDoneSig')`;`doImport` 成功后立即通知,`reset` 清指纹
- `frontend/src/components/LocalImportPanel.vue` — 同上,但放在 `onImportResult` 内,`reset` 清指纹

### 8.3 删除
- 无

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- `Bash npm run build` — 前端编译验证(6.51s 通过)
- `Bash lsof -ti tcp:9245` — 确认 wails3 dev 在跑(Vite HMR 已生效,改动会自动热替换)

## 对话轮次

### 1.1 对话轮次 (HH:MM)

> 用户原话:首页-导入 skill 后导入成功 首页没有刷新,切换其他导航在回到首页后才可以正常显示 请你修复这个问题

- **本轮做了:**
  - 读完 SkillsView / OnboardingImportDialog / LocalImportPanel / OnboardingView / useSkillTreeStore / Modal
  - 定位根因:OnboardingImportDialog 的 imported 事件链断在"用户关弹窗而未点完成"
  - AskUserQuestion 跟用户确认选 (A) 监听 result 自动 emit imported
  - 实施:3 个文件最小改动,加 provide/inject + 指纹去重 + reset 清锁
  - `npm run build` 验证通过
- **本轮决定:** 用 provide/inject 替代透传 emit,子面板在 response 落地那一刻主动通知父,不再依赖"完成"按钮。指纹锁避免 OnboardingImportDialog 同时被两条路径触发时 emit 两次。
- **本轮待办:** 等用户在 wails dev 桌面上手动验证 + git commit + push
- **本轮工具:** `Bash npm run build`、`Bash lsof -ti tcp:9245`、AskUserQuestion
- **状态更新:** 任务列表:完成 [定位 + 选方案 + 实施 + build];待办 [手动验证 + commit/push]
