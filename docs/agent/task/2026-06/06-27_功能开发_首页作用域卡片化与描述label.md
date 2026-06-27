# 首页作用域卡片化与描述 label

**日期:** 2026-06-27
**状态:** 已完成

## 1. 需求

用户对首页技能详情面板提出两点改造:

1. **作用域部分改成卡片样式** —— 与上下部分(标题 toolbar / 正文区)进行隔离,视觉上更美观。
2. **详情和触发词没占满整页宽度** —— 约 60% 处就自动换行,希望占满整个宽度后再换行。

后续补充调整(用户中途改需求):

3. **作用域卡片背景色改用白色**,与正文区域进行隔离。
4. **撤销第二点的"拆出独立卡片"方案** —— 描述和触发词改回 toolbar 内,在标题下方同一块里展示。
5. **描述前面加一个 label** —— 像触发词 label 一样(11px, uppercase, 字间距 0.3px)。
6. **修复作用域卡片下方边框没显示的问题** —— 加 detail-body 顶部分割线形成清晰分段。

## 2. 任务列表

- [x] 作用域区做独立卡片(白底 + 圆角 + border + margin)
- [x] 描述 + 触发词从 toolbar 拆出(占满 detail-pane 宽度)—— 中途被撤回
- [x] 撤销独立卡片方案,改回 toolbar 内行内展示
- [x] 描述前加 desc-label(与 triggers-label 同款)
- [x] 修复 scope-card 底部边框视觉断感(给 detail-body 加 border-top 形成分段)
- [x] i18n 加 descShort 字段(中英)
- [x] 前端构建验证
- [x] git commit + git push

## 3. 执行进度

- 第一轮:改造 scope 区为 info-card / scope-card 双卡片(灰底);拆出 description/triggers 到独立卡片占满 detail-pane
- 第二轮:用户改需求 → scope-card 改白底,info-card 改白底对称;加 detail-body 顶部分割线
- 第三轮:用户再改需求 → 撤销 info-card,description/triggers 回 toolbar 内;description 前加 desc-label;scope-card 边框问题修复

## 4. 问题与方案

**问题 A: 描述和触发词没占满宽度(60% 换行)**
- 现象:用户在 toolbar 内看到 description/triggers 只占 toolbar 内左侧 ~60% 空间就被右侧 6 个图标按钮挤到换行
- 定位:`.detail-toolbar` 是 `flex; space-between`,`.detail-title-block` 是 `flex:1`,所以 title-block 实际宽度 = toolbar 宽 - 212px(right actions)
- 方案:第一版拆出独立 info-card 占满整个 detail-pane 宽度;第二版用户撤回,改回 toolbar 内行内展示(此时占满的是 title-block 内的 100% 宽度,即 toolbar 左侧全部空间)
- 教训:用户对"占满整个页面宽度"的理解可能跟我不同,先问清楚比贸然拆布局要好。后续遇到布局歧义先用 AskUserQuestion 确认。

**问题 B: scope-card 底部边框没显示**
- 现象:作用域卡片底部 border 看起来"消失"了
- 定位:`.detail-section` 基础类有 `border-bottom: 1px solid var(--border)`(单类),`.detail-section.scope-card` 有 `border: 1px solid var(--border)`(双类)。后者优先级更高理论上覆盖前者。但 scope-card 用了 `border-radius: 6px`,底部圆角让 border 看起来弧度变浅,加上紧贴的 detail-body 没有顶部分割线,整体视觉割裂。
- 方案:scope-card 保持完整四边 border + `border-radius: 6px`;给 detail-body 加 `border-top: 1px solid var(--border)`,形成"▢ 卡片 → 12px 空白 → ━ 分隔线 → 正文"的清晰分段
- 教训:CSS cascade 中多类优先级高的 shorthand 会覆盖基础类的单边属性,但视觉效果仍依赖相邻元素的边界处理,需要从整体布局看。

## 5. 需求回流

无新需求外溢。

## 6. 测试报告

**自测时间:** 2026-06-27
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(1.41s)

### 6.2 手工验证
- [x] scope-card 白底卡片化、与 detail-body 之间有清晰分段 ✅
- [x] description 前面加 desc-label,样式与 triggers-label 一致 ✅
- [x] description/triggers 仍在 toolbar 内、标题下方同一块里展示 ✅

### 6.3 边界 / 异常
- [x] 无 description 时:行容器 v-if 不渲染,不会留空 label ✅
- [x] 无 triggers 时:行容器 v-if 不渲染,不会留空 label ✅

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题:无

## 7. 总结

- **完成了什么**:作用域区做白底卡片化(与正文区分段);description 前面加 desc-label 与 triggers-label 对齐;i18n 加 descShort 字段
- **留下了什么**:`SkillsView.vue` 的 scope-card / detail-desc-row / detail-body 样式 + i18n 文案
- **留给下次的事**:无
- **复盘**:
  - 做得好:中途用户改需求时,及时撤销错误的 info-card 拆出方案,避免双卡片冗余
  - 需改进:第一版就该先用 AskUserQuestion 确认"占满宽度"的精确语义(整个 detail-pane 还是 toolbar 左侧),免得来回返工

## 8. 改动的文件

### 8.1 修改
- `frontend/src/views/SkillsView.vue` — scope-card 卡片化样式 + detail-desc-row 容器 + desc-label + detail-body border-top
- `frontend/src/core/i18n/zh-CN.js` — 加 `skills.editor.descShort = '描述'`
- `frontend/src/core/i18n/en-US.js` — 加 `skills.editor.descShort = 'Desc'`

### 8.2 删除
- 无

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- `Skill ui-ux-pro-max` — 三段视觉布局方案评估(白底卡片风格与项目极简基调一致)

### 9.3 CLI
- `Bash npm run build` — 前端编译验证(1.41s 通过)