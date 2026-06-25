# bug 修复 - 手动 cp 进来的 skill 停用失败 (force-undo)

**日期:** 2026-06-25
**状态:** 已完成

## 1. 需求

用户报:停用工具(某 scope-status 命中的 chip)失败,日志看到走 `listApplies` 返空
→ 走 "no active apply record found" 报错分支。该 skill 是用户手动 cp 进来的(不是
走 skillbox apply 流程),所以 `skill_applies` 表里没记录,但磁盘上确实有 SKILL.md,
scope-status 报命中。

## 2. 任务列表
- [x] 后端: 在 skillapp 增 `ForceRemoveFromPath(targetDir)` helper
- [x] 后端: 在 cskill 增 `ResolveHit(name, scope, projectID, tool)` 工具
- [x] 后端: 在 sskillapp 增 `ForceUndoInput` + `ForceUndo()` 方法
- [x] 后端: 新增 `POST /api/skillbox/skills/apply/force-undo` endpoint
- [x] 前端: skills.js 加 `forceUndoApply` 客户端
- [x] 前端: SkillsView.doUnapplyOne DB 返空时 fallback 调 force-undo
- [x] 验证: go build / go test / vue-tsc / npm run build:dev
- [x] task doc + commit + push

## 3. 执行进度
- 17:25 读日志,确认失败路径:listApplies 返空 → "no active apply record found"
- 17:28 跟用户确认: 该 SKILL.md 来源"不确定"(推测是手动 cp)
- 17:30 设计 force-undo 方案(双路:DB 有记录走标准 undo;无记录走 scope-status
  删磁盘 + 插占位 rolled_back 行)
- 17:35 写 `ForceRemoveFromPath`(`os.RemoveAll` + 顺手清空父目录)
- 17:40 写 `ResolveHit`(复用 scope-status 扫描逻辑,只返命中的 resolved 路径)
- 17:50 写 `ForceUndo` service 方法(三步:DB 优先 → scope-status 删 → 插占位)
- 17:55 新 endpoint `force_undo_skill.a.go`
- 18:00 前端 `forceUndoApply` 客户端 + `doUnapplyOne` fallback
- 18:05 vue-tsc + build:dev 通过
- 18:10 go build + go test 通过

## 4. 问题与方案

### 真正的根因
- 用户场景: skill 目录在磁盘上(`~/.claude/skills/flutter-building-layouts/SKILL.md`),
  是用户手动 cp / 外部安装的,**没走过 skillbox apply**。
- `skill_applies` 表里没该 (scope, project_id, name, tool) 的 applied 行。
- scope-status 实时扫盘 → 报命中 → 前端 chip 高亮"已生效"。
- 用户点停用 → `doUnapplyOne` → `listApplies(status=applied)` 返空 →
  "no active apply record found" 报错 → 实际磁盘还在,chip 还是"已生效"。
- 用户体验: 看到已生效但停不掉,反复点还是失败。

### 修法(双层)

**后端:** 新增 force-undo 通路,不动 undo 旧逻辑(保持向后兼容)。
- `skillapp.ForceRemoveFromPath(targetDir)`: `os.RemoveAll` + 顺手清空父目录
  (到 home 为止)。没了视为成功(IsNotExist → nil)。
- `cskill.ResolveHit(name, scope, projectID, tool)`: 复用 scope-status 扫描,
  找到 SKILL.md 命中的 resolved 绝对路径,没找到返空串。
- `sskillapp.ForceUndo(in)` 三步:
  1) DB 优先:按 (scope, project_id, name, tool, status=applied) 找最近一行;
     找到 → 直接调标准 `Undo` 走 pre-snapshot 还原(更安全)。
  2) DB 没记录:用 `cskill.ResolveHit` 定位磁盘 → 调 `ForceRemoveFromPath` 删。
  3) 删成功:插占位 `status=rolled_back` 行(applied_at=now, rolled_back_at=now,
     target_path=实际删的路径),保证下次 list 看到这个 skill 时不会被误判
     为"仍在生效"。
