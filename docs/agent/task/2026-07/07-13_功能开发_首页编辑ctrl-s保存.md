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
- [x] 实现 Ctrl+S / Cmd+S 快捷键保存
- [x] 前端构建 + 自测 + 提交推送

## 3. 执行进度

- 14:00 定位到 `frontend/src/components/skill/SkillFileInlinePanel.vue`,保存逻辑在 `saveCurrent()`(L751)
- 14:00 编辑态用 `currentEditingPath`(L111)+ `isDirty` computed(L381)判断;`onMounted`/`onUnmounted` 已存在(L236 / L927)
- 14:00 设计:在 `onMounted` 挂 `window.keydown`,`onUnmounted` 移除,触发后 `preventDefault` + 调 `saveCurrent()`

## 4. 问题与方案

无。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-13 14:10
**自测人:** AI(本轮 Claude)
**自测范围:** `frontend/src/components/skill/SkillFileInlinePanel.vue` 的 onKeyDown + mount/unmount

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(11.71s,产 dist)
- 前端 `npm run lint` 结果: 不涉及(项目未配 lint script)
- 后端 `go test ./...` 结果: 不涉及(本次只改前端)

### 6.2 手工 / 接口验证
未在浏览器/桌面端实操验证(本环境无 GUI)。
代码静态检查通过项:
- [x] `onKeyDown` 在 `isComposing` 时直接返回,不会抢输入法输入
- [x] 仅 `isDirty.value === true` 或 `currentMode.value === 'edit'` 时触发,避免空保存
- [x] `selectedFile.value?.path` 不存在时直接放行,不抢键
- [x] macOS 走 `metaKey`,其它平台走 `ctrlKey`
- [x] `e.preventDefault()` 在判定通过后调用,避免触发浏览器默认"保存网页"
- [x] `saving.value` 锁定防止连按重复触发
- [x] `onUnmounted` 调 `removeEventListener`,`:key` 重 mount 不残留

### 6.3 边界 / 异常
- [x] 非编辑模式 + 非 dirty → 不抢键(浏览器可走默认行为)
- [x] Alt+Ctrl+S → `!e.altKey` 过滤掉,不抢
- [x] 焦点在 Monaco/Tiptap 编辑器内部 → 监听挂在 window,仍能捕获

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 实际键盘交互需在 wails3 dev 中手验(本环境无桌面端)

## 7. 总结

- 完成了什么: SkillFileInlinePanel 编辑态支持 Ctrl+S / Cmd+S 全局快捷键保存
- 留下了什么: `frontend/src/components/skill/SkillFileInlinePanel.vue` 加 `onKeyDown` + mount/unmount 挂卸载(共 +13 行代码)
- 留给下次的事: 桌面端 webview 实际按键验证 + 其他编辑场景(如 frontmatter 弹窗)是否需要同样快捷键
- 复盘: 设计选择"挂 window + 仅在编辑态触发"是兼顾可达性(焦点在编辑器内)和克制(不抢无关快捷键)的平衡点

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
- `Bash npm run build` — 前端编译验证(11.71s 通过)
- `Bash git commit && git push` — 提交并推送(0a353fb)

## 1.2 对话轮次 (14:10)

- **本轮做了:** 实现 `onKeyDown` 监听 + `onMounted` 挂载 + `onUnmounted` 移除;跑 `npm run build` 通过;commit + push 0a353fb
- **本轮决定:** 把 `onKeyDown` 函数定义放在 `saveCurrent` 之前,方便阅读"快捷键触发保存"链路一目了然
- **本轮待办:** 无
- **本轮工具:** `Bash npm run build`、`Bash git commit && git push`
- **状态更新:** 任务列表全部 completed

## 1.1 对话轮次 (14:00)

> 用户原话:首页技能详情编辑的时候支持快捷键 ctrl + s 触发保存

- **本轮做了:** 定位代码,确定 `saveCurrent()` 入口与编辑态判断(`currentEditingPath` / `isDirty`);写了 task 文档框架
- **本轮决定:** 监听挂在 `window`(而不是组件根 div),这样即使焦点在 CodeViewer/Monaco/Tiptap 内部也能捕获;仅在编辑模式或 dirty 时触发,跳过输入法组合中(`isComposing`);macOS 用 `metaKey`,其它用 `ctrlKey`
- **本轮待办:** 写 onKeyDown 实现 + mount/unmount + 自测 + 提交
- **本轮工具:** 无
- **状态更新:** 任务列表 #1 → completed,#2 → in_progress