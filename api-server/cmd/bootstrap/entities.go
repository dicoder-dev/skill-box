package bootstrap

import "ginp-api/internal/gapi/entity"

// EntityAutoMigrateList 自动迁移的实体列表。
// 业务模块如果新增 entity,应在这里登记;或者在调用方业务侧维护自己的
// 列表 + 调 dbs.GetWriteDb().AutoMigrate(...)。
//
// Skill Box 表(见 docs/project/需求规划.md 第 6 节):
// project / skill_file / skill_tag / skill_file_snapshot /
// skill_apply / audit_log / ai_provider / market_source / market_skill /
// onboarding_state / setting
//
// 2026-06-24 改造:skill 表(对应 entity.Skill)弃用,源数据走 ~/.skill-box/skills/<name>/SKILL.md;
// 下游表(skill_file / skill_apply / skill_tag / skill_file_snapshot / skill_test_*)
// 保留,关联键从 skill_id(uint)改为 (scope, project_id, name) 复合键。
var EntityAutoMigrateList = []any{
	new(entity.User),

	new(entity.Project),
	new(entity.SkillFile),
	new(entity.SkillTag),
	new(entity.SkillFileSnapshot),
	new(entity.SkillApply),
	new(entity.SkillTestRun),
	new(entity.SkillTestResult),
	new(entity.AuditLog),
	new(entity.AIProvider),
	new(entity.MarketSource),
	new(entity.MarketSkill),
	new(entity.OnboardingState),
	new(entity.Setting),
}

// EntityGenerationList 需要自动生成的实体(代码生成器使用,运行期不参与)。
var EntityGenerationList = []any{
	new(entity.User),
}
