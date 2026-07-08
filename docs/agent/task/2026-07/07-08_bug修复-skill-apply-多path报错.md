# Adapter 加 UserPath(scope) — 修复首页 skill apply 全局作用域失败

**日期:** 2026-07-08
**状态:** 已完成

## 1. 需求

用户在首页对某个 skill 做"作用到全局作用域 → 工具=claude"操作时,后端日志里出现

```
[INFO] [2026-07-08 16:52:04] audit {"action":"apply_failed","fields":
  {"error":"skillapp: tool claude has 2 paths for scope global (max 1 per (scope, category))",
   "scope":"global","tool":"claude"},"name":"code-review","target_type":"skill"}
```

POST /api/skillbox/skills/apply 返回 200 但 apply 全失败,前端看不到成功 chip。

**澄清后目标:**
- 首页 apply 必须能用,且写盘路径落在 user category(~/.agents/skills/<name>)
- 不动 DiscoverPaths 接口契约(scope-status / importer / onboarding / cproject 都依赖多 path)
- 走方案:Adapter interface 新加 UserPath(scope) 方法

## 2. 任务列表

- [x] 排查 skillapp/applier.go 报错根因
- [x] Adapter interface 加 UserPath(scope) 接口方法
- [x] BaseAdapter 默认实现 UserPath(Tools[scope] 单值校验)
- [x] applier.resolveTargetDir 改走 UserPath
- [x] 补 stub adapter 同步(UserPath 一行)
- [x] 新增 TestApplyOne_UsesUserPath_NotDiscoverPaths 回归
- [x] go build && go test 通过
- [x] 写 memory 沉淀
- [x] git commit && git push

## 3. 执行进度

- 16:54 完成根因定位:`e_tool_path` 的 uniqueIndex 是 (scope, category) 而非 scope;claude/codex 同 scope 有 user + system 两条 path,BaseAdapter.DiscoverPaths(scope) 合并后返 2 条 → applier 报错
- 16:55 EnterPlanMode 拟定 UserPath 方案
- 17:05 实施方案 5 步改动,完成实现 + 测试 + memory
- 17:08 git commit + push

## 4. 问题与方案

### 4.1 报错根因

`e_tool_path` 加 `uniqueIndex(tool_id, scope, category)` 后,seed 里 claude 同时有
- `(global, user)` → `~/.agents/skills`
- `(global, system)` → `~/.claude/plugins/marketplaces`

`(scope, category)` 唯一,**scope 维度上仍是 2 条**。`BaseAdapter.DiscoverPaths(scope)` 有意合并 user + system 后返多 path(scope-status 依赖多 path chip 展示),所以 `applier.resolveTargetDir` 注释"单 path 模型,e_tool_path uniqueIndex 已兜底"误读了那个索引 —— 它不限制 scope 维度多 path。

**方案 A**:Adapter interface 加 `UserPath(scope)`,BaseAdapter 默认从 Tools[scope] 取单值,applier 走新方法。
**方案 B**:`DiscoverPaths` 只返 user。
**方案 C**:applier 用 IsSystemPath 过滤。

**最终选 A**(用户 AskUserQuestion 选 A),理由:
- B 会破坏 scope-status 多 chip,phase2 的只读参考功能废掉
- C 在 system 不在时仍可能拿到错的(没有 user 时取 system),且每次 apply 一次遍历

## 5. 需求回流

> (无)

## 6. 测试报告

**自测时间:** 2026-07-08 17:08
**自测人:** AI(本轮 Claude)
**自测范围:** skilladapter 接口 / BaseAdapter / skillapp Applier / 4 个 stubAdapter / skillapp_test 新测试

### 6.1 自动化测试

```
$ go build ./...
(无输出,通过)

$ go test ./internal/skillapp/... ./internal/skilladapter/... ./internal/skillimporter/... ./internal/gapi/service/skillapp/...
ok      ginp-api/internal/skillapp               0.017s
ok      ginp-api/internal/skilladapter           0.037s
ok      ginp-api/internal/skilladapter/toolspecs 0.014s
ok      ginp-api/internal/skillimporter         0.090s
ok      ginp-api/internal/gapi/service/skillapp/sskillapp 0.129s

$ go test ./internal/skillapp/... -run TestApplyOne_UsesUserPath_NotDiscoverPaths -v
=== RUN   TestApplyOne_UsesUserPath_NotDiscoverPaths
--- PASS: TestApplyOne_UsesUserPath_NotDiscoverPaths (0.00s)
PASS
ok      ginp-api/internal/skillapp   0.010s
```

### 6.2 手工 / 接口验证

未启动 web 服务做 curl(留作 production 验证)。自动化测试已覆盖关键路径:
- multiPathFakeAdapter:DiscoverPaths 返 [user, system] / UserPath 返 user → apply 必须写 userRoot
- 实际断言 `res.TargetPath == userRoot/code-review` 且 `systemRoot/code-review` 不存在

### 6.3 边界 / 异常

