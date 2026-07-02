# apply 模式切换(copy / symlink)

**日期:** 2026-07-02
**状态:** 进行中

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

(待自测后填)

## 7. 总结

(任务结束填)

## 8. 改动的文件

### 8.1 新增
- `api-server/internal/gapi/controller/skillbox/cskillapply/migrate_mode.a.go` — POST /api/skillbox/skills/apply/migrate-mode 路由
- (待加) `frontend/src/...` — SettingsView UI

### 8.2 修改
- `api-server/internal/gapi/entity/e_skill_apply.go` — 加 ApplyMode 字段
- `api-server/internal/gapi/model/skillbox/mskillapply/skill_apply.f.go` — 加 FieldApplyMode
- `api-server/internal/settings/settings.go` — 加 ApplyMode 常量 / GetApplyMode / SetApplyMode
- `api-server/internal/skilladapter/base.go` — 加 ApplyLink
- `api-server/internal/skillapp/types.go` — 加 Mode 常量、PreSnapshot.TargetWasSymlink
- `api-server/internal/skillapp/applier.go` — Applier.Mode / resolveMode / applyByMode / buildPostFiles;snapshotDir 接 mode;restoreFromSnapshot 处理 symlink
- `api-server/internal/gapi/service/skillapp/sskillapp/skillapp.s.go` — WithSettings / currentApplyMode;recordApply 落 ApplyMode;MigrateMode / migrateOne / writeTargetFresh
- `api-server/internal/gapi/controller/skillbox/cskillapply/apply_skill.a.go` — newService 注入 settings

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- (本任务用得不多,部分地方用 WebFetch 查 Go os.Symlink 语义)

### 9.2 Skill
- (无)

### 9.3 CLI
- `Bash go build ./...` — 编译验证
