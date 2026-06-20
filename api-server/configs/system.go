package configs

import "ginp-api/pkg/cfg"

// System 全局配置变量
var System = new(SystemConfig)

// SystemConfig 系统配置
type SystemConfig struct {
	AppName       string `default:"dianji"`
	UserCenterUrl string `default:"http://localhost:8082"`
}

func init() {
	cfg.ParseConfigStruct(System)
}