# GINP 代码生成工具

基于 service 层的命令行代码生成工具，用于快速生成实体 CRUD 代码、API 接口和 Swagger 文档。

## 安装

```bash
# 构建工具
#直接运行
go run cmd/gencode/main.go
```

## 功能特性

- ✅ 实体管理（列表、生成、删除）
- ✅ API 接口管理（添加、列表）
- ✅ Swagger 文档生成
- ✅ 基于 service 层，与 Web 接口保持一致
- ✅ 支持交互式和命令行参数两种模式

## 使用指南

### 1. 实体管理

#### 列出所有实体

```bash
./bin/gencode entity list
```

输出示例：
```
共找到 10 个实体:

1. SysUser
   标题: 用户
   表名: sys_user
   字段数: 15
   父级目录: system

2. SysRole
   标题: 角色
   表名: sys_role
   字段数: 8
```

#### 查看实体详细信息

```bash
# 查看实体详细信息
./bin/gencode entity info SysUser

# 以 JSON 格式输出
./bin/gencode entity info SysUser --json
```

#### 生成实体 CRUD 代码

```bash
# 为指定实体生成 CRUD 代码
./bin/gencode entity gen SysUser
```

生成的文件包括：
- Controller 层（5个CRUD接口：create, update, delete, find_by_id, search）
- Service 层
- Model 层
- Router 配置

#### 删除实体 CRUD 代码

```bash
# 删除实体的所有 CRUD 代码（需要确认）
./bin/gencode entity delete SysUser

# 强制删除，不需要确认
./bin/gencode entity delete SysUser --force
```

### 2. API 接口管理

#### 列出所有 API

```bash
./bin/gencode api list
```

输出示例：
```
共找到 50 个API:

1. /api/system/user/create
   标题: 创建用户
   方法: POST
   权限: system.user.create.api
   需要登录: true
   需要权限: true
```

#### 添加新 API

**交互式添加：**

```bash
./bin/gencode api add
```

按照提示输入：
- API名称（大驼峰，如 GetUserInfo）
- API标题（如：获取用户信息）
- API描述（可选）
- 一级目录名称（如 system 或 center）
- 二级目录名称（可选，如 cuser）
- HTTP方法（GET/POST/PUT/DELETE，默认POST）
- 是否需要登录
- 是否需要权限

**从 JSON 文件添加：**

```bash
# 先生成模板
./bin/gencode api template -o my_api.json

# 编辑 my_api.json 文件

# 从文件添加
./bin/gencode api add --file my_api.json
```

JSON 模板示例：
```json
{
  "api_name": "ExampleApi",
  "title": "示例API",
  "description": "这是一个示例API",
  "first_dir_name": "system",
  "second_dir_name": "cexample",
  "method": "POST",
  "need_login": true,
  "need_permission": true,
  "api_path": "/api/system/example/example_api",
  "params": [
    {
      "param_name": "UserId",
      "param_json_tag": "user_id",
      "param_type": "int64",
      "param_location": "body",
      "param_comment": "用户ID",
      "is_required": true
    }
  ],
  "response_fields": [
    {
      "field_name": "Success",
      "field_json_tag": "success",
      "field_type": "bool",
      "field_comment": "是否成功"
    }
  ]
}
```

### 3. Swagger 文档生成

```bash
# 使用默认配置生成
./bin/gencode swagger

# 自定义配置
./bin/gencode swagger \
  --host "api.example.com" \
  --title "我的API文档" \
  --version "2.0.0" \
  --description "这是我的API文档" \
  --output "./docs"
```

参数说明：
- `--host, -H`: API主机地址（默认：localhost:8082）
- `--title, -t`: API文档标题（默认：GINP API 文档）
- `--version, -v`: API版本号（默认：1.0.0）
- `--description, -d`: API文档描述
- `--output, -o`: 文档保存目录（默认：./static/docs）

生成后访问：`http://localhost:8082/swagger/index.html`

## 命令速查

```bash
# 实体相关
gencode entity list                    # 列出所有实体
gencode entity info <实体名>           # 查看实体详情
gencode entity gen <实体名>            # 生成实体CRUD代码
gencode entity delete <实体名>         # 删除实体CRUD代码

# API相关
gencode api list                       # 列出所有API
gencode api add                        # 交互式添加API
gencode api add --file <文件>          # 从JSON文件添加API
gencode api template                   # 生成API模板JSON

# Swagger相关
gencode swagger                        # 生成Swagger文档
gencode swagger --host <主机>          # 指定主机地址
```

## 工作原理

该工具直接调用 `internal/app/gapi/service/system/sgen` 包中的 service 层函数，与 Web 接口使用相同的底层实现，确保功能一致性。

主要 service 函数：
- `sgen.GetEntityList()` - 获取实体列表
- `sgen.GenServerCrudFiles()` - 生成CRUD代码
- `sgen.DeleteCrudFolders()` - 删除CRUD代码
- `sgen.AddApiInfo()` - 添加API接口
- `sgen.GetApiList()` - 获取API列表

## 注意事项

1. **实体生成**：生成实体前，需要先在 `setting.EntityGenerationList` 中注册实体
2. **API生成**：生成的API文件需要手动实现业务逻辑（TODO部分）
3. **删除操作**：删除实体CRUD代码时会提示确认，使用 `--force` 可跳过确认
4. **路径问题**：命令需要在 `api-server` 目录下执行

## 版本历史

- v0.2.0 - 重构为基于 service 层的实现
- v0.1.2 - 旧版本（基于 genfunc 包）

## 相关文档

- [代码生成服务实现](../../internal/app/gapi/service/system/sgen/)
- [Web接口实现](../../internal/app/gapi/controller/system/cgen/)
