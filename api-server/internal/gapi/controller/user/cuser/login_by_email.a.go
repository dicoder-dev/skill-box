package cuser

import (
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

func LoginByEmail(c *ginp.ContextPlus) {
	var requestParams *RequestLoginByEmail
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.FailData("request param error:" + err.Error())
		return
	}

	// 直接调用 service 层的 LoginByEmail 函数
	userInfo, token, err := suser.LoginByEmail(requestParams.Email, requestParams.Password)
	if err != nil {
		c.FailData(err.Error())
		return
	}

	// 返回成功结果
	c.SuccessData(&RespondLogin{
		Token:    token,
		UserInfo: userInfo,
	})
}

const ApiLoginByEmail = "/api/sys_user/login_by_email" //API Path

type RequestLoginByEmail struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RespondLoginByEmail struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiLoginByEmail,                    //api路径
		Handlers:       ginp.RegisterHandler(LoginByEmail), //对应控制器
		HttpType:       ginp.HttpPost,                      //http请求类型
		NeedLogin:      false,                              //是否需要登录
		NeedPermission: false,                              //是否需要鉴权
		PermissionName: "User.login_by_email",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "login_by_email",
			Description: "",
			RequestDto:  RequestLoginByEmail{},
		},
	})
}
