package bootstrap

import (
	"fmt"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/skilladapter/toolspecs"
	"ginp-api/internal/toolseed"
)

// StartDB 根据配置中的 db.use_type 初始化数据库,完成表结构自动迁移。
//
// 注意:调用前必须已经 SetDbType() 过(useDbType 由 bootstrap 入口同步),
// 否则 GetWriteDb 内部 switch 会拿到错误的 DB 实现。
//
// 2026-06-30 二改:启动流程 = AutoMigrate → EnsureSeeded → ReloadAllFromDB。
// - AutoMigrate 创建/更新 e_tool + e_tool_path 表
// - EnsureSeeded 把 9 个默认工具 seed 进 DB(全新 DB 才 seed)
// - ReloadAllFromDB 从 DB 拉一次 enabled 工具,注册到 skilladapter.Registry
//
// 顺序关键:先 seed 完再 reload,reload 时才有数据可拉。
func StartDB() {
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

	if dbs.GetWriteDb() == nil {
		return
	}

	// 1) AutoMigrate
	if err := dbs.GetWriteDb().AutoMigrate(EntityAutoMigrateList...); err != nil {
		fmt.Println("迁移表结构失败:" + err.Error())
		panic(err)
	}

	// 2) Seed 9 个默认工具(全新 DB 才会真 seed)
	if err := toolseed.EnsureSeeded(dbs.GetWriteDb(), dbs.GetReadDb()); err != nil {
		fmt.Println("seed 工具元数据失败:" + err.Error())
		panic(err)
	}

	// 3) 从 DB 加载工具到 skilladapter.Registry
	if err := toolspecs.ReloadAllFromDB(dbs.GetWriteDb()); err != nil {
		fmt.Println("加载工具元数据到 registry 失败:" + err.Error())
		panic(err)
	}
}
