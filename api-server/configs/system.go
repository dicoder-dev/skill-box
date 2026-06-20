package configs

import "ginp-api/pkg/cfg"

// System 全局配置变量
var System = new(SystemConfig)

// SystemConfig 系统配置
type SystemConfig struct {
	AppName       string `default:"dianji"`
	UserCenterUrl string `default:"http://localhost:8082"`

	// RunMode 部署形态,影响鉴权默认策略与前端运行时配置注入。
	//   web     - Web 端,默认 NeedAuth=true
	//   desktop - 桌面端,默认 NeedAuth=false
	RunMode string `default:"web" configkey:"system.run_mode"`

	// NeedAuth 是否启用 JWT 鉴权中间件。
	//   true  - 业务接口走 JWT 鉴权,失败返回 401
	//   false - 中间件直接放行(桌面端单用户场景)
	NeedAuth bool `default:"true" configkey:"system.need_auth"`
}

func init() {
	cfg.ParseConfigStruct(System)
}