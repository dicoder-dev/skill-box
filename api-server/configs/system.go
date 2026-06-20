package configs

import "ginp-api/pkg/cfg"

const (
	ConfigKeySystemAppName       = "system.app.name"
	ConfigKeySystemUserCenterUrl = "system.usercenter.url"
	
)

const (
	defaultSystemAppName       = "dianji"
	defaultSystemUsercenterUrl = "http://localhost:8082"
)

func init() {
	cfg.SetDefault(ConfigKeySystemAppName, defaultSystemAppName)
	cfg.SetDefault(ConfigKeySystemUserCenterUrl, defaultSystemUsercenterUrl)
	
}

func SystemAppName() string {
	return cfg.GetString(ConfigKeySystemAppName)
}

func SystemUserCenterUrl() string {
	return cfg.GetString(ConfigKeySystemUserCenterUrl)
}

