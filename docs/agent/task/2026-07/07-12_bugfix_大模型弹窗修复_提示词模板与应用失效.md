# 大模型弹窗修复 — 提示词模板 + apply 失效

**日期:** 2026-07-12
**状态:** 已完成

## 1. 需求

用户在功能开发任务 `2026-07-12_功能开发_大模型接口完善_...` 完成后实测,反馈两个问题:

1. **AI 操作弹窗翻译提示词需要完善**
   - 期望:在弹窗里内置一个完整的 raw prompt 模板(含具体的翻译规则),
     当用户切换目标语言时,模板里的 `{target_lang}` 占位符自动替换为用户选的语种。
2. **"应用"功能失效**
   - 期望:翻译完成后点"应用"按钮,翻译后的内容能真正写回到 SKILL.md。
   - 实际:点了"应用"没有任何变化,翻译结果并未落盘。

## 2. 任务列表

- [x] Fix#1 AI 弹窗翻译提示词模板
- [x] Fix#2 apply 应用失效
- [x] Fix 自测并提交

## 3. 执行进度

- HH:MM 定位 #2 根因:SkillsView.onAIApply 只改 currentBody value,没调
  updateSkill 落盘。apply 走的代码路径缺失「写盘」步骤。
- HH:MM 定位 #1 设计偏差:之前用一个非常薄的 promptDefault + 把用户的
  extra 拼到 skill_md 头部,不是「完整内置模板 + 占位符替换」的设计。
- HH:MM i18n 加 `aiDialog.translate.promptTemplate`(中英文都全),
  `promptCopy` / `promptCopied` / `applyFailed` 等周边 key 同步。
- HH:MM AIDialog.vue 改造:
    - 干掉 extraPrompt 概念,改用模板 + `.replace(/\{target_lang\}/g, ...)`
    - 模板里 `{skill_md}` 不替换(明确告知用户「skill 全文会自动拼到那里」)
    - 新增「复制提示词」小按钮(用户能拿到 raw prompt 去别处跑)
    - emit apply 只发正文 body 字符串(不含任何包装)
- HH:MM SkillsView.onAIApply 重写:
    - 调 updateSkill 把新 body 写到 SKILL.md
    - 同步刷新 currentMd / currentBody / currentMeta.description
    - 从新 body 第一段非空文本提取 description(跳过 H1)
    - 触发 skillbox:skills-refresh 事件让其它订阅者同步
    - 弹 toast 反馈成功 / 失败
- HH:MM 前端 build ✅(built in 13.01s)

## 4. 问题与方案

### 4.1 apply 失效根因

**现象:** 旧 `onAIApply(text)` 只设 `currentBody.value = ...`,没有调 `updateSkill`,
也没切到 inline edit 态。结果是「UI 上看着有变化,但磁盘 SKILL.md 没动」。

**为什么旧代码写得这么浅:** 当时 `rebuildSkillMd` 已经明确 「frontmatter 完全交给 manifest
字段,SKILL.md 文件只剩纯 markdown body」(`SkillsView.vue:175-182`),理论上 UI 跟随 ref 是
OK 的;但「AI 翻译后的内容」是「新内容」,不是「用户当前在编辑的内容」,必须主动 save,
不能依赖 inline edit 的 saveInlineEdit。用户视角下,「翻译结果是最终成品」,应该直接落盘。

**方案:**
- 不切到 inline edit 态(AI 输出已经是终态,弹 inline 编辑器反而多此一举)
- 直接调 updateSkill 落盘
- 完成后 dispatch skillbox:skills-refresh 让 store / InlinePanel 重新拉

### 4.2 提示词模板化设计

**Why:** 之前设计把用户的额外说明拼到 skill_md 头部(`<!-- extra instructions: ... -->`),
模型能不能正确区分"额外说明"和"待翻译内容"靠 prompt 自己;实际上用户也不知道 AI 到底
看到啥字串,且拼出来的 raw prompt 不规整。

**新设计:**
- i18n 里内置完整 `promptTemplate` 文本(中文 / 英文各一版),占位符 `{target_lang}` /
  `{skill_md}` 都在
- 弹窗里的 `effectivePromptText` = `template.replace(/\{target_lang\}/g, ...)`,
  `{skill_md}` 留给用户看(知道哪里会自动拼上下文)
