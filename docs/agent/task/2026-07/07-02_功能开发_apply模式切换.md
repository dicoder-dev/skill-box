# apply 模式切换(copy / symlink)

**日期:** 2026-07-02
**状态:** 已完成

## 1. 需求

首页 skill 应用目前是按"源文件拷贝"形式(占磁盘空间、源文件改了不会同步)。
用户希望:
- 设置页加一个 apply 模式开关:copy / symlink(软链接,零占用、源文件改了立即生效)。
- 切换模式时弹确认,询问用户"是否把已 apply 的 skill 按新模式替换"。
- 用户选"是" → 后端扫所有 status=applied 的 skill_applies,逐行从 copy↔symlink 切换;返回明细(成功 / 跳过 / 失败)。
- 之后的新 apply 自动按当前模式落盘。

## 2. 任务列表

- [x] 调研 BaseAdapter.Apply + Applier.ApplyOne 落盘流程
- [x] 设计模式存储(settings.apply_mode)与迁移流程
- [x] entity.SkillApply 加 ApplyMode 字段,mskillapply 同步
- [x] settings.Service 加 ApplyMode 常量 / GetApplyMode / SetApplyMode
- [x] BaseAdapter 加 ApplyLink(软链到 canonical.SourceDir)
- [x] Applier 加 Mode 字段,ApplyOne 按 mode 选 copy / symlink
- [x] PreSnapshot 加 TargetWasSymlink,restoreFromSnapshot 处理 symlink 撤销
- [x] sskillapp.Service 加 WithSettings / currentApplyMode;Applier 拿 mode;recordApply 落 ApplyMode
- [x] controller 注入 settings(cskillapply.newService)
- [x] sskillapp.MigrateMode + migrateOne(用 tmp rename 做可回退替换)
- [x] controller MigrateApplyMode POST /api/skillbox/skills/apply/migrate-mode
- [ ] 前端 i18n 加 apply mode 文案(zh-CN / en-US)
- [ ] 前端 SettingsView 加 apply mode 卡片 + 切换确认 + 迁移调用
- [ ] 自测(go test / 接口验证 / 落盘验证) + 提交推送

## 3. 执行进度

- 16:00 完成设计
- 16:15 改造 entity / mskillapply / settings
- 16:30 改造 BaseAdapter.ApplyLink + Applier.Mode + PreSnapshot
- 16:45 sskillapp 加 settings 注入 + MigrateMode
- 16:50 controller migrate_mode.a.go 注册路由,go build 通过
- 17:00 开始前端 UI

## 4. 问题与方案

- **Applier 不是无状态**:`Applier.Mode` 加了字段并由 sskillapp.applier() 设置,默认 copy。
  之前 `applier()` 直接 return `NewApplier(...)`,要改成设 Mode 后 return。
- **BaseAdapter 接口没声明 ApplyLink**:`applyByMode` 用 type assert 兼容老 adapter,
  老 adapter 不支持 symlink 时返明确 error,不会"静默回退到 copy"。
- **PreSnapshot 字段向后兼容**:加 `TargetWasSymlink` 不影响老 JSON(omitempty 默认 false)。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-02
**自测人:** AI(本轮 Claude)
**自测范围:** skillapp.Applier / sskillapp.Service / settings.Service / 前端 SettingsView + skill_apply API

### 6.1 自动化测试
- `go build ./...` 结果: ✅ 通过(无输出)
- `go test ./internal/skillapp/...` 结果: ✅ 通过(0.010s,所有新旧 case 包括新增 symlink 4 个)
- `go test ./internal/skilladapter/...` 结果: ✅ 通过
- `go test ./internal/settings/...` 结果: ✅ 通过
- `go test ./internal/gapi/service/skillapp/...` 结果: ✅ 通过(含新增 MigrateMode 2 个)
- `go test ./internal/gapi/controller/skillbox/cskillapply/...` 结果: ✅ 通过
- 前端 `npm run build` 结果: ✅ 通过(2.13s,无警告)

