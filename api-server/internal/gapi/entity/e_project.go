package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameProject = "projects"

// Project 见 docs/project/需求规划.md 第 6 节。
type Project struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Name           string     `gorm:"type:varchar(64);column:name;comment:显示名" json:"name,omitempty"`
    Alias          string     `gorm:"type:varchar(64);column:alias;comment:唯一别名;uniqueIndex:idx_project_alias" json:"alias,omitempty"`
    RootPath       string     `gorm:"type:varchar(512);column:root_path;comment:项目根;uniqueIndex:idx_project_root" json:"root_path,omitempty"`
    Description    string     `gorm:"type:varchar(512);column:description;comment:描述" json:"description,omitempty"`
    CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
    UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*Project)(nil)

func (Project) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameProject,
	}
}

func (Project) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (Project) TableName() string {
	return tableNameProject
}
