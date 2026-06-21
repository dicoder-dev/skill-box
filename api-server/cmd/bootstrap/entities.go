package bootstrap

import "ginp-api/internal/gapi/entity"

// EntityAutoMigrateList 自动迁移的实体列表。
// 业务模块如果新增 entity,应在这里登记;或者在调用方业务侧维护自己的
// 列表 + 调 dbs.GetWriteDb().AutoMigrate(...)。
var EntityAutoMigrateList = []any{
	new(entity.User),
}

// EntityGenerationList 需要自动生成的实体(代码生成器使用,运行期不参与)。
var EntityGenerationList = []any{
	new(entity.User),
}
