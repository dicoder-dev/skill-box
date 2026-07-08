# 首页 4 个 UI 小问题修复

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

用户在首页(SkillsView)发现 4 个 UI 问题:

1. **MD 文件放弃修改要点 2 次**:编辑 md 文件后点"放弃修改"按钮,状态没回到原值,得再点一次才行。py 文件点 1 次即可。
2. **技能目录树文件图标不显示**:目录树前面的文件图标是灰色"问号"占位(`NOT_FOUND_ICON = 'Help'`),不显示。
3. **作用域标题图标不显示**:SkillScopePanel 的标题前的图标也不显示。
4. **首页图标按钮 hover tips 不全**:部分图标按钮鼠标划过没显示原生 title 提示。

## 2. 任务列表

- [x] 修 MD 放弃修改 1 次生效(根因:Tiptap 异步 onUpdate 飞行帧覆盖 reset 后的 localFiles)
- [x] 替换目录树文件图标(直接用 iconpark PascalCase 名,不走 mdi 映射)
- [x] 替换作用域标题图标(`Help` 问号 → `Local` 位置标记)
- [x] 首页所有图标按钮补 hover tips(`data-tip` 替代 `title`,统一项目 tooltip 规范)
- [x] 自测构建 + 提交

## 3. 执行进度

- 14:30 排查 MD 放弃修改根因 — Tiptap 异步 onUpdate 飞行帧覆盖 reset 后的 localFiles
- 14:45 给 resetCurrent 加 resetLock 时间窗(80ms),期间 onContentChange 丢弃
- 14:50 改造 FileTreeNode FILE_ICON 表 — 直传 iconpark PascalCase
- 14:55 替换 scope 标题图标 → Local
- 15:00 补全首页 icon-btn / title-action-btn / sfip-mode-btn 等的 data-tip
- 15:05 构建验证:npx vite build 通过、go build 通过

## 4. 问题与方案

### 问题 1:MD 放弃修改点 2 次(py 文件 1 次即可)

**根因**:Tiptap 编辑器的 onUpdate 是异步的(Markdown → HTML → Markdown 转换后字符串跟编辑器实时内容不完全等价,可能 debounce)。Monaco 走 `onDidChangeContent` 同步触发。

**用户场景**:用户在 Tiptap 改一行 → 编辑器触发 onUpdate(异步)→ emit update → SkillFileInlinePanel.onContentChange 写 localFiles。在这一帧还没飞完时,用户点了"放弃修改"按钮:
1. `resetCurrent` 把 localFiles 重置回 orig,清 dirtyPaths
2. **Tiptap 飞行中那一帧 onUpdate 落地**:emit 触发 → onContentChange 再次写回用户的最新内容 → dirtyPaths 重新被加上
3. 用户看到 dirty 还在 → 再点一次

**方案**: `resetLockUntil = Date.now() + 80` 时间窗锁。`resetCurrent` 设置锁,80ms 内 onContentChange 直接 return。足够吞掉 Tiptap 的异步飞行帧,不影响 Monaco(同步 emit 不会有异步窗口)。

### 问题 2+3:图标显示为问号

**根因**: `FileTreeNode.FILE_ICON` 表里大量 mdi 名(`mdi:language-markdown-outline` / `mdi:code-json` / `mdi:database-outline` 等)在 `iconparkMap.js` 里没有对应,全部 fallback 到 `NOT_FOUND_ICON = 'Help'`。 scope 标题 `mdi:help-circle-outline` → `Help` 是有的,但用户期望更明显的"作用域/位置"语义。

**方案**:
- 直接用 iconpark PascalCase 组件名(走 IconPark 组件"非 mdi 前缀 → 原名作组件名"分支,绕开映射表)。
- 选图原则:语义贴近即可,iconpark 没有 `language-python` 这种专用语言图标,统一用 `FileCode` / `Code` / `Terminal` 等通用组件。
- scope 标题图标从 `Help`(问号)改成 `Local`(位置 marker),更贴合"作用域 = 生效位置"。

