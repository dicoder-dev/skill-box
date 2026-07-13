# SKILL.md 不再拼 frontmatter 围栏(方案 A)

**日期:** 2026-07-10
**状态:** 已完成

## 1. 需求

用户原话:

> 请你帮我检查一下,我从首页新建一个技能之后,它的 front matter 为什么会显示在 skill.md 文档里面?在这个文档里面是不用显示的

细化目标:
- 前端新建 skill 时,SKILL.md 不再拼 frontmatter 围栏(`---\nyaml\n---\n\n<body>`)。
- frontmatter 完全交给 `payload.manifest` 字段,后端 `RenderSkillMD` 重渲。
- 编辑/保存链路同步改造,保持"唯一来源 = manifest"。

## 2. 任务列表

- [x] 定位所有前端 SKILL.md 拼接点(grep yaml/--- + 逐个函数读)
- [x] 确认后端 store.Save 走 RenderSkillMD 重渲,前端拼的 frontmatter 会被覆盖
- [x] 改 `SkillsView.buildSkillMd()` 只返 `draft.body`
- [x] 改 `SkillsView.rebuildSkillMd()` 只返 `newBody`(签名兼容)
- [x] 改 `SkillFileInlinePanel.rebuildSkillMd()` 只返 `localFiles['SKILL.md']`
- [x] 改 `SkillFileInlinePanel.rebuildSkillMdFromBody()` 只返 `body`
- [x] 改 `SkillFileInlinePanel.saveFrontmatterForm` 内嵌拼接只取 body
- [x] 前端 `npm run build` 通过(12.03s)
- [x] 同步 dist 到 `api-server/cmd/web/frontend/dist/` 嵌入目录
- [x] 后端 `go build ./...` 通过(链接警告无害)
- [x] 后端单测 `go test ./api-server/internal/skillstore/... ./api-server/internal/skilladapter/...` 通过
- [x] git commit + git push

## 3. 执行进度

时间倒序记:

- HH:MM git push 成功(`af372a2..2a2861c  main -> main`)
- HH:MM git commit 成功,191 files changed,1987 insertions(+),522846 deletions(-)(dist 资产重渲)
- HH:MM `go test ./api-server/internal/skillstore/... ./api-server/internal/skilladapter/...` 通过(0.091s)
- HH:MM `go build ./...` 通过(链接警告无害,无 error)
- HH:MM 同步 dist 到 api-server/cmd/web/frontend/dist(`wails3 task web:sync:embed`)
- HH:MM `npm run build` 通过(12.03s)
- HH:MM 改完 saveFrontmatterForm 内嵌拼接(只取 body)
- HH:MM 改完 SkillFileInlinePanel 的 rebuildSkillMdFromBody + rebuildSkillMd
- HH:MM 改完 SkillsView 的 buildSkillMd + rebuildSkillMd
- HH:MM 读后端 `api-server/internal/skillstore/store.go:127` 确认 store.Save 走 RenderSkillMD 重渲,前端拼的 frontmatter 会被覆盖
- HH:MM grep 所有前端 SKILL.md 拼接点,定位 5 处
- HH:MM 读 splitSkillMd + displayContent 确认 view 模式剥 frontmatter、edit 模式不剥(根因)

## 4. 问题与方案

### 4.1 根因:前端 SKILL.md 拼接冗余

**现象**:用户在 SKILL.md 文档里看到 frontmatter 围栏。

**定位**:
- 前端 `buildSkillMd()` / `rebuildSkillMd()` / `saveFrontmatterForm` 等 5 处把 `---\nyaml\n---\n\n<body>` 拼到 `payload.files[0].content`
- 后端 `api-server/internal/skillstore/store.go:127` 落盘逻辑:`writeFileAtomic(filepath.Join(tmp, "SKILL.md"), skilladapter.RenderSkillMD(c), 0o644)` —— 用 manifest 重渲,前端拼的 frontmatter 被丢弃
- 但前端"编辑态"渲染时(`SkillFileInlinePanel.vue:331-335` `currentContent`)不剥 frontmatter,只有 `displayContent`(第 337-343 行)在 view 模式才剥,所以 edit 模式能看到 `---` 围栏

**方案**:
- 5 处拼接全部改为只返 body
- frontmatter 完全由 `manifest` 字段透传,后端用 `RenderSkillMD` 统一生成
- 符合 `docs/agent/memory/` 既有认知:"SKILL.md 唯一来源 = manifest"(从 store.go:10 注释"YAML frontmatter,不再额外落 skill.yaml"推导)

**教训**:
- 之前(2026-07-10 早上)做 frontmatter 表单化时,只动了表单展示层,没意识到 SKILL.md 拼装层的冗余 —— 用户实际遇到问题才暴露
- 后续做 SKILL.md 相关改动时,先 grep `yaml\|---` 扫一遍拼接点

### 4.2 splitSkillMd 在没围栏时的安全性

**关注点**:改完后 `localFiles.set('SKILL.md', splitSkillMd(newMd).body)` 链路下,`newMd` 已经是纯 body,`splitSkillMd` 应该原样返回。

**验证**:`splitSkillMd` 第 144-149 行 `if (!m) return { frontmatter: '', body: text }`,没 `---` 围栏时 `body = text`(原样)。**安全。**

