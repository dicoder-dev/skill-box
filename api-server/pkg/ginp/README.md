# 🚀 ginp - 高效的 Go API 开发框架

> 基于 Gin 框架的强大扩展，提供自动化参数绑定、操作类型分类、简洁错误处理和完整的中间件支持

[![Go Report Card](https://goreportcard.com/badge/github.com/DicoderCn/ginp)](#)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](#)

---

## ✨ 核心特性

### 1. 🔴 **自动参数绑定** - BindParamsHandler
参数绑定失败 **直接返回错误**，**不进入 handler**
```go
func Create(ctx *ginp.ContextPlus, params *entity.User) error {
    user, err := service.Create(params)  // 参数已自动绑定
    if err != nil { return err }          // 只需 return error
    ctx.SuccessData(user)
    return nil
}

ginp.RouterAppend(ginp.RouterItem{
    Path:     "/api/user/create",
    Handlers: ginp.BindParamsHandler(Create, &entity.User{}),  // ✨ 自动绑定
    // ...
})
```

### 2. 🏷️ **操作类型** - 14 种标准操作
CRUD + 导入/导出/审核/批准等
```go
ginp.OpCreate, ginp.OpRead, ginp.OpUpdate, ginp.OpDelete, ginp.OpSearch,
ginp.OpImport, ginp.OpExport, ginp.OpDownload, ginp.OpUpload, ginp.OpSync,
ginp.OpAudit, ginp.OpApprove, ginp.OpReject, ginp.OpCancel
```

### 3. 📊 **简化的参数绑定**
```go
if !ginp.MustBindJSON(ctx, &params) { return }  // 自动处理错误
result := ginp.BindJSON(ctx, &params)           // 自定义错误处理
```

### 4. 🛡️ **通用中间件**
- `LoggingMiddleware()` - 请求日志
- `CORSMiddleware()` - 跨域支持
- `RecoveryMiddleware()` - Panic 恢复
- `RequestIDMiddleware()` - 请求追踪

### 5. 🔄 **完全向后兼容**
现有代码无需修改，可逐步迁移

---

## 📦 快速安装

```bash
go get github.com/DicoderCn/ginp
```

---

## 🚀 5 分钟快速开始

### Step 1: 定义 Handler（带自动绑定）

```go
package cuser

import (
    "ginp-api/internal/app/gapi/entity"
    "ginp-api/internal/app/gapi/service/user"
    "ginp-api/pkg/ginp"
)

// 参数在函数签名中，自动绑定
func Create(ctx *ginp.ContextPlus, params *entity.User) error {
    user, err := service.Create(params)
    if err != nil {
        return err  // 错误自动转为响应
    }
    ctx.SuccessData(user)
    return nil
}
```

### Step 2: 注册路由

```go
func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/create",
        Handlers:      ginp.BindParamsHandler(Create, &entity.User{}),  // ✨ 关键
        HttpType:      ginp.HttpPost,
        NeedLogin:     false,
        NeedPermission: false,
        OperationType: ginp.OpCreate,  // 标记操作类型
        Swagger: &ginp.SwaggerInfo{
            Title:      "创建用户",
            RequestDto: entity.User{},
        },
    })
}
```

### Step 3: 调用接口

```bash
curl -X POST http://localhost:8080/api/user/create \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com"}'
```

**响应**：
```json
{"code":1,"msg":"success","data":{"id":1,"username":"john","email":"john@example.com"}}
```

---

## 🔍 深度理解：错误自动转为响应

### 问题：怎么理解"只需 return error，自动转为错误响应"？

`BindParamsHandler` 使用**反射**自动处理 Handler 的返回值。

### 核心原理

```
Handler 执行完毕
    ↓
BindParamsHandler 检查返回值
    ├─ 检查最后一个返回值是否是 error 类型
    └─ 检查这个 error 是否 != nil
    ↓
如果是 error 且不为 nil
    ↓
自动调用 ctx.Fail(err.Error())
    ↓
自动返回 JSON 错误响应
```

### 代码实现（convert.go 第 89-98 行）

```go
// 调用原始 handler
handlerFunc := reflect.ValueOf(handler)
results := handlerFunc.Call(params)  // ← 获取返回值

// 自动处理返回值
if len(results) > 0 {
    // 获取最后一个返回值
    if errVal := results[len(results)-1]; 
       // 检查是否实现了 error 接口
       errVal.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
        
        // 如果不是 nil（有错误）
        if !errVal.IsNil() {
            err := errVal.Interface().(error)
            ctx.Fail(err.Error())  // ← 自动返回错误响应
            return
        }
    }
}

// 如果返回 nil（成功），自动返回成功响应
ctx.Success()
```

### 三个场景

#### 场景 1️⃣：参数校验失败

```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    if params.Extra.PageNum <= 0 {
        return errors.New("invalid page")  // ← 返回错误
    }
    ctx.SuccessData(data)
    return nil
}
```

**执行流程**：
```
1. Search() 检查参数
2. 发现 PageNum <= 0
3. return errors.New("invalid page")
4. BindParamsHandler 接收到返回值
5. 发现返回值是 error 且不为 nil
6. 自动调用 ctx.Fail("invalid page")
7. 客户端收到：{"code":0,"msg":"invalid page"}
```

#### 场景 2️⃣：业务逻辑失败

```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    if params.Extra.PageNum <= 0 {
        return errors.New("invalid page")
    }
    
    list, err := service.Search(params)
    if err != nil {
        return err  // ← 返回错误
    }
    
    ctx.SuccessData(list)
    return nil
}
```

**执行流程**：
```
1. 参数校验通过
2. 调用 service.Search()
3. 数据库查询失败，service 返回 error
4. Handler 执行 return err
5. BindParamsHandler 检查返回值
6. 发现是 error 且不为 nil
7. 自动调用 ctx.Fail(err.Error())
8. 客户端收到：{"code":0,"msg":"database error"}
```

#### 场景 3️⃣：成功

```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    if params.Extra.PageNum <= 0 {
        return errors.New("invalid page")
    }
    
    list, err := service.Search(params)
    if err != nil {
        return err
    }
    
    ctx.SuccessData(list)
    return nil  // ← 返回 nil
}
```

**执行流程**：
```
1. 参数校验通过
2. 业务逻辑执行成功
3. 调用 ctx.SuccessData(list)
4. return nil
5. BindParamsHandler 检查返回值
6. 发现是 nil
7. 自动调用 ctx.Success()
8. 客户端收到：{"code":1,"msg":"success","data":[...]}
```

### 对比：旧方式 vs 新方式

**❌ 旧方式（手动处理）**：
```go
func Search(ctx *ginp.ContextPlus) {
    var params *comdto.ReqSearch
    if err := ctx.ShouldBindJSON(&params); err != nil {
        ctx.Fail("error: " + err.Error())  // ← 手动 ctx.Fail()
        return
    }
    
    list, err := service.Search(params)
    if err != nil {
        ctx.Fail("failed: " + err.Error())  // ← 手动 ctx.Fail()
        return
    }
    
    ctx.SuccessData(list)
}
```

**✅ 新方式（自动处理）**：
```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    if params.Extra.PageNum <= 0 {
        return errors.New("invalid page")  // ← 只需 return
    }
    
    list, err := service.Search(params)
    if err != nil {
        return err  // ← 只需 return
    }
    
    ctx.SuccessData(list)
    return nil  // ← 成功也是 return nil
}
```

### 类比理解

就像 Promise / async-await 一样：

```javascript
// JavaScript 的 async/await
async function search(params) {
    if (params.page <= 0) {
        throw new Error("invalid page")  // ← throw error
    }
    
    try {
        const list = await service.search(params)
        return list  // ← return data
    } catch (err) {
        throw err  // ← throw error
    }
}

// 上层自动处理
try {
    const result = await search(params)
    response.json({code: 1, data: result})  // ← 成功
} catch (err) {
    response.json({code: 0, msg: err.message})  // ← 错误
}
```

**Go 中的 BindParamsHandler 也做同样的事**：
- `return error` = throw error
- `return nil` = success
- BindParamsHandler = 自动的 try-catch

### 关键要点

| 要点 | 说明 |
|------|------|
| **返回值必须是 error** | Handler 最后一个返回值必须是 error 类型 |
| **error 为 nil** | 表示成功，自动调用 ctx.Success() |
| **error 不为 nil** | 表示失败，自动调用 ctx.Fail() |
| **自动处理** | 无需手动调用 ctx.Fail() 或 ctx.Success() |
| **统一方式** | 所有错误都通过 return error 处理 |

---

| 方案 | 代码行数 | 特点 | 推荐 |
|------|---------|------|------|
| **原始方式** | 15 | 手动绑定，手动检查 | ⚠️ 仅兼容旧代码 |
| **MustBindJSON** | 12 | 自动处理错误 | ✅ 现有代码改造 |
| **BindParamsHandler** | 6 | 绑定失败不进 handler | ⭐⭐⭐ **推荐新代码** |

### 代码对比

**原始方式（15 行）**：
```go
func Create(ctx *ginp.ContextPlus) {
    var params *entity.User
    if err := ctx.ShouldBindJSON(&params); err != nil {
        ctx.Fail("请求参数有误" + err.Error())
        return
    }
    user, err := service.Create(params)
    if err != nil {
        ctx.Fail("创建失败" + err.Error())
        return
    }
    ctx.SuccessData(user)
}
```

**改进方式（6 行）**：
```go
func Create(ctx *ginp.ContextPlus, params *entity.User) error {
    user, err := service.Create(params)
    if err != nil { return err }
    ctx.SuccessData(user)
    return nil
}
// 使用 ginp.BindParamsHandler(Create, &entity.User{})
```

**改进**：📉 减少 60% 代码！

---

## 🎯 核心 API

### ContextPlus - 增强的上下文

继承 `gin.Context` 的所有方法，额外提供：

```go
// 成功响应
ctx.Success()                    // {"code":1,"msg":"success"}
ctx.SuccessData(data)            // 带数据

// 失败响应
ctx.Fail("错误")                 // {"code":0,"msg":"错误"}
ctx.FailData(data, "错误")       // 带数据

// 获取用户信息
userID := ctx.GetUserID()        // 从 JWT token 获取

// 访问 Gin 原生方法
ctx.JSON(200, data)              // 标准 Gin 方法
ctx.GetString("key")             // Gin 上下文存储
```

### 参数绑定

```go
// 方式 1：简洁方式（推荐）- 自动处理错误
if !ginp.MustBindJSON(ctx, &params) { return }
if !ginp.MustBindQuery(ctx, &params) { return }
if !ginp.MustBindURI(ctx, &params) { return }

// 方式 2：灵活方式 - 自定义错误处理
result := ginp.BindJSON(ctx, &params)
if !result.Success {
    ctx.Fail("自定义错误: " + result.Message)
    return
}

// 方式 3：自动绑定方式（最推荐）
func Handler(ctx *ginp.ContextPlus, params *Type) error {
    // params 已自动绑定，绑定失败不会执行到这里
}
ginp.BindParamsHandler(Handler, &Type{})
```

### 操作类型

```go
// CRUD 基础操作
ginp.OpCreate   // 创建
ginp.OpRead     // 查询
ginp.OpUpdate   // 修改
ginp.OpDelete   // 删除
ginp.OpSearch   // 搜索

// 其他操作
ginp.OpImport, ginp.OpExport, ginp.OpDownload, ginp.OpUpload,
ginp.OpSync, ginp.OpAudit, ginp.OpApprove, ginp.OpReject, ginp.OpCancel
```

### 路由注册

```go
ginp.RouterAppend(ginp.RouterItem{
    Path:           "/api/user/create",
    AliasePaths:    []string{"/api/v2/user/create"},  // 别名
    Handlers:       ginp.BindParamsHandler(Create, &entity.User{}),
    HttpType:       ginp.HttpPost,
    NeedLogin:      false,
    NeedPermission: false,
    PermissionName: "system.user.create",
    OperationType:  ginp.OpCreate,  // 操作类型
    Swagger: &ginp.SwaggerInfo{
        Title:      "创建用户",
        Description: "创建新用户",
        RequestDto: entity.User{},
    },
})
```

---

## 📚 完整示例（CRUD）

### Create - 创建
```go
func Create(ctx *ginp.ContextPlus, params *entity.User) error {
    user, err := service.Create(params)
    if err != nil { return err }
    ctx.SuccessData(user)
    return nil
}

func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/create",
        Handlers:      ginp.BindParamsHandler(Create, &entity.User{}),
        HttpType:      ginp.HttpPost,
        OperationType: ginp.OpCreate,
    })
}
```

### Read - 查询
```go
func FindByID(ctx *ginp.ContextPlus, params *comdto.ReqFindById) error {
    user, err := service.FindByID(params.ID)
    if err != nil { return err }
    ctx.SuccessData(user)
    return nil
}

func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/find_by_id",
        Handlers:      ginp.BindParamsHandler(FindByID, &comdto.ReqFindById{}),
        HttpType:      ginp.HttpPost,
        OperationType: ginp.OpRead,
    })
}
```

### Update - 修改
```go
func Update(ctx *ginp.ContextPlus, params *UpdateRequest) error {
    err := service.Update(params.ID, params.UpdateData, params.Fields...)
    if err != nil { return err }
    ctx.Success()
    return nil
}

func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/update",
        Handlers:      ginp.BindParamsHandler(Update, &UpdateRequest{}),
        HttpType:      ginp.HttpPost,
        OperationType: ginp.OpUpdate,
    })
}
```

### Delete - 删除
```go
func Delete(ctx *ginp.ContextPlus, params *comdto.ReqDelete) error {
    err := service.Delete(params.ID)
    if err != nil { return err }
    ctx.Success()
    return nil
}

func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/delete",
        Handlers:      ginp.BindParamsHandler(Delete, &comdto.ReqDelete{}),
        HttpType:      ginp.HttpPost,
        OperationType: ginp.OpDelete,
    })
}
```

### Search - 搜索
```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    list, total, err := service.Search(params.Wheres, params.Extra)
    if err != nil { return err }
    ctx.SuccessData(map[string]interface{}{
        "list": list,
        "total": total,
    })
    return nil
}

func init() {
    ginp.RouterAppend(ginp.RouterItem{
        Path:          "/api/user/search",
        Handlers:      ginp.BindParamsHandler(Search, &comdto.ReqSearch{}),
        HttpType:      ginp.HttpPost,
        OperationType: ginp.OpSearch,
    })
}
```

---

## 🔧 高级用法

### 多参数绑定

同时绑定 JSON body 和 URL Query：

```go
func Search(ctx *ginp.ContextPlus, 
            bodyParams *RequestBody,
            queryParams *QueryParams) error {
    // bodyParams 从 JSON 自动绑定
    // queryParams 从 Query 自动绑定
    // 任何一个失败都不会进入此函数
    return nil
}

ginp.BindParamsHandler(Search, &RequestBody{}, &QueryParams{})
```

### 使用中间件

```go
func main() {
    r := gin.Default()
    
    // 应用中间件
    r.Use(ginp.LoggingMiddleware())        // 请求日志
    r.Use(ginp.CORSMiddleware())           // 跨域
    r.Use(ginp.RecoveryMiddleware())       // Panic 恢复
    r.Use(ginp.RequestIDMiddleware())      // 请求 ID
    
    // 注册所有路由
    ginp.RegisterRouter(r)
    
    // 配置日志级别
    ginp.SetLogLevel(ginp.LogLevelDebug)
    
    r.Run(":8080")
}
```

### 操作处理器注册表

```go
registry := ginp.NewOperationRegistry()

registry.Register(ginp.OpCreate, func(ctx *ginp.ContextPlus, op ginp.OperationType, params interface{}) error {
    user := params.(*entity.User)
    return service.Create(user)
})

registry.Register(ginp.OpDelete, func(ctx *ginp.ContextPlus, op ginp.OperationType, params interface{}) error {
    user := params.(*entity.User)
    return service.Delete(user.ID)
})

err := registry.Execute(ctx, ginp.OpCreate, &user)
if err != nil {
    ctx.Fail(err.Error())
}
```

---

## 📈 性能对比

| 指标 | 原始方式 | MustBindJSON | BindParamsHandler |
|------|---------|--------------|-----------------|
| **代码行数** | 15 | 12 | **6** |
| **参数绑定在 handler 内** | ✅ | ✅ | ❌ |
| **绑定失败进入 handler** | ✅ | ✅ | ❌ |
| **反射开销** | - | - | < 1μs |
| **可读性** | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

---

## 🔄 迁移指南

### 从原始方式迁移到 BindParamsHandler

**Step 1: 修改 Handler 签名**
```go
// 原始
func Create(ctx *ginp.ContextPlus) { ... }

// 改为
func Create(ctx *ginp.ContextPlus, params *entity.User) error { ... }
```

**Step 2: 移除参数绑定代码**
```go
// 删除这些
var params *entity.User
if err := ctx.ShouldBindJSON(&params); err != nil {
    ctx.Fail("请求参数有误" + err.Error())
    return
}
```

**Step 3: 简化错误返回**
```go
// 改为
if err != nil {
    return err  // 自动转为响应
}
```

**Step 4: 使用 BindParamsHandler**
```go
Handlers: ginp.BindParamsHandler(Create, &entity.User{})
```

---

## 🛡️ 请求拦截 (Request Interceptor)

### 问题
使用 `BindParamsHandler` 时，有时需要在参数绑定**后**、业务逻辑**前**做额外的校验。比如：
- 校验搜索条件的合法性
- 校验业务权限
- 校验用户配额

### 解决方案：验证函数

**步骤 1: 创建验证函数**
```go
// validateSearchParams 校验搜索参数
func validateSearchParams(params *comdto.ReqSearch) error {
    // 校验分页参数
    if params.Extra != nil {
        if params.Extra.PageNum <= 0 {
            return errors.New("pageNum must be greater than 0")
        }
        if params.Extra.PageSize > 1000 {
            return errors.New("pageSize cannot exceed 1000")
        }
    }
    
    // 校验搜索字段数量
    if params.Wheres != nil && len(params.Wheres) > 10 {
        return errors.New("search conditions cannot exceed 10")
    }
    
    return nil
}
```

**步骤 2: 在 Handler 中调用验证**
```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    // 1️⃣ 参数拦截 - 验证参数合法性
    if err := validateSearchParams(params); err != nil {
        return err  // 自动转为错误响应
    }
    
    // 2️⃣ 参数已自动绑定，直接使用
    if where.Check(params.Wheres) != nil {
        return where.Check(params.Wheres)
    }
    
    // 3️⃣ 调用业务逻辑
    list, total, err := service.Search(params.Wheres, params.Extra)
    if err != nil {
        return err
    }
    
    // 4️⃣ 返回结果
    ctx.SuccessData(map[string]interface{}{
        "list": list,
        "total": total,
    })
    return nil
}
```

### 完整流程图

```
请求来临
    ↓
Gin 路由匹配
    ↓
BindParamsHandler 拦截
    ├─ 解析 JSON
    ├─ 绑定参数
    └─ 如果失败 → 直接返回错误响应 ❌
    ↓
Handler 执行
    ├─ 1️⃣ validateSearchParams() - 参数拦截
    │   ├─ 校验分页参数
    │   ├─ 校验搜索字段
    │   └─ 如果失败 → return error ❌
    ├─ 2️⃣ 业务逻辑
    └─ 3️⃣ 返回结果
    ↓
返回给客户端
```

### 参数直接使用

关键点：**参数在函数签名中**，BindParamsHandler 会自动填充

```go
// ❌ 错误方式 - 参数不在签名中
func Search(ctx *ginp.ContextPlus) error {
    var params *comdto.ReqSearch  // ← 需要手动声明
    // ...
}

// ✅ 正确方式 - 参数在签名中
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    // params 已经自动绑定，直接使用
    if params.Extra.PageNum <= 0 { ... }
}
```

### 实际例子

**search.a.go** - 完整的参数拦截示例：
```go
func Search(ctx *ginp.ContextPlus, params *comdto.ReqSearch) error {
    // 1️⃣ 请求拦截
    if err := validateSearchParams(params); err != nil {
        return err  // 自动转为错误响应
    }
    
    // 2️⃣ 参数已经自动绑定，可以直接使用
    if where.Check(params.Wheres) != nil {
        return where.Check(params.Wheres)
    }
    
    // 3️⃣ 调用业务逻辑
    list, total, err := sdjcategory.Model().FindListWithRelations(
        params.Wheres, params.Extra, params.Relations,
    )
    if err != nil {
        return err
    }
    
    // 4️⃣ 返回结果
    ctx.SuccessData(map[string]interface{}{
        "list": list,
        "total": total,
    })
    return nil
}

// validateSearchParams 是请求拦截的实现
func validateSearchParams(params *comdto.ReqSearch) error {
    if params.Extra != nil {
        if params.Extra.PageNum <= 0 {
            return errors.New("pageNum must be greater than 0")
        }
        if params.Extra.PageSize <= 0 || params.Extra.PageSize > 1000 {
            return errors.New("pageSize must be between 1 and 1000")
        }
    }
    
    if params.Wheres != nil && len(params.Wheres) > 10 {
        return errors.New("search conditions cannot exceed 10")
    }
    
    return nil
}
```

### 与中间件的区别

| 对比 | 中间件 | 参数拦截 |
|------|--------|----------|
| **执行时机** | 路由之前 | 参数绑定之后 |
| **能否访问参数** | ❌ 不能 | ✅ 能 |
| **通常用途** | 认证、日志、CORS | 参数验证 |
| **位置** | 应用级 | 路由级 |

---

**Q: BindParamsHandler 和 MustBindJSON 有什么区别？**

A: 关键区别是参数绑定失败时：
- `MustBindJSON`：在 handler 内检查，仍会进入 handler
- `BindParamsHandler`：在 ginp 内检查，**不进入 handler**（更高效）

**Q: 现有代码需要改吗？**

A: 不需要。完全向后兼容。新代码推荐使用 BindParamsHandler，旧代码可逐步迁移。

**Q: OperationType 必须添加吗？**

A: 不必须。这是可选的，用于 API 分类和统计。

**Q: 如何自定义错误消息？**

A: 使用 `BindJSON()` 而非 `MustBindJSON()`：
```go
result := ginp.BindJSON(ctx, &params)
if !result.Success {
    ctx.Fail("自定义错误: " + result.Message)
    return
}
```

**Q: 支持哪些参数源？**

A: 支持 JSON Body、URL Query、URL Path：
```go
ginp.MustBindJSON(ctx, &params)   // JSON
ginp.MustBindQuery(ctx, &params)  // Query
ginp.MustBindURI(ctx, &params)    // Path
```

---

## ✅ 检查清单（迁移现有 API）

对每个 API，检查以下项目：

- [ ] 修改 handler 签名，添加参数
- [ ] 删除 `var params` 声明
- [ ] 删除参数绑定检查代码
- [ ] 将 `ctx.Fail()` 改为 `return err`
- [ ] 更新路由注册，使用 `BindParamsHandler`
- [ ] 添加 `OperationType` 字段
- [ ] 测试无效参数（确认不进入 handler）
- [ ] 测试有效参数（确认正常执行）

---

## 📝 设置方法

```go
// 成功/失败状态码
ginp.SetSuccessCode(1)        // 默认 1
ginp.SetFailCode(0)           // 默认 0

// 成功/失败消息
ginp.SetSuccessMsg("ok")      // 默认 "success"
ginp.SetFailMsg("error")      // 默认 "fail"

// HTTP 状态码
ginp.SetSuccessHttpCode(200)  // 默认 200
ginp.SetFailHttpCode(200)     // 默认 200

// 日志
ginp.SetShowLog(true)         // 默认 true
ginp.SetLogLevel(ginp.LogLevelInfo)  // 日志级别
```

---

## 🎉 总结

ginp 框架让 Go API 开发：

- 📉 **代码减少 60%** - 参数绑定从 15 行到 6 行
- ⚡ **性能更好** - 参数绑定失败不进入 handler
- 🎯 **意图更清晰** - 参数在函数签名中
- 🔄 **完全兼容** - 现有代码无需修改
- 📊 **更易维护** - 统一的操作类型和错误处理

---

## 📄 许可证

MIT License

---

## 🤝 贡献

欢迎 PR 和 Issue！

---

**开始使用 ginp，让你的 API 开发更高效！** 🚀