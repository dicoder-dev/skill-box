package configs

import "ginp-api/pkg/cfg"

// Db 全局配置变量
var Db = new(DbConfig)

// DbConfig 数据库配置
type DbConfig struct {
	UseType string `default:"mysql"` // 适用的数据库类型: mysql,pgsql,sqlite
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