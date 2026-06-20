package ccommon

import (
	scommon "ginp-api/internal/gapi/service/system/common"
	"regexp"

	"ginp-api/pkg/ginp"
)

const ApiSendEmailCode = "/api/common/send_email_code" //API Path

// 验证邮箱格式
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

func SendEmailCode(c *ginp.ContextPlus) {
	var requestParams *RequestSendEmailCode
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.Fail("request param error:" + err.Error())
		return
	}

	//检查邮箱格式
	if !ValidateEmail(requestParams.Email) {
		c.Fail("email format error")
		return
	}

	err := scommon.EmailInstance.SendCode(requestParams.Email)
	if err != nil {
		c.Fail("send email code error:" + err.Error())
		return
	}

	c.Success("send email code success")
}

type RequestSendEmailCode struct {
	Type  string `json:"type"`
	Email string `json:"email"`
}

type RespondSendEmailCode struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiSendEmailCode,                    //api路径
		Handlers:       ginp.RegisterHandler(SendEmailCode), //对应控制器
		HttpType:       ginp.HttpPost,                       //http请求类型
		NeedLogin:      false,                               //是否需要登录
		NeedPermission: false,                               //是否需要鉴权
		PermissionName: "common.send_email_code",            //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "send_email_code",
			Description: "",
			RequestDto:  RequestSendEmailCode{},
		},
	})
}
