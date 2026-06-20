package dbs

import (
	"ginp-api/internal/db/mysql"
	"ginp-api/internal/db/pgsql"
	"ginp-api/internal/db/sqlite"

	"gorm.io/gorm"
)

const (
	DbTypeMysql  = "mysql"
	DbTypePgsql  = "pgsql"
	DbTypeSqlite = "sqlite"
)

var (
	// useDbType 由 SetDbType() 在 main 入口 cfg 加载完后显式设置。
	// 注意:历史代码曾在 init() 里用 configs.SystemDbType() 推断,但 init 阶段 cfg
	// 还没加载,推断结果不可靠;现在改为 main 显式调用,语义更清晰。
	useDbType = DbTypeMysql
)

// SetDbType 显式设定数据库类型(供 main 入口在 cfg 加载完后调用)。
func SetDbType(t string) {
	switch t {
	case "mysql":
		useDbType = DbTypeMysql
	case "pgsql", "postgresql":
		useDbType = DbTypePgsql
	case "sqlite":
		useDbType = DbTypeSqlite
	default:
		useDbType = DbTypeMysql
	}
}

var (
	DbRead  *gorm.DB
	DbWrite *gorm.DB
)

func GetReadDb() *gorm.DB {
	switch useDbType {
	case DbTypeMysql:
		return mysql.GetReadDb()
	case DbTypePgsql:
		return pgsql.GetReadDb()
	case DbTypeSqlite:
		db, err := sqlite.GetReadDb()
		if err != nil {
			panic(err)
		}
		return db
	default:
		return mysql.GetReadDb()
	}
}

func GetWriteDb() *gorm.DB {
	switch useDbType {
	case DbTypeMysql:
		return mysql.GetWriteDb()
	case DbTypePgsql:
		return pgsql.GetWriteDb()
	case DbTypeSqlite:
		db, err := sqlite.GetWriteDb()
		if err != nil {
			panic(err)
		}
		return db
	default:
		return mysql.GetWriteDb()
	}
}
