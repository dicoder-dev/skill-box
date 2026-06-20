package configs

import "ginp-api/pkg/cfg"

// 适用的数据库类型: mysql,pgsql,sqlite
const ConfigKeySystemDbType = "db.use_type"

const defaultSystemDbType = "mysql"

// -------------------------Mysql-------------------------
const ConfigKeyMysqlPort = "db.mysql.port"
const ConfigKeyMysqlIp = "db.mysql.ip"
const ConfigKeyMysqlUser = "db.mysql.user"
const ConfigKeyMysqlPwd = "db.mysql.pwd"
const ConfigKeyMysqlDb = "db.mysql.dbname"

func init() { // 设置默认值
	cfg.SetDefault(ConfigKeySystemDbType, defaultSystemDbType)
	cfg.SetDefault(ConfigKeyMysqlIp, "127.0.0.1")
	cfg.SetDefault(ConfigKeyMysqlPort, "3306")
	cfg.SetDefault(ConfigKeyMysqlUser, "root")
	cfg.SetDefault(ConfigKeyMysqlDb, "")
	cfg.SetDefault(ConfigKeyMysqlPwd, "123456")
}
func MysqlIp() string {
	return cfg.GetString(ConfigKeyMysqlIp)
}
func MysqlPort() string {
	return cfg.GetString(ConfigKeyMysqlPort)
}

func MysqlUser() string {
	return cfg.GetString(ConfigKeyMysqlUser)
}
func MysqlPwd() string {
	return cfg.GetString(ConfigKeyMysqlPwd)
}
func MysqlDb() string {
	return cfg.GetString(ConfigKeyMysqlDb)
}

func SystemDbType() string {
	return cfg.GetString(ConfigKeySystemDbType)
}

// -------------------------Sqlite-------------------------
// sqlite数据库文件路径
const ConfigKeySqliteDbPath = "db.sqlite.db_path"

// 设置默认值
func init() {
	cfg.SetDefault(ConfigKeySqliteDbPath, "data.db")
}

func SqliteDbPath() string {
	return cfg.GetString(ConfigKeySqliteDbPath)
}

// -------------------------Pgsql-------------------------
const ConfigKeyPgsqlPort = "db.pgsql.port"
const ConfigKeyPgsqlIp = "db.pgsql.ip"
const ConfigKeyPgsqlUser = "db.pgsql.user"
const ConfigKeyPgsqlPwd = "db.pgsql.pwd"
const ConfigKeyPgsqlDb = "db.pgsql.dbname"

func init() {
	cfg.SetDefault(ConfigKeyPgsqlIp, "127.0.0.1")
	cfg.SetDefault(ConfigKeyPgsqlPort, "5432")
	cfg.SetDefault(ConfigKeyPgsqlUser, "root")
	cfg.SetDefault(ConfigKeyPgsqlDb, "")
	cfg.SetDefault(ConfigKeyPgsqlPwd, "123456")
}
func PgsqlIp() string {
	return cfg.GetString(ConfigKeyPgsqlIp)
}
func PgsqlPort() string {
	return cfg.GetString(ConfigKeyPgsqlPort)
}
func PgsqlUser() string {
	return cfg.GetString(ConfigKeyPgsqlUser)
}
func PgsqlPwd() string {
	return cfg.GetString(ConfigKeyPgsqlPwd)
}
func PgsqlDb() string {
	return cfg.GetString(ConfigKeyPgsqlDb)
}
