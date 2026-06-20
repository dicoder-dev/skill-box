package dbs

import (
	"ginp-api/configs"
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

var useDbType = DbTypePgsql

func init() {
	dbType := configs.SystemDbType()
	switch dbType {
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
