package configs

// TencentCosBucketName 返回腾讯云 COS 存储桶名称。
// 用于 pkg/cos 直接读取配置,避免调用方反复深拷贝 Tencent.Cos。
func TencentCosBucketName() string {
	return Tencent.Cos.BucketName
}

// TencentCosBucketAppID 返回腾讯云 COS 存储桶 AppID。
func TencentCosBucketAppID() string {
	return Tencent.Cos.BucketAppID
}

// TencentCosRegion 返回腾讯云 COS 区域(例如 ap-guangzhou)。
func TencentCosRegion() string {
	return Tencent.Cos.Region
}

// TencentCosSecretID 返回腾讯云 API 密钥 ID。
func TencentCosSecretID() string {
	return Tencent.Cos.SecretID
}

// TencentCosSecretKey 返回腾讯云 API 密钥 Key。
func TencentCosSecretKey() string {
	return Tencent.Cos.SecretKey
}

// TencentCosDuration 返回临时密钥有效期(秒),<=0 表示走长期密钥方案。
func TencentCosDuration() int {
	return Tencent.Cos.Duration
}

// TencentCosAllowPrefix 返回允许上传/操作的路径前缀(用于 STS 权限收敛)。
func TencentCosAllowPrefix() string {
	return Tencent.Cos.AllowPrefix
}
