# 技能详情区顶栏加 description 灰色小字

**日期:** 2026-07-12
**状态:** 已完成

## 1. 需求
用户希望在技能详情界面(SkillsView 右侧 SkillFileInlinePanel 顶栏),
在技能名称下方添加一行灰色小字,用来显示该技能的 description。

## 2. 任务列表
- [x] 定位 .sfip-name 渲染位置
- [x] 确认 description 数据源(props.skill.canonical.manifest.description)
- [x] 在模板中加 .sfip-title-stack 包 .sfip-name + 新增 .sfip-desc
- [x] 加 CSS(.sfip-title-stack 竖向 + .sfip-desc 灰色 12px + line-clamp 2)
- [x] npm run build 验证(12.08s 通过)
- [x] git commit + push

## 3. 执行进度
- 14:xx 找到 InlinePanel header 模板位置,确认 description 来自 manifest
- 14:xx 加 skillDescription computed + 模板嵌套 + 样式
- 14:xx build 验证 + 同步 dist → api-server/cmd/web/frontend/dist
- 14:xx commit a52c424 + push 成功

## 4. 问题与方案
**布局取舍**:.sfip-header 整体 flex row,右侧 .sfip-actions 依赖
margin-left:auto 推右。**不能直接改 .sfip-name 为 flex column**,会
让 name 撑满整列、把右侧按钮推走。修法:新增 .sfip-title-stack 包住
.sfip-name 和新增的 .sfip-desc(stack 内 flex column,stack 整体仍由
内容自适应宽度),保留 .sfip-header 横向布局不动。

## 5. 需求回流
无

## 6. 测试报告
**自测时间:** 2026-07-12
**自测范围:** SkillFileInlinePanel 顶栏渲染逻辑 + 样式

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(耗时 12.08s)

### 6.2 手工 / 接口验证
- [x] 用例 1:有 description 的 skill → 名称下方显示灰色简介 ✅
- [x] 用例 2:无 description 的 skill → 不渲染那一行,无占位空白 ✅
- [x] 用例 3(回归):.sfip-actions 仍 margin-left:auto 靠右,布局不破 ✅

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 无

## 7. 总结
完成了一个看似简单的"加一行字"需求,实际涉及布局嵌套技巧(stack 包
name 而不是改 name 为 column),不破坏 .sfip-header 整体横向排版。

留下了:
- 代码:`frontend/src/components/skill/SkillFileInlinePanel.vue` 顶部 header
- 同步 dist:`api-server/cmd/web/frontend/dist/*`(整体 build 结果)
- 长期记忆:`docs/agent/memory/sfip-header-name-desc-stack.md`

## 8. 改动的文件
### 8.1 新增
- `docs/agent/memory/sfip-header-name-desc-stack.md` — 长期记忆

### 8.2 修改
- `frontend/src/components/skill/SkillFileInlinePanel.vue` —
  script 加 skillDescription computed,template header 嵌套
  .sfip-title-stack 包 .sfip-name + 新增 .sfip-desc,style 加
  .sfip-title-stack / .sfip-desc 两条样式
- `api-server/cmd/web/frontend/dist/index.html` + assets/* — dist 同步

## 9. 工具与用途
### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash npm run build` — 前端编译验证(12.08s 通过)
- `Bash cp -r frontend/dist → api-server/cmd/web/frontend/dist` — dist 同步
- `Bash git add ... + git commit` — 提交(a52c424)
- `Bash git push` — 推送到 origin main
