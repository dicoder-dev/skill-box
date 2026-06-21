# ifthen - Go 三元运算符工具包

`ifthen` 包提供了类似于其他编程语言中三元运算符 `condition ? trueValue : falseValue` 的功能，支持多种数据类型。

## 功能特性

- 🚀 **类型安全**：编译时类型检查，避免运行时错误
- 🎯 **泛型支持**：使用 Go 1.18+ 泛型，支持任意类型
- 📦 **多类型覆盖**：数字、字符串、布尔、指针、切片等常用类型
- ⚡ **高性能**：零运行时开销，编译时优化

## 安装

```go
import "your-project/pkg/ifthen"
```

## 函数列表

| 函数 | 支持类型 | 描述 |
|------|----------|------|
| `Number[T]` | 所有数字类型 | 整数和浮点数的三元运算 |
| `Bool` | bool | 布尔值的三元运算 |
| `String` | string | 字符串的三元运算 |
| `Any[T]` | 任意类型 | 通用泛型三元运算 |
| `Ptr[T]` | 指针类型 | 指针的三元运算 |
| `Slice[T]` | 切片类型 | 切片的三元运算 |
| `Func[T]` | 无参数函数 | 返回值类型为T的函数三元运算 |
| `FuncWithArgs[T,R]` | 带参数函数 | 参数类型T，返回值类型R的函数三元运算 |
| `Handler` | 无参数处理器 | 无返回值的处理器函数三元运算 |
| `HandlerWithArgs[T]` | 带参数处理器 | 参数类型T，无返回值的处理器函数三元运算 |

## 使用示例

### 数字类型 (Number)

```go
package main

import (
    "fmt"
    "your-project/pkg/ifthen"
)

func main() {
    // 整数
    age := 25
    status := ifthen.Number(age >= 18, 1, 0) // 返回 1
    
    // 浮点数
    isVip := true
    price := ifthen.Number(isVip, 99.9, 199.9) // 返回 99.9
    
    // 不同数字类型
    var count int64 = ifthen.Number(hasItems, int64(10), int64(0))
    var score float32 = ifthen.Number(passed, float32(85.5), float32(0))
}
```

### 布尔类型 (Bool)

```go
func main() {
    hasPermission := true
    canEdit := false
    
    // 权限检查
    isEnabled := ifthen.Bool(hasPermission && canEdit, true, false)
    
    // 功能开关
    showButton := ifthen.Bool(isLoggedIn, true, false)
}
```

### 字符串类型 (String)

```go
func main() {
    isError := false
    username := "admin"
    
    // 消息提示
    message := ifthen.String(isError, "操作失败", "操作成功")
    
    // 显示名称
    displayName := ifthen.String(username != "", username, "匿名用户")
    
    // CSS 类名
    className := ifthen.String(isActive, "btn-primary", "btn-secondary")
}
```

### 任意类型 (Any)

```go
type User struct {
    ID   int
    Name string
}

func main() {
    isLoggedIn := true
    currentUser := User{ID: 1, Name: "张三"}
    anonymousUser := User{ID: 0, Name: "游客"}
    
    // 结构体选择
    user := ifthen.Any(isLoggedIn, currentUser, anonymousUser)
    
    // 接口类型
    var result interface{} = ifthen.Any(success, data, nil)
    
    // Map 类型
    config := map[string]string{"env": "prod"}
    emptyConfig := map[string]string{}
    activeConfig := ifthen.Any(isProduction, config, emptyConfig)
}
```

### 指针类型 (Ptr)

```go
func main() {
    user := &User{ID: 1, Name: "张三"}
    userExists := true
    
    // 指针选择
    var selectedUser *User = ifthen.Ptr(userExists, user, nil)
    
    // 可选值处理
    var optionalData *string
    if hasData {
        data := "some data"
        optionalData = ifthen.Ptr(true, &data, nil)
    }
}
```

### 切片类型 (Slice)

```go
func main() {
    hasData := true
    dataList := []string{"item1", "item2", "item3"}
    emptyList := []string{}
    
    // 切片选择
    items := ifthen.Slice(hasData, dataList, emptyList)
    
    // 数字切片
    numbers := []int{1, 2, 3}
    result := ifthen.Slice(len(numbers) > 0, numbers, []int{0})
    
    // 结构体切片
    users := []User{{ID: 1, Name: "张三"}}
    activeUsers := ifthen.Slice(hasActiveUsers, users, []User{})
}
```

