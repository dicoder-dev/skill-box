# 首页左侧 skill 列表改造成多级分组 + 右键菜单

**日期:** 2026-06-29
**状态:** 进行中

## 1. 需求

把首页(SkillsView)左侧的扁平 skill 列表改造成**类文件目录树**的多级分组结构,核心功能:

1. **多级分组**:分组可嵌套(类似文件目录),支持右键新建子分组、删除分组
2. **拖拽**:skill 可拖到指定分组、分组可拖到另一分组嵌套
3. **删除分组时同步删除组内全部 skill**
4. **右键 skill 弹菜单**:删除 / 打 tag / 在 Finder 打开
5. **删除时询问"是否同步清理已应用到工具目录的副本"**(默认勾选;范围 = 5 个工具的全局 / 项目级目录,codex/claude/opencode/cursor/trae)
6. **删除走 `force-undo` 链路**:DB 没 apply 记录也能定位到磁盘目录

## 2. 任务列表

- [x] 计划阶段:与用户确认决策(分组存储方式、删除询问方式、右键组件、拖拽范围)
- [ ] 后端 #1: skillstore 支持 path 分组子目录(Save/Load/Delete/List/Exists + 新增 Move)
- [ ] 后端 #2: skilladapter.NormalizeGroupName + Manifest.GroupPath 字段
- [ ] 后端 #3: sskill service 增 ListTree / CreateGroup / DeleteGroup / MoveSkill
- [ ] 后端 #4: cskill 控制器增 create_group / delete_group / move_skill,改造 list_skills / delete_skill
- [ ] 后端 #5: go test ./... 通过
- [ ] 前端 #6: 新建 ContextMenu.vue(轻量自研)+ TreeNode.vue 递归组件
- [ ] 前端 #7: 新建 core/store/skill-tree.js Pinia + api/skillbox/skills.js 增 4 个封装
- [ ] 前端 #8: SkillsView 左侧改树形 + 右键 + 拖拽 + 删除复选框 + i18n
- [ ] 前端 #9: npm run build 通过 + 手工验证(wails3 dev)

## 3. 执行进度

- 15:30 完成计划编写,已获用户审批(plan 文件: `/Users/brody/.claude/plans/crystalline-spinning-shamir.md`)
- 关键决策:
  - 分组 → 映射到文件系统子目录 (`~/.skill-box/skills/<group>/<skill>/SKILL.md`)
  - 删除 → 弹窗带"同步清理工具目录"复选框(默认勾选)
  - 右键菜单 → 自研轻量 ContextMenu 组件(零依赖)
  - 拖拽 → skill 与分组都可拖,分组可嵌套
- 约束: skill 叶子名仍走 `NormalizeName`(不含 `/`);分组名独立规约(允许 `/` 嵌套,走 `safeRelPath` 防穿越)

## 4. 问题与方案

待记录。

## 5. 需求回流

无。

## 6. 测试报告

待任务完成后填。

## 7. 总结

待任务完成后填。

## 8. 改动的文件

待任务完成后填。

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- 暂无
