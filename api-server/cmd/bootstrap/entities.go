package bootstrap

import "ginp-api/internal/gapi/entity"

// EntityAutoMigrateList 自动迁移的实体列表。
// 业务模块如果新增 entity,应在这里登记;或者在调用方业务侧维护自己的
// 列表 + 调 dbs.GetWriteDb().AutoMigrate(...)。
//
// Skill Box 表(见 docs/project/需求规划.md 第 6 节):
// project / skill_file / skill_tag / skill_file_snapshot /
// skill_apply / ai_provider / market_source / market_skill /
// onboarding_state / setting / tool / tool_path
//
// 2026-06-24 改造:skill 表(对应 entity.Skill)弃用,源数据走 ~/.skill-box/skills/<name>/SKILL.md;
// 下游表(skill_file / skill_apply / skill_tag / skill_file_snapshot / skill_test_*)
// 保留,关联键从 skill_id(uint)改为 (scope, project_id, name) 复合键。
// 2026-07-04 改造:audit_log 表下线,业务事件改写到 ~/.skill-box/logs/<YYYY-MM>/INFO.csv。
var EntityAutoMigrateList = []any{
	new(entity.User),

	new(entity.Project),
	new(entity.SkillFile),
	// 2026-07-17 改造:skill_tags / skill_file_snapshots 表下线,
	// 版本管理改走 ~/.skill-box/skills/ 的 git 仓库(go-git)。
	new(entity.SkillApply),
	new(entity.SkillTestRun),
	new(entity.SkillTestResult),
	new(entity.AIProvider),
	new(entity.MarketSource),
	new(entity.MarketSkill),
	new(entity.OnboardingState),
	new(entity.Setting),
	// 2026-06-30 二改:工具元数据从 yaml embed 迁到 DB,前端可编辑
	new(entity.Tool),
	new(entity.ToolPath),
}

// EntityGenerationList 需要自动生成的实体(代码生成器使用,运行期不参与)。
var EntityGenerationList = []any{
	new(entity.User),
}
