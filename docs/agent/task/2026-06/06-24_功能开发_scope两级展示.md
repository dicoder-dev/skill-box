# scope 两级:工具 × 作用域

**日期:** 2026-06-24
**状态:** 已完成

## 1. 需求

之前 SkillsView 的 scope chip 是"全局 + 1...N 项目"单级多选,点击 toggle,意图是"把这个 skill 标记为作用于这些 scope"。但实际生效位置是由用户本地文件系统决定的(用户手动把 skill 拷到 `~/.codex/skills/xxx` / `<project>/.claude/skills/xxx` 等等),这个 toggle 既不写库,也不动文件,纯属误导。

用户希望改成"两级只读"展示,直接看文件系统的真实情况:

- **第一级 = 编程工具**(5 个):codex / claude / opencode / cursor / trae
- **第二级 = 作用域**:全局 + 各项目(取自 listProjects)
- 选中某项的展示规则:**该 (tool, scope, project) 组合下,磁盘上真实存在 SKILL.md** → 高亮

## 2. 任务列表

- [x] 后端:`GET /api/skillbox/skills/scope-status?name=&version=` 实时扫所有 adapter 路径
- [x] 前端 API client:`getSkillScopeStatus`
- [x] 前端 SkillsView 改造 scope chip 区域为两级布局
- [x] 删旧 toggleScope / isScopeActive / projectLabel / activeScopes
- [x] i18n 加 `scopeToolsRow` / `scopeTargetsRow` / `scopeEmpty` / `scopeHitCount`
- [x] 移动端响应式:scope-row 改竖排
- [x] `go build ./...` 通过
- [x] `npm run build` 通过
- [x] commit + push

## 3. 执行进度

- 00:30 读 skilladapter/types.go / registry.go / base.go / 5 个 adapter 源文件
- 00:40 读 onboarding status 端点,了解 adapter 信息暴露模式
- 00:50 写后端 `scope_status.a.go`:遍历 `skilladapter.All()`,global 直接 `DiscoverPaths(ScopeGlobal)`,project 拿 `listProjects` 后用 `root_path + DiscoverPaths(ScopeProject)` 拼绝对路径
- 01:00 写 `skillDirExists`:`<resolved>/SKILL.md` 存在即命中
- 01:10 go build 通过
- 01:20 前端 API client + i18n + 删旧逻辑
- 01:35 SkillsView 改造 scope chip 区域为"工具行 + 作用域行"两级
- 01:45 加 chip-tool / chip-scope-target / chip-muted / chip-mini-list 样式
- 01:50 npm run build 通过

## 4. 问题与方案

**问题:旧 scope chip 是"勾选式",但实际不影响文件,纯属装饰。**

方案:不存数据库,每次请求实时扫磁盘。理由:
- 用户在外部(文件管理器 / cp 命令)把 skill 拷到 `~/.codex/skills/xxx` 时,如果不重新 import,旧数据无法感知
- 数据库存"该 skill 作用于 X"会跟实际状态脱节,容易误导
- adapter 已经把每个 (tool, scope) 的候选目录路径定义好了,直接 `os.Stat(<path>/<name>/SKILL.md)` 就行

**实现细节:**

1. **Global scope 候选路径**:adapter 自己给的是绝对路径(`~/.codex/skills` 等),直接用。
2. **Project scope 候选路径**:adapter 给的是**相对路径**(`.codex/skills` 等),要跟每个 project 的 `root_path` 拼起来。一个 project × 一个 rel → 一条候选。
3. **命中判定**:`<resolved>/SKILL.md` 存在(对齐 `BaseAdapter.readSkillDir` 入口,不做深度递归 — scope-status 关心"我有没有放在工具期望的位置",不是"我有没有放在任意子目录里",避免误报)。
4. **System 区分**:用 `IsSystemPath` 区分 user / system,前端 chip 暂时不区分(等 phase2 再加"只读参考"标签)。
5. **顺序稳定性**:`skilladapter.All()` 内部按 ToolID 排序,前端 `scopeTargets` 排序规则:global 永远第一,其余按 project_id 升序。

