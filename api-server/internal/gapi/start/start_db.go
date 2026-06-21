package start

import (
	"fmt"
	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
)

func startDB() {
	// 根据配置文件中的 db.use_type 决定使用哪种数据库
	dbType := configs.Db.UseType

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
