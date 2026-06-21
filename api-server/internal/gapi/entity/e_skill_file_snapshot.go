package entity

import (
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameSkillFileSnapshot = "skill_file_snapshots"

// SkillFileSnapshot 见 docs/project/需求规划.md 第 6 节。
type SkillFileSnapshot struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    SkillTagID     uint       `gorm:"column:skill_tag_id;index;comment:所属 tag;uniqueIndex:idx_skill_filesnap_unique" json:"skill_tag_id,omitempty"`
    Path           string     `gorm:"type:varchar(512);column:path;comment:相对路径;uniqueIndex:idx_skill_filesnap_unique" json:"path,omitempty"`
    Content        string     `gorm:"type:longtext;column:content;comment:文件内容" json:"content,omitempty"`
    ContentHash    string     `gorm:"type:varchar(64);column:content_hash;comment:SHA-256" json:"content_hash,omitempty"`
}

var _ typ.IEntity = (*SkillFileSnapshot)(nil)

func (SkillFileSnapshot) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillFileSnapshot,
	}
}

func (SkillFileSnapshot) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillFileSnapshot) TableName() string {
	return tableNameSkillFileSnapshot
}