### 函数类型 (Func/Handler)

```go
func main() {
    // 无参数函数选择
    useCache := true
    getData := ifthen.Func(useCache, getCachedData, getFreshData)
    result := getData() // 调用选中的函数
    
    // 带参数函数选择
    isProduction := false
    processor := ifthen.FuncWithArgs(isProduction, prodProcessor, devProcessor)
    output := processor("input data") // 传入参数并调用
    
    // 无参数处理器选择
    hasPermission := true
    handler := ifthen.Handler(hasPermission, successHandler, errorHandler)
    handler() // 执行选中的处理器
    
    // 带参数处理器选择
    isDebug := true
    logger := ifthen.HandlerWithArgs(isDebug, debugLog, productionLog)
    logger("debug message") // 传入参数并执行
}

// 示例函数定义
func getCachedData() string { return "cached" }
func getFreshData() string { return "fresh" }

func prodProcessor(input string) string { return "prod: " + input }
func devProcessor(input string) string { return "dev: " + input }

func successHandler() { fmt.Println("Success!") }
func errorHandler() { fmt.Println("Error!") }

func debugLog(msg string) { fmt.Printf("[DEBUG] %s\n", msg) }
func productionLog(msg string) { fmt.Printf("[INFO] %s\n", msg) }
```

## 实际应用场景

### Web API 响应

```go
func GetUserProfile(userID int) map[string]interface{} {
    user, exists := getUserByID(userID)
    
    return map[string]interface{}{
        "success": ifthen.Bool(exists, true, false),
        "message": ifthen.String(exists, "获取成功", "用户不存在"),
        "data":    ifthen.Any(exists, user, nil),
        "code":    ifthen.Number(exists, 200, 404),
    }
}
```

### 配置管理

```go
func GetConfig() Config {
    isProd := os.Getenv("ENV") == "production"
    
    return Config{
        Debug:    ifthen.Bool(isProd, false, true),
        LogLevel: ifthen.String(isProd, "error", "debug"),
        Port:     ifthen.Number(isProd, 80, 8080),
        Database: ifthen.String(isProd, "prod_db", "dev_db"),
    }
}
```

### 前端数据处理

```go
func FormatUserList(users []User, showAll bool) []User {
    activeUsers := filterActiveUsers(users)
    
    return ifthen.Slice(showAll, users, activeUsers)
}

func GetUserDisplayName(user *User) string {
    hasUser := user != nil
    userName := ifthen.Ptr(hasUser, &user.Name, nil)
    
    return ifthen.String(
        hasUser && userName != nil, 
        *userName, 
        "未知用户",
    )
}
```

### 策略模式与函数选择

```go
// 数据处理策略选择
func ProcessData(data []byte, useAdvanced bool) []byte {
    processor := ifthen.FuncWithArgs(useAdvanced, advancedProcess, basicProcess)
    return processor(data)
}

// 事件处理器选择
func SetupEventHandler(isVip bool) {
    handler := ifthen.HandlerWithArgs(isVip, vipEventHandler, normalEventHandler)
    eventBus.Subscribe("user_action", handler)
}

// 中间件选择
func GetAuthMiddleware(strictMode bool) func(http.Handler) http.Handler {
    return ifthen.Func(strictMode, strictAuthMiddleware, basicAuthMiddleware)
}

// 验证器选择
func GetValidator(isProduction bool) func(interface{}) error {
    return ifthen.FuncWithArgs(isProduction, productionValidator, developmentValidator)
}

// 回调函数选择
func ProcessAsync(data interface{}, success bool) {
    callback := ifthen.Handler(success, onSuccess, onFailure)
    go func() {
        // 异步处理
        time.Sleep(time.Second)
        callback()
    }()
}
```

## 性能说明

所有函数都是内联函数，编译器会在编译时进行优化，运行时性能与直接使用 `if-else` 语句相同，但代码更加简洁易读。

## 注意事项

1. **类型一致性**：`trueValue` 和 `falseValue` 必须是相同类型
2. **泛型约束**：`Number` 函数只支持数字类型，其他类型请使用 `Any`
3. **空指针安全**：使用 `Ptr` 函数时注意空指针检查
4. **Go 版本**：需要 Go 1.18+ 版本支持泛型语法