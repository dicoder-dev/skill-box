package entity

import (
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameAIProvider = "ai_providers"

// AIProvider 见 docs/project/需求规划.md 第 6 节。
type AIProvider struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Name           string     `gorm:"type:varchar(64);column:name;comment:显示名;uniqueIndex:idx_ai_provider_name" json:"name,omitempty"`
    Kind           string     `gorm:"type:varchar(16);column:kind;comment:openai/anthropic/deepseek/openai_compat" json:"kind,omitempty"`
    BaseURL        string     `gorm:"type:varchar(256);column:base_url;comment:base URL" json:"base_url,omitempty"`
    APIKeyRef      string     `gorm:"type:varchar(128);column:api_key_ref;comment:OS keychain 引用" json:"api_key_ref,omitempty"`
    Model          string     `gorm:"type:varchar(64);column:model;comment:默认模型" json:"model,omitempty"`
    Priority       int        `gorm:"column:priority;index;comment:数字越小越优先" json:"priority,omitempty"`
    Enabled        bool       `gorm:"column:enabled;comment:是否启用" json:"enabled,omitempty"`
}

var _ typ.IEntity = (*AIProvider)(nil)

func (AIProvider) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameAIProvider,
	}
}

func (AIProvider) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (AIProvider) TableName() string {
	return tableNameAIProvider
}
