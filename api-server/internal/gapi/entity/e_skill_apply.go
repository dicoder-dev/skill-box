package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillApply = "skill_applies"

// SkillApply 一次 apply 的落库记录。
// 2026-06-24 改造:用 (scope, project_id, name) 关联 skill,不再依赖 entity.Skill 的数字 ID。
// skill_id 字段保留为 deprecated,用于过渡期回溯旧数据。
// 2026-07-02 增:ApplyMode 记录本次 apply 落盘用的模式 (copy / symlink),用于
// 后续模式切换时只迁"对得上"的行;也方便前端在 SkillsView 区分展示。值为
// "copy"(默认)/"symlink";老数据为空串,迁时按"等价 copy"处理。
type SkillApply struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SkillID      uint       `gorm:"column:skill_id;index;comment:所属 skill(已弃用,用 scope+name 定位)" json:"skill_id,omitempty"`
	Scope        string     `gorm:"type:varchar(16);column:scope;index;comment:作用域;uniqueIndex:idx_skill_apply_lookup" json:"scope,omitempty"`
	ProjectID    uint       `gorm:"column:project_id;comment:项目ID;uniqueIndex:idx_skill_apply_lookup" json:"project_id,omitempty"`
	Name         string     `gorm:"type:varchar(128);column:name;comment:skill 名;uniqueIndex:idx_skill_apply_lookup" json:"name,omitempty"`
	Tool         string     `gorm:"type:varchar(32);column:tool;index;comment:目标工具 ID" json:"tool,omitempty"`
	Status       string     `gorm:"type:varchar(16);column:status;index;comment:状态" json:"status,omitempty"`
	ApplyMode    string     `gorm:"type:varchar(16);column:apply_mode;index;comment:apply 模式(copy/symlink)" json:"apply_mode,omitempty"`
	TargetPath   string     `gorm:"type:varchar(512);column:target_path;comment:落盘路径" json:"target_path,omitempty"`
	PreSnapshot  string     `gorm:"type:text;column:pre_snapshot;comment:apply 前目标目录状态" json:"pre_snapshot,omitempty"`
	AppliedAt    time.Time  `json:"applied_at,omitempty"`
	RolledBackAt *time.Time `gorm:"column:rolled_back_at;comment:回滚时间" json:"rolled_back_at,omitempty"`
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