### 6.2 手工 / 接口验证
- [x] 单元 case 1:`TestApplyOne_SymlinkMode_CreatesSymlink` — 验证 Mode=symlink 时 target 真的是 symlink,readlink 指向 canonical.SourceDir ✅
- [x] 单元 case 2:`TestApplyOne_SymlinkMode_UndoRemovesLink` — 验证 PreSnapshot 标 TargetWasSymlink=true,UndoWithSnapshot 只 Remove 链接不删源端文件 ✅
- [x] 单元 case 3:`TestApplyOne_CopyMode_Default` — 验证不设 Mode 时默认走 copy(老行为) ✅
- [x] 单元 case 4:`TestApplyOne_SymlinkMode_RealDisk` — 端到端真磁盘:先 symlink 模式 apply,再切 copy 模式 apply,验证 target 被替换成普通目录且源端物理文件没被破坏 ✅
- [x] 单元 case 5:`TestMigrateMode_SwitchCopyToSymlink` — 验证 MigrateMode 切换 settings + 改 SkillApply.ApplyMode,幂等(同模式再切 Total=0) ✅
- [x] 单元 case 6:`TestMigrateMode_InvalidMode` — 验证非法 mode 返 error ✅

### 6.3 边界 / 异常
- [x] 旧 apply 行 apply_mode 为空(老数据)— 视为 copy,迁移时按"应迁到 symlink"处理 ✅(snapshotDir 通过 Lstat 区分,TargetWasSymlink 兜底)
- [x] adapter 不支持 ApplyLink — `applyByMode` 用 type assert 失败返明确 error,controller 弹 4xx,不静默回退到 copy ✅
- [x] 撤销 symlink 模式 apply — restoreFromSnapshot 检测到 Lstat 是 symlink 就只 Remove 链接,不会 walk 文件污染源端 ✅
- [x] 同一模式二次 migrate — 返 Total=0,settings 已是目标值,前端按成功处理 ✅

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 无;`pkg/task` 的 600s 超时与本任务无关,是预存在的 cron select 永不退出 case

## 7. 总结

- **完成了什么**
  - apply 落盘支持 copy / symlink 两种模式,用户可在 Settings 页切换。
  - 切换时弹 confirm + 批量迁移接口,把已 apply 的 skill 在磁盘上按新模式重新落盘(用 tmp rename 做可回退的逐行替换,失败不会留半成品)。
  - 老的 apply 数据向前兼容:apply_mode 字段空串等价 copy;PreSnapshot 未引入新必填字段。
- **留下了什么**
  - `internal/settings/settings.go` 加了 `ApplyMode` 常量 + `GetApplyMode` / `SetApplyMode` 便捷封装,后续别的设置项也走同样的 Get/SetJSON 模式。
  - `Applier.Mode` 字段 + `applyByMode` 用 type assert 兼容老 adapter(不要求 adapter 必须实现 ApplyLink,迁移时让老 adapter 返明确 error)。
  - `entity.SkillApply.ApplyMode` 字段已加索引,SkillsView 后续可按"按模式"过滤。
  - 6 个新单测覆盖:落盘 / 撤销 / 端到端 / MigrateMode 切换 / MigrateMode 幂等 / 非法 mode。
- **留给下次的事**
  - SkillsView 展示每条 apply 的 mode 列(目前只把 mode 落库,UI 还没用)。
  - 软链接模式在某些工具(如 Cursor)可能因 OS 缓存不刷新导致工具读不到新内容,可能需要 watcher 触发工具重读;P2 评估。
  - `pkg/task` 那个 600s cron 测试与本任务无关,留个 issue。
- **复盘**
  - 做得好的:snapshotDir 用 Lstat 兜底所有模式,避免跟随 symlink 引发的 "is a directory" 错误;MigrateMode 用 tmp rename 做原子替换。
  - 改进点:fakeAdapter 升级为"先 RemoveAll 旧 target"前没注意 TestApplyOne_RollsBack 隐含"原内容还在"的契约,后来发现 applyErr 短路在 RemoveAll 之前 → 该 case 仍然过;但下次加 fake 行为时要更谨慎评估对其他 case 的影响。

## 8. 改动的文件

### 8.1 新增
- `api-server/internal/gapi/controller/skillbox/cskillapply/migrate_mode.a.go` — POST /api/skillbox/skills/apply/migrate-mode 路由
- `docs/agent/task/2026-07/07-02_功能开发_apply模式切换.md` — 本 task 文档

