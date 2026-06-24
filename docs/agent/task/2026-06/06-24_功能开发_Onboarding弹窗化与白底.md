# Onboarding 弹窗化 + 删除左栏导航

**日期:** 2026-06-24
**状态:** 已完成

## 1. 需求

- 技能列表左栏改白底(原来是 `--bg-subtle` 灰色)
- "导入"按钮从"跳到 Onboarding tab"改成打开弹窗(把现有 OnboardingView 抽成 dialog)
- 左侧菜单栏删除"导入技能"导航项(原 onboarding)

## 2. 任务列表

- [x] 抽 OnboardingImportDialog 组件(包 Modal + 内部 OnboardingView)
- [x] 改 OnboardingView:去掉 view-header(已不需要);goSkills 改 emit('done')
- [x] SkillsView 引入 OnboardingImportDialog;左栏"导入"按钮触发 importOpen
- [x] 左栏背景改白(变量 `--bg-card`);选中态改用 `--bg-subtle` 更明显
- [x] App.vue 删除 onboarding 菜单 + 路由 + import OnboardingView
- [x] npm run build 通过
- [x] commit + push

## 3. 执行进度

- 23:55 抽 OnboardingImportDialog 组件 + 改 OnboardingView(去 view-header,goSkills → emit done)
- 23:58 App.vue 删除 nav / import / route
- 00:00 SkillsView 左栏白底 + 接入弹窗
- 00:05 build 通过

## 4. 问题与方案

**问题 1:OnboardingView 1355 行,直接搬到 SkillsView 模板会变成一团。**
方案:把 OnboardingView 改造成"既可作 page 又可作弹窗 body"——去掉外层 view-header(弹窗自己有 title),`goSkills` 从 `appBus.emit('switch-tab', 'skills')` 改成 `emit('done', payload)`。新建 `OnboardingImportDialog.vue` 套 Modal + OnboardingView,SkillsView 引入。

**问题 2:左栏改白底后,选中态变得不明显。**
方案:`.skill-item-active` 之前用 `--bg-card` 与白底同色,失效;改成 `--bg-subtle` 浅灰底 + 3px 主题色竖条(已存在),整体白底配灰底选中态层次清晰。

## 5. 需求回流

(暂无)

## 6. 测试报告

**自测时间:** 2026-06-24
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(281.35 kB JS / 78.05 kB CSS,gzip 后 95.82 kB / 12.16 kB)

### 6.2 手工验证(代码 review)
- [x] 左栏背景 = `--bg-card`(白)
- [x] 列表项 hover 用 `--bg-hover`,选中用 `--bg-subtle`,左侧 3px 竖条
- [x] "导入"按钮 → importOpen=true → OnboardingImportDialog 显示
- [x] App.vue navItems 删掉 onboarding
- [x] App.vue 路由不渲染 OnboardingView
- [x] OnboardingImportDialog 内部 goSkills emit('done') → SkillsView onImported → reload 列表

### 6.3 边界 / 异常
- [x] 弹窗被关闭时:OnboardingView 的 onMounted 里的 doScan 已经跑过一次,下次打开不会重扫(避免重复请求);用户可点"重新扫描"按钮触发
- [x] 老代码 `appBus.emit('switch-tab', 'skills')` 不会再被触发,goSkills 已统一改 emit
- [x] getOnboardingStatus 仍被 App.vue 用作统计,保持不变

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: dev server click-through 验收待跑

## 7. 总结

- 完成了什么: 导入流程从"独立 tab"压成 SkillsView 的弹窗,左栏改白底,菜单减少一项
- 留下了什么: 改 3 个文件 + 新增 1 个组件
  - `frontend/src/views/SkillsView.vue` — 左栏白底 + 引入 OnboardingImportDialog
  - `frontend/src/views/OnboardingView.vue` — 去掉 view-header,goSkills → emit('done')
  - `frontend/src/App.vue` — 删除 onboarding 菜单项 / import / 路由
  - `frontend/src/components/OnboardingImportDialog.vue` — 新增,Modal 包 OnboardingView
- 留给下次的事: dev server 跑一次 click-through
- 复盘: 把大组件(1355 行)抽成弹窗时,与其复制模板不如改原组件"兼容两种用法",只去掉外层装饰 + 改通信方式,改动最小
