# 根下分组挪到根(dst="")被误报 "target already exists"

**日期:** 2026-06-29
**状态:** 已完成

## 1. 需求

用户报告:首页实现可以将 skill 从分组内移动到根目录,但**拖根下的分组"挪到根"**(srcGroupPath="aa", dstGroupPath="")报错:

```
[WARN] [2026-06-29 22:56:38] skill move group: skillstore: target group "/Users/brody/.skill-box/skills/aa" already exists
[127.0.0.1] POST /api/skillbox/skills/group/move |  409 |  554µs | ...
```

预期:因为 `aa` 分组本身已经在根下,这次操作实际是 no-op,应直接返 OK 才对。

## 2. 任务列表

- [x] 定位后端 MoveGroupDir 的 dstAbs 计算逻辑
- [x] 修复:加 no-op 短路(dstAbs == srcAbs 直接返 nil)
- [x] 加回归测试 TestMoveGroupDir_NoOp_ToRoot 覆盖"挪到根"+"挪到父级下"两种 case
- [x] go test ./internal/skillstore/... 通过
- [x] go build ./... 通过
- [x] git commit + push

## 3. 执行进度

- 22:58 复现用户报错,查 `~/.skill-box/logs/2026-06/06-29-error_request.txt` 拿到连续 4 次
  `{"src_group_path":"aa","dst_group_path":""}` 都报 target group .../aa already exists
- 22:59 定位 `api-server/internal/skillstore/store.go:348-393` 的 MoveGroupDir:
  `dstAbs := filepath.Join(s.root, filepath.FromSlash(dstGroupPath), srcBase)`
  当 `dstGroupPath=""` 时,`filepath.Join` 跳过空段 → `dstAbs = root/<srcBase> = srcAbs` → 撞"目标已存在"判断
- 23:01 加 no-op 短路 + 注释说明
- 23:02 加 TestMoveGroupDir_NoOp_ToRoot 回归测试
- 23:03 go test 通过(4 个 MoveGroupDir 测试全部 PASS)+ go build 通过
- 23:04 git commit `daffa17` + push origin/main 成功

## 4. 问题与方案

### 4.1 `dstGroupPath=""` 让 `filepath.Join` 跳过空段

**现象:** `MoveGroupDir("aa", "")` 报错 target group .../aa already exists。
**定位:** `store.go:364`
```go
dstAbs := filepath.Join(s.root, filepath.FromSlash(dstGroupPath), srcBase)
// 当 dstGroupPath="" 时,filepath.Join 跳过空段
// dstAbs = filepath.Join(s.root, srcBase) = srcAbs
```
注释里其实写了 "src=aa,dst=""     → 合法(挪到根,目标 = root/aa)",但实现没做 no-op 短路,直接撞到下面的"目标已存在"判断。
**方案:** 在 `isDescendantOrSame` 检查前先判断 `dstAbs == srcAbs`,直接返 nil。这同时覆盖了"挪到根"和"挪到自己父级下"两种同义操作(后者注释里也说要拒,实际上走 os.Rename 也是 noop,但前端会更早撞到"目标已存在"check,所以在 store 层先拦更稳妥)。
**教训:** 注释里写"合法"但实现没短路是个常见坑,加显式 short-circuit 防御性更好。

### 4.2 改前没看日志文件,差点瞎猜

**现象:** 用户只贴了一段 GIN 日志 WARN 行。
**定位:** 按 memory 提示查 `~/.skill-box/logs/2026-06/06-29-error_request.txt` 拿到 4 次连续失败的请求体,确认是 `{"src_group_path":"aa","dst_group_path":""}`。
**教训:** 改 bug 前先翻请求日志,能直接看到真实的请求/响应体,省去猜的步骤。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-06-29 23:03
**自测人:** AI(本轮 Claude)
**自测范围:** skillstore.MoveGroupDir + 回归测试

### 6.1 自动化测试

- `go test ./internal/skillstore/... -run "TestMoveGroupDir" -v` 结果: ✅ PASS(4 个)
  - `TestMoveGroupDir`(原有)— 分组挪到另一分组下
  - `TestMoveGroupDir_AncestorCheck`(原有)— 防止挪到自己的子目录
  - `TestMoveGroupDir_ToRoot`(原有)— 把分组挪到根下
  - `TestMoveGroupDir_NoOp_ToRoot`(新增)— 顶层分组"挪到根" + 嵌套分组挪到父级下 都 no-op 返 OK ✅
- `go test ./internal/skillstore/...` 结果: ✅ PASS(整个包)
- `go build ./...` 结果: ✅ 通过

### 6.2 手工 / 接口验证

- [x] 用例 1(用户报告 case): `MoveGroupDir("aa", "")` → 预期 nil → 实际 nil ✅
- [x] 用例 2(原有): `MoveGroupDir("aa/bb", "aa/cc")` → 预期 移到 aa/cc/bb → 实际 ✅(旧测试覆盖)
- [x] 用例 3(回归): `MoveGroupDir("aa", "aa/yy")` → 预期 拒(死循环防御) → 实际 拒 ✅(旧 AncestorCheck 覆盖)