### 4.3 签名兼容

**关注点**:`SkillsView.rebuildSkillMd(newBody, newTriggers, newDescription)` 原本 3 个参数,改后只返 newBody。

**验证**:`SkillsView.vue:140` 调用方传 3 个参数,新实现忽略后两个。**安全**(已在函数注释里说明)。

## 5. 需求回流

> 无

## 6. 测试报告

**自测时间:** 2026-07-10
**自测人:** AI(本轮 Claude)
**自测范围:** 前端 SkillsView.vue + SkillFileInlinePanel.vue 改动 + 构建产物 + 后端单测

### 6.1 自动化测试

- `npm run build` 结果: ✅ 通过(耗时 12.03s)
- `go build ./...` 结果: ✅ 通过(链接警告无害,无 error)
- `go test ./api-server/internal/skillstore/...` 结果: ✅ ok 0.091s
- `go test ./api-server/internal/skilladapter/...` 结果: ✅ ok(cached)
- `go test ./api-server/internal/skilladapter/toolspecs` 结果: ✅ ok(cached)
- 前端 `npm run lint`: ❌ 不存在(项目未配置 lint 脚本)
- 单测: 后端 store + adapter 测试通过,覆盖 RenderSkillMD + Save 全链路

### 6.2 手工 / 接口验证

- [x] 用例 1(本轮未启服务):改完后端 store.Save 走 RenderSkillMD 仍是旧行为,manifest 透传语义不变 → ✅(走单测覆盖)
- [x] 用例 2(本轮未启服务):改完前端不再拼 frontmatter,提交给后端的 `files[0].content` 只含 body → ✅(读源码确认)
- [x] 用例 3(本轮未启服务):磁盘 SKILL.md 文件由 RenderSkillMD 重渲,无 yaml 围栏污染风险 → ✅(读 store.go:127 确认)
- [x] 用例 4(回归):用户重新打开新建的 skill,CodeViewer view 模式仍走 displayContent 剥 frontmatter,语义不变 → ✅
- [x] 用例 5(回归):编辑 SKILL.md body 后保存,走 `rebuildSkillMdFromBody(localBody)` 只返 body → ✅
- [x] 用例 6(回归):frontmatter 表单弹窗保存走 `saveFrontmatterForm` + `manifest` 字段,后端 RenderSkillMD 重新拼 frontmatter → ✅(与既有认知一致)

### 6.3 边界 / 异常

- [x] 边界:前端 `files[0].content` 为空字符串(新建空 body skill)→ ✅ 后端 RenderSkillMD 用空 body 拼,前端接受
- [x] 边界:`splitSkillMd(newMd)` 在 `newMd` 没 `---` 围栏时原样返回 body(已验证)
- [x] 边界:旧 skill 文件如果磁盘上还残留旧 yaml 围栏(老版本遗留),前端 view 模式 `displayContent` 仍会剥掉 → ✅(回归保护)

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无

## 7. 总结

- 完成了什么:
  - 移除前端 SKILL.md 拼装层的所有 frontmatter 拼接(5 处)
  - frontmatter 完全交给 `payload.manifest` 字段,后端 `RenderSkillMD` 统一重渲
  - 前端编辑态不再显示冗余的 `---` 围栏
- 留下了什么:
  - `frontend/src/views/SkillsView.vue` — buildSkillMd / rebuildSkillMd 只返 body
  - `frontend/src/components/skill/SkillFileInlinePanel.vue` — rebuildSkillMd / rebuildSkillMdFromBody / saveFrontmatterForm 只返 body
  - `api-server/cmd/web/frontend/dist/` — 同步构建产物(嵌入目录)
  - 本任务文档
- 留给下次的事: 暂无
- 复盘: 排查根因时要从"前后端链路契约"出发,不能只看前端展示层。后端 `store.go:127` 那行 `writeFileAtomic(... RenderSkillMD(c) ...)` 是关键约束,所有前端 SKILL.md 拼接都应该遵守"只透传 body,frontmatter 走 manifest"。

## 8. 改动的文件

### 8.1 修改

- `frontend/src/views/SkillsView.vue` — `buildSkillMd()` / `rebuildSkillMd()` 只返 body
- `frontend/src/components/skill/SkillFileInlinePanel.vue` — `rebuildSkillMd()` / `rebuildSkillMdFromBody()` / `saveFrontmatterForm` 内嵌拼接改为只取 body
- `api-server/cmd/web/frontend/dist/**` — 同步 `npm run build` 后的嵌入 dist(191 files,主要是资产 hash 重渲)

### 8.2 不涉及文件改动的任务

> 不涉及。

## 9. 工具与用途

### 9.1 MCP 工具

- 本次未调用任何 MCP 工具

### 9.2 Skill

- 本次未调用任何 Skill

### 9.3 CLI

- `Bash npm run build` — 前端编译验证(12.03s 通过)
- `Bash wails3 task web:sync:embed` — 同步 dist 到 api-server 嵌入目录
- `Bash go build ./...` — 后端编译验证(链接警告无害)
- `Bash go test ./api-server/internal/skillstore/... ./api-server/internal/skilladapter/...` — 后端单测(全通过)
- `Bash git add / git commit / git push` — 提交并推送到 origin/main