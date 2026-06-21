package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameMarketSkill = "market_skills"

// MarketSkill 三方市场 skill 缓存。
//
// 由 internal/skillmarket 刷写,smarket 列表接口直接查这张表(避免每次都打三方)。
// 一行 = 一个三方源里的一个 skill 版本,跟 source_name + remote_id 唯一。
type MarketSkill struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	SourceID    uint      `gorm:"column:source_id;index;comment:market_sources.id" json:"source_id,omitempty"`
	SourceName  string    `gorm:"type:varchar(64);column:source_name;index;comment:冗余,便于按源筛选" json:"source_name,omitempty"`
	RemoteID    string    `gorm:"type:varchar(256);column:remote_id;index;comment:三方源里的唯一 ID" json:"remote_id,omitempty"`
	Name        string    `gorm:"type:varchar(128);column:name;index;comment:display name" json:"name,omitempty"`
	Version     string    `gorm:"type:varchar(32);column:version;comment:三方源的版本号" json:"version,omitempty"`
	Description string    `gorm:"type:text;column:description;comment:描述" json:"description,omitempty"`
	Author      string    `gorm:"type:varchar(64);column:author" json:"author,omitempty"`
	License     string    `gorm:"type:varchar(32);column:license" json:"license,omitempty"`
	Tags        string    `gorm:"type:varchar(256);column:tags;comment:逗号分隔" json:"tags,omitempty"`
	InstallRef  string    `gorm:"type:varchar(512);column:install_ref;comment:下载/安装引用 URL" json:"install_ref,omitempty"`
	DetailURL   string    `gorm:"type:varchar(512);column:detail_url;comment:三方详情页 URL" json:"detail_url,omitempty"`
	ExtraJSON   string    `gorm:"type:text;column:extra_json;comment:其它元数据 JSON" json:"extra_json,omitempty"`
	FetchedAt   time.Time `gorm:"column:fetched_at;comment:最近一次拉取时间" json:"fetched_at,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*MarketSkill)(nil)

func (MarketSkill) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameMarketSkill,
	}
}

func (MarketSkill) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (MarketSkill) TableName() string {
	return tableNameMarketSkill
}
