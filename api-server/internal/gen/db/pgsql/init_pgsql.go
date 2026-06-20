package pgsql

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// InitDb 初始化PostgreSQL数据库连接
// 连接方式：IP:端口形式
// 示例：InitDb("192.168.1.100", "5432", "user", "db", "pass")
func InitDb(ip, port, userName, dbName, dbPwd string) {
	// 验证必要参数
	if ip == "" || port == "" || userName == "" || dbName == "" || dbPwd == "" {
		panic("PostgreSQL连接参数不能为空：ip, port, userName, dbName, dbPwd 都必须提供")
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // Don't include params in the SQL log
			Colorful:                  true,        // Disable color
		},
	)

	// 生成DSN连接字符串
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=Asia/Shanghai",
		ip, userName, dbPwd, dbName, port)

	fmt.Printf("PostgreSQL连接配置：主机=%v, 端口=%v, 数据库=%v\n", ip, port, dbName)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger, //日志参数
	})

	if err != nil {
		fmt.Println("PostgreSQL连接失败: " + err.Error())
		panic(err)
	}

	fmt.Println("PostgreSQL数据库连接成功！")
}

func GetReadDb() *gorm.DB {
	//返回数据库实例的副本
	copyDb := *db
	return &copyDb
}

// GetDbInstance 获取gorm示例的副本
func GetWriteDb() *gorm.DB {
	//返回数据库实例的副本
	copyDb := *db
	return &copyDb
}
