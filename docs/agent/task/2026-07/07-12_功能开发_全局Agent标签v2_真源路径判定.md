# 2026-07-12 功能开发 — 全局 Agent 标签 v2(基于 ~/.agents/skills/ 真源路径)

## 需求来源

v3 提交(commit af5f77c)用 `node.path === 'agents' || path.startsWith('agents/')` 判定全局 Agent —— **这是错的**。用户在 v3 上线后反馈:真实"全局 Agent"指的是物理路径在 `~/.agents/skills/<name>/` 下的 skill(用户家目录的全局 skills 池,所有工具可自动读取),而不是用户在 skillbox UI 里手建一个叫 `agents` 的分组。

例如 "聚能" 在 `~/.agents/skills/canvas-design` 下,同时也是 skillbox 里的一个 skill,这时它就需要显示"全局 Agent"标签。

## v3 错在哪

| 场景 | 真实情况 | v3 判定 | v3 结果 |
|------|---------|---------|---------|
| `~/.agents/skills/canvas-design/` 导入到 `~/.skill-box/skills/canvas-design/` | 是全局 Agent | path = "canvas-design" | ❌ 不显示(漏判) |
| 用户在 UI 拖建 `agents/` 分组,把 skill 放进去 | 不是全局 Agent | path = "agents/foo" | ❌ 误显示(误判) |

## v4 修复方案

### 后端改动(`api-server/internal/skillstore/store.go`)

1. **SkillTreeMeta 加 `SourcePath` 字段**(行 894-908):从 sidecar 读 source_path 注入到 tree 节点
2. **新增 sidecar 读写 helper**(行 1247-1296):
   - `sourceMetaFile = ".skillbox-source.json"` —— 以 `.` 开头,ListTree 扫描会自动跳过
   - `sourceMeta` struct:承载 `SourcePath string`(`json:"source_path"`)
   - `WriteSourcePath(absDir, sourcePath)` —— exported 供 caller 写入
   - `readSourcePath(absDir)` —— buildTreeNode 读取用
3. **buildTreeNode 读 sidecar**(行 967-978):skill 叶子节点构造时调 `readSourcePath(absDir)` 注入到 SkillTreeMeta.SourcePath
4. **import 加 encoding/json import**(行 21)

### 后端改动(`api-server/internal/skillpkg/local_import.go`)

- **`importOneFromDir` 写 sidecar**(行 320-360):在 `store.Save` 成功后,把 `dir` 走 `filepath.EvalSymlinks` 解析成真实路径(macOS 真实路径在 `/private/var/...` 下,需要归一化),然后调 `skillstore.WriteSourcePath` 写到落盘目录
- 失败不阻断导入流程(sidecar 缺失只少一个标签,不影响 skill 本身可用)

### 前端改动(`frontend/src/components/TreeNode.vue`)

- **`isGlobalAgent` 改用 source_path 正则**(行 195-211):`/[\\/]\.agents[\\/]skills[\\/]/.test(src)`
  - 跨平台:macOS/Linux 用 `/`,Windows 用 `\`
  - 不能用 `startsWith('~/.agents/...')` —— 后端给的是 EvalSymlinks 后的绝对路径,不是 shell 缩写形式
- **`visibleTools` 注释更新**(行 213-219):解释为什么全局 Agent 卡片不显示 chip

## 正则单元测试结果(6/6 通过)

| 输入路径 | 期望 | 实际 |
|---------|------|------|
| `/Users/brody/.agents/skills/canvas-design` | true | ✅ true |
| `/Users/brody/.skill-box/skills/commit-msg` | false | ✅ false |
| `/Users/brody/.skill-box/skills/agents/demo-global-agent` | false | ✅ false(修复 v3 误判) |
| ``(空) | false | ✅ false |
| `C:\Users\brody\.agents\skills\foo` | true | ✅ true(Windows 路径) |
| `/home/user/.agents/skills/flutter-animating-apps` | true | ✅ true(Linux 路径) |

## 视觉验证(`docs/agent/task/skill_card_v4_global_agent.png`)

4 张 mock 卡片浮窗验证(都在 1100x720 viewport 居中):

1. **canvas-design**(`~/.agents/skills/canvas-design`):name 左侧 ✅ 显示"全局 Agent"翠绿色标签,无 chip
2. **commit-msg**(`~/.skill-box/skills/commit-msg`):name 左侧 ✅ 无 tag,有 Claude + Codex chip
3. **demo-global-agent**(`~/.skill-box/skills/agents/demo-global-agent`):name 左侧 ✅ 无 tag(v3 误判场景,已修复),有 Codex chip
4. **aa**(source_path 空):name 左侧 ✅ 无 tag,卡片处于选中态

## 数据迁移注意

**已有 skill 没有 sidecar 文件**,所以 v4 上线后**历史导入的全局 Agent skill 不会显示"全局 Agent"标签**。修复方案:

- 方案 A(简单,推荐):加一个一次性迁移脚本,扫描每个 skill 目录,如果没有 `.skillbox-source.json` 就**根据当前 store 路径反推 + 用户历史导入记录补全**。但当前 store 没记"原始 source"信息,反推不出来 —— 不行
- 方案 B(实用,推荐):加一个**回扫功能**,ImportGlobalPaths 时(或专门的"扫描 ~/.agents/skills"按钮)比对磁盘上同名 skill,补 sidecar
- 方案 C(简单兜底):v4 上线后,用户重新走"导入技能 → 全局目录"流程,自动补全 sidecar。**已有 skill 维持原样**(只是少了 tag,功能不受影响)

本次实现 v4 后,先走方案 C 兜底;方案 B 留作后续优化。

## 关键修改点

| 文件 | 行号 | 改动 |
|------|------|------|
| `api-server/internal/skillstore/store.go` | 21 | 加 `encoding/json` import |
| `api-server/internal/skillstore/store.go` | 894-908 | `SkillTreeMeta` 加 `SourcePath` 字段 |
| `api-server/internal/skillstore/store.go` | 965-980 | `buildTreeNode` 读 sidecar 注入 SourcePath |
| `api-server/internal/skillstore/store.go` | 1247-1296 | 新增 `WriteSourcePath`/`readSourcePath`/`sourceMeta` + sidecar 常量 |
| `api-server/internal/skillpkg/local_import.go` | 320-360 | `importOneFromDir` 在 Save 后写 sidecar + 加 `skillpkgNormalizeGroup` 辅助 |
| `frontend/src/components/TreeNode.vue` | 195-219 | `isGlobalAgent` 改用 source_path 正则 + 注释更新 |

## 验证

- ✅ `go build ./api-server/...` 通过
- ✅ `npm run build:dev` 通过
- ✅ 6/6 正则单元测试通过(覆盖 macOS/Linux/Windows + v3 误判场景)
- ✅ 4/4 mock 视觉验证通过

## 提交

(待 git commit + push)
