# COS Package

腾讯云对象存储（COS）Go SDK封装包，提供两种签名方式以满足不同场景需求。

## 功能特性

- 基于长期密钥的签名方案
- 基于临时密钥的签名方案
- 预签名URL生成
- 授权签名生成
- 完整的测试用例

## 两种签名方式的区别

### 基于长期密钥的签名 (sign.go)

使用长期有效的SecretID和SecretKey进行签名。

**优势：**
- 实现简单，直接使用腾讯云API密钥
- 适用于服务端直接操作COS的场景

**劣势：**
- 密钥安全性较低，一旦泄露影响较大
- 权限控制不够灵活

**适用场景：**
- 服务端后台批量操作COS
- 不涉及客户端直传的场景

### 基于临时密钥的签名 (sts_sign.go)

通过腾讯云CAM服务获取临时密钥进行签名，临时密钥具有时效性和权限限制。

**优势：**
- **权限安全**：可以有效限定安全的权限范围，只能用于指定的文件路径
- **路径安全**：由服务端决定随机的COS文件路径，避免文件覆盖风险
- **传输安全**：在服务端生成签名，避免临时密钥在传输过程中泄漏

**劣势：**
- 实现相对复杂
- 需要额外调用CAM服务获取临时密钥

**适用场景：**
- 客户端直传场景
- 对安全性要求较高的场景

## 使用说明

### 安装

```bash
go get github.com/tencentyun/cos-go-sdk-v5
go get github.com/tencentyun/qcloud-cos-sts-sdk/go
```

### 基于长期密钥的签名使用示例

```go
import "your-project/pkg/cos"

// 初始化签名器
bucketURL := "https://your-bucket.cos.ap-guangzhou.myqcloud.com"
signer, err := cos.NewSigner(bucketURL, "your-secret-id", "your-secret-key")
if err != nil {
    log.Fatal(err)
}

// 生成预签名URL
presignedURL, err := signer.GeneratePresignedURL("path/to/file.jpg", "PUT", 30*time.Minute)
if err != nil {
    log.Fatal(err)
}

fmt.Println("预签名URL:", presignedURL)
```

### 基于临时密钥的签名使用示例

```go
import "your-project/pkg/cos"

// 配置临时密钥参数
config := &cos.STSConfig{
    SecretID:    "your-secret-id",
    SecretKey:   "your-secret-key",
    Bucket:      "your-bucket-name",
    Region:      "ap-guangzhou",
    Duration:    1800, // 30分钟
    AllowPrefix: "uploads/", // 限制只能上传到uploads目录
}

// 初始化STSSigner
stsSigner, err := cos.NewSTSSigner(config)
if err != nil {
    log.Fatal(err)
}

// 生成预签名URL
presignedURL, err := stsSigner.GeneratePresignedURL("uploads/file.jpg", "PUT", 30*time.Minute)
if err != nil {
    log.Fatal(err)
}

fmt.Println("预签名URL:", presignedURL)

// 获取临时密钥信息（可用于返回给客户端）
credential := stsSigner.GetCredential()
fmt.Printf("临时密钥ID: %s\n", credential.Credentials.TmpSecretID)
```

## 客户端直传最佳实践

1. 客户端向业务服务器发送上传请求
2. 业务服务器调用STSSigner生成预签名URL和临时密钥
3. 业务服务器将预签名URL和临时密钥返回给客户端
4. 客户端使用预签名URL直接上传文件到COS

这种方式既保证了安全性，又避免了业务服务器的带宽消耗。

### 客户端直传文件服务端处理示例

以下是一个完整的客户端直传文件的服务端处理示例：

```go
// 服务端处理函数
func HandleFileUpload(c *gin.Context) {
    // 获取客户端传递的文件后缀
    fileExt := c.PostForm("file_ext")
    if fileExt == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "文件后缀不能为空"})
        return
    }

    // 生成随机文件名
    fileName := fmt.Sprintf("%d_%s.%s", time.Now().Unix(), utils.GenerateRandomString(8), fileExt)
    
    // 配置STSSigner
    config := &cos.STSConfig{
        SecretID:    os.Getenv("COS_SECRET_ID"),
        SecretKey:   os.Getenv("COS_SECRET_KEY"),
        Bucket:      "your-bucket-name",
        Region:      "ap-guangzhou",
        Duration:    1800, // 30分钟
        AllowPrefix: "uploads/", // 限制只能上传到uploads目录
    }

    // 初始化STSSigner
    stsSigner, err := cos.NewSTSSigner(config)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "签名器初始化失败"})
        return
    }

    // 生成预签名URL
    presignedURL, err := stsSigner.GeneratePresignedURL(fmt.Sprintf("uploads/%s", fileName), "PUT", 30*time.Minute)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "生成预签名URL失败"})
        return
    }

    // 获取临时密钥信息
    credential := stsSigner.GetCredential()

    // 返回给客户端
    c.JSON(http.StatusOK, gin.H{
        "presigned_url": presignedURL,
        "file_key":      fmt.Sprintf("uploads/%s", fileName),
        "tmp_secret_id": credential.Credentials.TmpSecretID,
        "tmp_secret_key": credential.Credentials.TmpSecretKey,
        "session_token": credential.Credentials.Token,
    })
}
```

客户端在获取到预签名URL和其他凭证信息后，可以直接使用HTTP PUT或POST请求将文件上传到COS，无需再经过业务服务器中转。

## 文件说明

- `sign.go`：基于长期密钥的签名实现
- `sign_test.go`：基于长期密钥的签名测试
- `sts_sign.go`：基于临时密钥的签名实现
- `sts_sign_test.go`：基于临时密钥的签名测试
- `sts_sign_example.go`：基于临时密钥的使用示例

## 相关文档

- [COS_SIGNING_PRACTICE.md](../../COS_SIGNING_PRACTICE.md)：COS服务端签名实践详细说明