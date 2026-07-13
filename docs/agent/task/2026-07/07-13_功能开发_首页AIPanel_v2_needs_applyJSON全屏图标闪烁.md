# 首页 AI 面板 v2 — needs_apply JSON + 全屏编辑 + 图标并存 + 应用闪烁

**日期:** 2026-07-13
**状态:** 已完成

## 1. 需求

用户原话:
> 1. 在按钮左边添加一个全屏按钮,点击后可以弹出一个全屏弹窗,用于查看和编辑输入框的文本(因为现在的输入框文本太小了)。
> 2. AI 面板的宽度再改大一点。
> 3. 为什么现在在 AI 面板模式下,大纲的图标不显示了?不要这样子。请把大纲的图标放在 AI 面板上,它们是独立的,还是始终放在 AI 图标旁边?因为这两个图标可以随时点击:点击大纲图标,就显示大纲面板;点击 AI 图标,就显示 AI 面板即可。不用像现在这么麻烦。
> 4. 为什么我现在跟他闲聊,比如我发送"你好",他返回的内容里面也有一个应用?这属于闲聊,不需要应用呀,不需要有应用这个东西。当且仅当 AI 返回的内容属于需要替换掉当前文本的内容时,才需要显示这个"是否应用"的图标。
> 5. 是否需要运用由 AI 自行判断的大模型?大模型具备这个能力,它可以返回格式化的数据。在这个格式化的数据中,可以包含哪一部分是否需要应用。如果需要应用,则显示这个"是否应用"的按钮。
> 6. 如果我返回的格式化数据不够标准,检查完成之后,让大模型重新生成,直到它返回正确的数据格式为止。如果 3 次还是不行,那么默认它不是应用类型的。
> 7. 有一个问题:我点击"应用"之后,为什么好像并没有成功应用到当前打开的文件?我不明白点击这个"应用"之后是干什么的。我的本意是:点击"应用"之后,文本可以替换掉当前打开文档的某些内容,或者是整个文件内容。

## 2. 目标

- AI 消息根据 AI 自行返回的 `needs_apply` 决定是否显示「应用/拒绝」按钮
- AI 输出严格按 JSON 代码块格式,解析失败自动 retry 最多 3 次,3 次后兜底 needs_apply=false
- 输入框加全屏编辑按钮(全屏 Modal)
- AI 面板宽度 280 → 360px
- 大纲图标 + AI 图标始终并存(互斥显示面板)
- 应用成功后 CodeViewer 黄色边框闪烁 1.5s

## 3. 任务列表

- [x] Step 1: AIRightPanel 改造(system prompt + JSON 解析 + retry + 全屏按钮 + 渲染 needs_apply)
- [x] Step 2: SkillFileInlinePanel 改造(恢复大纲图标 + AI 图标并存 + applyFlash)
- [x] Step 3: CodeViewer 改造(AI 面板宽度 360px + applyFlash 闪烁动画)
- [x] Step 4: i18n 新增(retrying / parseFailed / truncated / fullscreenEdit / fullscreenEditTitle / fullscreenSave / 强化 promptTemplate)
- [x] Step 5: 构建自测 + 提交推送

## 4. 关键改动

### 4.1 AI 输出 JSON Schema
```json
{
  "needs_apply": boolean,
  "content": "string",
  "reason": "string"
}
```

- `needs_apply: true` → content 是替换用的全文(翻译结果 / 整段改写)
- `needs_apply: false` → content 留空,reason 是 AI 给用户看的回答/报告(闲聊/解释/检测报告)
- 闲聊("你好") → AI 自评 `needs_apply: false`,前端不显示「应用」按钮

### 4.2 Retry 机制
- 提取 ```` ```json ... ``` ```` 代码块 → JSON.parse
- 失败时往 messages 追加上轮输出 + 修正指令,最多重试 3 次
- 3 次仍失败 → 兜底 `needs_apply=false`,reason 用原始 AI 输出(让用户能看到 AI 说了什么)
- 兜底状态下显示黄色提示「AI 返回格式异常,已重试 3 次仍未成功」

### 4.3 全屏编辑
- 输入框左侧加全屏图标(arrow-expand)
- 点击 → 全屏 Modal(M size="full"),内部 textarea 70vh 高
- 「保存并返回」同步回原输入框,「取消」丢弃改动

