package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillTag = "skill_tags"

// SkillTag 一次手工 tag 记录。
// 2026-06-24 改造:用 (scope, project_id, name) 关联 skill,不再依赖 entity.Skill 的数字 ID。
type SkillTag struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SkillID    uint      `gorm:"column:skill_id;index;comment:所属 skill(已弃用,用 scope+name 定位)" json:"skill_id,omitempty"`
	Scope      string    `gorm:"type:varchar(16);column:scope;comment:作用域;uniqueIndex:idx_skill_tag_unique" json:"scope,omitempty"`
	ProjectID  uint      `gorm:"column:project_id;comment:项目ID;uniqueIndex:idx_skill_tag_unique" json:"project_id,omitempty"`
	Name       string    `gorm:"type:varchar(128);column:name;comment:skill 名;uniqueIndex:idx_skill_tag_unique" json:"name,omitempty"`
	Tag        string    `gorm:"type:varchar(64);column:tag;comment:tag 名;uniqueIndex:idx_skill_tag_unique" json:"tag,omitempty"`
	Message    string    `gorm:"type:varchar(256);column:message;comment:描述" json:"message,omitempty"`
	IsImplicit bool      `gorm:"column:is_implicit;comment:true = 预回滚隐式 tag" json:"is_implicit,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
}

var _ typ.IEntity = (*SkillTag)(nil)

func (SkillTag) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillTag,
	}
}

func (SkillTag) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillTag) TableName() string {
	return tableNameSkillTag
}
