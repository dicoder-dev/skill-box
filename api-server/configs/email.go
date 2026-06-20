package configs

import "ginp-api/pkg/cfg"

const ConfigKeyEmailClientAccount = "email.client.account"
const ConfigKeyEmailClientPwd = "email.client.pwd"
const ConfigKeyEmailClientPort = "email.client.port"
const ConfigKeyEmailClientHost = "email.client.host"

func init() {
	cfg.SetDefault(ConfigKeyEmailClientAccount, "dicoder@126.com")
	cfg.SetDefault(ConfigKeyEmailClientPwd, "12345")
	cfg.SetDefault(ConfigKeyEmailClientPort, 465)
	cfg.SetDefault(ConfigKeyEmailClientHost, "smtp.126.com")
}

func EmailClientAccount() string {
	return cfg.GetString(ConfigKeyEmailClientAccount)
}
func EmailClientPwd() string {
	return cfg.GetString(ConfigKeyEmailClientPwd)
}
func EmailClientPort() int {
	return cfg.GetInt(ConfigKeyEmailClientPort)
}
func EmailClientHost() string {
	return cfg.GetString(ConfigKeyEmailClientHost)
}