**响应结构(节选):**
```json
{
  "name": "review-code",
  "version": "1.0.0",
  "tools": [
    { "tool_id": "claude", "display_name": "Claude Code", "icon": "" },
    ...
  ],
  "projects": [
    { "id": 1, "name": "skill-box", "alias": "skill-box", "root_path": "..." }
  ],
  "hits": [
    { "tool_id": "claude", "scope": "global", "project_id": 0, "path": "/Users/.../.claude/skills", "resolved": "/Users/.../.claude/skills/review-code", "exists": true, "is_system": false },
    { "tool_id": "claude", "scope": "project", "project_id": 1, "project_label": "skill-box", "path": "<root>/.claude/skills", "resolved": "<root>/.claude/skills/review-code", "exists": false, "is_system": false },
    ...
  ]
}
```

**前端两级布局:**

- 第一行(`scopeToolsRow` = "工具"):5 个 chip,代码化的短名(Codex / Claude / OpenCode / Cursor / Trae),右侧徽章 = 该工具下命中数;命中用主色背景,未命中用 muted 灰
- 第二行(`scopeTargetsRow` = "生效位置"):全局 + 各项目 chip,命中用蓝色背景,右侧小角标列出哪些工具里命中了(用 mdi icon 紧凑排列);未命中 muted 灰
- `title` 属性附完整命中信息(工具名 + 绝对路径),hover 可看

## 5. 需求回流

(暂无)

## 6. 测试报告

**自测时间:** 2026-06-24
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 后端 `go build ./...` 结果: ✅ 通过
- 前端 `npm run build` 结果: ✅ 通过(286.04 kB JS / 81.08 kB CSS,gzip 后 96.96 kB / 12.57 kB)

### 6.2 手工验证(代码 review)
- [x] 后端 controller 路径在 `cskill` 包下,沿用 `ginp.RouterAppend` 模式
- [x] `skillDirExists` 只判 `<resolved>/SKILL.md`,不递归(避免误报)
- [x] 入参 `name` 必填校验,缺失返回 400
- [x] project 路径拼接前判 `root_path == ""` 跳过(避免在空 root 上拼相对路径)
- [x] 异常 adapter 不会让整个接口挂(每个 adapter 单独 log warn 后继续)
- [x] 前端 `loadScopeStatus` 在 `loadCurrent` 末尾触发,切换 skill 自动刷新
- [x] 旧 `toggleScope / isScopeActive / projectLabel / activeScopes` 已清干净
- [x] i18n key 完整覆盖中英
- [x] 移动端 `@media (max-width: 720px)` scope-row 改竖排

### 6.3 边界 / 异常
- [x] 入参 name 缺失:返回 400
- [x] listProjects 失败:log warn 后 projects=[],继续扫 global
- [x] 单个 adapter 路径不可访问:`os.Stat` 失败,`exists=false`,不影响其他
- [x] project 数量为 0:scope 行只显示"全局"chip(用 muted 表示无项目)
- [x] skill 在所有位置都不存在:hit 数组存在但 `exists` 全 false,前端仍展示 muted chips,行尾有"该技能尚未写入任何工具/位置"提示

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题:dev server click-through 验收待跑;前端 `TOOL_ICON_MAP` 暂用静态映射,后续可从后端 icon 字段统一(目前后端 icon 为空,前端没真依赖)

## 7. 总结

- 完成了什么: scope chip 改成两级(工具 × 作用域)只读展示,实时反映磁盘真实情况
- 留下了什么:
  - `api-server/internal/gapi/controller/skillbox/cskill/scope_status.a.go` — 新接口
  - `frontend/src/api/skillbox/skills.js` — 加 `getSkillScopeStatus`
  - `frontend/src/views/SkillsView.vue` — 删旧 scope 多选,加两级布局 + 配套样式
  - `frontend/src/core/i18n/{zh-CN,en-US}.js` — 4 个新 key
- 留给下次的事:
  - dev server 跑一次 click-through
  - phase2 可加 system 档位 chip 视觉区分(目前 chip 不区分 user/system,后端 `is_system` 字段已就绪)
  - 如果后端需要给前端透图标(adapter.Icon 字段当前为空),前端 TOOL_ICON_MAP 可改成后端驱动
- 复盘: "勾选式 scope" 是个常见反模式 — 看起来能操作但实际上不写库,会让用户怀疑系统是否生效。改成"读取式"展示,既消除了误导,又零成本对齐实际状态,符合"系统状态 = 文件系统"的真实语义。
