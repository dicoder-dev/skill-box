---
name: fe-md-outline-nav
description: CodeViewer 右侧大纲导航 — md-it heading_open 重写 + extractHeadings 抽 title + 可收起 + 全局状态持久化
metadata:
  type: reference
---

# md 文件大纲导航实现要点(2026-07-10)

**位置**:`frontend/src/core/utils/markdown_view.js` + `frontend/src/components/skill/CodeViewer.vue` +
`frontend/src/core/composables/useMdOutlineVisible.js` + `frontend/src/components/skill/SkillFileInlinePanel.vue`

**实现三件套**:
1. **markdown_view.js heading_open 重写** — 给每个 h1-h6 加 `id="md-h-{slug}"`,slug 走
   `slugifyHeading`(小写 + 去标点 + 空格转 -)。同名标题用 env 内的 _headingIdCounts 自动
   追加 -1 / -2,id 必须全局唯一。
2. **extractHeadings(src)** — 走 md.parse 拿 tokens,按 heading_open + inline 顺序组装
   `{level, text, id}` 列表。**不 render html**,只 parse,省 CPU。
3. **CodeViewer .cv-md-wrap 改两列** — 左侧 cv-md-content 占满,右侧 cv-md-outline 220px
   固定宽,只在 view 模式 + 有标题 + outlineVisible 时显示。

**关键坑**:
- `inline.content` 是 markdown 源文本(含 `*italic*` 残留),`inline.children` 是解析后的
  token 列表(content 字段是纯文本)。**抽 text 优先用 children**
- `getElementById` + `scrollIntoView` 接受任意 id 字符串(中文/括号/emoji 都行),无需 ASCII-only
- 缩进按"最小 level 提一档"算(`h.level - minHeadingLevel + 1`),避免"全是缩进很深的
  小标题";`.cv-md-outline-l1` ~ `.cv-md-outline-l6` 6 个级别,每级 +14px
- 大纲 aside 条件 `v-if="!editable && mdHeadings.length && outlineVisible"`,短 md / 无标题
  / 用户已收起时都不渲染
- 滚动后给目标加 `cv-md-heading-active` 临时高亮 class(1200ms 后移除),`setTimeout` 兜底

**全局可收起状态(2026-07-10 v2)**:
- `useMdOutlineVisible` composable:模块级 Ref<boolean> 单例 + localStorage 持久化
  (`skillbox.mdOutlineVisible`,值为 `'0'` / `'1'`),跨刷新 + 跨文件保留
- 两处控制入口共享同一份 state:
  1. SkillFileInlinePanel 顶栏编辑按钮右侧(同 sfip-mode-btn 风格,
     bookmark-plus/minus 图标暗示当前状态,data-tip 同步文案)
  2. CodeViewer 大纲 header 右上角(22×22 小尺寸按钮,跟 count 徽章同排)
- watch(_visible, ...) 触发 localStorage 写入,try/catch 兜底隐私模式 / 磁盘满
- 默认 `true`(显示),用户首次使用不会看到空 panel

**反向高亮(scroll 同步)**:可用 IntersectionObserver,本期没做。

**相关**:
- [[fe-icon-library]] — 大纲 header 图标 mdi:format-list-bulleted;顶栏按钮 mdi:bookmark-plus/minus-outline
- [[avoid-violet-as-primary-color]] — 大纲 panel 用纯白 #ffffff + border,无紫色
