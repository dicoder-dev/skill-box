package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillTestRun = "skill_test_runs"

// SkillTestRun 一次 skill 测试的运行记录。
//
// 触发一次 run 会顺序执行:static lint -> script execute -> ai walkthrough。
// 每一步的结果落 skill_test_results 表(本表只存汇总状态 + skill key 引用)。
// 状态约定:
//   - passed  全部 check 通过
//   - failed  至少一个 check failed
//   - errored 至少一个 check 抛错(脚本崩溃 / AI 异常)
//   - skipped 没有可跑的内容(无 lint 项 / 无 test.sh / 无 AI provider)
type SkillTestRun struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SkillID    uint      `gorm:"column:skill_id;index;comment:所属 skill" json:"skill_id,omitempty"`
	Scope      string    `gorm:"type:varchar(16);column:scope;comment:作用域" json:"scope,omitempty"`
	ProjectID  uint      `gorm:"column:project_id;index;comment:项目ID" json:"project_id,omitempty"`
	Name       string    `gorm:"type:varchar(128);column:name;index;comment:skill 名" json:"name,omitempty"`
	Version    string    `gorm:"type:varchar(32);column:version;comment:版本" json:"version,omitempty"`
	Status     string    `gorm:"type:varchar(16);column:status;index;comment:passed/failed/errored/skipped" json:"status,omitempty"`
	Trigger    string    `gorm:"type:varchar(16);column:trigger;index;comment:manual/auto" json:"trigger,omitempty"`
	Summary    string    `gorm:"type:text;column:summary;comment:一句话总结" json:"summary,omitempty"`
	StartedAt  time.Time `gorm:"column:started_at;index;comment:开始时间" json:"started_at,omitempty"`
	FinishedAt time.Time `gorm:"column:finished_at;comment:完成时间" json:"finished_at,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
}

var _ typ.IEntity = (*SkillTestRun)(nil)

func (SkillTestRun) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillTestRun,
	}
}

func (SkillTestRun) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillTestRun) TableName() string {
	return tableNameSkillTestRun
}