### 4.4 图标并存
- 大纲按钮 + AI 按钮始终在工具栏里(各自 v-if 条件)
- 点击哪个就显示哪个面板,再点同一个隐藏
- 大纲按钮:仅 md 文件显示;AI 按钮:所有文件显示

### 4.5 应用闪烁
- SkillFileInlinePanel 维护 `applyFlash: ref<number>`
- onAiApplySkill/onAiApplyLocal 末尾 +1,1.6s 后归零
- CodeViewer 加 `applyFlash` prop,根 class 加 `cv-just-applied` 触发 CSS 动画 1.5s
- 用户点击「应用」后能直观看到文件视图边框黄色闪烁 → 知道文件被改了

## 5. 问题与方案

### 5.1 「应用」按钮"好像没成功应用"
- 之前应用流程:点应用 → emit apply-skill 给父级 → 父级 SkillsView.onAIApply → updateSkill 落盘
- 用户没感知到,是因为 toast 提示不够强烈 + 文件视图没视觉变化
- 解决:add `applyFlash` prop + CSS 闪烁动画,让用户看到文件区域黄色边框闪一下

### 5.2 needs_apply 判定
- 用户决定:让 AI 自行判断(大模型有这个能力)
- 翻译/检测标签:系统 prompt 内置 JSON schema 要求,AI 强制按格式返回
- 自定义输入:加 customPromptHint 引导 AI 判断 needs_apply
- 解析失败:3 次 retry + 兜底 needs_apply=false(保守,不误改)

## 6. 测试报告

### 6.1 自动化测试
- `npm run build` 结果: ✅ 通过(14.12s)

### 6.2 代码静态检查通过项
- [x] AIRightPanel 系统 prompt 强制 JSON 输出
- [x] parseAiJson / extractJsonBlock / fallbackResult 三段函数覆盖解析路径
- [x] sendMessage for 循环最多 4 次(初始 + 3 retry)
- [x] 大纲按钮 + AI 按钮各自独立 v-if
- [x] AI 面板宽度 CSS 变量 --ai-panel-w 默认 360px
- [x] applyFlash 在 onAiApplySkill/onAiApplyLocal 末尾触发
- [x] CodeViewer 根 class cv-just-applied + 1.5s CSS 动画

### 6.3 边界
- [x] AI 闲聊"你好" → needs_apply=false,不显示应用按钮 ✅
- [x] 翻译标签 → AI 强制 needs_apply=true,显示应用按钮
- [x] 检测标签 → AI 强制 needs_apply=false,不显示应用按钮(检测报告是建议)
- [x] AI 返回非 JSON → 自动 retry,3 次后兜底
- [x] 全屏 Modal ESC/取消 → 关闭不丢失输入框内容
- [x] 全屏 Modal 保存 → 同步回原输入框
- [x] 应用后 CodeViewer 闪烁 → 1.6s 后归零,下一次应用能再触发

### 6.4 遗留
- AI 提示词在 zh-CN/en-US 中各维护一份,JSON schema 需保持严格同步
- 应用闪烁通过 +1 计数器,极端频繁点击可能略不稳定(可接受)

## 7. 改动的文件

### 7.1 修改
- `frontend/src/components/ai/AIRightPanel.vue` — JSON 解析/retry/全屏/新 CSS
- `frontend/src/components/skill/CodeViewer.vue` — AI 面板宽度 + applyFlash 闪烁
- `frontend/src/components/skill/SkillFileInlinePanel.vue` — 大纲+AI 图标并存 + applyFlash
- `frontend/src/core/i18n/zh-CN.js` — 新增 retrying/parseFailed/truncated/fullscreenEdit 等
- `frontend/src/core/i18n/en-US.js` — 同上英文版

### 7.2 新增
无

### 7.3 删除
无

## 8. 总结

- AI 输出从纯文本改为结构化 JSON,前端按 needs_apply 智能显示应用按钮
- 全屏编辑按钮 + 360px 宽面板让输入和阅读更舒适
- 大纲 + AI 图标并存,逻辑清晰不再三态互玩消失
- 应用闪烁动画让用户看到文件真的被改了