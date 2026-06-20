package configs

import "ginp-api/pkg/cfg"

// Tencent 全局配置变量
var Tencent = new(TencentConfig)

type TencentConfig struct {
	Cos Cos `default:""`
}

// Cos 腾讯云COS配置
type Cos struct {
	SecretID    string `default:""`
	SecretKey   string `default:""`
	BucketName  string `default:""`
	BucketAppID string `default:""`
	Region      string `default:""`
	Duration    int    `default:"0"`
	AllowPrefix string `default:""`
}

func init() {
	cfg.ParseConfigStruct(Tencent)
}