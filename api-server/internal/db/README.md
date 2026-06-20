# 数据库连接模块总览

## 概述

本项目提供了三种数据库的连接模块，都基于GORM框架实现，支持读写分离和连接池管理。

## 支持的数据库

### 1. MySQL 数据库
- **模块路径**: `internal/db/mysql`
- **默认端口**: 3306
- **驱动**: `gorm.io/driver/mysql`
- **特性**: 支持utf8mb4字符集，时区设置为Local

### 2. PostgreSQL 数据库
- **模块路径**: `internal/db/pgsql`
- **默认端口**: 5432
- **驱动**: `gorm.io/driver/postgres`
- **特性**: 支持SSL配置，时区设置为Asia/Shanghai

### 3. SQLite 数据库
- **模块路径**: `internal/db/sqlite`
- **特性**: 基于文件，无需网络连接
- **用途**: 适合本地数据存储和测试

## 端口为空时的兼容处理

### 设计理念

为了提供更好的用户体验，所有网络数据库模块（MySQL和PostgreSQL）都支持端口为空的情况。当`port`参数为空字符串时，系统会自动使用该数据库的默认端口。

### 兼容处理逻辑

```go
// 端口为空时的处理逻辑
if port == "" {
    // 使用默认端口
    // MySQL: 3306
    // PostgreSQL: 5432
} else {
    // 使用指定的端口
}
```

### 使用场景

1. **域名连接**: 当使用域名连接数据库时，通常使用默认端口
   ```go
   // 使用域名，端口为空
   mysql.InitDb("db.example.com", "", "username", "database", "password")
   pgsql.InitDb("db.example.com", "", "username", "database", "password")
   ```

2. **标准端口**: 当数据库使用标准端口时，可以省略端口参数
   ```go
   // 使用标准端口
   mysql.InitDb("localhost", "", "username", "database", "password")
   pgsql.InitDb("localhost", "", "username", "database", "password")
   ```

3. **配置灵活性**: 配置文件中的端口可以为空，提高配置的灵活性

### 实现细节

#### MySQL模块
- 当`port`为空时，自动使用端口3306
- 连接字符串格式：`username:password@tcp(host:3306)/database_name?...`

#### PostgreSQL模块
- 当`port`为空时，自动使用端口5432
- 连接字符串格式：`host=host port=5432 user=username...`

## 统一的使用模式

所有数据库模块都遵循相同的使用模式：

```go
// 1. 初始化连接
database.InitDb(ip, port, userName, dbName, dbPwd)

// 2. 获取数据库实例
readDb := database.GetReadDb()
writeDb := database.GetWriteDb()
```

## 配置建议

### 环境变量配置
```bash
# MySQL
MYSQL_HOST=db.example.com
MYSQL_PORT=          # 为空时使用默认端口3306
MYSQL_USER=username
MYSQL_PASSWORD=password
MYSQL_DATABASE=database

# PostgreSQL
PGSQL_HOST=db.example.com
PGSQL_PORT=          # 为空时使用默认端口5432
PGSQL_USER=username
PGSQL_PASSWORD=password
PGSQL_DATABASE=database
```

### 配置文件示例
```yaml
database:
  mysql:
    host: db.example.com
    port: ""          # 为空时使用默认端口3306
    username: username
    password: password
    database: database
  
  pgsql:
    host: db.example.com
    port: ""          # 为空时使用默认端口5432
    username: username
    password: password
    database: database
```

## 测试

每个模块都包含完整的测试用例，包括端口为空时的兼容处理测试：

```bash
# 运行MySQL测试
cd internal/db/mysql && go test -v

# 运行PostgreSQL测试
cd internal/db/pgsql && go test -v

# 运行SQLite测试
cd internal/db/sqlite && go test -v
```

## 注意事项

1. **端口为空时**：系统会自动使用默认端口，确保数据库服务在默认端口上运行
2. **域名解析**：当使用域名时，确保DNS解析正常
3. **防火墙设置**：确保默认端口在防火墙中开放
4. **连接池配置**：根据实际需求调整连接池参数

## 依赖管理

所有模块都使用Go modules管理依赖，确保版本兼容性：

```go
require (
    gorm.io/gorm v1.25.0
    gorm.io/driver/mysql v1.5.0
    gorm.io/driver/postgres v1.5.0
    gorm.io/driver/sqlite v1.5.0
)
``` 