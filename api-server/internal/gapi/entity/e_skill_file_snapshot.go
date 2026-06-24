package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkillFileSnapshot = "skill_file_snapshots"

// SkillFileSnapshot 单个 tag 时刻的某个文件快照(由 ctag 写入,rollback 时回写)。
// 2026-06-24 改造:所属 skill 由 (scope, project_id, name) 定位。
type SkillFileSnapshot struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SkillTagID uint  `gorm:"column:skill_tag_id;index;comment:所属 tag" json:"skill_tag_id,omitempty"`
	Scope     string `gorm:"type:varchar(16);column:scope;comment:作用域" json:"scope,omitempty"`
	ProjectID uint   `gorm:"column:project_id;comment:项目ID" json:"project_id,omitempty"`
	Name      string `gorm:"type:varchar(128);column:name;comment:skill 名" json:"name,omitempty"`
	Path      string `gorm:"type:varchar(512);column:path;comment:相对路径" json:"path,omitempty"`
	Content   string `gorm:"type:longtext;column:content;comment:文件内容" json:"content,omitempty"`
	ContentHash string `gorm:"type:varchar(64);column:content_hash;comment:SHA-256" json:"content_hash,omitempty"`
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
