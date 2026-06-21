# 华为推送服务封装包

本包封装了华为Push Kit服务端REST API，用于向华为设备推送通知消息。

## 功能特性

- 支持华为Push Kit V3 API
- 支持通知消息推送
- 支持测试消息发送
- 支持自定义消息参数
- 内置重试机制和错误处理
- 完整的类型定义和参数验证

## 快速开始

### 方式一：使用已有JWT Token

```go
config := &push.HuaweiPushConfig{
    ProjectID:   "your_project_id",     // 华为开发者项目ID
    AccessToken: "your_access_token",   // JWT格式访问令牌
}

client := push.NewHuaweiPushClient(config)
```

### 方式二：使用服务账号自动生成JWT Token

```go
// 1. 解析私钥
privateKeyPEM := `-----BEGIN PRIVATE KEY-----
YOUR_PRIVATE_KEY_HERE
-----END PRIVATE KEY-----`
privateKey, err := ParsePrivateKeyFromPEM(privateKeyPEM)
if err != nil {
    log.Fatal(err)
}

// 2. 创建服务账号配置
serviceConfig := &HuaweiServiceAccountConfig{
    SubAccount: "your_sub_account",  // 服务账号ID
    KeyID:      "your_key_id",      // 密钥ID
    PrivateKey: privateKey,
}

// 3. 创建推送配置
config := &HuaweiPushConfig{
    ProjectID: "your_project_id",
}

// 4. 创建带服务账号的推送客户端
client := NewHuaweiPushClientWithServiceAccount(config, serviceConfig)

// 5. 自动刷新token
err = client.AutoRefreshToken()
if err != nil {
    log.Fatal(err)
}
```

### 方式三：使用客户端凭证模式获取Token

```go
// 1. 创建推送客户端
config := &HuaweiPushConfig{
    ProjectID: "your_project_id",
}
client := NewHuaweiPushClient(config)

// 2. 通过客户端凭证获取访问令牌
tokenResponse, err := client.GetAccessToken(
    "your_client_id",
    "your_client_secret",
)
if err != nil {
    log.Fatal(err)
}

// 3. 更新客户端配置中的访问令牌
client.config.AccessToken = tokenResponse.AccessToken
```

## 消息推送示例

### 2. 发送简单通知

```go
tokens := []string{"device_push_token"}

response, err := client.SendSimpleNotification(
    tokens,
    "通知标题",
    "通知内容",
    "MARKETING", // 消息类型
)

if err != nil {
    log.Printf("推送失败: %v", err)
    return
}

log.Printf("推送成功: %s", response.RequestID)
```

### 3. 发送测试消息

```go
response, err := client.SendTestNotification(
    tokens,
    "测试标题",
    "测试内容",
    "MARKETING",
)
```

### 4. 发送自定义消息

```go
request := &push.HuaweiPushRequest{
    Payload: push.PushPayload{
        Notification: push.NotificationMessage{
            Category: "MARKETING",
            Title:    "自定义标题",
            Body:     "自定义内容",
            ClickAction: push.ClickAction{
                ActionType: 1, // 0:首页 1:内页
            },
            ForegroundShow: true,
            NotifyID:       12345,
        },
    },
    Target: push.PushTarget{
        Token: tokens,
    },
    PushOptions: push.PushOptions{
        TestMessage: false,
        TTL:         3600, // 缓存时间(秒)
    },
}

response, err := client.SendNotification(request)
```

## 配置说明

### 基本配置
- `ProjectID`: 华为开发者控制台中的项目ID
- `AccessToken`: JWT格式的访问令牌，可通过以下三种方式获取
- `Token`: 设备的Push Token，需要在客户端应用中获取
- `Category`: 消息分类，影响推送频控限制
  - `MARKETING`: 资讯营销类消息，受频控限制
  - 其他类别请参考华为推送服务文档

### JWT Token获取方式

