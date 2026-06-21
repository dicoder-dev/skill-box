package cuser

import (
	"ginp-api/internal/gapi/entity"
	scommon "ginp-api/internal/gapi/service/system/common"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/user/register_by_email",                                     //api路径
		Handler:        ginp.BindParamsHandler(RegisterByEmail, RequestRegisterByEmail{}), //对应控制器
		HttpType:       ginp.HttpPost,                                                     //http请求类型
		NeedLogin:      false,                                                             //是否需要登录
		NeedPermission: false,                                                             //是否需要鉴权
		PermissionName: "user.register_by_email",                                          //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "register_by_email",
			Description:   "",
			RequestParams: RequestRegisterByEmail{},
		},
	})
}

// 验证邮箱格式
// func ValidateEmail(email string) bool {
// 	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
// 	match, _ := regexp.MatchString(pattern, email)
// 	return match
// }

func RegisterByEmail(c *ginp.ContextPlus, requestParams *RequestRegisterByEmail) {
	//验证验证码
	isPass := scommon.EmailInstance.VerifyCode(requestParams.Email, requestParams.EmailCode)
	if !isPass {
		c.Fail("email verify code error")
		return
	}
	userInfo, token, err := suser.Register(&entity.User{
		Username: requestParams.Username,
		Password: requestParams.Password,
		Email:    requestParams.Email,
	})
	if err != nil {
		c.Fail("fail:" + err.Error())
		return
	}
	c.SuccessData(&RespondLogin{
		Token:    token,
		UserInfo: userInfo,
	})
}

type RequestRegisterByEmail struct {
	Email     string `json:"email" binding:"required,email"`
	EmailCode string `json:"email_code" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type RespondRegisterByEmail struct {
}
