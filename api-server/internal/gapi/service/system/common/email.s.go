package scommon

import (
	"fmt"
	"ginp-api/pkg/email"
	"math/rand"
	"strconv"
	"time"
)

const (
	// 最大每日发送次数
	MaxDailySendCount = 30
	// 最小发送间隔 秒
	MinSendInterval = 30 * time.Second
)

var EmailInstance = EmailService{}

func InitEmailInstance(emailClient *email.EmailClient) {
	EmailInstance.EmailClient = emailClient
}

// 初始化时设置缓存默认过期时间为 5 分钟，清理间隔为 10 分钟
func init() {
	InitCacheInstance(5*time.Minute, 10*time.Minute)
}

type EmailService struct {
	EmailClient *email.EmailClient
}

func (s *EmailService) SendEmail(toEmail, title, content string) error {
	return s.EmailClient.SendEmail(toEmail, title, content)
}

// 发送邮箱验证码
func (s *EmailService) SendCode(toEmail string) error {
	// 检查缓存中是否有该邮箱的发送记录
	lastSendTimeKey := "email_send_time:" + toEmail
	if lastSendTime, found := CacheInstance.Get(lastSendTimeKey); found {
		if time.Since(lastSendTime.(time.Time)) < MinSendInterval {
			return fmt.Errorf("You can only send verification code once per 30 seconds")
		}
	}

	// 检查当天发送次数
	todayKey := "email_send_count:" + toEmail + "_" + time.Now().Format("2006-01-02")
	count, _ := CacheInstance.Get(todayKey)
	var sendCount int
	if count != nil {
		sendCount = count.(int)
	}
	if sendCount >= MaxDailySendCount {
		return fmt.Errorf("you can send a maximum of 30 verification codes per day")
	}

	// 生成一个四位数的随机数字验证码
	// Go 1.20 及以上版本无需调用 rand.Seed，直接使用 rand 包即可
	// 原代码已移除 rand.Seed 调用
	code := rand.Intn(9000) + 1000
	codeStr := strconv.Itoa(code)
	title := "Verify Code : " + codeStr
	htmlContent := "Your Verify Code is: <b>" + codeStr + "</b> ,The validity period is 5 minutes. Please complete the verification within the specified time"
	// 发送验证码邮件
	if err := s.EmailClient.SendEmail(toEmail, title, htmlContent); err != nil {
		return err
	}
	// 将验证码存入缓存，设置过期时间为 5 分钟
	CacheInstance.Set("email_code:"+toEmail, codeStr, 5*time.Minute)

	// 更新发送时间和次数
	CacheInstance.Set(lastSendTimeKey, time.Now(), time.Minute)
	CacheInstance.Set(todayKey, sendCount+1, 24*time.Hour)

	return nil
}

// 验证邮箱验证码
func (s *EmailService) VerifyCode(toEmail, code string) bool {
	cacheKey := "email_code:" + toEmail
	cachedCode, found := CacheInstance.Get(cacheKey)
	if !found {
		return false
	}
	if cachedCode.(string) == code {
		CacheInstance.Delete(cacheKey)
		return true
	}
	return false
}
