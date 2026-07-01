# 首页工具视图 UI 修复

**日期:** 2026-07-01
**状态:** 已完成

## 1. 需求

修首页 → 工具(ToolsView)界面的 3 个 UI 小问题:
- 工具栏右上"新建"按钮前面显示两个图标(放大镜/plus 重复)
- 新建弹窗"成熟度"select 选择框高度跟其他 input 不一致
- 新建弹窗"添加路径"按钮前面显示两个加号图标

## 2. 任务列表

- [x] 1. 排查双图标根因
- [x] 2. 修复 i18n 文案里的手写 '+' 字符
- [x] 3. 修复 select 行高统一
- [x] 4. build 验证 + commit + push

## 3. 执行进度

- 17:30 收到反馈,定位根因
- 17:35 修改 zh-CN / en-US 的 btnNew / paths.add / emptyHint
- 17:40 ToolsView CSS 加 `.form-field select/input { height: 36px; min-height: 36px }`
- 17:45 build 通过(2.22s) + commit + push

## 4. 问题与方案

### 4.1 双图标根因
- 模板里 button 已经放 `<Icon icon="mdi:plus" />`,但 i18n 文案 `btnNew: '+ 新建'` / `paths.add: '+ 添加路径'` 里又手写了 '+'
- 模板渲染输出 = Icon(图标) + 文案里的 '+' + 文案正文 → 视觉上两个 + 重叠
- 修复:去掉文案里的 '+',只保留纯文字

### 4.2 select 行高不一致
- 全局 CSS `input,select,textarea { padding: 8px 12px }` 设了 padding,但 macOS Chrome 的 native select 会因右侧下拉箭头占位 + 系统字体 SF Pro 的 line-height 计算,渲染高度比 input 多 2-3px
- 按 [[project.md]] 第 12 条 textarea 锁定行高的经验:`height + min-height` 双锁避免被外层规则覆盖
- 修复:`.form-field select, .form-field input { height: 36px; min-height: 36px; line-height: 1.4 }`

## 5. 需求回流

> 暂无

## 6. 测试报告

**自测时间:** 2026-07-01
**自测人:** AI(本轮 Claude)
**自测范围:** 静态修复 + build 验证

- `npm run build`:✅ 通过(2.22s)
- 没启服务做端到端 UI 验证(用户在工作区同步改 MarketPullConfirm.vue 样式)

## 7. 总结

- **完成了什么**:ToolsView 3 个 UI bug 全修(i18n 双 + 行高),1 个 commit `1e1a01e` 已 push
- **留下了什么**:用户的 MarketPullConfirm.vue 改动不在 staged 里(用户自己在改样式),保留其 working tree
- **复盘**:i18n 文案和模板图标语义重叠是常见坑,以后写 button 文案先看模板里有没有 Icon,有就别再加 '+'

## 8. 改动的文件

### 8.1 修改
- `frontend/src/views/ToolsView.vue` — CSS 加 `.form-field select/input` 锁定 36px 行高
- `frontend/src/core/i18n/zh-CN.js` — `btnNew` / `paths.add` / `emptyHint` 去 '+'
- `frontend/src/core/i18n/en-US.js` — 同上
- `api-server/cmd/web/frontend/dist/index.html` — embed 同步(mtime 改但内容不变,实际无需 add)
- `api-server/cmd/web/frontend/dist/assets/index-CiaKw_Rw.js` — 新 build 产物
- `api-server/cmd/web/frontend/dist/assets/index-DTKXLiFr.css` — 新 build 产物

### 8.2 删除
无

## 对话轮次

### 1.1 对话轮次 (17:30)

> 用户原话:"检查一下首页工具界面新建按钮,它前面为什么会显示两个图标?有新建弹窗的成熟度,它的输入框高度为什么跟其他输入框的高度没有保持一致?键弹窗的添加路径,它前面也显示了两个图标。两个加号图标。"

- **本轮做了**:grep i18n 锁定 3 处 '+' 重叠 + ToolsView CSS 加 select/input 行高锁
- **本轮决定**:文案和模板图标语义重叠,改文案;select 行高用 height+min-height 双锁
- **本轮待办**:build + commit + push
- **本轮工具**:`Read ToolsView.vue`、`Read style.css`、`Edit`、`Bash npm run build`、`Bash git add/commit/push`
- **状态更新**:commit `1e1a01e` 已 push origin main