- `BaseAdapter.UserPath` 在 Tools[scope] 多条时返 `ErrMultipleUserPaths`(DB 脏数据兜底)
- `BaseAdapter.UserPath` 在 Tools[scope] 为空时返 `("", nil)`,applier 转成 4xx "has no user path",提示用户去 ToolsView 配置
- DiscoverPaths 契约不动,scope-status / importer / onboarding / cproject 仍多 path,不受影响

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 无(线上未跑集成 curl,但单元测试覆盖路径已锁死,scope-status / 现有 5 个 adapter 配置不变)

## 7. 总结

**完成了什么:**
- Adapter interface 加 UserPath(scope) 方法
- BaseAdapter 提供默认实现,Tools[scope] 单值校验
- applier.resolveTargetDir 改走 UserPath,头注释同步改成"user path 模型"
- 4 个 stubAdapter 补一行 UserPath
- 新增 TestApplyOne_UsesUserPath_NotDiscoverPaths 回归,锁死"applier 必须走 UserPath 不是 DiscoverPaths"
- memory 沉淀一条 adapter-userpath-vs-discoverpaths

**留下了什么:**
- 内存文件 `~/.claude/projects/.../memory/adapter-userpath-vs-discoverpaths.md`
- 5 个文件修改 + 1 个测试新增,全部走过单测

**留给下次的事:**
- production 跑一次首页 apply,确认 code-review → claude/global 真实写盘在 ~/.agents/skills/code-review
- (可选)ToolsView 的"无 user path"4xx 弹个中文 toast,目前是英文 "has no user path for scope"

**复盘:**
- 好在 Phase 1 的 Explore agent 把 DiscoverPaths 的 6 处 caller 全部查到,所以方案讨论时无需再到代码里翻
- 好在 applier_test.go 已存在,新建的 fakeAdapter 复用了既有 Apply/ApplyLink 实现,没复制大段代码
- 教训:`e_tool_path` uniqueIndex 是 (scope, category) 不是 scope,**注释写"已兜底"前要核对索引列定义**;这种"看起来对了其实不对"的注释很容易骗到后人

## 8. 改动的文件

### 8.1 新增
- (无)

### 8.2 修改
- `api-server/internal/skilladapter/types.go` — Adapter interface 加 UserPath(scope) 接口方法 + ErrMultipleUserPaths 哨兵 + import errors
- `api-server/internal/skilladapter/base.go` — BaseAdapter 实现 UserPath(Tools[scope] 单值校验,>1 条返 ErrMultipleUserPaths)
- `api-server/internal/skillapp/applier.go` — resolveTargetDir 改走 ad.UserPath(scope),头注释改成"user path 模型"
- `api-server/internal/skilladapter/registry_test.go` — stubAdapter 加 UserPath stub
- `api-server/internal/skillimporter/importer_test.go` — fakeAdapter 加 UserPath stub
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s_test.go` — fakeAdapter 加 UserPath stub
- `api-server/internal/skillapp/applier_test.go` — fakeAdapter 加 UserPath stub + 新增 multiPathFakeAdapter 类型 + 新增 TestApplyOne_UsesUserPath_NotDiscoverPaths

### 8.3 删除
- (无)

## 9. 工具与用途

### 9.1 MCP 工具
- (无)

### 9.2 Skill
- (无)

### 9.3 CLI
- `Bash go build ./...` — 编译验证(通过,无输出)
- `Bash go test ./internal/skillapp/... ./internal/skilladapter/... ./internal/skillimporter/... ./internal/gapi/service/skillapp/...` — 5 个包单测全过
- `Bash go test ./internal/skillapp/... -run TestApplyOne_UsesUserPath_NotDiscoverPaths -v` — 单独验证新增回归

## 1.1 对话轮次 (16:54)

> 用户原话:"检查一下首页 skill 作用域作用 skill 时提示失败..."

- **本轮做了:** 排查 + 读 applier.go / e_tool_path / builtin.go / cskill/scope_status.a.go,定位根因(DiscoverPaths 返 2 条 vs applier 期望 1 条)
- **本轮决定:** EnterPlanMode 走 UserPath 路线
- **本轮待办:** 实施 5 步改动
- **本轮工具:** `Bash grep -rn "DiscoverPaths"` / `Read 关键 4 个文件`
- **状态更新:** 任务状态 已完成 → 进行中(进入实施)

## 1.2 对话轮次 (17:05)

> 用户原话:Plan 已批准,继续

- **本轮做了:** 改 5 个文件 + 新增 1 个测试;go build + 5 个包 go test 全过;写 memory + MEMORY.md 更新
- **本轮决定:** memory 命名 adapter-userpath-vs-discoverpaths,便于 grep "adapter" 找到
- **本轮待办:** git commit + push
- **本轮工具:** `Bash go build` / `Bash go test`(两个) / `Edit` × 7 + `Write` × 1(memory)
- **状态更新:** 进行中 → 已完成(待 push)
