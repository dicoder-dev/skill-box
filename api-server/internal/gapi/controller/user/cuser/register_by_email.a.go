package cuser

import (
	"ginp-api/internal/gapi/entity"
	scommon "ginp-api/internal/gapi/service/system/common"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

const ApiRegisterByEmail = "/api/user/register_by_email" //API Path

func RegisterByEmail(c *ginp.ContextPlus) {
	var requestParams *RequestRegisterByEmail
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.Fail("request param error:" + err.Error())
		return
	}

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
	//TODO:
	//suser.Model().Create(&entity.user{})
	//c.SuccessData(&RespondRegisterByEmail{})
}

type RequestRegisterByEmail struct {
	Email     string `json:"email" binding:"required,email"`
	EmailCode string `json:"email_code" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type RespondRegisterByEmail struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiRegisterByEmail,                    //api路径
		Handlers:       ginp.RegisterHandler(RegisterByEmail), //对应控制器
		HttpType:       ginp.HttpPost,                         //http请求类型
		NeedLogin:      false,                                 //是否需要登录
		NeedPermission: false,                                 //是否需要鉴权
		PermissionName: "user.register_by_email",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "register_by_email",
			Description: "",
			RequestDto:  RequestRegisterByEmail{},
		},
	})
}
