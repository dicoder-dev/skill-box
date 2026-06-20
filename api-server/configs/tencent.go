package configs

import "ginp-api/pkg/cfg"

const (
	//SecretID 腾讯云API密钥ID
	ConfigKeyTencentCosSecretID = "tencent.cos.secret_id"
	//SecretKey 腾讯云API密钥Key
	ConfigKeyTencentCosSecretKey = "tencent.cos.secret_key"
	//BucketName 存储桶名称
	ConfigKeyTencentCosBucketName = "tencent.cos.bucket_name"
	//BucketAppID 存储桶appid
	ConfigKeyTencentCosBucketAppId = "tencent.cos.bucket_appid"
	//Region 区域
	ConfigKeyTencentCosRegion = "tencent.cos.region"
	//Duration 签名有效期
	ConfigKeyTencentCosDuration = "tencent.cos.duration"
	//AllowPrefix 允许访问的前缀
	ConfigKeyTencentCosAllowPrefix = "tencent.cos.allow_prefix"
)

// 设置默认值
func init() {
	cfg.SetDefault(ConfigKeyTencentCosSecretID, "")
	cfg.SetDefault(ConfigKeyTencentCosSecretKey, "")
	cfg.SetDefault(ConfigKeyTencentCosBucketName, "")
	cfg.SetDefault(ConfigKeyTencentCosRegion, "")
	cfg.SetDefault(ConfigKeyTencentCosDuration, 0)
	cfg.SetDefault(ConfigKeyTencentCosAllowPrefix, "")
	cfg.SetDefault(ConfigKeyTencentCosBucketAppId, "")
}

func TencentCosSecretID() string {
	return cfg.GetString(ConfigKeyTencentCosSecretID)
}

func TencentCosSecretKey() string {
	return cfg.GetString(ConfigKeyTencentCosSecretKey)
}

func TencentCosBucketName() string {
	return cfg.GetString(ConfigKeyTencentCosBucketName)
}

func TencentCosRegion() string {
	return cfg.GetString(ConfigKeyTencentCosRegion)
}

func TencentCosDuration() int {
	return cfg.GetInt(ConfigKeyTencentCosDuration)
}

func TencentCosAllowPrefix() string {
	return cfg.GetString(ConfigKeyTencentCosAllowPrefix)
}

func TencentCosBucketAppId() string {
	return cfg.GetString(ConfigKeyTencentCosBucketAppId)
}