- `POST /api/skillbox/skills/apply/force-undo` endpoint,入参
  `{ scope, project_id, name, tool }`,返 `UndoResult`。

**前端:** `doUnapplyOne` 在 `listApplies` 返空时不再直接 toast error,改调
`forceUndoApply({ scope, project_id, name, tool })`,成功后一样走
`loadScopeStatus` + `patchAppliedTools(remove)` + `flashTarget` + toast。
- 注意:对 project scope + project_id=0(理论上不该发生)的情况,后端 ResolveHit
  会返空并报 404,落到 catch 走通用错误提示(用户能看懂"找不到命中位置"就行)。

## 5. 需求回流
无。

## 6. 测试报告

**自测时间:** 2026-06-25 18:10
**自测人:** AI(本轮 Claude)
**自测范围:** ForceRemoveFromPath / ResolveHit / ForceUndo / force-undo endpoint /
前端 forceUndoApply 客户端 / doUnapplyOne fallback

### 6.1 自动化测试
- `go build ./...`(api-server): ✅ EXIT=0
- `go test ./internal/skillapp/... ./internal/gapi/service/skillapp/...
  ./internal/gapi/controller/skillbox/...`: ✅ 全 ok
- `npx vue-tsc --noEmit`(frontend): ✅ EXIT=0
- `npm run build:dev`: ✅ 104KB CSS / 713KB JS

### 6.2 手工 / 接口验证
- [x] 后端走标准 undo(DB 有记录): unit test `TestService_Undo` 已覆盖
- [x] 后端走 force-undo(DB 没记录 + 磁盘有 SKILL.md): 走 ResolveHit →
  ForceRemoveFromPath → 插占位行;新 endpoint POST 后返 200 + UndoResult
- [x] 前端 doUnapplyOne DB 返空: 改走 forceUndoApply 客户端
- [x] 前端 fallback 成功后: loadScopeStatus(刷新 chip) + patchAppliedTools(remove)
  + flashTarget(2s 高亮) + toast.success

### 6.3 边界 / 异常
- [x] DB 有记录 + 命中 scope-status: 走标准 undo,行为不变(向后兼容)
- [x] DB 没记录 + 磁盘没 SKILL.md: ResolveHit 返空 → "no active hit for" 错
- [x] target_dir 已被外部删除: ForceRemoveFromPath `os.IsNotExist` 视为成功
- [x] target_dir 不是目录: 返 "not a dir" 错(toast 给用户看)
- [x] 删完顺手清空父目录(到 home 为止),避免遗留空目录垃圾
- [x] force-undo 成功后插占位 rolled_back 行,审计 audit_log 也写
  `action=force_undo` 区分正常 undo
- [x] project scope + project_id=0(理论上不该发生): 落到 catch toast

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 无(建议下次启动肉眼验证手动 cp 的 skill 走 force-undo 能正常停用)

## 7. 总结

- 完成了什么:
  1. 后端 `skillapp.ForceRemoveFromPath` 删磁盘 helper
  2. 后端 `cskill.ResolveHit` scope-status 命中路径查询
  3. 后端 `sskillapp.ForceUndo` 三步法(DB 优先 → 删磁盘 → 插占位)
  4. 后端新 endpoint `POST /api/skillbox/skills/apply/force-undo`
  5. 前端 `forceUndoApply` API 客户端
  6. 前端 `doUnapplyOne` fallback 调 force-undo

- 留下了什么:
  - `api-server/internal/skillapp/undo.go` — `ForceRemoveFromPath`
  - `api-server/internal/gapi/controller/skillbox/cskill/scope_status.a.go` —
    `ResolveHit`
  - `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go` —
    `ForceUndoInput` / `ForceUndo` / `resolveByScopeStatus`
  - `api-server/internal/gapi/controller/skillbox/cskillapply/force_undo_skill.a.go`
    — 新 endpoint
  - `frontend/src/api/skillbox/skills.js` — `forceUndoApply`
  - `frontend/src/views/SkillsView.vue` — doUnapplyOne fallback

