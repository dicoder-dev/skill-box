# 文件预览编辑状态独立化 + SKILL.md 损坏文件修复

**日期:** 2026-07-05
**状态:** 已完成

## 1. 需求

用户在首页技能详情区反馈两个问题:

1. **编辑状态全局污染**: 点击了某个文件的编辑按钮,再切换到其他文件时,该文件还停留在编辑态;期望每个文件独立记住自己的 view/edit 模式。
2. **Markdown 渲染乱码**: 部分 SKILL.md 渲染为乱码,例如 `/Users/brody/.skill-box/skills/pravidhi-ddgs-internet-search/SKILL.md`。

## 2. 任务列表

- [x] 问题 1: 修复 `editMode` 全局污染,改为按文件 path 独立记忆
- [x] 问题 2-1: 修复磁盘上损坏的 `pravidhi-ddgs-internet-search/SKILL.md` 文件(脚本清理)
- [x] 问题 2-2: 后端读文件后做 UTF-8 校验,非 UTF-8 返回明确错误
- [x] 问题 2-3: 前端拿到非 UTF-8 内容时给清晰提示
- [x] 自测与 commit

## 3. 执行进度

- 22:54 读取 SkillFileInlinePanel.vue 与 CodeViewer.vue,定位问题 1 根因
- 22:55 用 xxd 检查 pravidhi-ddgs-internet-search/SKILL.md,确认为磁盘文件被破坏(.DS_Store 8512 字节被写入 SKILL.md 文件)
- 23:00 与用户确认修复方案:按 path 独立记忆 + 脚本清理 + 加防御
- 23:05 改 SkillFileInlinePanel.vue:editMode ref → editModeMap reactive,currentMode computed
- 23:08 commit d829426 + push(问题 1)
- 23:10 改 skillstore.store.go:加 utf8.Valid 校验 + ErrCorruptedFile sentinel
- 23:11 改 skill.s.go:翻译 store sentinel 为本服务 sentinel(Get / GetByPath)
- 23:12 改 get_skill.a.go:识别 ErrCorruptedFile 返 422 + code=corrupted_file + hint
- 23:13 备份磁盘 SKILL.md → .bak.20260705,然后用 python 正则截到合法 frontmatter 末尾(343 字节 ASCII text)
- 23:13 删除磁盘上的 .DS_Store(污染源)
- 23:14 改 SkillsView.vue:识别 422 + corrupted_file 弹 toast + 错误区
- 23:14 加 i18n key(zh-CN + en-US):fileBrowser.corruptedHint
- 23:15 加 3 个单测覆盖 ErrCorruptedFile 链路
- 23:16 跑 skillstore / sskill / skillpkg / skilladapter 单测全过(0.5s)
- 23:18 commit 3932ed2 + push(问题 2)

## 4. 问题与方案

### 问题 1 根因

`SkillFileInlinePanel.vue:46` 的 `editMode` 是组件级 ref,虽然 `onSelectFile` 有
`if (selectedKey.value !== file.path) editMode.value = 'view'` 重置逻辑,
但用户期望的是"每个文件独立记忆自己的 view/edit 态",所以即使该文件此前是 edit,
切走再回来又变 view,体验不对。

**方案:** 改为 `editModeMap: reactive({})`,模板用 `currentMode = computed(() => getMode(selectedFile.path))`
代替旧的 `editMode`;切换文件不再 reset 该文件已记录的 mode;跨 skill 不会串扰
(SkillFileInlinePanel 通过 v-if 卸载重建)。

### 问题 2 根因

`pravidhi-ddgs-internet-search/SKILL.md` 文件 8855 字节,头部 200 字节是合法
YAML frontmatter,之后紧跟 .DS_Store / iCloud sync 的 plist 二进制片段
(`bplist00` / `scriptsbwspblob` 等键名)+ 大量 `0x00` 控制字符。
`file` 命令判定为 `data` 而不是 `ASCII text`。
后端 `os.ReadFile` 全字节读后 `string(bytes)` 走 UTF-8 解码,非法字节被替换为
`U+FFFD`(`EF BF BD`),前端 markdown-it 渲染出豆腐块。

**方案:**

- **磁盘:** 备份 → python 正则截到合法 frontmatter 末尾(343 字节)→ 删除污染源 .DS_Store
- **后端 4 层防御:** skillstore.ErrCorruptedFile sentinel + loadFromDir utf8.Valid 校验
  + sskill 翻译 store sentinel 为服务层 sentinel + cskill.get_skill 返 422 + code=corrupted_file
- **前端:** SkillsView loadCurrent catch 分支识别 422 + data.code=corrupted_file,
  toast 弹 8s + 错误区展示后端 hint
- **日志:** skillstore.List 遇到损坏文件跳过 + stderr log 一行(用户能从 ~/.skill-box/logs 看到)
- **测试:** 3 个新单测覆盖 ErrCorruptedFile 三入口(LoadByName / LoadByPath / List 跳过)

## 5. 需求回流

(暂无)

## 6. 测试报告

**自测时间:** 2026-07-05 23:16
**自测人:** AI(本轮 Claude)
**自测范围:** skillstore + sskill service + cskill controller + 前端 SkillsView/i18n

### 6.1 自动化测试

- `go test -count=1 ./internal/skillstore/... ./internal/gapi/service/skill/sskill/... ./internal/gapi/controller/skillbox/cskill/... ./internal/skillpkg/... ./internal/skilladapter/...` 结果: ✅ 全过(0.5s)
  - skillstore: ok 0.167s
  - sskill: ok 0.130s
  - cskill: no test files(没新增)
  - skillpkg: ok 0.160s
  - skilladapter: ok 0.016s
  - skilladapter/toolspecs: ok 0.005s
