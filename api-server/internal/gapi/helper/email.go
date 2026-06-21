package helper

import (
	"fmt"
	"ginp-api/configs"
	"ginp-api/pkg/email"
)

// emailClient 邮件客户端实例
var emailClient *email.EmailClient

// initEmailClient 初始化邮件客户端
func initEmailClient() {
	if emailClient != nil {
		return
	}

	host := configs.Email.Client.Host
	port := configs.Email.Client.Port
	account := configs.Email.Client.Account
	pwd := configs.Email.Client.Pwd

	if host == "" || account == "" || pwd == "" {
		fmt.Println("邮箱配置不完整，无法初始化邮件客户端")
		return
	}

	emailClient = email.NewEemailClient(
		email.EmailConfig{
			Host:          host,
			Port:          port,
			Email:         account,
			Pwd:           pwd,
			EmailUserName: account,
		},
	)
}

// getEmailClient 获取邮件客户端
func getEmailClient() *email.EmailClient {
	if emailClient == nil {
		initEmailClient()
	}
	return emailClient
}

// SendEmail 快速发送邮件的助手函数
// 参数：
//   - toEmail: 目标邮箱地址
//   - title: 邮件标题
//   - content: 邮件内容（支持HTML）
//
// 返回值：
//   - error: 发送失败时返回错误
func SendEmail(toEmail, title, content string) error {
	client := getEmailClient()
	if client == nil {
		return fmt.Errorf("邮件客户端未初始化，请检查邮箱配置")
	}
	return client.SendEmail(toEmail, title, content)
}

// SendEmailWithDefaultTitle 使用默认标题发送邮件
// 参数：
//   - toEmail: 目标邮箱地址
//   - content: 邮件内容（支持HTML）
//
// 返回值：
//   - error: 发送失败时返回错误
func SendEmailWithDefaultTitle(toEmail, content string) error {
	return SendEmail(toEmail, "通知", content)
}

// SendEmailWithTemplate 使用模板发送邮件
// 参数：
//   - toEmail: 目标邮箱地址
//   - title: 邮件标题
//   - template: 模板名称
//   - data: 模板数据
//
// 返回值：
//   - error: 发送失败时返回错误
//
// 模板示例：
//   - "order_created": 订单创建通知
//   - "order_paid": 订单支付成功通知
//   - "order_shipped": 订单发货通知
func SendEmailWithTemplate(toEmail, title, template string, data map[string]any) error {
	content := renderEmailTemplate(template, data)
	if content == "" {
		return fmt.Errorf("未知的邮件模板: %s", template)
	}
	return SendEmail(toEmail, title, content)
}

// renderEmailTemplate 渲染邮件模板
func renderEmailTemplate(template string, data map[string]any) string {
	switch template {
	case "order_created":
		return renderOrderCreatedTemplate(data)
	case "order_paid":
		return renderOrderPaidTemplate(data)
	case "order_shipped":
		return renderOrderShippedTemplate(data)
	default:
		return ""
	}
}

// renderOrderCreatedTemplate 订单创建通知模板
func renderOrderCreatedTemplate(data map[string]any) string {
	orderNo := getStringValue(data, "orderNo")
	amount := getStringValue(data, "amount")
	return fmt.Sprintf(`
		<h2>订单创建成功</h2>
		<p>您好，您的订单已成功创建。</p>
		<table style="border-collapse: collapse; width: 100%%; max-width: 600px;">
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">订单号</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">订单金额</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
		</table>
		<p>请尽快完成支付。</p>
	`, orderNo, amount)
}

// renderOrderPaidTemplate 订单支付成功通知模板
func renderOrderPaidTemplate(data map[string]any) string {
	orderNo := getStringValue(data, "orderNo")
	amount := getStringValue(data, "amount")
	return fmt.Sprintf(`
		<h2>支付成功</h2>
		<p>您好，您的订单已支付成功。</p>
		<table style="border-collapse: collapse; width: 100%%; max-width: 600px;">
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">订单号</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">支付金额</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
		</table>
		<p>感谢您的购买！</p>
	`, orderNo, amount)
}

// renderOrderShippedTemplate 订单发货通知模板
func renderOrderShippedTemplate(data map[string]any) string {
	orderNo := getStringValue(data, "orderNo")
	expressCompany := getStringValue(data, "expressCompany")
	expressNo := getStringValue(data, "expressNo")
	return fmt.Sprintf(`
		<h2>订单已发货</h2>
		<p>您好，您的订单已发货。</p>
		<table style="border-collapse: collapse; width: 100%%; max-width: 600px;">
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">订单号</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">快递公司</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
			<tr>
				<td style="padding: 8px; border: 1px solid #ddd;">快递单号</td>
				<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
			</tr>
		</table>
		<p>请留意查收。</p>
	`, orderNo, expressCompany, expressNo)
}

// getStringValue 从map中获取字符串值
func getStringValue(data map[string]any, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
