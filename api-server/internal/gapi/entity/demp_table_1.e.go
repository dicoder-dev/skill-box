package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
	"time"
)

const tableNameDemoTable1 = "demo_table1"

type DemoTable1 struct {
	ID int `json:"id"`
	//... other fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at" `
}

var _ typ.IEntity = (*DemoTable)(nil) // U实体必须实现接口GenConfig

func (DemoTable1) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameDemoTable,
	}
}

func (DemoTable1) TableName() string {
	return tableNameDemoTable
}
