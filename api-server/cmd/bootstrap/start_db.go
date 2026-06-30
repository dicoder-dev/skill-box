package bootstrap

import (
	"fmt"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/toolseed"
)

// StartDB 根据配置中的 db.use_type 初始化数据库,完成表结构自动迁移。
//
// 注意:调用前必须已经 SetDbType() 过(useDbType 由 bootstrap 入口同步),
// 否则 GetWriteDb 内部 switch 会拿到错误的 DB 实现。
//
// 2026-06-30 二改:AutoMigrate 之后追加 EnsureSeeded,全新 DB 会自动
// 写入 9 个默认 AI 编程工具到 e_tool + e_tool_path;已初始化过的 DB
// 跳过(查 e_tool.Count()==0 判定)。
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

	//迁移表
	if dbs.GetWriteDb() != nil {
		//自动迁移表结构
		err := dbs.GetWriteDb().AutoMigrate(EntityAutoMigrateList...)
		if err != nil {
			fmt.Println("迁移表结构失败" + err.Error())
			panic(err)
		}
	}

	// 2026-06-30 增:启动期 seed 9 个默认 AI 编程工具
	// 全新 DB(Count==0)才 seed;已有任何 tool 行(不论系统 / 用户)即跳过。
	if dbs.GetWriteDb() != nil {
		if err := toolseed.EnsureSeeded(dbs.GetWriteDb(), dbs.GetReadDb()); err != nil {
			fmt.Println("seed 工具元数据失败:" + err.Error())
			panic(err)
		}
	}
}
