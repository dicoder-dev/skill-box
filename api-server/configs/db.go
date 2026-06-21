package configs

import "ginp-api/pkg/cfg"

// Db 全局配置变量
var Db = new(DbConfig)

// DbConfig 数据库配置
type DbConfig struct {
	// UseType 适用的数据库类型: mysql,pgsql,sqlite。
	// 默认 sqlite — 在没有任何显式配置时(例如桌面端首次运行且项目根
	// configs.yaml 缺失),结构体默认值会兜底为 sqlite,确保开箱即用。
	// web/cli 场景通过 -config 指向 mysql/pgsql 配置即可覆盖。
	UseType string `default:"sqlite"`
	Mysql   MysqlConfig
	Sqlite  SqliteConfig
	Pgsql   PgsqlConfig
}

// MysqlConfig MySQL配置
type MysqlConfig struct {
	Ip   string `default:"127.0.0.1"`
	Port string `default:"3306"`
	User string `default:"root"`
	Pwd  string `default:"123456"`
	Db   string `default:""`
}

// SqliteConfig Sqlite配置
type SqliteConfig struct {
	DbPath string `default:"data.db"`
}

// PgsqlConfig Pgsql配置
type PgsqlConfig struct {
	Ip   string `default:"127.0.0.1"`
	Port string `default:"5432"`
	User string `default:"root"`
	Pwd  string `default:"123456"`
	Db   string `default:""`
}

func init() {
	cfg.ParseConfigStruct(Db)
}