- 新增单测覆盖:
  - `TestLoadByName_CorruptedSKILL` ✅
  - `TestLoadByPath_CorruptedSKILL` ✅
  - `TestList_SkipsCorruptedAndLogs` ✅
- `npm run build` 结果: ✅ 通过(6.9s)

### 6.2 手工 / 接口验证

- [x] 磁盘文件修复: `file SKILL.md` 改前 `data`,改后 `ASCII text`,343 字节,UTF-8 合法
- [x] 备份保留: `SKILL.md.bak.20260705` 仍在目录里,用户可从备份恢复原内容
- [x] 污染源清理: `~/.skill-box/skills/pravidhi-ddgs-internet-search/.DS_Store` 已删

### 6.3 边界 / 异常

- [x] utf8.Valid 用 0xff 0xfe 测试字节触发 → 正确返回 ErrCorruptedFile
- [x] List 跳过损坏文件 → 0 items returned,stderr log 包含路径
- [x] frontmatter-only 文件(无 body)→ utf8.Valid 通过,正常 load

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: pkg/task 600s 超时是项目原有 cron 测试,与本改动无关(已在 /tmp 输出确认)

## 7. 总结

### 完成了什么

- **问题 1**: SkillFileInlinePanel 改为按文件 path 独立记忆 view/edit 状态,切走再回来仍保留模式。
- **问题 2**: 4 层防御体系(skillstore sentinel → service 翻译 → controller 422 → 前端 toast),
  任何后续遇到磁盘文件损坏都会返清晰提示,而不是渲染豆腐块。
- 磁盘上 pravidhi-ddgs-internet-search/SKILL.md 修复(343 字节合法 frontmatter,备份在 .bak.20260705)。
- 3 个新单测覆盖 ErrCorruptedFile 三入口。

### 留下了什么

- 2 个 git commit(d829426 + 3932ed2),都已 push 到 origin/main。
- 新增 sentinel `skillstore.ErrCorruptedFile` / `sskill.ErrCorruptedFile`,后续其他损坏场景复用。
- 新增 i18n key `skills.fileBrowser.corruptedHint`(zh-CN + en-US)。
- 磁盘文件 `SKILL.md.bak.20260705`(8855 字节,含原始二进制,用户可从备份人工恢复正文)。

### 留给下次的事

- 用户需要在 UI 上把 SKILL.md 的 body 部分重新编辑(截断时只剩 frontmatter)。
- 其它 skill 目录里可能也有 .DS_Store 残留(全局排查是 macOS 用户习惯问题,本次未批量清理)。

### 复盘

- **做得好:** 在用 AskUserQuestion 让用户先选方案后再动手,避免做完发现选错方向。
- **做得好:** 加 sentinel error 而不是 magic string,errors.Is 跨包识别更可靠。
- **可改进:** 先用 chrome 跑前端验证行为(用户说"还在编辑状态"),而不是只看代码逻辑——代码上明明 reset 了,但用户的"独立记忆"是更高阶的 UX 期望。

## 8. 改动的文件

### 8.1 新增
- (无新文件,均为修改)

### 8.2 修改

- `frontend/src/components/skill/SkillFileInlinePanel.vue` — editMode 组件级 ref 改为 editModeMap reactive + currentMode computed(36 行新增,21 行删除)
- `api-server/internal/skillstore/store.go` — 加 unicode/utf8 import + ErrCorruptedFile sentinel + loadFromDir utf8.Valid 校验 + collectSkillsRecursive 损坏文件 stderr log
- `api-server/internal/skillstore/store_test.go` — 加 3 个单测覆盖 ErrCorruptedFile
- `api-server/internal/gapi/service/skill/sskill/skill.s.go` — 加 ErrCorruptedFile sentinel + Get/GetByPath 翻译 store sentinel
- `api-server/internal/gapi/controller/skillbox/cskill/get_skill.a.go` — 识别 ErrCorruptedFile 返 422 + code=corrupted_file + hint
- `frontend/src/views/SkillsView.vue` — loadCurrent catch 分支识别 corrupted_file 弹 toast + 错误区展示 hint
- `frontend/src/core/i18n/zh-CN.js` — 加 fileBrowser.corruptedHint
- `frontend/src/core/i18n/en-US.js` — 加 fileBrowser.corruptedHint

### 8.3 删除

(无)

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI

- `Bash ls /Users/brody/.skill-box/skills/` — 看磁盘 skill 列表
- `Bash file SKILL.md` — 确认损坏文件被 file 判定为 data(改后变 ASCII text)
- `Bash xxd` — 取字节确认二进制污染源是 .DS_Store
- `Bash stat -f` — 对比正常 SKILL.md 文件大小
- `Bash python3` — 备份 + 正则截断损坏文件到合法 frontmatter 末尾
- `Bash cp /Users/brody/.skill-box/skills/pravidhi-ddgs-internet-search/SKILL.md ...bak.20260705` — 损坏文件备份
- `Bash rm .DS_Store` — 删除污染源
- `Bash go build ./...` — 后端编译验证(无错误)
- `Bash go test ./internal/skillstore/...` — 3 个新单测全过
- `Bash go test -count=1 ./internal/skillstore/... ./internal/gapi/service/skill/sskill/... ./internal/gapi/controller/skillbox/cskill/... ./internal/skillpkg/... ./internal/skilladapter/...` — 改动涉及的所有包全过
- `Bash npm run build` — 前端构建验证(6.9s 通过)
- `Bash git commit && git push` — 2 个 commit 都已 push 到 origin/main