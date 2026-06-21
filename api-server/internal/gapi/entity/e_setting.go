package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameSetting = "settings"

// Setting 见 docs/project/需求规划.md 第 6 节。
type Setting struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Key            string     `gorm:"type:varchar(64);column:key;comment:键名;uniqueIndex:idx_setting_key" json:"key,omitempty"`
    Value          string     `gorm:"type:text;column:value;comment:值" json:"value,omitempty"`
    UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*Setting)(nil)

func (Setting) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameSetting,
	}
}

func (Setting) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (Setting) TableName() string {
	return tableNameSetting
}
