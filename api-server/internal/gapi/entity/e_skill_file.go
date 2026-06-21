package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameSkillFile = "skill_files"

// SkillFile 见 docs/project/需求规划.md 第 6 节。
type SkillFile struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    SkillID        uint       `gorm:"column:skill_id;index;comment:所属 skill;uniqueIndex:idx_skill_file_unique" json:"skill_id,omitempty"`
    Path           string     `gorm:"type:varchar(512);column:path;comment:相对路径;uniqueIndex:idx_skill_file_unique" json:"path,omitempty"`
    Content        string     `gorm:"type:longtext;column:content;comment:文件内容" json:"content,omitempty"`
    ContentHash    string     `gorm:"type:varchar(64);column:content_hash;comment:SHA-256" json:"content_hash,omitempty"`
    UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*SkillFile)(nil)

func (SkillFile) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkillFile,
	}
}

func (SkillFile) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (SkillFile) TableName() string {
	return tableNameSkillFile
}
