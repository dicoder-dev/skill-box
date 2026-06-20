package configs

import "ginp-api/pkg/cfg"

// Server 全局配置变量
var Server = new(ServerConfig)

// ServerConfig 服务配置
type ServerConfig struct {
	Port string `default:"8082"`
}

func init() {
	cfg.ParseConfigStruct(Server)
}