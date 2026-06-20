// Package emails
// @Author: zhangdi
// @File: send_test
// @Version: 1.0.0
// @Date: 2023/6/25 15:17
package email

import "testing"

func TestSend(t *testing.T) {

	// 发送邮件
	email := NewEemailClient(EmailConfig{
		Host:          "smtpout.secureserver.net",
		Port:          465,
		Email:         "",
		Pwd:           "",

	})
	err := email.SendEmail("dicoder@126.com", "验证码", "您的验证码为0092")
	if err != nil {
		println(err.Error())
	}

}
