package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillTestResult = "skill_test_results"

// SkillTestResult 单个 check 的结果,挂在 SkillTestRun 下。
//
// Check 字段值:
//   - static  静态 lint
//   - script  脚本执行
//   - ai      AI 走查
//
// Status 字段值:
//   - passed  通过
//   - failed  不通过
//   - errored 抛错
//   - skipped 跳过(无内容 / 无 provider)
type SkillTestResult struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	RunID     uint      `gorm:"column:run_id;index;comment:所属 run" json:"run_id,omitempty"`
	Check     string    `gorm:"type:varchar(16);column:check;index;comment:static/script/ai" json:"check,omitempty"`
	Status    string    `gorm:"type:varchar(16);column:status;comment:passed/failed/errored/skipped" json:"status,omitempty"`
	Message   string    `gorm:"type:text;column:message;comment:一句话说明" json:"message,omitempty"`
	Detail    string    `gorm:"type:text;column:detail;comment:详细结果 JSON" json:"detail,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
}

var _ typ.IEntity = (*SkillTestResult)(nil)

func (SkillTestResult) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillTestResult,
	}
}

func (SkillTestResult) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillTestResult) TableName() string {
	return tableNameSkillTestResult
}
