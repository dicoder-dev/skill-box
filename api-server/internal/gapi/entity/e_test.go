package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
	"time"
)

const tableNameTest = "test"

type TestEnum struct {
	ID uint `json:"id"`
	//... other fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at" `
}

var _ typ.IEntity = (*TestEnum)(nil) // U实体必须实现接口GenConfig

func (TestEnum) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameTest,
	}
}

func (TestEnum) TableName() string {
	return tableNameTest
}
