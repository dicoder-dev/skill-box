---
name: fe-compact-source-list-sidebar
description: 左侧技能栏紧凑化改造 — 用间距分组代替 border-bottom 分割线、技能卡片改扁平行范式
metadata:
  type: feedback
---

用户在 skill-box 技能页左侧栏提出的改造要求(2026-10-30):「左侧的布局改紧凑一点,
还有美化一下左侧栏目,现在的样式有点丑,新建和导入下方有下划线,搜索框在也有下划线」。

**Why:** 用户明确不喜欢"横线切割"的侧栏,也不喜欢浮夸的渐变卡片风。
这与顶级桌面应用(Linear / VS Code / Arc / macOS Source List)的范式一致 ——
密集侧栏靠**留白 + 背景色块**分层,不靠 border-bottom 分割线;列表项靠
**淡底 + 左侧色条**表达选中,不靠光晕阴影。

**How to apply:**
- 侧栏分区**不要**加 `border-bottom`,改用 padding/gap 做间距分组。
- 搜索框改**填充式**(filled):`--bg-subtle` 底 + 透明边框,focus 时才显强调色描边。
- 列表项改**扁平行**:去渐变、去 box-shadow、去 translateY 浮起;
  hover = 中性底 `--bg-hover`;选中 = `--accent-blue-bg` 淡底 + 2px 左色条 + 强调色文字。
- 主次按钮分权重:主操作实心强调色,次操作 ghost 无边框。两个同款描边按钮没有视觉焦点。
- 强调色实心底上的文字色走 `--accent-blue-fg` token,暗色模式自动切深色
  (暗色下 accent-blue 是亮蓝,白字对比度仅 ~2.3:1,不合格)。
- 分组行字号可以跟内容行一样(13px),靠**字重** 600 拉层级,不用字号,更紧凑。

相关:[[fe-icon-library]] [[fe-hide-scrollbar-everywhere]] [[avoid-violet-as-primary-color]]