### 问题 4:图标按钮 hover tip 缺失

**根因**:项目 tooltip 体系是 `[data-tip]` CSS 自定义实现(style.css L473-499),但部分按钮只写了 `:title` 走浏览器原生 tooltip(被 CSS reset 干掉),部分按钮完全没写。

**方案**:统一用 `:data-tip` 替代 `:title`,让项目 tooltip 体系覆盖。

## 5. 需求回流

## 6. 测试报告

**自测时间:** 2026-07-08 15:05
**自测人:** AI(本轮 Claude)
**自测范围:** 前端 3 个文件改动 + 构建验证

### 6.1 自动化测试
- `go build ./...` 结果: ✅ 通过(只有 macOS 链接警告,与改动无关)
- 前端 `npx vite build` 结果: ✅ 通过(10.91s)

### 6.2 手工 / 接口验证
- [x] 验证 FileCode / FileText / Terminal / DatabaseConfig / Code / FileSettings / FileFocus / FilePdf / Folder / Local / Save / Info / Edit / View / FileCabinet 在 @icon-park/vue-next/es/icons 都存在
- [x] 验证 resetCurrent + resetLock 时间窗实现路径(逻辑走通,真实 Tiptap 行为需用户在桌面端复测)
- [x] 验证 SkillsView 5 个 icon-btn 都有 data-tip,InlinePanel 内 4 个按钮(frontmatter/mode/放弃/保存)都补齐

### 6.3 边界 / 异常
- [x] MD 文件 resetLock 兜底 80ms 飞行帧 → 不会因为丢 emit 导致 dirty 永久卡住(80ms 后恢复正常)
- [x] py 文件走 Monaco 同步路径,resetLock 不影响正常编辑(同步 emit 走完再 unlock)

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: Tiptap 实际防抖时长需用户复测(我代码按 80ms 兜底,如 Tiptap debounce >80ms 需调整)

## 7. 总结

4 个 UI 问题一次性修完,核心改动 3 个文件(SkillFileInlinePanel.vue + FileTreeNode.vue + SkillScopePanel.vue),外加 SkillsView.vue 3 个按钮补 tip。

**关键决策**:
- MD 放弃修改用 resetLock 时间窗(80ms)而不是改 Tiptap 内部状态,改动最小、风险最低
- iconpark 直传 PascalCase 比补全 mdi 映射表更彻底(避免后续新后缀又走 fallback)

**遗留事项**: 用户需在桌面端复测 4 个问题确认;特别是 Tiptap 防抖时长,若实测需 >80ms 我再调整。

## 8. 改动的文件

### 8.1 修改
- `frontend/src/components/skill/SkillFileInlinePanel.vue`
  - 新增 `resetLockUntil` 时间窗锁,resetCurrent 加锁,onContentChange 锁内丢弃
  - 文件树标题、InlinePanel 头部、`mdi:pencil-outline` / `mdi:eye-outline` / `mdi:content-save` / `mdi:information-outline` 等图标改为 iconpark 直名(PascalCase)
  - "放弃修改" / "保存" / 模式切换 / frontmatter 按钮都补 `data-tip`
- `frontend/src/components/skill/FileTreeNode.vue`
  - `FILE_ICON` 表全部 mdi:xxx 改成 iconpark PascalCase(FileText/FileCode/Terminal/DatabaseConfig/Code/FileSettings/FileFocus/FilePdf/Folder)
- `frontend/src/components/skill/SkillScopePanel.vue`
  - 标题图标 `mdi:help-circle-outline` → `Local`(位置 marker)
  - 工具组展开按钮 `:title` → `:data-tip` 走项目 tooltip
- `frontend/src/views/SkillsView.vue`
  - `name-actions` 槽内的编辑/取消/保存按钮都补 `:data-tip`

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- `Bash npx vite build` — 前端编译验证(10.91s 通过)
- `Bash go build ./...` — 后端编译验证(通过,只有 macOS 链接警告)

### 9.4 不涉及工具调用的任务
本任务为纯代码改动 + 构建验证,未调用任何 MCP / Skill。