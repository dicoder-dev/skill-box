package push

import (
	"log"
)

// ExampleHuaweiPush 华为推送使用示例
func ExampleHuaweiPush() {

	// ========== 方式二：使用服务账号自动生成JWT token ==========
	ExampleWithServiceAccount()

}

// ExampleWithServiceAccount 使用服务账号自动生成JWT token的示例
func ExampleWithServiceAccount() {
	// 1. 解析私钥
	privateKeyPEM := `-----BEGIN PRIVATE KEY-----
YOUR_PRIVATE_KEY_HERE
-----END PRIVATE KEY-----`
	privateKey, err := ParsePrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		log.Printf("解析私钥失败: %v", err)
		return
	}

	// 2. 创建服务账号配置
	serviceConfig := &HuaweiServiceAccountConfig{
		SubAccount: "your_sub_account", // 服务账号ID
		KeyID:      "your_key_id",      // 密钥ID
		PrivateKey: privateKey,
		// Audience 会自动设置为华为OAuth服务地址
	}

	// 3. 创建推送配置
	config := &HuaweiPushConfig{
		ProjectID: "your_project_id",
		// AccessToken 会通过服务账号自动生成
	}

	// 4. 创建带服务账号的推送客户端
	client := NewHuaweiPushClientWithServiceAccount(config, serviceConfig)

	// 5. 自动刷新token
	err = client.RefreshToken()
	if err != nil {
		log.Printf("自动刷新token失败: %v", err)
		return
	}

	// 6. 发送推送消息
	tokens := []string{"device_push_token"}
	response, err := client.SendSimpleNotification(
		tokens,
		"服务账号推送标题",
		"通过服务账号自动生成JWT token发送的消息",
		"MARKETING",
	)

	if err != nil {
		log.Printf("发送推送失败: %v", err)
		return
	}

	log.Printf("服务账号推送成功: %s", response.RequestID)
}

// 使用说明：
// 1. ProjectID: 登录AppGallery Connect网站获取项目ID
// 2. AccessToken: 可通过以下方式获取JWT格式的访问令牌：
//    - 方式一：直接提供已有的JWT token
//    - 方式二：使用服务账号配置自动生成JWT token
//    - 方式三：使用客户端凭证模式获取访问令牌
// 3. Token: 设备的Push Token，需要在客户端获取
// 4. Category: 消息分类，影响推送频控限制
//    - MARKETING: 资讯营销类消息，受频控限制
//    - 其他类别请参考华为推送服务文档
// 5. TestMessage: 测试消息每个项目每天限制1000条
// 6. TTL: 消息缓存时间，单位秒，默认86400(1天)
// 7. 服务账号配置：需要从华为开发者控制台下载服务账号JSON文件获取相关参数
// 8. 客户端凭证：需要从华为开发者控制台获取client_id和client_secret
