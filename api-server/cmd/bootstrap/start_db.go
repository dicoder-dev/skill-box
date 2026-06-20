package bootstrap

import (
	"fmt"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
)

// StartDB 根据配置中的 db.use_type 初始化数据库,完成表结构自动迁移。
//
// 注意:调用前必须已经 SetDbType() 过(useDbType 由 bootstrap 入口同步),
// 否则 GetWriteDb 内部 switch 会拿到错误的 DB 实现。
func StartDB() {
	dbType := configs.SystemDbType()

	switch dbType {
	case "pgsql", "postgresql":
		dbs.InitDb(dbs.DbTypePgsql)
	case "mysql":
		dbs.InitDb(dbs.DbTypeMysql)
	case "sqlite":
		dbs.InitDb(dbs.DbTypeSqlite)
	default:
		// 默认使用 MySQL
		dbs.InitDb(dbs.DbTypeMysql)
	}

	//迁移表
	if dbs.GetWriteDb() != nil {
		//自动迁移表结构
		err := dbs.GetWriteDb().AutoMigrate(EntityAutoMigrateList...)
		if err != nil {
			fmt.Println("迁移表结构失败" + err.Error())
			panic(err)
		}
	}
}
