# 2026-07-11 功能开发 — 首页 skill 卡片美化 v2(删卡顶蓝色条)

## 需求来源

v1（commit affbc5c）上线后用户反馈：skill 卡片顶部的 4px 蓝色实色横条"有点用力过猛的感觉，删掉这个部分"。

## 改动

| 项 | v1 | v2 |
|---|---|---|
| 卡顶 4px 实色 accent 条 | ✅ `border-top: 4px solid var(--card-accent)` | ❌ 删除 |
| 卡顶浅蓝渐变背景 | ✅ 6% accent → bg-card 70% | ✅ 保留(更克制,几乎看不出,只留"色温") |
| hover 边框带 accent | ✅ `color-mix(40%, accent)` | ✅ 保留 |
| hover 阴影带 accent 光晕 | ✅ `30% accent` | ✅ 保留 |
| 选中态边框 + 渐变 + 三层阴影 | ✅ | ✅ 保留 |
| 选中态 `border-top-color` 同步 | ✅ | ❌ 随卡顶条一并删除(已无意义) |
| chip 按工具分色 | ✅ 5 色 | ✅ 保留 |

## 修改点

`frontend/src/components/TreeNode.vue`：

1. `.tree-row-skill` 删除 `border-top: 4px solid var(--card-accent);`（卡顶 4px 条）
2. `.tree-row-skill:hover` 删除 `border-top-color: var(--card-accent);`（hover 时同步卡顶条）
3. `.tree-node-selected > .tree-row-skill` 删除 `border-top-color: var(--accent-blue);`
4. `.tree-node-selected > .tree-row-skill:hover` 删除 `border-top-color: var(--accent-blue);`
5. 注释更新为 v2

未修改其它文件，未改 chip 配色、未改选中态层次（边框 + 渐变背景 + 三层阴影 + name 变蓝全部保留）。

## 验证

执行：
1. ✅ `npm run dev` 已在 5173 端口运行
2. ✅ chrome-devtools MCP 重新加载页面
3. ✅ 截图浅色模式：`docs/agent/task/skill_card_v2_light.png`
4. ✅ 截图暗黑模式：`docs/agent/task/skill_card_v2_dark.png`

视觉确认（来自 MiniMax understand_image 分析）：
- ✅ 卡顶 4px 蓝色实色横条已删除（全部卡片统一处理）
- ✅ 卡顶只剩非常浅的蓝色渐变（6% accent），几乎看不出，只留"色温"暗示
- ✅ 选中态（aa 卡片）层次依然清晰：边框 + 渐变背景 + 左侧色条 + name 变蓝，四重信号表达选中
- ✅ 整体观感更克制、未选中卡片不再有"色块侵入"，滚动浏览时眼睛更舒服

## 后续可优化（非本次范围）

- Trae 红饱和度偏高，可换成更柔和的玫红/珊瑚色
- 卡片之间间距可加 1-2px，让卡片"分隔"作用更突出
- 如果未来希望卡片仍保留明显的"主色锚点"，可考虑在卡片左侧加 3-4px 竖向 accent 色条（比顶部横条更内敛，仍能识别 skill 类型）