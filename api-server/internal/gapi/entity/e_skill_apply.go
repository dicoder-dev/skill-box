package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameSkillApply = "skill_applies"

// SkillApply 见 docs/project/需求规划.md 第 6 节。
type SkillApply struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    SkillID        uint       `gorm:"column:skill_id;index;comment:所属 skill" json:"skill_id,omitempty"`
    Tool           string     `gorm:"type:varchar(32);column:tool;index;comment:目标工具 ID" json:"tool,omitempty"`
    Status         string     `gorm:"type:varchar(16);column:status;index;comment:状态" json:"status,omitempty"`
    TargetPath     string     `gorm:"type:varchar(512);column:target_path;comment:落盘路径" json:"target_path,omitempty"`
    PreSnapshot    string     `gorm:"type:text;column:pre_snapshot;comment:apply 前目标目录状态" json:"pre_snapshot,omitempty"`
    AppliedAt      time.Time  `json:"applied_at,omitempty"`
    RolledBackAt   *time.Time `gorm:"column:rolled_back_at;comment:回滚时间" json:"rolled_back_at,omitempty"`
}

var _ typ.IEntity = (*SkillApply)(nil)

func (SkillApply) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillApply,
	}
}

func (SkillApply) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillApply) TableName() string {
	return tableNameSkillApply
}
