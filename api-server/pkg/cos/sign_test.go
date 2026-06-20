package cos

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

//预签名 URL 更适合用于临时访问资源，而授权签名更适合用于 API 请求的身份验证。
// 两者都依赖于签名算法来确保安全性,都是用于验证请求合法性的机制，但使用方式和适用场景有所不同

// TestSigner_GeneratePresignedURL 测试基于长期密钥的预签名URL生成功能
// 预签名 URL 是一种包含了签名信息的 URL，可以直接用于访问受保护的资源。
// 它通常用于临时授权第三方访问特定资源，而无需暴露长期有效的凭证。
// 预签名 URL 包含了访问资源所需的所有信息，包括签名、过期时间等。
// 适用于需要临时访问资源的场景，例如上传或下载文件。
// 对于客户端直传场景，推荐使用基于临时密钥的签名方案(sts_sign.go)以提高安全性
func TestSigner_GeneratePresignedURL(t *testing.T) {
	// 初始化 Signer
	bucketURL := "https://your-bucket.cos.ap-guangzhou.myqcloud.com" // 替换为你的存储桶 URL
	secretID := "your-secret-id"                                     // 替换为你的 SecretID
	secretKey := "your-secret-key"                                   // 替换为你的 SecretKey

	signer, err := NewSigner(bucketURL, secretID, secretKey)
	if err != nil {
		t.Fatalf("创建 Signer 失败: %v", err)
	}

	// 生成预签名 URL
	objectKey := "test/example.jpg" // 替换为你要签名的对象键
	method := "PUT"
	expire := 10 * time.Minute

	presignedURL, err := signer.GeneratePresignedURL(objectKey, method, expire)
	if err != nil {
		t.Fatalf("生成预签名 URL 失败: %v", err)
	}

	fmt.Printf("预签名 URL: %s\n", presignedURL)
}

// TestSigner_GenerateAuthorization 测试基于长期密钥的授权签名生成功能
// 授权签名是用于在请求头中传递的签名信息，用于验证请求的合法性。
// 它通常用于 API 请求中，确保请求是由合法用户发起的。
// 授权签名需要在请求头中手动添加，通常以 Authorization 字段的形式出现。
// 适用于需要对每个请求进行身份验证的场景。
// 对于客户端直传场景，推荐使用基于临时密钥的签名方案(sts_sign.go)以提高安全性
func TestSigner_GenerateAuthorization(t *testing.T) {
	// 初始化 Signer
	bucketURL := "https://your-bucket.cos.ap-guangzhou.myqcloud.com" // 替换为你的存储桶 URL
	secretID := "your-secret-id"                                     // 替换为你的 SecretID
	secretKey := "your-secret-key"                                   // 替换为你的 SecretKey

	signer, err := NewSigner(bucketURL, secretID, secretKey)
	if err != nil {
		t.Fatalf("创建 Signer 失败: %v", err)
	}

	// 生成授权签名
	method := "PUT"
	uri := "https://your-bucket.cos.ap-guangzhou.myqcloud.com/test/example.jpg" // 替换为实际的 URI
	headers := http.Header{}
	headers.Set("Content-Type", "image/jpeg")
	expire := 10 * time.Minute

	auth, err := signer.GenerateAuthorization(method, uri, headers, expire)
	if err != nil {
		t.Fatalf("生成授权签名失败: %v", err)
	}

	fmt.Printf("授权签名: %s\n", auth)
}
