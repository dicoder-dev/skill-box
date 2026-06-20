# GINP-API 命令行工具

## 简介

GINP-API 是一个基于 Gin 框架的 API 开发工具，提供了代码生成、实体管理等功能。

## 安装

1. 克隆仓库
```bash
git clone https://github.com/dicoder-cn/ginp-api.git
cd ginp-api
```

2. 编译命令行工具
```bash
./scripts/build_gapi.sh
```

3. 添加到 PATH（可选）
```bash
export PATH="/path/to/ginp-api/build:$PATH"
```

## 使用方法

### 生成swagger文档
cd ./cmd/gencode && go run main.go swagger

### 查看版本
```bash
gapi -v
```

### 创建实体并生成 CRUD 代码
```bash
# 交互式方式
gapi gen entity

# 直接指定实体名称
gapi gen entity -c UserGroup
```

### 生成实体字段常量
```bash
gapi gen const
```

### 新增 API 接口
```bash
# 交互式方式
gapi gen api

# 直接指定 API 名称和目录
gapi gen api -a GetUserInfo -d user/cuser
```

## 命令说明

### 根命令
- `gapi`: 显示帮助信息
- `gapi -v`: 显示版本信息

### 创建 UserGroup 实体
```bash
gapi gen entity -c UserGroup -p user
```

### 新增一个接口
```bash
# -a API名称,大驼峰命名法 -d指定api所在文件夹,
# 存放于controller/user/cuser文件夹，命名为get_user_info.go 采用一个api接口一个文件的方式
gapi gen api -d user/cuser -a GetUserInfo 
```

### 生成实体字段常量  
```bash 
gapi gen const 
```

### 现有实体生成crud
```bash
# 指定多个实体 -p指定父级目录 -e 指定实体名称列表
gapi gen crud -e tableNameDemoTable1,tableNameDemoTable2 -p demo
# 指定一个实体 -p指定父级目录 -e 指定实体名称
gapi gen crud -e tableNameDemoTable1 -p demo
```

### 删除现有实体的crud文件
```bash
# 删除多个实体的CRUD文件 -p指定父级目录 -e 指定实体名称列表
gapi gen rm crud -e tableNameDemoTable1,tableNameDemoTable2 -p demo
# 删除一个实体的CRUD文件 -p指定父级目录 -e 指定实体名称
gapi gen rm crud -e tableNameDemoTable1 -p demo
# 交互式删除（不指定参数时进入交互模式）
gapi gen rm crud
```

## 贡献

欢迎提交 Pull Request 或提出 Issue。

## 许可证

[MIT](LICENSE)