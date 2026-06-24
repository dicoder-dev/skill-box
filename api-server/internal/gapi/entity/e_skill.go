package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameSkill = "skills"

// Skill 见 docs/project/需求规划.md 第 6 节。
//
// 2026-06-24 改造:此 entity 已弃用,源数据走 ~/.skill-box/skills/<name>/SKILL.md;
// 保留 struct 是为了 mskill 包编译不挂,以及旧迁移期把已有数据回填到 store 后再删表。
// 不再加入 AutoMigrate 列表,新代码禁止再读/写这张表。
type Skill struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	Scope        string    `gorm:"type:varchar(16);column:scope;index;comment:作用域" json:"scope,omitempty"`
	ProjectID    uint      `gorm:"column:project_id;index;comment:项目ID" json:"project_id,omitempty"`
	Name         string    `gorm:"type:varchar(128);column:name;comment:canonical 名;uniqueIndex:idx_skill_scope_proj_name_ver;uniqueIndex:idx_skill_scope_proj_name" json:"name,omitempty"`
	Version      string    `gorm:"type:varchar(32);column:version;comment:semver;uniqueIndex:idx_skill_scope_proj_name_ver" json:"version,omitempty"`
	Source       string    `gorm:"type:varchar(16);column:source;index;comment:local/imported/market" json:"source,omitempty"`
	SourceRef    string    `gorm:"type:varchar(256);column:source_ref;comment:来源引用" json:"source_ref,omitempty"`
	ManifestJSON string    `gorm:"type:text;column:manifest_json;comment:完整 manifest JSON" json:"manifest_json,omitempty"`
	CurrentTagID uint      `gorm:"column:current_tag_id;comment:最新手动 tag" json:"current_tag_id,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*Skill)(nil)

func (Skill) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSkill,
	}
}

func (Skill) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (Skill) TableName() string {
	return tableNameSkill
}
