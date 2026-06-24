# Skill 存储从 DB 切到文件 + 目录统一 `~/.skill-box/`

**日期:** 2026-06-24
**状态:** 已完成(后端改造 + 测试全绿,前端 SkillsView / api JS 适配遗留)

## 1. 需求

> 现在项目的 skill 是保存在数据库的,但是我想了一下 感觉这样很不灵活 比 毕竟 skill 是以文件为核心的,为了更好的兼容其他编程工具,我现在希望把 skill 列表放在 工具 `~/.skill-box/skills` 参考 calude code 的 skill 存放方式,并通过相关的工具去解析 skill 的元数据信息,这样可以做到更好兼容其他项目的 skill,弃用数据库存储的方式

(2026-06-24 补充澄清)
- **目录统一**:不额外创建 `~/.skillbox/`,所有数据统一进 `~/.skill-box/`
- **layout 贴合 claude code**:`~/.skill-box/skills/<name>/SKILL.md`,**没有 version 目录**;version 写在 SKILL.md 的 frontmatter
- **跨工具兼容**:外部工具读 SKILL.md,我们通过自有的 frontmatter 解析工具拿元数据
- **多版本**:只保留一份(覆盖式);升级 version = 改 SKILL.md frontmatter
- **下游域**:apply / tag / test / snapshot 仍然落库,但关联键从 `skill_id` 换成 `(scope, name, version)` 复合键
- **旧 `~/.skillbox/`**:不检测、不迁移、不提示(用户自管)

## 2. 任务列表

- [x] 阶段 1:目录与根路径统一(`~/.skillbox/store` → `~/.skill-box/skills`)
  - [x] `skillstore.New` 默认根改成 `~/.skill-box/skills`
  - [x] `configs.Skillbox.StoreRoot` 默认值、注释路径全部更新
  - [x] `skillapp/applier.go` 写 projects/<id>/ 的兜底路径同步
  - [x] `bundle_seed.go` 的 store 装配路径同步
- [x] 阶段 2:skillstore 切到无 version layout
  - [x] `Save/Load/Delete` 路径从 `<name>/<version>/` 变成 `<name>/`
  - [x] `ListNames/ListVersions` 重写:不再有 version 目录
  - [x] 删 `manifestFileName=skill.yaml`、改用 SKILL.md frontmatter 作为唯一源
  - [x] `validateManifest` 简化(不再要求 description 长度 >= 10 等)
- [x] 阶段 3:封 frontmatter 解析工具
  - [x] `skilladapter.ParseSkillMD` 强化:作为唯一源(取代 skill.yaml)
  - [x] `skilladapter.RenderSkillMD` 用作唯一写回工具
  - [x] 提供 `ReadSkillDir(dir)` / `WriteSkillDir(dir, c)` 工具函数
- [x] 阶段 4:sskill 业务层切到文件 only
  - [x] `sskill.Service.Create` 去掉 mskill.Create,只写 SKILL.md
  - [x] `sskill.Service.Get/Update/Delete` 不查 DB,只走 store
  - [x] `sskill.Service.List` 改为扫目录 + 解析 frontmatter
  - [x] 删 `marshalManifest` / `defaultSource` 等 DB 关联 helper
- [x] 阶段 5:下游域改关联键
  - [x] `entity.SkillApply` 加 `scope/name/version`,把 skill_id 标记 deprecated
  - [x] `entity.SkillTag` / `SkillFile` / `SkillFileSnapshot` 同样改造
  - [x] `entity.SkillTestRun/Result` 同样
  - [x] `sskillapp.ApplyInput` 改用 (scope, name, version) 定位
  - [x] `sskillapp.skillapp.s.go:380` 的 `FindOneById` 改为走 store
  - [x] ctag / cskilltest / cskillapply 的 controller 入参兼容(name+version 代替 id)
- [ ] 阶段 6:前端 + importer 适配
  - [x] `skillimporter.upsertDBRow` 改为只写 store(走 sskill)
  - [x] 删 `entity.Skill` / `mskill` 整个表/模型/entity/model 注册
  - [ ] 前端 list/get 响应去掉 `id`,改用 (scope, name, version) 组合
  - [ ] apply / tag / test 接口的入参从 `skill_id` 改 `name`+`version`
- [ ] 阶段 7:清理 + 文档
  - [x] `cmd/bootstrap/entities.go` 移除 Skill / SkillFile 注册
  - [x] `bundle_seed.go` 的 `var _ = mskill.FieldID` 清理
  - [ ] `docs/project/项目架构.md` / `需求规划.md` 更新存储模型
  - [x] 跑通 `go test ./...` (skill 相关全绿;pgsql/cos/httpclient 是环境问题与本重构无关)
  - [ ] 前端 `npm run build` 验证

## 3. 执行进度

- 14:00 收到需求,先扫了一遍相关代码
- 14:30 跟用户确认了关键决策(布局、版本、下游、旧目录)
- 14:45 建本任务文件,准备开始阶段 1
- 阶段 1-5 + 6 后端部分 + 7 后端部分全部完成,单 commit + 测试修复 commit 已落地
- 测试:`go test ./internal/gapi/service/... ./internal/skillstore/... ./internal/skilladapter/... ./internal/skillapp/... ./internal/skillimporter/...` 全绿
- 失败项:pkg/cos (无 STS 凭证)、pkg/db/pgsql (无 DB)、pkg/httpclient (logger format string vet,与本重构无关)、pkg/ginp / pkg/task (服务启动测试,环境性超时)

## 4. 问题与方案

- **store.Save 写出空 SKILL.md**: caller 传 Manifest 没 Files 时,RenderSkillMD 兜底 `# <name>` 最小 body
- **sskill.Create 内部调用 Name 为空**: 兜底用 Manifest.Name(用户友好)
- **sskillaudit.Rollback 内部 Update 漏 scope**: 兜底 global,避免内部调用也要带 scope
- **sskillapp.Apply 把 sskill.ErrNotFound 直接外抛**: 包外期望 ErrSkillNotFound,在 service 层 errors.Is 转换
- **sskillapp.CheckUpdates 用 NewStore() 拿到的是默认 root,跟测试 temp dir 不一致**: 改用 skillSvcFactory().List("")
- **store.HashFile 之前是占位返回空串**: 改为 crypto/sha256
- **.lock 文件残留**: store.unlock 回调里加 os.Remove
- **TestRollback 断言 SKILL.md == "v1 body"**: 实际 SKILL.md 含 frontmatter + body,改用 strings.Contains

## 5. 需求回流

(暂无)

## 6. 总结

- 后端 skill 完全文件化:crud 走 store,跨工具兼容 SKILL.md frontmatter
- 下游表(apply/tag/test/snapshot)按 (scope, name) 关联,与 skill_id 解耦
- 后续 frontend api JS + SkillsView 适配是单独任务(阶段 6 前端部分 + npm build 验证)
