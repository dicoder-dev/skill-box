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

	// 检查用户权限
	checkUserPermissions(db, userName, dbName)
}

// checkUserPermissions 检查用户权限
func checkUserPermissions(db *gorm.DB, userName, dbName string) {
	// 检查用户是否有创建表的权限
	var hasCreatePermission bool
	err := db.Raw(`
		SELECT has_schema_privilege($1, 'public', 'CREATE') 
		AND has_database_privilege($1, $2, 'CREATE')
	`, userName, dbName).Scan(&hasCreatePermission).Error

	if err != nil {
		fmt.Printf("警告：无法检查用户权限: %v\n", err)
		return
	}

	if !hasCreatePermission {
		fmt.Printf("警告：用户 %s 没有在数据库 %s 的 public schema 中创建表的权限\n", userName, dbName)
		fmt.Println("请执行以下SQL命令授予权限：")
		fmt.Printf("GRANT CREATE ON SCHEMA public TO %s;\n", userName)
		fmt.Printf("GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO %s;\n", userName)
		fmt.Printf("GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO %s;\n", userName)
		fmt.Printf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO %s;\n", userName)
		fmt.Printf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO %s;\n", userName)
	} else {
		fmt.Printf("用户 %s 具有创建表的权限\n", userName)
	}
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
