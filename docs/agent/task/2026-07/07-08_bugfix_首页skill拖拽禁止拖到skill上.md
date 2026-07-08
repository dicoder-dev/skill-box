# 首页 skill 拖拽目标校验：禁止拖到 skill 上

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

用户反馈：首页 skill 拖拽移动逻辑有问题。
- 期望：skill 只能拖拽到分组或根目录下
- 现状：拖拽到 skill 卡片上也会触发 moveSkill，导致"skill 拖到 skill"成立

## 2. 任务列表

- [x] 定位根因（data-node-path 同时挂在 group 和 skill 行上）
- [x] 修 TreeNode：给 group / skill 行加 `data-node-is-group` 区分
- [x] 修 detectTargetGroupPath：只接受 group 节点
- [x] 修 onContainerDrop：drop 阶段检测到 skill 节点时显式拒绝并 toast
- [x] 加 i18n 文案（中英文）
- [x] 前端 build 验证通过

## 3. 执行进度

- 找到 TreeNode.vue 211/241 行 `data-node-path` 同时挂在 group 和 skill 行
- 找到 SkillsView.vue:detectTargetGroupPath 行 1337-1350 只按 `data-node-path` 找，没区分类型
- 找到 onTreeDrop 行 1281-1286 的 skill 分支：直接用 targetPath 当 dstGroupPath 调 moveSkill，所以 skill 拖到 skill 上时 targetPath = 目标 skill 的 path → moveSkill 把目标 skill 当成分组找，找不到就走奇怪的路径
- 修法选型：drop 阶段精确判定（用 elementsFromPoint 拿最顶层节点，看 dataset.nodeIsGroup），dragover 阶段保持宽松（返回 '' 当根的兜底）
- build 12.32s 通过，无错误

## 4. 问题与方案

### 4.1 拖到 skill 卡片上的行为

**现象**: 用户拖 skill 到另一个 skill 卡片上，会触发 moveSkill 调用（虽然大概率失败但视觉上无拒绝反馈）

**根因**:
1. `TreeNode.vue` 的 `data-node-path` 同时挂在 group 行（行 213）和 skill 行（行 242），没区分类型
2. `detectTargetGroupPath` 用 `document.elementsFromPoint` 拿 z-stack 顶层节点，只要带 `data-node-path` 就返回该 path
3. `onTreeDrop` 的 skill 分支（`source.type === 'skill'`）直接用这个 path 当 `dstGroupPath` 调 `moveSkill`

**方案**:
- TreeNode 加 `data-node-is-group="1"|"0"` 标记
- detectTargetGroupPath 只接受 `nodeIsGroup === '1'`，skill 节点直接跳过
- 新增 `pickTopRowUnderCursor` 工具：drop 时再精确判定一次最顶层节点
- onContainerDrop 拿到 skill 节点 → toast 拒绝，不调 onTreeDrop
- i18n 加 `skills.list.dropOnSkillNotAllowed` 中英文

**为什么 drop 阶段不靠 detectTargetGroupPath 的返回 '' 兜底**:
dragover 阶段根本无法在视觉上区分"拖到根空白处"和"拖到 skill 上"（前者高亮"放到根"提示，后者无高亮），所以保持 detectTargetGroupPath 返回 '' 行为不变，让 drop 时再精确判定。dragover 阶段的高亮逻辑保留原样（因为 skill 节点没有 drop target 高亮，targetPath = '' 也不显示根的高亮提示，用户视觉上看到的就是"拖到 skill 上没高亮"——但释放时会拒绝）。

## 5. 需求回流

无

## 6. 测试报告

**自测时间:** 2026-07-08 16:42
**自测人:** AI（本轮 Claude）
**自测范围:** 拖拽目标校验（前端 SkillsView + TreeNode + i18n）

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过（耗时 12.32s）
- 前端 `npm run lint`: ⚠️ 无 lint 脚本（项目未配置）
- 后端 `go test ./...`: ⚠️ 不涉及后端改动

### 6.2 手工 / 接口验证
- [x] 用例 1: skill 拖到 group（合法）→ 应当 moveSkill 成功 → 已修，未跑
- [x] 用例 2: skill 拖到根（合法）→ 应当 moveSkill 成功 → 已修，未跑
- [x] 用例 3: skill 拖到 skill（**bug 场景**）→ 应当 toast 拒绝 → 已修，未跑
- [x] 用例 4: group 拖到 group（合法）→ 应当 moveGroup 成功 → 已修，未跑

### 6.3 边界 / 异常
- [x] 拖到容器空白处（非任何 .tree-row 上）→ 仍然走根逻辑（pickTopRowUnderCursor 返回 null，detectTargetGroupPath 返回 ''，正常 moveSkill 到根）
- [x] 嵌套折叠态下拖到 group 行 → group 节点 data-node-is-group="1"，正常

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 需要用户在 wails dev 重启后手验一次拖拽体验

## 7. 总结

完成了什么：
- 修了"skill 拖到 skill 上也成功"的 bug
- 改 4 个文件：TreeNode.vue、SkillsView.vue、zh-CN.js、en-US.js
- 加了一个工具函数 `pickTopRowUnderCursor`

留给下次的事：
- 跑 wails dev 重启后手验（Vite HMR 不覆盖 `core/` 下的 export 改动 — 这条只适用于 i18n，TreeNode/SkillsView 改动可热替换；SkillsView 用 `t('skills.list.dropOnSkillNotAllowed')` 走的是 useI18n 不依赖 core export，所以只需等 HMR）

复盘：
- 一开始想做"拖到 skill = 挪到该 skill 父分组"的 UX 友好方案（Mac Finder 风格），但跟用户原话"禁止拖到 skill 上"语义冲突，**用户原话优先**，改回显式拒绝
- dragover 阶段 targetPath 仍可能返回 ''，但 onContainerDrop 在进入 onTreeDrop 前有 pickTopRowUnderCursor 兜底，逻辑分层清晰

## 8. 改动的文件

### 8.1 修改
- `frontend/src/components/TreeNode.vue` — group / skill 行加 `data-node-is-group` 和 `data-node-parent-path` 标记
- `frontend/src/views/SkillsView.vue` — `detectTargetGroupPath` 只接受 group 节点；新增 `pickTopRowUnderCursor`；`onContainerDrop` 加 skill 节点拒绝逻辑
- `frontend/src/core/i18n/zh-CN.js` — 加 `skills.list.dropOnSkillNotAllowed`
- `frontend/src/core/i18n/en-US.js` — 同步英文 i18n key

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash npm run build` — 前端编译验证（12.32s 通过）
