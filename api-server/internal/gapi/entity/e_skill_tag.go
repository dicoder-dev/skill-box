package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameSkillTag = "skill_tags"

// SkillTag 见 docs/project/需求规划.md 第 6 节。
type SkillTag struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    SkillID        uint       `gorm:"column:skill_id;index;comment:所属 skill;uniqueIndex:idx_skill_tag_unique" json:"skill_id,omitempty"`
    Tag            string     `gorm:"type:varchar(64);column:tag;comment:tag 名;uniqueIndex:idx_skill_tag_unique" json:"tag,omitempty"`
    Message        string     `gorm:"type:varchar(256);column:message;comment:描述" json:"message,omitempty"`
    IsImplicit     bool       `gorm:"column:is_implicit;comment:true = 预回滚隐式 tag" json:"is_implicit,omitempty"`
    CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
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
