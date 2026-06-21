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
	if !filepath.IsAbs(dbPath) && configs.System.RunMode == "desktop" {
		if abs := sharefunc.DbPath(); abs != "" {
			dbPath = abs
		}
	}
	sqlite.InitdDb(dbPath)
}
