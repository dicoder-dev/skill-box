# 首页技能详情编辑支持 Ctrl+S / Cmd+S 触发保存

**日期:** 2026-07-13
**状态:** 进行中

## 1. 需求

用户原话:
> 首页技能详情编辑的时候支持快捷键 ctrl + s 触发保存

细化目标:
- 在首页右侧 `SkillFileInlinePanel` 内编辑文件时,按 `Ctrl+S`(Linux/Windows) 或 `Cmd+S`(macOS) 触发保存
- 阻止浏览器/桌面 webview 默认的"保存网页"行为
- 仅在编辑模式(`currentEditingPath` 有值)或当前文件 dirty 时触发,避免空保存
- 不破坏现有的"保存"按钮链路
- 跨平台:macOS 上 Cmd+S 也走相同分支

## 2. 任务列表

- [x] 定位 SkillFileInlinePanel 编辑入口与保存逻辑
- [ ] 实现 Ctrl+S / Cmd+S 快捷键保存
- [ ] 前端构建 + 自测 + 提交推送

## 3. 执行进度

- 14:00 定位到 `frontend/src/components/skill/SkillFileInlinePanel.vue`,保存逻辑在 `saveCurrent()`(L751)
- 14:00 编辑态用 `currentEditingPath`(L111)+ `isDirty` computed(L381)判断;`onMounted`/`onUnmounted` 已存在(L236 / L927)
- 14:00 设计:在 `onMounted` 挂 `window.keydown`,`onUnmounted` 移除,触发后 `preventDefault` + 调 `saveCurrent()`

## 4. 问题与方案

无。

## 5. 需求回流

无。

## 6. 测试报告

(待自测后补)

## 7. 总结

(任务结束时填)

## 8. 改动的文件

### 8.1 新增
无

### 8.2 修改
- `frontend/src/components/skill/SkillFileInlinePanel.vue` — 加 `onKeyDown` 监听 + `onMounted/onUnmounted` 挂卸载

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
无

### 9.2 Skill
无

### 9.3 CLI
无

## 1.1 对话轮次 (14:00)

> 用户原话:首页技能详情编辑的时候支持快捷键 ctrl + s 触发保存

- **本轮做了:** 定位代码,确定 `saveCurrent()` 入口与编辑态判断(`currentEditingPath` / `isDirty`);写了 task 文档框架
- **本轮决定:** 监听挂在 `window`(而不是组件根 div),这样即使焦点在 CodeViewer/Monaco/Tiptap 内部也能捕获;仅在编辑模式或 dirty 时触发,跳过输入法组合中(`isComposing`);macOS 用 `metaKey`,其它用 `ctrlKey`
- **本轮待办:** 写 onKeyDown 实现 + mount/unmount + 自测 + 提交
- **本轮工具:** 无
- **状态更新:** 任务列表 #1 → completed,#2 → in_progress