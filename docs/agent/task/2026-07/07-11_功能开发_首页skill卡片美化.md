# 2026-07-11 功能开发 — 首页 skill 卡片美化

## 需求

用户反馈首页技能列表中的盒子（skill 卡片）太单调——目前只有灰色主配色（白底 + 灰边框 + 浅灰阴影），与同仓库里 MarketView（卡顶 4px accent 条 + 渐变 + color-mix 配色）和 ToolsView（teal→emerald 渐变主题）相比，首页 skill 卡片缺少视觉重点和语义区分。

要求：给 skill 卡片加上配色，跟其它视图保持统一的设计语言。

## 方案

采纳"蓝色主体 + chip 分色"模式：

1. **卡片主体**：统一走蓝色 accent（MarketView 风格）
   - 卡顶 4px 蓝色 accent 条
   - 卡顶 6% 蓝色渐变 → bg-card（顶部 70% 范围，底部保持纯白）
   - hover 时边框 + 阴影带蓝色色温（border 用 color-mix 40% accent，阴影 30% accent）
   - 引入 `--card-accent` 局部变量，未来想给不同 skill 走不同 accent 只需覆盖这一个变量

2. **chip 按工具分色**（5 色，对应 5 个工具）
   - codex → accent-blue（蓝）
   - claude → accent-emerald（翠）
   - cursor → accent-amber（橙）
   - opencode → accent-violet（紫，仅 chip 用，克制）
   - trae → accent-rose（玫）
   - 兜底 → 现有灰底灰字
   - chip hover 加深用 `filter: brightness(0.97)`

3. **选中态升级**
   - 从单一蓝色边框升级为：accent 渐变背景（18% → 6% → bg-card）+ 三层 accent 色光阴影（内描边 ring + 主光晕 6px/18px 35% accent + 微近景 2px/6px 20% accent）
   - hover 时阴影升级到 8px/22px 45% accent
   - 选中卡片内的 name 变蓝（强化焦点）

4. **暗黑模式**
   - 不写独立 dark 规则，复用 style.css `.dark {}` 已有的深色 accent 变量
   - 浅色 accent-blue（#2563eb）→ 暗黑 #60a5fa，color-mix 自动适配 `--bg-card`（#171717）

## 配色约束（来自项目 memory）

- `avoid-violet-as-primary-color.md` 禁止紫色作为项目主色 → 本方案仅在 opencode chip 用紫色（chip 强调色），不蔓延到卡片主体，符合"主色禁用，辅助允许"
- 主色候选：蓝/绿/橙/红/灰 → 卡片主体走蓝色（候选中第一个），与 MarketView / ToolsView 主题形成连贯
- 项目统一图标库 iconpark → 本次未涉及图标修改，沿用现有 chip icon

## 关键修改点

| 文件 | 位置 | 改动 |
|---|---|---|
| `frontend/src/components/TreeNode.vue` | 行 161-167 | 新增 `TOOL_ACCENT` 常量（5 个工具 → accent 名映射） |
| `frontend/src/components/TreeNode.vue` | 行 276 | 模板 chip `:class` 加 `'tool-chip-' + (TOOL_ACCENT[tid] \|\| 'default')` |
| `frontend/src/components/TreeNode.vue` | 行 410-430 | `.tree-row-skill` 主体加 `--card-accent` / 卡顶 4px 条 / 卡顶渐变 |
| `frontend/src/components/TreeNode.vue` | 行 431-438 | `.tree-row-skill:hover` 升级（border / 阴影带 accent 色温） |
| `frontend/src/components/TreeNode.vue` | 行 478-523 | `.tree-tool-chip` 微调 + 5 个 `.tool-chip-accent-*` 变体 + hover 加深 |
| `frontend/src/components/TreeNode.vue` | 行 540-565 | `.tree-node-selected > .tree-row-skill` 升级（渐变背景 + 三层阴影 + name 变蓝） |

未修改：
- `frontend/src/style.css`（accent 变量已有，浅色 / 暗黑都已定义好）
- `frontend/src/views/SkillsView.vue`（容器样式无需改）
- `frontend/src/views/MarketView.vue` / `ToolsView.vue`（参考样板，只读不改）

## 复用现有资源

- 全局变量 `--accent-blue / -emerald / -amber / -rose / -violet` 及 `-bg`（50-tint 浅底）/ `-border`（200-tint 浅边）— `frontend/src/style.css` 行 40-54（浅色）、行 123-137（暗黑）
- `color-mix(in srgb, var(--accent) X%, transparent)` 渐变和阴影 — `frontend/src/views/MarketView.vue` 行 925、940、994 已大量使用，浏览器兼容已确认

## 验证

执行：
1. ✅ `cd frontend && npm run dev` 启动 vite dev server（端口 5173）
2. ✅ chrome-devtools MCP 打开首页 skill 列表
3. ✅ 截图浅色模式：`docs/agent/task/skill_card_v1_light.png`
4. ✅ 截图选中态（aa 卡片）：`docs/agent/task/skill_card_v1_light_selected.png`
5. ✅ 切到暗黑模式截图：`docs/agent/task/skill_card_v1_dark.png`

视觉确认（来自 MiniMax understand_image 分析）：
- ✅ 卡顶 4px 蓝色横条 — 全部卡片统一有
- ✅ chip 按工具分色 — Claude=绿、Codex=蓝、Cursor=橙、Trae=红、OpenCode=紫 五色清晰可读
- ✅ 选中态层次 — 蓝色边框 + 蓝色光晕 + 顶部横条三重信号，name 变蓝形成同色呼应
- ✅ 暗黑模式自适应 — 深色背景下 chip 文字对比度达标，无灰阶断层

## 后续可优化（非本次范围）

用户截图反馈提到的两点（已记入文档，未在本次实现）：
1. Trae 红饱和度偏高 → 可换成更柔和的玫红/珊瑚色，让卡片之间更平衡
2. 卡片之间间距可加 1-2px，让顶部蓝条"分隔"作用更突出

如果用户后续提需求再迭代，本次以最小改动验证视觉方案。