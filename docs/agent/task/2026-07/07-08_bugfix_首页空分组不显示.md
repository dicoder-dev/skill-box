# 首页空分组不显示：分组 a 移空 skill 后仍要可见

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

用户反馈：首页 skill 树中，分组 a 里最后一个 skill 移走后，**分组 a 整个消失**。
- 期望：哪怕没有 skill 也要显示空分组
- 现状：移走最后一个 skill 后，磁盘上 `a/` 目录被自动清掉，ListTree 看不到 a

## 2. 任务列表

- [x] 复现 bug：写测试 TestRepro_EmptyGroupAfterMove
- [x] 定位根因：MoveGroupPath 末尾 `removeIfEmpty(srcParent)` 把空 group 目录删了
- [x] 修 store.go:MoveGroupPath — 移除自动清理空 group 的逻辑
- [x] 加正式回归测试 TestEmptyGroupVisibleAfterMove
- [x] 后端 skillstore 全部测试通过
- [x] 前端 build 验证通过

## 3. 执行进度

- 17:05 收到用户反馈"分组 a 移空后消失"
- 17:08 查 buildTreeNode 行 910-915 逻辑：看起来空 group 在 kw=='' 时应保留
- 17:10 写复现测试，发现磁盘上 `a/` 目录**根本不存在** → 不是 ListTree 过滤问题，是被自动删了
- 17:12 查 MoveGroupPath 行 450-452 `_ = removeIfEmpty(srcParent)` → 移走 skill 后 `a/` 变空被删
- 17:13 修 store.go：注释说明为什么删这行（保留空 group 是用户意图，删分组走 DeleteGroupDir 显式）
- 17:15 新加 `TestEmptyGroupVisibleAfterMove` 覆盖 4 个场景：磁盘保留 / ListTree 返回 / IsGroup 标记 / 移回 a 复活
- 17:17 skillstore 全部测试通过；前端 build 11.62s 通过

## 4. 问题与方案

### 4.1 根因 — MoveGroupPath 自动清理空 group 目录

**现象**: 把分组 a 里最后一个 skill 挪到根下后，a 从首页消失（磁盘上 `a/` 也消失）。

**根因**:
- `api-server/internal/skillstore/store.go:MoveGroupPath` 行 450-452 在 os.Rename/copy 完成后调 `removeIfEmpty(srcParent)`
- `srcParent` = `filepath.Dir(srcDir)`，即源 skill 的父目录
- 移走 skill 后父目录变空 → removeIfEmpty 把它删了
- 下次 ListTree 读磁盘，根本没有 `a/` 目录可列

**方案**:
- 移除 MoveGroupPath 末尾的 `removeIfEmpty(srcParent)` 调用
- 改为注释说明：保留空 group 目录是用户意图（"分类"占位），删分组必须走 `DeleteGroupDir` 显式操作
- 不动 `MoveGroupDir`（行 526）的 `removeIfEmpty(filepath.Dir(srcAbs))` — 那行是挪走整个 group 时清理**它的父级**，语义不同（用户挪走 react 时本意就是让父层少东西）

### 4.2 为什么是后端 bug 而不是前端

- 前端 TreeNode 模板行 190 用 `v-for="node in nodes"` 渲染 group 行，group 行**只要 tree 数据里有**就一定渲染
- 前端 `displayNodes` 是死代码（`flattenSingleLevel` 函数体根本不存在，但模板没用它），跟空 group 无关
- ListTree 行 910-915 也有"空 group kw=='' 时保留"的兜底逻辑
- 唯一缺口在 MoveGroupPath 末尾的清理调用 — 后端从源头就抹掉了

## 5. 需求回流

无

## 6. 测试报告

**自测时间:** 2026-07-08 17:15
**自测人:** AI（本轮 Claude）
**自测范围:** 后端 skillstore.MoveGroupPath + ListTree + 磁盘状态

### 6.1 自动化测试
- `go test ./internal/skillstore/ -v` 结果: ✅ 全部通过（20+ 用例，含新加 TestEmptyGroupVisibleAfterMove）
- 前端 `npm run build` 结果: ✅ 通过（11.62s）

### 6.2 手工 / 接口验证
- [x] 用例 1: a 下 1 个 skill x 移出到根 → a 仍存在 + ListTree 返回 IsGroup=true / Children=[] / Path=a → ✅
- [x] 用例 2: x 移回 a → a 复活，children = [x] → ✅
- [x] 用例 3: 磁盘上 a/ 目录保留（不依赖 ListTree 二次过滤） → ✅

### 6.3 边界 / 异常
- [x] 嵌套空 group (frontend/react/x 移出 → react/frontend 变空两层) → 两层都保留
- [x] MoveGroupDir（挪整个 group）行为不变 → 全部旧测试通过

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 需要 wails dev 重启让后端代码生效（Go 端不会 HMR）

## 7. 总结

完成了什么：
- 修了"分组 a 移空 skill 后整组消失"的 bug
- 改 1 个后端文件 + 1 个新测试文件
- 加 TestEmptyGroupVisibleAfterMove 回归测试，4 个断言

留给下次的事：
- 用户在 wails dev 重启后手验：拖空分组 a 是否仍可见

复盘：
- 一开始以为是 ListTree 过滤问题（行 910-915 那段），浪费了 5 分钟
- **快速写复现测试** 立刻定位到磁盘上 `a/` 根本不存在，把问题从"前端 / 后端过滤"缩小到"后端 MoveGroupPath 清理逻辑"
- 应该第一时间写复现测试，不要靠看代码脑补

## 8. 改动的文件

### 8.1 修改
- `api-server/internal/skillstore/store.go` — MoveGroupPath 移除 `removeIfEmpty(srcParent)` 调用

### 8.2 新增
- `api-server/internal/skillstore/empty_group_test.go` — 回归测试 TestEmptyGroupVisibleAfterMove

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash go test ./internal/skillstore/ -v` — 跑 skillstore 全量测试（含新加回归）全通过
- `Bash go test ./internal/...` — 跑 internal 全量（skillstore 全部 cached 通过；其他无关包失败是环境问题）
- `Bash npm run build` — 前端编译验证（11.62s 通过）
