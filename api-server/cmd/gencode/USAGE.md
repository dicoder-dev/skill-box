# gencode 工具使用指南

## 快速开始

```bash
# 推荐：直接使用 go run（无需构建）
go run cmd/gencode/main.go

# 或者先构建再使用
go build -o bin/gencode cmd/gencode/main.go
./bin/gencode
```

## 核心功能

### 1. 实体管理

#### 列出所有实体
```bash
go run cmd/gencode/main.go entity list
```

#### 查看实体详情
```bash
# 查看基本信息
go run cmd/gencode/main.go entity info SysUser

# 以 JSON 格式输出（包含所有字段详情）
go run cmd/gencode/main.go entity info SysUser --json
```

#### 生成实体 CRUD 代码
```bash
go run cmd/gencode/main.go entity gen SysUser
```

**生成的内容：**
- ✅ 后端 Controller（5个CRUD接口）
- ✅ 后端 Service 层
- ✅ 后端 Model 层
- ✅ 前端 Vue 页面（列表页、编辑页）
- ✅ 前端字段配置
- ✅ 前端服务层
- ✅ 配置文件（gen-config/*.json）
- ✅ 自动更新 router 导入列表
- ✅ 自动生成菜单和权限（需要数据库）

#### 删除实体 CRUD 代码
```bash
# 需要确认
go run cmd/gencode/main.go entity delete SysUser

# 强制删除，不需要确认
go run cmd/gencode/main.go entity delete SysUser --force
```

### 2. API 接口管理

#### 列出所有 API
```bash
go run cmd/gencode/main.go api list
```

#### 添加新 API

**方式1：交互式添加**
```bash
go run cmd/gencode/main.go api add
```

按提示输入：
- API名称（大驼峰）
- API标题
- 一级目录（system/center）
- 二级目录（可选）
- HTTP方法
- 是否需要登录/权限

**方式2：从 JSON 文件添加**
```bash
# 生成模板
go run cmd/gencode/main.go api template -o my_api.json

# 编辑 my_api.json

# 从文件添加
go run cmd/gencode/main.go api add --file my_api.json
```

### 3. Swagger 文档生成

```bash
# 使用默认配置
go run cmd/gencode/main.go swagger

# 自定义配置
go run cmd/gencode/main.go swagger \
  --host "localhost:8082" \
  --title "我的API文档" \
  --version "1.0.0" \
  --output "./static/docs"
```

## 工作流程示例

### 场景：创建新的实体并生成完整代码

假设要创建一个 `Article`（文章）实体：

#### 步骤 1：创建实体文件

在 `internal/app/gapi/entity/` 创建 `article.e.go`：

```go
package entity

import (
	"ginp-api/internal/app/gapi/dto/system"
	"ginp-api/pkg/gencode/gen"
	"time"
)

const tableNameArticle = "articles"

type Article struct {
	ID        uint      `gorm:"primaryKey" json:"id,omitempty"`
	Title     string    `gorm:"type:varchar(200);comment:标题" json:"title,omitempty"`
	Content   string    `gorm:"type:text;comment:内容" json:"content,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*Article)(nil)

func (Article) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName:         tableNameArticle,
		Title:             "文章",
		FatherFolderName:  "system",
		OptionsLabelField: "Title",
		OptionsValueField: "id",
	}
}

func (Article) TableName() string {
	return tableNameArticle
}
```

#### 步骤 2：注册实体

在 `internal/app/gapi/setting/setting.go` 中添加：

```go
var EntityAutoMigrateList = []any{
	// ... 其他实体
	new(entity.Article),  // 添加新实体
}

var EntityGenerationList = []any{
	// ... 其他实体
	new(entity.Article),  // 添加新实体
}
```

#### 步骤 3：生成 CRUD 代码

```bash
go run cmd/gencode/main.go entity gen Article
```

#### 步骤 4：验证生成结果

```bash
# 检查后端文件
ls internal/app/gapi/controller/system/carticle/
ls internal/app/gapi/service/system/sarticle/
ls internal/app/gapi/model/system/marticle/

# 检查前端文件
ls frontend-vue3/src/views/admin/system/article/

# 检查配置文件
cat gen-config/article.json

# 检查 router 导入
cat internal/app/gapi/router/routers_import.go
```

#### 步骤 5：启动服务测试

```bash
# 启动后端服务
cd cmd/gapi
go run main.go

# 访问 API
curl http://localhost:8082/api/system/article/search
```

## 注意事项

### 1. 运行目录

**推荐：** 在 `api-server` 目录下运行
```bash
cd api-server
go run cmd/gencode/main.go entity list
```

### 2. 数据库初始化

生成代码时会尝试创建菜单和权限，需要数据库连接。如果数据库未初始化，会出现 panic，但不影响代码生成：

```
panic: 数据库未初始化，请先调用 InitDb() 函数
```

**解决方案：**
- 代码已经生成成功，可以忽略这个错误
- 或者先启动 Web 服务初始化数据库，再使用 Web 接口生成

### 3. 实体注册

生成代码前，必须先在 `setting.go` 中注册实体到 `EntityGenerationList`。

### 4. 路径问题

工具已修复路径问题，支持：
- ✅ 从 `api-server` 目录运行
- ✅ 使用 `go run` 运行
- ✅ 使用构建后的 `bin/gencode` 运行

## 命令速查表

| 功能 | 命令 |
|------|------|
| 列出实体 | `go run cmd/gencode/main.go entity list` |
| 查看实体详情 | `go run cmd/gencode/main.go entity info <实体名>` |
| 生成 CRUD | `go run cmd/gencode/main.go entity gen <实体名>` |
| 删除 CRUD | `go run cmd/gencode/main.go entity delete <实体名>` |
| 列出 API | `go run cmd/gencode/main.go api list` |
| 添加 API | `go run cmd/gencode/main.go api add` |
| 生成 Swagger | `go run cmd/gencode/main.go swagger` |
| 查看帮助 | `go run cmd/gencode/main.go --help` |

## 技术细节

### 代码生成流程

```
用户执行命令
  ↓
cmd/gencode/cmd/entity.go (命令行层)
  ↓
sgen.SaveEntityInfo() (service 层)
  ↓
├─ 生成实体文件 (.e.go)
├─ 保存配置文件 (.json)
├─ 更新 setting.go
├─ GenerateFilesFromTemplates() (模板系统)
│  ├─ 生成 Controller (5个文件)
│  ├─ 生成 Service
│  ├─ 生成 Model
│  └─ 生成前端文件
├─ GenServerCrudFiles() (更新 router 导入)
├─ GenCrudFront() (生成前端字段配置)
└─ GenMenuAndPermission() (生成菜单权限)
```

### 路径计算

工具使用 `sgen/func.go` 中的路径函数：
- `GetProjectBaseDir()` - 获取项目基础目录
- `GetModuleDir()` - 获取模块目录
- `GetGenerateConfigSavePath()` - 获取配置保存路径
- `GetFrontViewDir()` - 获取前端视图目录

这些函数已修复，支持从不同目录运行。

### Router 导入自动更新

生成实体后，会自动更新 `router/routers_import.go`：
- 收集所有实体的 controller 路径
- 合并固定导入和实体导入
- 去重并写入文件
- 确保新生成的 controller 被正确导入

## 故障排除

### 问题：找不到实体

**原因：** 实体未注册到 `EntityGenerationList`

**解决：** 在 `setting.go` 中添加实体

### 问题：路径错误

**原因：** 工作目录不正确

**解决：** 确保在 `api-server` 目录下运行

### 问题：数据库 panic

**原因：** 命令行工具未初始化数据库

**解决：** 忽略错误（代码已生成），或使用 Web 接口生成

## 相关文档

- [重构总结](./REFACTOR_SUMMARY.md)
- [路径修复总结](../../PATH_FIX_SUMMARY.md)
- [Router 导入修复](../../ROUTER_IMPORT_FIX.md)
