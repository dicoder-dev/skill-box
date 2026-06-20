package configs

import "ginp-api/pkg/cfg"

const (
	ConfigKeyServerPort = "server.port"
)

const (
	defaultServerPort = "8082"
)

// 初始化配置
func init() {
	cfg.SetDefault(ConfigKeyServerPort, defaultServerPort)
}

// 获取服务端口
func ServerPort() string {
	port := cfg.GetString(ConfigKeyServerPort)
	if port == "" {
		return defaultServerPort
	}
	return port
}
