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

	// useRunMode 由 SetRunMode() 在 main 入口显式注入,标识当前部署形态。
	// 用途:sqlite 数据路径在桌面端需要重定向到 ~/.<AppName>/data.db;
	// 之前从 configs.System.RunMode 读,但该字段已删除,改成从启动命令注入。
	useRunMode = "web"
)

// SetRunMode 显式设定当前部署形态("web" / "desktop"),由 main 在 Boot 阶段
// 从 BootOptions.RunMode 透传过来。dbs 包内部 sqlite 路径解析依赖这个值。
func SetRunMode(m string) {
	if m == "desktop" {
		useRunMode = "desktop"
	} else {
		useRunMode = "web"
	}
}

// IsDesktop 报告当前是否桌面端,供其他包(sharefunc 等)判断是否需要
// 把数据目录重定向到 ~/.<AppName>/。
func IsDesktop() bool {
	return useRunMode == "desktop"
}

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
