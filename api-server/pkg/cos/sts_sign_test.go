package cos

import (
	"fmt"
	"testing"
)

// TestSTSSigner 测试 STSSigner 的功能
func TestSTSSigner(t *testing.T) {
	// 注意：在实际运行测试时，需要替换为有效的配置信息
	config := &STSConfig{
		SecretID:  "your-secret-id",   // 替换为你的 SecretID
		SecretKey: "your-secret-key",  // 替换为你的 SecretKey
		Bucket:    "your-bucket-name", // 替换为你的存储桶名称
		Region:    "ap-guangzhou",     // 替换为你的存储桶所在地域
		Duration:  1800,               // 临时密钥有效期（秒）
		AppID:     "1250000000",       // 替换为你的 APPID
	}

	// 创建 STSSigner 实例
	signer, err := NewSTSSigner(config)
	if err != nil {
		t.Fatalf("创建 STSSigner 失败: %v", err)
	}

	// 验证临时密钥信息
	credential := signer.GetCredential()
	if credential.Credentials.TmpSecretID == "" {
		t.Error("临时密钥 TmpSecretID 为空")
	}
	if credential.Credentials.TmpSecretKey == "" {
		t.Error("临时密钥 TmpSecretKey 为空")
	}
	if credential.Credentials.SessionToken == "" {
		t.Error("临时密钥 SessionToken 为空")
	}

	fmt.Printf("临时密钥信息:\n")
	fmt.Printf("TmpSecretID: %s\n", credential.Credentials.TmpSecretID)
	fmt.Printf("TmpSecretKey: %s\n", credential.Credentials.TmpSecretKey)
	fmt.Printf("SessionToken: %s\n", credential.Credentials.SessionToken)
	fmt.Printf("ExpiredTime: %d\n", credential.ExpiredTime)

	// 测试生成预签名 URL
	presignedURL, _, err := signer.GeneratePresignedURL("jpg")
	if err != nil {
		t.Fatalf("生成预签名 URL 失败: %v", err)
	}

	if presignedURL == "" {
		t.Error("生成的预签名 URL 为空")
	}

	fmt.Printf("预签名 URL: %s\n", presignedURL)
}

// TestSTSSigner_RefreshCredential 测试刷新临时密钥功能
func TestSTSSigner_RefreshCredential(t *testing.T) {
	// 注意：在实际运行测试时，需要替换为有效的配置信息
	config := &STSConfig{
		SecretID:    "your-secret-id",   // 替换为你的 SecretID
		SecretKey:   "your-secret-key",  // 替换为你的 SecretKey
		Bucket:      "your-bucket-name", // 替换为你的存储桶名称
		Region:      "ap-guangzhou",     // 替换为你的存储桶所在地域
		Duration:    1800,               // 临时密钥有效期（秒）
		AllowPrefix: "",                 // 允许操作的路径前缀
		AppID:       "1250000000",       // 替换为你的 APPID
	}

	// 创建 STSSigner 实例
	signer, err := NewSTSSigner(config)
	if err != nil {
		t.Fatalf("创建 STSSigner 失败: %v", err)
	}

	// 获取原始临时密钥信息
	originalCredential := signer.GetCredential()

	// 刷新临时密钥
	err = signer.RefreshCredential(config)
	if err != nil {
		t.Fatalf("刷新临时密钥失败: %v", err)
	}

	// 获取刷新后的临时密钥信息
	newCredential := signer.GetCredential()

	// 验证临时密钥是否已更新
	if originalCredential.Credentials.TmpSecretID == newCredential.Credentials.TmpSecretID {
		t.Error("临时密钥未更新")
	}

	fmt.Printf("原始临时密钥 TmpSecretID: %s\n", originalCredential.Credentials.TmpSecretID)
	fmt.Printf("新临时密钥 TmpSecretID: %s\n", newCredential.Credentials.TmpSecretID)
}

func TestUploadHandler(t *testing.T) {
	// 获取客户端传递的文件后缀
	fileExt := "png" //不含.
	if fileExt == "" {
		println("文件后缀不能为空")
		return
	}

	// 配置STSSigner
	config := &STSConfig{
		SecretID:  "",
		SecretKey: "",
		Bucket:    "1343515495",
		Region:    "na-siliconvalley", //硅谷
		Duration:  1800,               // 30分钟
		AppID:     "1343515495",       // 从 Bucket 中提取的 APPID
	}

	// 初始化STSSigner
	stsSigner, err := NewSTSSigner(config)
	if err != nil {

		println("签名器初始化失败" + err.Error())
		return
	}

	// 生成预签名URL
	presignedURL, file_key, err := stsSigner.GeneratePresignedURL(fileExt)
	if err != nil {
		println("生成预签名URL失败")
		return
	}

	// 获取临时密钥信息
	credential := stsSigner.GetCredential()

	// 返回给客户端
	res := map[string]any{
		"presigned_url":  presignedURL, // 文件url带有鉴权信息的
		"file_key":       file_key,     //保存的相对路径
		"tmp_secret_id":  credential.Credentials.TmpSecretID,
		"tmp_secret_key": credential.Credentials.TmpSecretKey,
		"session_token":  credential.Credentials.SessionToken,
	}

	// presigned_url ：预签名 URL，将签名嵌入到 URL 中生成的签名链接，可用于对象的上传和下载。客户端可直接使用该链接对指定对象进行操作，无需额外的认证请求 2 。
	// file_key ：对象在 COS 存储桶中的唯一标识，类似文件路径。通过 fmt.Sprintf("uploads/%s", fileName) 生成，表明文件存放在 uploads 目录下，文件名由 fileName 确定 4 。
	// tmp_secret_id ：临时密钥 ID，与 tmp_secret_key 和 session_token 配合使用，用于临时访问 COS 服务。该密钥有使用期限，过期后需重新获取，提升了访问的安全性 1 。
	// tmp_secret_key ：临时密钥，与 tmp_secret_id 成对出现，用于对请求进行签名认证，保证请求的合法性和完整性 1 。
	// session_token ：会话令牌，是临时访问凭证的一部分，与临时密钥配合使用，确保临时密钥的安全使用。客户端在请求时需携带该令牌，服务端会验证其有效性

	println(res)
}
