package configs

import "ginp-api/pkg/cfg"

// System 全局配置变量
var System = new(SystemConfig)

// SystemConfig 系统配置
//
// 部署形态(runMode = "web" / "desktop")不再放这里。运行形态由启动命令
// 显式声明(BootOptions.RunMode / web:dev:frontend env),单源真相,避免
// 配置文件 + 启动命令双源歧义。
type SystemConfig struct {
	AppName       string `default:"dianji"`
	UserCenterUrl string `default:"http://localhost:8082"`

	// NeedAuth 是否启用 JWT 鉴权中间件。
	//   true  - 业务接口走 JWT 鉴权,失败返回 401
	//   false - 中间件直接放行(桌面端单用户场景)
	NeedAuth bool `default:"true" configkey:"system.need_auth"`
}

func init() {
	cfg.ParseConfigStruct(System)
}