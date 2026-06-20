package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
	"time"
)

const tableNameDemoTable = "demo_table"

type DemoTable struct {
	ID int `json:"id"`
	//... other fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at" `
}

var _ typ.IEntity = (*DemoTable)(nil) // U实体必须实现接口GenConfig

func (DemoTable) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameDemoTable,
	}
}

func (DemoTable) TableName() string {
	return tableNameDemoTable
}
