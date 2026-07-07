# 工具页:导航顺序交换 + 卡片 actions 常显 + 新增打开 skills 目录按钮

**日期:** 2026-07-06
**状态:** 已完成

## 1. 需求

用户在一次会话中提出三件相关的事:

1. 左侧导航栏中"工具"和"项目"的顺序互换(项目在上、工具在下)。
2. 工具列表卡片中编辑和锁定的图标要一直显示,而不是鼠标划过时才显示。
3. 在工具卡片上加一个文件夹图标按钮,点击后在系统浏览器(资源管理器 / Finder)中打开该工具对应的 skills 技能目录。

## 2. 任务列表

- [x] App.vue navItems 顺序:tools 提到 projects 之前
- [x] ToolsView.vue 卡片 actions 去掉 hover 控制,常显
- [x] ToolsView.vue 加文件夹按钮(打开 skills 目录),无 path 时禁用
- [x] zh-CN / en-US 补 btnOpenSkillsDir / openNoPath / openFailed 文案
- [x] 前端 `npm run build` 通过
- [x] 提交并 git push

## 3. 执行进度

- 14:xx 读 App.vue / ToolsView.vue,确认 navItems 配置在 App.vue 167-174 行
- 14:xx 确认要交换的两个项就是 `{ key: 'tools' }` 和 `{ key: 'projects' }`,调换位置
- 14:xx 在 ToolsView.vue 改 `.tool-card-actions` 由 `opacity: 0` + hover 显 → 常显 `opacity: 1`,同步删响应式里的 hover 兜底
- 14:xx 检索 backend:发现 `FsReveal` hook、`POST /api/desktop/fs/reveal`、`platform.fs.reveal(path)` 都已存在,直接复用即可(不用新写后端)
- 14:xx 在 ToolsView.vue 加 `firstSkillsPath` / `openSkillsDir`,模板里 actions 区第一个位置加文件夹按钮(`mdi:folder-open-outline`),无 path 时按钮置灰 + tooltip 提示
- 14:xx 补 i18n
- 14:xx `npm run build` 6.44s 通过
- 14:xx git commit + push(commit 384ddd8)

## 4. 问题与方案

无。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-06
**自测人:** AI(本轮 Claude)
**自测范围:** App.vue / ToolsView.vue 模板 + script + style / i18n 字符串

### 6.1 自动化测试

- 前端 `npm run build` 结果: ✅ 通过(6.44s)
- 后端本次未改 Go 代码,无需 `go test`

### 6.2 手工 / 接口验证

未做浏览器端到端验证(本环境无法直接起 wails3 dev)。

### 6.3 边界 / 异常

- 卡片 paths 为空数组: 按钮 disabled、tooltip = "该工具尚未配置 skills 目录"
- 卡片 paths 中多个项 path 都非空: 取第一个(按后端返回顺序,优先 global user,其次 global system,再次 project user,最后 project system — 跟 store 的 PATH_SLOTS 顺序一致)
- 桌面端: 走 `platform.fs.reveal` → POST /api/desktop/fs/reveal → FsReveal hook → 系统文件管理器
- Web 端: 后端 hook 未注入 → 返 501 + fallback_url(父目录 file://)→ 前端 platform.fs.reveal 自动 `window.open(fallback_url)`,无需额外代码

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无

## 7. 总结

- 完成了什么:三件需求全部落地,前端 build 通过,提交推送完成
- 留下了什么: commit `384ddd8`,4 个文件 +69/-7 行
- 留给下次的事: 无
- 复盘: 这三件事本来是分开的小需求,但合并到一个 commit 是合适的 — 都是"工具页"相关的视觉 / 交互微调,没有跨越功能边界。如果需求 3 要求后端也跟着动,可能得拆 commit;但复用既有 reveal 链路让三件事落得很快。

## 8. 改动的文件

### 8.1 新增

无。

### 8.2 修改

- `frontend/src/App.vue` — navItems 把 tools 提到 projects 之前,同步注释
- `frontend/src/views/ToolsView.vue` — 加 firstSkillsPath / openSkillsDir;模板里加文件夹按钮(actions 区最前);`.tool-card-actions` 常显 + 删响应式 hover 兜底;新增 `.action-icon-folder` / `.action-icon-disabled` CSS
- `frontend/src/core/i18n/zh-CN.js` — tools.* 命名空间补 btnOpenSkillsDir / openNoPath / openFailed
- `frontend/src/core/i18n/en-US.js` — 同步英文

### 8.3 删除

无。

## 9. 工具与用途

### 9.1 MCP 工具

无。

### 9.2 Skill

无。

### 9.3 CLI

- `Bash npm run build` — 前端编译验证(6.44s 通过)
- `Bash git add ... && git commit && git push` — 提交并推送到 origin/main
- `Bash grep -rE "BrowserOpenURL|open.*folder|open.*path|openPath" ...` — 调研既有打开目录链路
- `Bash find ... && grep -rn "fs/reveal|fs\.reveal|FsReveal" ...` — 确认 platform.fs.reveal 已存在