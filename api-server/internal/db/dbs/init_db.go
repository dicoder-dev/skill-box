package dbs

import (
	"path/filepath"

	"ginp-api/configs"
	"ginp-api/internal/db/mysql"
	"ginp-api/internal/db/pgsql"
	"ginp-api/internal/db/sqlite"
	sharefunc "ginp-api/share/func"
)

func InitDb(dbType string) {
	switch dbType {
	case DbTypeMysql:
		initMysql()
	case DbTypePgsql:
		initPgsql()
	case DbTypeSqlite:
		initSqlite()
	default:
		panic("db type not support")
	}

}

func initMysql() {
	mysql.InitDb(
		configs.Db.Mysql.Ip,
		configs.Db.Mysql.Port,
		configs.Db.Mysql.User,
		configs.Db.Mysql.Db,
		configs.Db.Mysql.Pwd,
	)
}

func initPgsql() {
	pgsql.InitDb(
		configs.Db.Pgsql.Ip,
		configs.Db.Pgsql.Port,
		configs.Db.Pgsql.User,
		configs.Db.Pgsql.Db,
		configs.Db.Pgsql.Pwd,
	)
}

func initSqlite() {
	dbPath := configs.Db.Sqlite.DbPath
	// 桌面端需要把 sqlite 数据文件重定向到 ~/.<AppName>/data.db,
	// IsDesktop() 由 main 入口在 cfg 加载后通过 dbs.SetRunMode 注入,
	// 替代已删除的 configs.System.RunMode 字段,避免双源歧义。
	if !filepath.IsAbs(dbPath) && IsDesktop() {
		if abs := sharefunc.DbPath(); abs != "" {
			dbPath = abs
		}
	}
	sqlite.InitdDb(dbPath)
}
