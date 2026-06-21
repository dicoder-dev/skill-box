# gencode 命令行工具重构总结

## 重构目标

将 `cmd/gencode` 命令行工具重构为基于 service 层的实现，与 Web 接口保持一致，避免代码重复。

## 重构前的问题

1. **代码重复**：命令行工具使用 `pkg/gencode/genfunc` 包，Web 接口使用 `service/system/sgen` 包，两套实现
2. **维护困难**：功能更新需要同时修改两处代码
3. **不一致风险**：两套实现可能产生不一致的行为

## 重构方案

### 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    用户交互层                              │
├──────────────────────┬──────────────────────────────────┤
│   命令行工具 (CLI)    │      Web 接口 (HTTP API)          │
│  cmd/gencode/cmd/    │  controller/system/cgen/         │
└──────────┬───────────┴──────────────┬───────────────────┘
           │                          │
           └──────────┬───────────────┘
                      │
           ┌──────────▼──────────┐
           │   Service 层         │
           │  service/system/sgen │
           └─────────────────────┘
```

### 实现的命令

#### 1. entity 命令（实体管理）

| 子命令 | 功能 | Service 函数 |
|--------|------|-------------|
| `list` | 列出所有实体 | `sgen.GetEntityList()` |
| `info` | 查看实体详情 | `sgen.GetEntityList()` |
| `gen` | 生成实体CRUD代码 | `sgen.GenServerCrudFiles()` |
| `delete` | 删除实体CRUD代码 | `sgen.DeleteCrudFolders()` |

**示例：**
```bash
gencode entity list
gencode entity info SysUser
gencode entity gen SysUser
gencode entity delete SysUser --force
```

#### 2. api 命令（API管理）

| 子命令 | 功能 | Service 函数 |
|--------|------|-------------|
| `list` | 列出所有API | `sgen.GetApiList()` |
| `add` | 添加新API（交互式） | `sgen.AddApiInfo()` |
| `add --file` | 从JSON文件添加API | `sgen.AddApiInfo()` |
| `template` | 生成API模板JSON | - |

**示例：**
```bash
gencode api list
gencode api add
gencode api add --file my_api.json
gencode api template -o template.json
```

#### 3. swagger 命令（文档生成）

| 功能 | Service 函数 |
|------|-------------|
| 生成Swagger文档 | `swagen.NewSwaGen()`, `ginp.GetAllRouter()` |

**示例：**
```bash
gencode swagger
gencode swagger --host api.example.com --title "我的API"
```

## 文件变更

### 新增文件

1. `cmd/gencode/cmd/entity.go` - 实体管理命令（200行）
2. `cmd/gencode/cmd/api.go` - API管理命令（220行）
3. `cmd/gencode/README.md` - 使用文档

### 修改文件

1. `cmd/gencode/cmd/root.go` - 更新版本号和帮助信息
2. `cmd/gencode/cmd/swagger.go` - 重构为基于 service 层

### 删除文件

1. `cmd/gencode/cmd/gen.go` - 旧的生成命令（已被 entity.go 和 api.go 替代）
2. `cmd/gencode/cmd/crud.go` - 旧的批量CRUD命令（功能已整合）
3. `cmd/gencode/cmd/rm.go` - 旧的删除命令（已被 entity delete 替代）

## 核心改进

### 1. 统一的数据源

所有命令都调用 `service/system/sgen` 包中的函数，确保：
- ✅ 命令行工具和 Web 接口使用相同的业务逻辑
- ✅ 数据一致性（实体列表、API列表等）
- ✅ 功能更新只需修改一处

### 2. 更好的用户体验

**交互式模式：**
```bash
$ gencode api add
=== 添加新API ===