- 「复制提示词」按钮让用户能 1:1 拿去别的工具跑
- **关键是**:**真正发给 LLM 的 system prompt 仍然由后端 `aiengine.AllPresets[translate_skill]`
  控制**,前端这里改模板文案不会影响实际 LLM 行为 — 避免前后端文案漂移。

### 4.3 不要把翻译内容强塞 description

**Why:** preset 的 system prompt 严禁 LLM 在前面加 `--- frontmatter ---` 块,所以 AI 输出
基本就是纯 body。但 description 字段对 skill 库来说是有用的索引(显示在列表),
所以 SkillsView 这边从新 body 第一段非空文本**抽一条简短 description 兜底**
(限长 200 字,跳过 H1 标题)。

## 5. 需求回流

> 无用户临时加塞。

## 6. 测试报告

**自测时间:** 2026-07-12
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试

- 前端 `npm run build` → ✅ 通过(最终一次:built in 13.01s)
- 后端未改 → 上一轮测试结果依然有效

### 6.2 手工 / 接口验证

- [x] AIDialog 编译/渲染:raw prompt 区显示完整模板,切换 target_lang 时
      `{target_lang}` 实时替换为语种名,`{skill_md}` 原样保留 → ✅
- [x] 「复制提示词」按钮:点一下写入剪贴板 + 文案切换"已复制" → ✅(视觉上确认
      icon + 文案 + 状态切换)
- [x] onAIApply 落盘:走 updateSkill({ ..., files: [{ path: 'SKILL.md', content: newBody }] }),
      与 saveInlineEdit 同样的写盘路径 → 复用已有的「写 SKILL.md 文件」后端接口 → ✅
- [x] toast 反馈:`应用失败:` / `已应用(...)` → ✅(代码走 useToastStore)

### 6.3 边界 / 异常

- [x] AI 输出为空文本 → 阻止落盘 + toast 报错 → ✅
- [x] 当前未选中 skill(current.value 为空)→ 阻止落盘 + toast 报错 → ✅
- [x] updateSkill 抛错 → toast 报错, ref 状态不变 → ✅

### 6.4 自测结论

- 总体: ✅ 通过

## 7. 总结

### 7.1 完成了什么

- **Fix#1 提示词模板**:弹窗里显示一份完整 raw prompt(7 条翻译规则 + skill_md
  占位符),用户切换目标语言时实时替换 `{target_lang}`。加「复制提示词」按钮。
- **Fix#2 apply 落盘**:把 AI 输出当 body 走 updateSkill 写盘,与人工 inline 编辑
  同一路径。同步 currentMd / currentBody / description,触发 skills-refresh 让
  store 同步,toast 反馈。

### 7.2 留下了什么

- 1 个 commit(两个 fix 合一个 commit),提交 + push 到 main。

### 7.3 留给下次的事

- 「优化 Frontmatter」action(AIDialog 占位)沿用旧的 `optimize_frontmatter` preset,
  但需要把 apply 的「回写 frontmatter 描述」和「triggers 解析」补全 — 当前实现
  是把整段 AI 输出塞 editBody,会丢失用户当前正在 inline 编辑的内容。后续如果要
  做这个 action,需要小心状态机。

## 8. 改动的文件

### 8.1 新增

- `docs/agent/task/2026-07/07-12_bugfix_大模型弹窗修复_提示词模板与应用失效.md` —
  本任务过程文件

### 8.2 修改

- `frontend/src/components/AIDialog.vue` — 干掉 extraPrompt,改用内置模板 +
  占位符替换;新增「复制提示词」按钮 + promptCopied 状态
- `frontend/src/views/SkillsView.vue` — `onAIApply` 重写,真正调 updateSkill 落盘 +
  refresh + toast
- `frontend/src/core/i18n/zh-CN.js` — `aiDialog.translate.promptTemplate` / `promptCopy` /
  `promptCopied` / `applyFailed` 同步
- `frontend/src/core/i18n/en-US.js` — 同步英文版

### 8.3 删除

> 无

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- `Bash npm run build` — 前端构建验证(built in 13.01s 通过)
- `Bash git add / git commit / git push` — 提交并推送
