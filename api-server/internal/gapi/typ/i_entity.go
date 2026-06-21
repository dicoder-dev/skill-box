package typ

import "ginp-api/internal/gen"

type EntityEnumOption struct {
	FieldName string
	Options   map[string]string
}

type IEntity interface {
	// GetEntityName 获取实体名称
	GenConfig() *gen.EntityConfig
	GenEnumOptions() []EntityEnumOption
}