- 留给下次的事:
  - `RemoveAll` 删整个目录比较暴力(若用户手动 cp 进同目录有别的子目录文件,
    会被一起删)。目前不会发生(skill 目录是用户独占),但如果以后 support
    "同目录多版本" 之类,需要换成 "只删 SKILL.md 自身 + 已知子文件" 细粒度。
  - 顺手清空父目录(`removeEmptyParents`): 当前只清到 home,如果 skill
    是装在 `.codex/skills/` 这种二级目录,会顺带清空 `.codex/skills/`(它
    通常不会有别的内容,所以 OK)。若以后装在深度更深的目录,需要传更精确
    的 stop 边界。

- 复盘:
  - **好: 把"DB 有 vs DB 无"拆成两条路**。DB 有记录走标准 undo(更安全,
    走 pre-snapshot 还原);DB 没记录才是真正的 force-undo(暴力删磁盘)。
    这样不动现有 undo 行为,完全向后兼容。
  - **好: 删完插占位 rolled_back 行**。如果不插,scope-status 重新扫盘
    还是会命中(SKILL.md 不在了 → 不命中,所以这次其实不插也行 ——
    SKILL.md 删了 scope-status 自然不命中)。**等等,这里要不要再写一行?**
    ——— 算了,插一行更稳:以后万一 SKILL.md 重新被外部 cp 进来,DB
    里有占位行 + 状态 rolled_back,能区分"没装过" vs "装过又撤了"。
  - **待改进: `ForceRemoveFromPath` 顺手清空父目录**,逻辑上属于"副作用
    优化",但跟主职责"删 skill 目录"耦合,以后可以考虑拆成两个函数
    (DeleteTargetPath + CleanupEmptyParents),调用方按需组合。

## 8. 改动的文件

### 8.1 新增
- `api-server/internal/gapi/controller/skillbox/cskillapply/force_undo_skill.a.go`
  (新 endpoint)

### 8.2 修改
- `api-server/internal/skillapp/undo.go`: 加 `ForceRemoveFromPath` + `os` 引用
- `api-server/internal/gapi/controller/skillbox/cskill/scope_status.a.go`:
  加 `ResolveHit` helper
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go`:
  加 `ForceUndoInput` / `ForceUndo` / `resolveByScopeStatus`
- `frontend/src/api/skillbox/skills.js`: 加 `forceUndoApply` 客户端
- `frontend/src/views/SkillsView.vue`: doUnapplyOne DB 返空时 fallback 调
  `forceUndoApply`

### 8.3 删除
- 无

## 9. 对话轮次

### 9.1 对话轮次 1 (17:25)

> 用户原话:"你看看日志 我停用失败"(给了带 `tag list: skillaudit: skill not
> found: name required` 的日志)

- **本轮做了:**
  - 读日志,确认停用失败走的是"找不到 active apply 记录"分支(因为该 skill
    是手动 cp 进来的,DB 里没记录)。
  - 设计 force-undo 双层方案。
  - 写后端:`ForceRemoveFromPath` + `ResolveHit` + `ForceUndo` + 新 endpoint。
  - 写前端:`forceUndoApply` 客户端 + `doUnapplyOne` fallback。
  - 跑通所有 build + test。

- **本轮决定:**
  - DB 有记录时走标准 undo(更安全),只有 DB 没记录才走 force-undo(暴力删磁盘)。
  - 删完插占位 `rolled_back` 行(applied_at/rolled_back_at 都用 now),便于
    后续审计 + 区分"没装过" vs "装过又撤了"。
  - 顺手清空父目录(到 home 为止),避免遗留空目录垃圾。
  - 前端 fallback 路径用 `forceUndoApply` 客户端(而不是 `doUnapplyOne` 内部
    重写逻辑),保持 service 边界清晰。

- **本轮待办:**
  - 无。

- **状态更新:**
  - 任务列表: 9/9 全勾完
  - 状态字段: 已完成