### 6.3 边界 / 异常

- [x] `dstAbs == srcAbs`(本次 bug 触发 case)— 返 nil 不报错 ✅
- [x] `dstGroupPath=""` + 顶层分组 src — no-op 返 OK ✅
- [x] `dstGroupPath="<parent>"` + src 是其子 — no-op 返 OK(防御性覆盖) ✅

### 6.4 自测结论

- 总体: ✅ 通过
- 遗留问题: 需要用户重启 wails3 dev 让后端代码生效(`wails3 dev` 不会自动监听 Go 文件变更,memory 已记录)

## 7. 总结

### 7.1 完成了什么

- 修复了根下分组"挪到根"被误报 target already exists 的 bug
- 新增 no-op 短路 + 回归测试,锁住"挪到根"和"挪到父级下"两种同义操作
- 已 push 到 origin/main commit `daffa17`

### 7.2 留下了什么

- 代码: 1 个 store.go 改 16 行 + 1 个 group_tree_test.go 加 45 行
- 文档: 本 task 文件

### 7.3 留给下次的事

- 用户需要手动 `pkill -f "wails3 dev"` + 重新跑 `wails3 dev` 让新代码生效

### 7.4 复盘

- 做得好的: 改 bug 前先翻请求日志,直接拿到 4 次失败的真实请求体,省去猜;注释里写"合法"但实现没短路,加显式 short-circuit 防御性更好
- 能改进的: 之前加 MoveGroupDir 时就考虑 no-op 短路,而不是等用户撞上才补

## 8. 改动的文件

### 8.1 新增
- `docs/agent/task/2026-06/06-29_bug修复_根下分组挪到根误报target-exists.md` — 本任务过程文件

### 8.2 修改
- `api-server/internal/skillstore/store.go` — MoveGroupDir 加 no-op 短路(dstAbs == srcAbs 返 nil)
- `api-server/internal/skillstore/group_tree_test.go` — 加 TestMoveGroupDir_NoOp_ToRoot 回归测试
- `frontend/src/core/i18n/zh-CN.js` — 加 `dropToRoot` / `alreadyAtRoot` 两个 key
- `frontend/src/core/i18n/en-US.js` — 同步英文 i18n key
- `frontend/src/views/SkillsView.vue` — .tree-container 加 drop handler + 视觉反馈 + no-op 本地拦截
- `api-server/cmd/web/frontend/dist/index.html` — npm run build 同步
- `api-server/cmd/web/frontend/dist/assets/index-CE4Wepi8.css` — 新 build 产物
- `api-server/cmd/web/frontend/dist/assets/index-D9hawwvI.js` — 新 build 产物

### 8.3 删除
- 无

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash go test ./internal/skillstore/... -run "TestMoveGroupDir" -v` — 跑 move group 相关测试(4 个全 PASS)
- `Bash go build ./...` — 编译验证
- `Bash git add / commit / push` — 提交并推送(`daffa17`)

## 10. 对话轮次

## 1.1 对话轮次 (22:55)

> 用户原话:"查看一下首页实现可以将 skill 从分组内移动到根目录,现在尝试移动到根路径下的分组报错...提示 aa 分组已存在 但是实际上aa 分组是原本 skill 在的路径,我需要移动到根路径 也就是无父分组"

- **本轮做了:** 定位后端 `MoveGroupDir` 的 bug,加 no-op 短路 + 回归测试,提交并推送
- **本轮决定:** 用 `dstAbs == srcAbs` 短路返 nil(覆盖"挪到根"和"挪到父级下"两种 case)
- **本轮待办:** 用户需要重启 wails3 dev 让代码生效
- **本轮工具:** `Bash go test` / `Bash go build` / `Bash git commit && push`
- **状态更新:** 任务完成

## 1.2 对话轮次 (23:30)

> 用户原话:"现在移动没有报错 但是并没有移动成功 你可以检查一下请求日志"

- **本轮做了:** 查日志发现 `move_group` 返 200 + `{"ok":true}` 是 no-op(`aa` 本身就在根,移动无意义),用户期望的"让 skill 脱离分组"是另一个操作。AskUserQuestion 确认后,加 .tree-container drop handler 支持"拖到根",并把 no-op 改成显式 toast
- **本轮决定:**
  1. 区分"拖整组到根"(no-op,本地拦下给"已在根"toast)和"拖 skill 到根"(真实移动,后端处理)
  2. .tree-container 加 @drop / @dragover / @dragenter / @dragleave,接管空白处的 drop 事件
  3. 视觉反馈用 `::before` + `attr(data-drop-text)` 拿 i18n,支持中英文切换
  4. i18n 加 2 个 key:`dropToRoot` / `alreadyAtRoot`
- **本轮待办:** 用户测试新功能(拖 skill 到根、拖顶层 group 到根的两种 toast 反馈)
- **本轮工具:** `Bash npm run build` / `Bash rsync` / `Bash git add / commit / push`
- **状态更新:** 任务继续 — 主 bug 修复已完成,新增"拖到根"功能已提交 commit `8f542b8`,待用户验收
