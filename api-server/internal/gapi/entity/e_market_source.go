package entity

import (
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameMarketSource = "market_sources"

// MarketSource 见 docs/project/需求规划.md 第 6 节。
type MarketSource struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Name           string     `gorm:"type:varchar(64);column:name;comment:显示名;uniqueIndex:idx_market_source_name" json:"name,omitempty"`
    Type           string     `gorm:"type:varchar(16);column:type;comment:skillhub/skillssh/custom_http/git" json:"type,omitempty"`
    ConfigJSON     string     `gorm:"type:text;column:config_json;comment:适配器私有配置" json:"config_json,omitempty"`
    Enabled        bool       `gorm:"column:enabled;comment:是否启用" json:"enabled,omitempty"`
}

var _ typ.IEntity = (*MarketSource)(nil)

func (MarketSource) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameMarketSource,
	}
}

func (MarketSource) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (MarketSource) TableName() string {
	return tableNameMarketSource
}
