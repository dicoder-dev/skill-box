# Skill 存储从 DB 切到文件 + 目录统一 `~/.skill-box/`

**日期:** 2026-06-24
**状态:** 进行中

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

- [ ] 阶段 1:目录与根路径统一(`~/.skillbox/store` → `~/.skill-box/skills`)
  - [ ] `skillstore.New` 默认根改成 `~/.skill-box/skills`
  - [ ] `configs.Skillbox.StoreRoot` 默认值、注释路径全部更新
  - [ ] `skillapp/applier.go` 写 projects/<id>/ 的兜底路径同步
  - [ ] `bundle_seed.go` 的 store 装配路径同步
- [ ] 阶段 2:skillstore 切到无 version layout
  - [ ] `Save/Load/Delete` 路径从 `<name>/<version>/` 变成 `<name>/`
  - [ ] `ListNames/ListVersions` 重写:不再有 version 目录
  - [ ] 删 `manifestFileName=skill.yaml`、改用 SKILL.md frontmatter 作为唯一源
  - [ ] `validateManifest` 简化(不再要求 description 长度 >= 10 等)
- [ ] 阶段 3:封 frontmatter 解析工具
  - [ ] `skilladapter.ParseSkillMD` 强化:作为唯一源(取代 skill.yaml)
  - [ ] `skilladapter.RenderSkillMD` 用作唯一写回工具
  - [ ] 提供 `ReadSkillDir(dir)` / `WriteSkillDir(dir, c)` 工具函数
- [ ] 阶段 4:sskill 业务层切到文件 only
  - [ ] `sskill.Service.Create` 去掉 mskill.Create,只写 SKILL.md
  - [ ] `sskill.Service.Get/Update/Delete` 不查 DB,只走 store
  - [ ] `sskill.Service.List` 改为扫目录 + 解析 frontmatter
  - [ ] 删 `marshalManifest` / `defaultSource` 等 DB 关联 helper
- [ ] 阶段 5:下游域改关联键
  - [ ] `entity.SkillApply` 加 `scope/name/version`,把 skill_id 标记 deprecated
  - [ ] `entity.SkillTag` / `SkillFile` / `SkillFileSnapshot` 同样改造
  - [ ] `entity.SkillTestRun/Result` 同样
  - [ ] `sskillapp.ApplyInput` 改用 (scope, name, version) 定位
  - [ ] `sskillapp.skillapp.s.go:380` 的 `FindOneById` 改为走 store
  - [ ] ctag / cskilltest / cskillapply 的 controller 入参兼容(name+version 代替 id)
- [ ] 阶段 6:前端 + importer 适配
  - [ ] 前端 list/get 响应去掉 `id`,改用 (scope, name, version) 组合
  - [ ] apply / tag / test 接口的入参从 `skill_id` 改 `name`+`version`
  - [ ] `skillimporter.upsertDBRow` 改为只写 store(走 sskill)
  - [ ] 删 `entity.Skill` / `mskill` 整个表/模型/entity/model 注册
- [ ] 阶段 7:清理 + 文档
  - [ ] `cmd/bootstrap/entities.go` 移除 Skill / SkillFile 注册
  - [ ] `bundle_seed.go` 的 `var _ = mskill.FieldID` 清理
  - [ ] `docs/project/项目架构.md` / `需求规划.md` 更新存储模型
  - [ ] 跑通 `go test ./...` + 前端 `npm run build`

## 3. 执行进度

- 14:00 收到需求,先扫了一遍相关代码
- 14:30 跟用户确认了关键决策(布局、版本、下游、旧目录)
- 14:45 建本任务文件,准备开始阶段 1

## 4. 问题与方案

(待补:开发中遇到的具体问题)

## 5. 需求回流

(暂无)

## 6. 总结

(任务结束时填)
