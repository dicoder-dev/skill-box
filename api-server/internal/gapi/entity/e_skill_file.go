package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillFile = "skill_files"

// SkillFile skill 附属文件(用于打 tag / diff 等场景的快照存储)。
// 2026-06-24 改造:用 (scope, project_id, name) 关联 skill。
type SkillFile struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SkillID     uint      `gorm:"column:skill_id;index;comment:所属 skill(已弃用)" json:"skill_id,omitempty"`
	Scope       string    `gorm:"type:varchar(16);column:scope;comment:作用域;uniqueIndex:idx_skill_file_unique" json:"scope,omitempty"`
	ProjectID   uint      `gorm:"column:project_id;comment:项目ID;uniqueIndex:idx_skill_file_unique" json:"project_id,omitempty"`
	Name        string    `gorm:"type:varchar(128);column:name;comment:skill 名;uniqueIndex:idx_skill_file_unique" json:"name,omitempty"`
	Path        string    `gorm:"type:varchar(512);column:path;comment:相对路径;uniqueIndex:idx_skill_file_unique" json:"path,omitempty"`
	Content     string    `gorm:"type:longtext;column:content;comment:文件内容" json:"content,omitempty"`
	ContentHash string    `gorm:"type:varchar(64);column:content_hash;comment:SHA-256" json:"content_hash,omitempty"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
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
