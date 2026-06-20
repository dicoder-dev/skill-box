package cos

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// Signer 用于生成基于长期密钥的 COS 签名
// 适用于服务端直接操作COS的场景
// 对于客户端直传场景，推荐使用更安全的临时密钥签名方案(sts_sign.go)
type Signer struct {
	client *cos.Client
}

// NewSigner 创建一个新的 Signer 实例
// bucketURL: COS存储桶访问域名，格式如 https://bucket-name.cos.ap-guangzhou.myqcloud.com
// secretID: 腾讯云API密钥ID
// secretKey: 腾讯云API密钥Key
func NewSigner(bucketURL, secretID, secretKey string) (*Signer, error) {
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}

	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	return &Signer{client: client}, nil
}

// GeneratePresignedURL 生成预签名 URL
func (s *Signer) GeneratePresignedURL(objectKey string, method string, expire time.Duration) (string, error) {
	presignedURL, err := s.client.Object.GetPresignedURL2(context.Background(), method, objectKey, expire, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

// GenerateAuthorization 生成授权签名
func (s *Signer) GenerateAuthorization(method, uri string, headers http.Header, expire time.Duration) (string, error) {
	// 获取凭证
	secretID := s.client.GetCredential().GetSecretId()
	secretKey := s.client.GetCredential().GetSecretKey()

	// 创建请求
	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return "", err
	}
	req.Header = headers

	// 生成授权签名
	authTime := cos.NewAuthTime(expire)
	cos.AddAuthorizationHeader(secretID, secretKey, "", req, authTime)
	return req.Header.Get("Authorization"), nil
}
