package push

import (
	"crypto/rsa"
	"fmt"
	"ginp-api/pkg/httpclient"
	"ginp-api/pkg/logger"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HuaweiServiceAccountConfig 华为服务账号配置，用于生成JWT token
type HuaweiServiceAccountConfig struct {
	SubAccount string          // 服务账号ID (json配置文件中的sub_account值)
	KeyID      string          // 密钥ID (json配置文件中的key_id值)
	PrivateKey *rsa.PrivateKey // RSA私钥
	Audience   string          // 受众，默认为华为OAuth服务地址
}

// JWTTokenResponse JWT token响应
type JWTTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewHuaweiPushClientWithServiceAccount 创建带服务账号配置的华为推送客户端
func NewHuaweiPushClientWithServiceAccount(config *HuaweiPushConfig, serviceConfig *HuaweiServiceAccountConfig) *HuaweiPushClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://push-api.cloud.huawei.com"
	}
	if serviceConfig.Audience == "" {
		serviceConfig.Audience = "https://oauth-login.cloud.huawei.com/oauth2/v3/token"
	}

	// 创建基础客户端
	client := NewHuaweiPushClient(config)

	// 添加服务账号配置
	client.serviceConfig = serviceConfig
	return client
}

// GenerateJWTToken 生成JWT访问令牌
func (c *HuaweiPushClient) GenerateJWTToken() (string, error) {
	if c.serviceConfig == nil {
		return "", fmt.Errorf("服务账号配置为空")
	}

	// 创建JWT claims
	claims := jwt.MapClaims{
		"iss": c.serviceConfig.SubAccount,       // 签发者
		"sub": c.serviceConfig.SubAccount,       // 主题
		"aud": c.serviceConfig.Audience,         // 受众
		"exp": time.Now().Add(time.Hour).Unix(), // 过期时间(1小时)
		"iat": time.Now().Unix(),                // 签发时间
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = c.serviceConfig.KeyID

	// 签名token
	tokenString, err := token.SignedString(c.serviceConfig.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("签名JWT token失败: %v", err)
	}

	return tokenString, nil
}

// GetAccessToken 通过客户端凭证获取访问令牌
func (c *HuaweiPushClient) GetAccessToken(clientID, clientSecret string) (*JWTTokenResponse, error) {
	url := "https://oauth-login.cloud.huawei.com/oauth2/v3/token"

	// 构建请求参数
	params := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	// 发送POST表单请求
	postParams := &httpclient.PostFormParams{
		Url:  url,
		Data: params,
		Header: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
	}

	var response JWTTokenResponse
	err := httpclient.PostForm(postParams, &response)
	if err != nil {
		return nil, fmt.Errorf("获取访问令牌失败: %v", err)
	}

	logger.Info("成功获取访问令牌，类型: %s, 过期时间: %d秒", response.TokenType, response.ExpiresIn)
	return &response, nil
}

// RefreshAccessToken 刷新访问令牌
func (c *HuaweiPushClient) RefreshAccessToken(clientID, clientSecret, refreshToken string) (*JWTTokenResponse, error) {
	url := "https://oauth-login.cloud.huawei.com/oauth2/v3/token"

	// 构建请求参数
	params := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	}

	// 发送POST表单请求
	postParams := &httpclient.PostFormParams{
		Url:  url,
		Data: params,
		Header: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
	}

	var response JWTTokenResponse
	err := httpclient.PostForm(postParams, &response)
	if err != nil {
		return nil, fmt.Errorf("刷新访问令牌失败: %v", err)
	}

	logger.Info("成功刷新访问令牌，类型: %s, 过期时间: %d秒", response.TokenType, response.ExpiresIn)
	return &response, nil
}

// RefreshToken 自动刷新token并更新客户端配置
func (c *HuaweiPushClient) RefreshToken() error {
	if c.serviceConfig == nil {
		return fmt.Errorf("服务账号配置为空，无法自动刷新token")
	}

	// 生成JWT token
	jwtToken, err := c.GenerateJWTToken()
	if err != nil {
		return fmt.Errorf("生成JWT token失败: %v", err)
	}

	// 更新客户端配置中的访问令牌
	c.Config.AccessToken = jwtToken
	logger.Info("成功自动刷新JWT token")
	return nil
}

// ParsePrivateKeyFromPEM 从PEM格式字符串解析RSA私钥
func ParsePrivateKeyFromPEM(privateKeyPEM string) (*rsa.PrivateKey, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("解析RSA私钥失败: %v", err)
	}
	return privateKey, nil
}
