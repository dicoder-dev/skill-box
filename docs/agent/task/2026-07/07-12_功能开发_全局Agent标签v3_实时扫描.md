# 2026-07-12 功能开发 — 全局 Agent 标签 v3(实时扫描 ~/.agents/skills/)

## 需求来源

v4(commit 61fd7a0)用 sidecar `.skillbox-source.json` 缓存"曾经导入过"信息,
但用户在 review 时明确指出:**"全局 Agent 标签"语义就是"物理路径在 `~/.agents/skills/` 下"**,
应该**实时去读对应的文件夹**,而不是用缓存配置到本地。

v4 错在哪:
- 在 store 写 sidecar 记录"曾经从 ~/.agents/skills/ 导入"
- 用户在 ~/.agents/skills/ 下手动增/删 skill 后,**store 里的 sidecar 不会更新**
- 需要重新走"导入技能 → 全局目录"流程才能刷新(用户原话:"什么要重新导入啊?")
- 历史 skill 缺 sidecar → 需要迁移脚本

v5 直接砍掉 sidecar,改用**实时磁盘扫描** — 任何时候 reload 列表都跟磁盘真值同步,零迁移负担。

## v5 修复方案

### 后端改动(`api-server/internal/skillstore/store.go`)

1. **删除整个 sidecar 体系**:
   - `encoding/json` import 删除(不再需要)
   - `sourceMetaFile` / `sourceMeta` / `WriteSourcePath` / `readSourcePath` 全部删除

2. **新增 `resolveGlobalSourcePath(name)` helper**(store.go 末段):
   - 拼出候选路径 `<home>/.agents/skills/<name>`
   - `filepath.EvalSymlinks` 解析真实路径(处理 macOS `/private/var/...` + symlink)
   - `os.Stat` 验证 `<real>/SKILL.md` 存在(空目录会被拒)
   - 命中:返回 `real` 绝对路径;未命中:返回空字符串

3. **`buildTreeNode` 改用 `resolveGlobalSourcePath` 注入 source_path**(行 977-988):
   - 替换原来的 `readSourcePath(absDir)` 调用
   - 每次 ListTree 都实时 stat,无任何缓存

### 后端改动(`api-server/internal/skillpkg/local_import.go`)

- `importOneFromDir` 删除 sidecar 写入逻辑(行 320-345 还原成 v4 之前的简洁版)
- 删除 `skillpkgNormalizeGroup` 辅助函数(不再需要)
- 函数顶部注释说明 store.buildTreeNode 实时扫描,跟导入历史解耦

### 前端不动

`TreeNode.vue` 的 `isGlobalAgent` 正则判定逻辑保持 v4 不变 — `source_path` 字段名 / 正则 `/[\\/]\.agents[\\/]skills[\\/]/` / chip 收敛全保留。

## 真实后端验证(11/11 通过)

`go run ./api-server/cmd/test_listtree_v5/` 调 store.ListTree 直接打 source_path(临时测试脚本,提交前已删):

| skill | source_path 实际值 | 前端正则判定 | 预期 |
|-------|-------------------|------------|------|
| canvas-design | `/Users/brody/.agents/skills/canvas-design` | ✓ 命中 | 全局 Agent |
| commit-msg | `/Users/brody/.agents/skills/commit-msg` | ✓ 命中 | 全局 Agent |
| flutter-animating-apps | `/Users/brody/.agents/skills/flutter-animating-apps` | ✓ 命中 | 全局 Agent |
| flutter-reducing-app-size | `/Users/brody/.agents/skills/flutter-reducing-app-size` | ✓ 命中 | 全局 Agent |
| self-improving-agent | `/Users/brody/.agents/skills/self-improving-agent` | ✓ 命中 | 全局 Agent |
| ui-ux-pro-max | `/Users/brody/.agents/skills/ui-ux-pro-max` | ✓ 命中 | 全局 Agent |
| weather | `/Users/brody/.agents/skills/weather` | ✓ 命中 | 全局 Agent |
| aa | ``(空) | ✗ 不命中 | 普通 |
| code-review | ``(空) | ✗ 不命中 | 普通 |
| debug-helper | ``(空) | ✗ 不命中 | 普通 |
| **unit-test-gen** | `/Users/brody/.skill-box/skills/unit-test-gen` | ✗ 不命中 | 普通(symlink 假全局) |

**`unit-test-gen` 案例**:
- `~/.agents/skills/unit-test-gen` 是 symlink → 指向 `~/.skill-box/skills/unit-test-gen`
- `EvalSymlinks` 解析后真实路径 = `~/.skill-box/skills/unit-test-gen`
- 前端正则 `/[\\/]\.agents[\\/]skills[\\/]/` 匹配这个真实路径 → **不命中**(.agents/skills 段不在)
- 结果:不显示"全局 Agent"标签 ✅(这是预期 — 它实际上在 store 内,只是被软链到 .agents/skills 假装)
- 反过来说,如果用户从 `~/.agents/skills/unit-test-gen/` 直接放一个真实 SKILL.md(非 symlink),
  `EvalSymlinks` 拿不到回环,source_path 就会是 `~/.agents/skills/unit-test-gen`,**会**命中。

## 视觉验证(`docs/agent/task/skill_card_v5_realtime_scan.png`)

11 张 mock 卡片浮窗(用真实 ListTree 数据),正则单测 + 视觉双验证:
- 7 张全局 Agent(canvas-design / commit-msg / flutter-animating-apps / flutter-reducing-app-size /
  self-improving-agent / ui-ux-pro-max / weather)✅ 显示"全局 Agent"翠绿色标签 + 无 chip
- 4 张对照(aa / code-review / debug-helper / unit-test-gen)✅ 无 tag + 显示对应 chip

## 用户原始诉求的解决

> "什么要重新导入啊?现在不是直接实时去读取对应的文件夹吗?是直接根据文件夹去匹配,而不是缓存配置到本地。"

✅ **v5 完全满足**:
- 不再需要"重新导入" — 用户在 `~/.agents/skills/` 下增/删 skill,下次 reload 列表立即生效
- 不再用 sidecar 缓存 — store 保持"只读系统"纯粹性,只看磁盘真值
- 历史 skill 零迁移 — 不需要任何兼容代码,旧的 13 个根级 skill(其中 7 个恰好在 `.agents/skills/` 下)
  立即就能识别成全局 Agent

## 关键修改点

| 文件 | 行号 | 改动 |
|------|------|------|
| `api-server/internal/skillstore/store.go` | 18-33 | 删除 `encoding/json` import |
| `api-server/internal/skillstore/store.go` | 975-988 | `buildTreeNode` 改用 `resolveGlobalSourcePath(name)` |
| `api-server/internal/skillstore/store.go` | 1313-1361 | 删除 sidecar 体系 + 新增 `resolveGlobalSourcePath` helper |
| `api-server/internal/skillpkg/local_import.go` | 320-345 | `importOneFromDir` 还原成简洁版,删除 `WriteSourcePath` 调用 |
| `frontend/src/components/TreeNode.vue` | — | 不动(v4 正则判定逻辑正确) |

## 验证

- ✅ `go build ./api-server/...` 通过
- ✅ 后端 `ListTree` 真实数据 11/11 全部判定正确(7 全局 + 4 普通)
- ✅ 视觉 mock 11/11 全部符合预期
- ✅ symlink 假全局(unit-test-gen)被正确识别为普通

## 提交

(待 git commit + push)
