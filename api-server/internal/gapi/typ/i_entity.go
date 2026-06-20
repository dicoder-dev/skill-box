package typ

import "ginp-api/internal/gen"

type IEntity interface {
	// GetEntityName 获取实体名称
	GenConfig() *gen.EntityConfig
}
