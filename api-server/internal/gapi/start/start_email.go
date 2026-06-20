package start

import (
	"ginp-api/configs"
	"ginp-api/pkg/email"
)

var emailClient *email.EmailClient

func InitEmail() {
	emailClient = email.NewEemailClient(
		email.EmailConfig{
			Host:  configs.EmailClientHost(),    // 邮箱服务器地址，例如 smtp.qq.com
			Port:  configs.EmailClientPort(),    // 邮箱服务器端口，例如 465
			Email: configs.EmailClientAccount(), // 发件人邮箱地址
			Pwd:   configs.EmailClientPwd(),     // 发件人邮箱密码
		},
	)
}

func GetEmailClient() *email.EmailClient {
	if emailClient == nil {
		InitEmail()
		if emailClient == nil {
			panic("email client is nil")
		}
	}
	return emailClient
}