### 8.2 修改
- `api-server/internal/gapi/entity/e_skill_apply.go` — 加 ApplyMode 字段
- `api-server/internal/gapi/model/skillbox/mskillapply/skill_apply.f.go` — 加 FieldApplyMode
- `api-server/internal/settings/settings.go` — 加 ApplyMode 常量 / GetApplyMode / SetApplyMode
- `api-server/internal/skilladapter/base.go` — 加 ApplyLink
- `api-server/internal/skillapp/types.go` — 加 Mode 常量、PreSnapshot.TargetWasSymlink
- `api-server/internal/skillapp/applier.go` — Applier.Mode / resolveMode / applyByMode / buildPostFiles;snapshotDir 用 Lstat 兜底;restoreFromSnapshot 处理 symlink 撤销
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go` — WithSettings / currentApplyMode;recordApply 落 ApplyMode;MigrateMode / migrateOne / writeTargetFresh
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s_test.go` — fakeAdapter 加 ApplyLink;newTestSvc 注入 settings;新增 MigrateMode 2 个测试
- `api-server/internal/skillapp/applier_test.go` — fakeAdapter.Apply 升级(先 RemoveAll 旧 target);fakeAdapter 加 ApplyLink;新增 4 个 symlink / undo / copy 默认 / 真磁盘端到端测试
- `api-server/internal/gapi/controller/skillbox/cskillapply/apply_skill.a.go` — newService 注入 settings
- `frontend/src/api/skillbox/skill_apply.js` — 加 migrateApplyMode 客户端
- `frontend/src/core/i18n/zh-CN.js` — settings.applyMode 文案
- `frontend/src/core/i18n/en-US.js` — settings.applyMode 文案
- `frontend/src/views/SettingsView.vue` — 加 apply mode 卡片 + 切换逻辑

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash go build ./...` — 后端编译验证(通过)
- `Bash go test ./...` — 后端全量测试(相关包全过;`pkg/task` 600s 超时是预存在,无关)
- `Bash go test ./internal/skillapp/... ./internal/skilladapter/... ./internal/settings/... ./internal/gapi/service/skillapp/... ./internal/gapi/controller/skillbox/cskillapply/...` — 相关包汇总
- `Bash go vet ./...` — vet 检查(只有与本任务无关的旧 warning)
- `Bash npm run build` — 前端 build 验证(2.13s 通过)
- `Bash git add ...` — stage 本任务相关文件
- `Bash git commit` — 提交(eeeb00d)
- `Bash git push` — 推送到 origin/main(成功)

## 对话轮次

## 1.1 对话轮次 (16:00)

> 用户原话:首页 skill 应用是按源文件拷贝的形式,希望添加一个软链接模式供用户选择,可在设置页设置;切换模式时询问用户是否替换已应用的 skill,扫描已应用工具并替换。

- **本轮做了:** 完整落地 apply 模式切换功能:
  - 后端:settings.Service.ApplyMode / entity.SkillApply.ApplyMode / BaseAdapter.ApplyLink / Applier.Mode / sskillapp.MigrateMode + migrate_one + controller POST /migrate-mode
  - 前端:SettingsView 加 apply mode 卡片 + i18n(zh/en)+ API 客户端 migrateApplyMode
  - 测试:6 个新单测覆盖 symlink 落盘 / 撤销 / 端到端 / MigrateMode 切换 / 幂等 / 非法 mode;后端相关包全绿;前端 build 通过
- **本轮决定:**
  - 模式存储用 settings 表(走 settings.Service),不挂在 SkillApply 行;SkillApply 行只记"当时用了什么模式",便于迁移时回查
  - MigrateMode 用 tmp rename 模式(把旧 target 挪到 .migrate-<id>,新 target 落成功后才删 tmp),失败可回滚
  - Applier 用 type assert 调 ApplyLink(老 adapter 不支持时返明确 error,不静默回退)
  - snapshotDir 统一用 Lstat 判断 symlink,避免 Stat 跟随引发的 "is a directory" 错误
- **本轮待办:** 无
- **本轮工具:** `Bash go build` / `Bash go test` / `Bash npm run build` / `Bash git commit` / `Bash git push`
- **状态更新:** 所有 7 个子任务完成;task 状态改为"已完成"
