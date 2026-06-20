package start

import "ginp-api/internal/gapi/entity"

const ConfigFile = "../configs/configs.yaml"

// EntityAutoMigrateList 自动迁移的实体列表
var EntityAutoMigrateList = []any{
	new(entity.User),
	new(entity.DemoTable),
}

// EntityGenerationList 需要自动生成的实体
// 开始公用EntityAutoMigrateList，但是考虑到隔离，还是分开了，可以按需将生成实体写入
var EntityGenerationList = []any{
	new(entity.User),
	new(entity.DemoTable),
}