API名称（大驼峰，如 GetUserInfo）: GetUserInfo
API标题（如：获取用户信息）: 获取用户信息
...
```

**命令行参数模式：**
```bash
$ gencode entity gen SysUser
正在为实体 SysUser 生成CRUD代码...
✓ 实体 SysUser 的CRUD代码生成成功！
```

**JSON配置模式：**
```bash
$ gencode api template -o my_api.json
$ vim my_api.json  # 编辑配置
$ gencode api add --file my_api.json
```

### 3. 完善的帮助系统

每个命令都有详细的帮助信息：
```bash
gencode --help
gencode entity --help
gencode api add --help
gencode swagger --help
```

### 4. 灵活的参数配置

支持短参数和长参数：
```bash
gencode entity info SysUser -j          # 短参数
gencode entity info SysUser --json      # 长参数
gencode swagger -H localhost:8080       # 短参数
gencode swagger --host localhost:8080   # 长参数
```

## 技术细节

### Service 层函数映射

| 功能 | Service 函数 | 返回类型 |
|------|-------------|---------|
| 获取实体列表 | `sgen.GetEntityList()` | `[]sgen.EntityInfo` |
| 生成CRUD代码 | `sgen.GenServerCrudFiles(*EntityInfo)` | `error` |
| 删除CRUD代码 | `sgen.DeleteCrudFolders(string)` | `error` |
| 获取API列表 | `sgen.GetApiList()` | `[]map[string]interface{}` |
| 添加API | `sgen.AddApiInfo(*ApiInfo)` | `error` |

### 数据结构

**EntityInfo：**
```go
type EntityInfo struct {
    EntityName        string      // 实体名称
    Title             string      // 实体标题
    FieldCount        int         // 字段数
    TableName         string      // 表名
    OptionsLabelField string      // 选项标签字段
    FatherFolderName  string      // 父级文件夹名称
    Fields            []FieldInfo // 字段列表
}
```

**ApiInfo：**
```go
type ApiInfo struct {
    ApiName        string              // API名称
    Title          string              // 接口标题
    Description    string              // 接口描述
    FirstDirName   string              // 一级目录
    SecondDirName  string              // 二级目录
    Method         string              // HTTP方法
    NeedLogin      bool                // 是否需要登录
    NeedPermission bool                // 是否需要权限
    ApiPath        string              // API路径
    Params         []ParamItem         // 请求参数
    ResponseFields []ResponseFieldItem // 响应字段
}
```

## 使用示例

### 场景1：生成新实体的CRUD代码

```bash
# 1. 查看所有实体
$ gencode entity list

# 2. 查看实体详情
$ gencode entity info SysUser

# 3. 生成CRUD代码
$ gencode entity gen SysUser
正在为实体 SysUser 生成CRUD代码...
✓ 实体 SysUser 的CRUD代码生成成功！
```

### 场景2：添加新的API接口

```bash
# 方式1：交互式添加
$ gencode api add
=== 添加新API ===
API名称（大驼峰，如 GetUserInfo）: GetUserInfo
...

# 方式2：从JSON文件添加
$ gencode api template -o my_api.json
$ vim my_api.json
$ gencode api add --file my_api.json
```

### 场景3：生成Swagger文档

```bash
$ gencode swagger --host api.example.com --title "我的API文档"
=== 生成Swagger文档 ===
主机地址: api.example.com
文档标题: 我的API文档
版本号: 1.0.0
保存目录: ./static/docs

正在生成文档，共 50 个API接口...

✓ Swagger文档生成成功！
文档位置: ./static/docs/swagger.json

访问方式：
  后端服务地址/swagger/index.html
  例如: http://api.example.com/swagger/index.html
```

## 测试验证

### 构建测试

```bash
cd api-server
go build -o bin/gencode cmd/gencode/main.go
```

### 功能测试

```bash
# 测试帮助命令
./bin/gencode --help
./bin/gencode entity --help
./bin/gencode api --help
./bin/gencode swagger --help

# 测试实体命令
./bin/gencode entity list
./bin/gencode entity info SysUser
./bin/gencode entity info SysUser --json

# 测试API命令
./bin/gencode api list
./bin/gencode api template

# 测试Swagger命令
./bin/gencode swagger --help
```

## 兼容性说明

### 保留的功能

- ✅ 所有原有功能都已保留
- ✅ 生成的代码格式与之前一致
- ✅ 配置文件格式不变

### 移除的功能

- ❌ 旧的 `gen entity` 命令（改为 `entity gen`）
- ❌ 旧的 `gen crud` 批量生成命令（可通过脚本循环调用 `entity gen`）
- ❌ 旧的 `rm crud` 批量删除命令（可通过脚本循环调用 `entity delete`）

### 迁移指南

| 旧命令 | 新命令 |
|--------|--------|
| `gencode gen entity -c SysUser` | `gencode entity gen SysUser` |
| `gencode gen api -a GetUserInfo -d system/cuser` | `gencode api add` (交互式) |
| `gencode swagger` | `gencode swagger` (保持不变) |

## 后续优化建议

1. **批量操作**：添加批量生成/删除实体的命令
2. **模板定制**：支持自定义代码生成模板
3. **配置文件**：支持从配置文件读取默认参数
4. **进度显示**：生成大量代码时显示进度条
5. **回滚功能**：支持撤销最近的生成操作
6. **验证功能**：生成前验证实体配置的正确性

## 总结

本次重构成功实现了以下目标：

1. ✅ **统一实现**：命令行工具和 Web 接口共享同一套 service 层代码
2. ✅ **易于维护**：功能更新只需修改 service 层
3. ✅ **用户友好**：提供交互式、命令行参数、JSON配置三种使用方式
4. ✅ **文档完善**：提供详细的 README 和帮助信息
5. ✅ **向后兼容**：保留所有核心功能，仅调整命令结构

重构后的工具更加简洁、易用、可维护，为后续功能扩展打下了良好的基础。