#### 方式一：直接使用已有Token
如果您已经通过其他方式获取了JWT访问令牌，可以直接在配置中设置。

#### 方式二：服务账号自动生成（推荐）
使用华为开发者控制台下载的服务账号配置文件：
- `SubAccount`: 服务账号ID（从服务账号JSON文件中获取）
- `KeyID`: 密钥ID（从服务账号JSON文件中获取）
- `PrivateKey`: RSA私钥（从服务账号JSON文件中获取）
- `Audience`: OAuth服务地址（自动设置为华为OAuth服务）

#### 方式三：客户端凭证模式
使用华为开发者控制台的应用凭证：
- `ClientID`: 应用的客户端ID
- `ClientSecret`: 应用的客户端密钥

### HuaweiPushConfig

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ProjectID | string | 是 | 华为开发者项目ID |
| AccessToken | string | 是 | JWT格式的访问令牌 |
| BaseURL | string | 否 | 推送服务地址，默认华为云地址 |

### 消息类型 (Category)

- `MARKETING`: 资讯营销类消息，受频控限制(每日每设备2-5条)
- 其他类型请参考华为推送服务文档

### 点击行为 (ActionType)

- `0`: 点击消息进入应用首页
- `1`: 点击消息进入应用内页

## 重要说明

1. **API版本**: 使用V3版本API，仅支持HarmonyOS Next/5.x及之后版本
2. **频控限制**: 资讯营销类消息受到每日推送数量限制
3. **测试消息**: 每个项目每天限制1000条测试消息
4. **Token数量**: 单次推送Token数不超过10个(测试消息)
5. **消息缓存**: TTL设置消息缓存时间，默认86400秒(1天)

## 获取必要参数

### 基本参数获取

#### 1. 项目ID (ProjectID)

1. 登录 [AppGallery Connect](https://developer.huawei.com/consumer/cn/service/josp/agc/index.html)
2. 选择"开发与服务"
3. 在项目列表中选择对应项目
4. 左侧导航栏选择"项目设置"
5. 获取项目ID

#### 2. 设备Token

需要在客户端应用中获取设备的Push Token，具体方法请参考华为推送客户端集成文档。

### JWT Token获取参数

#### 服务账号方式（推荐）
1. 登录[华为开发者控制台](https://developer.huawei.com/consumer/cn/)
2. 进入项目 → API管理 → 凭据
3. 创建服务账号并下载JSON配置文件
4. 从JSON文件中获取以下参数：
   - `sub`: 服务账号ID（对应SubAccount）
   - `key_id`: 密钥ID（对应KeyID）
   - `private_key`: RSA私钥（对应PrivateKey）

#### 客户端凭证方式
1. 登录[华为开发者控制台](https://developer.huawei.com/consumer/cn/)
2. 进入项目 → API管理 → 凭据
3. 创建OAuth 2.0客户端ID
4. 获取以下参数：
   - `client_id`: 客户端ID
   - `client_secret`: 客户端密钥

#### 手动获取JWT Token
如果您需要手动获取JWT Token，可以参考华为官方文档：
- [JWT Token获取指南](https://developer.huawei.com/consumer/cn/doc/harmonyos-guides/push-jwt-token)
- [推送服务API请求结构](https://developer.huawei.com/consumer/cn/doc/harmonyos-references/push-scenariozed-api-request-struct)

## 错误处理

包内置了完整的错误处理机制：

- 参数验证错误
- 网络请求错误
- HTTP状态码错误
- 响应解析错误

所有错误都会通过返回值传递，建议在调用时进行适当的错误处理。

## 文件说明

- `huawei.go`: 华为推送服务主要实现文件（消息推送功能）
- `huawei_token.go`: JWT token获取和管理相关功能
- `example_huawei.go`: 使用示例代码
- `README.md`: 详细使用文档

## 示例代码

完整的使用示例请参考 `example_huawei.go` 文件。