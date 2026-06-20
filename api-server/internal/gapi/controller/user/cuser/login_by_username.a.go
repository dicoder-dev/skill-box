package cuser

import (
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

func LoginByUsername(c *ginp.ContextPlus) {
	var requestParams *RequestLoginByUsername
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.FailData("request param error:" + err.Error())
		return
	}
	userInfo, token, err := suser.LoginByUsername(requestParams.Username, requestParams.Password)
	if err != nil {
		c.FailData(err.Error())
		return
	}
	c.SuccessData(&RespondLogin{
		Token:    token,
		UserInfo: userInfo,
	})
}

const ApiLoginByUsername = "/api/sys_user/login_by_username" //API Path

type RequestLoginByUsername struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RespondLoginByUsername struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiLoginByUsername,                    //api路径
		Handlers:       ginp.RegisterHandler(LoginByUsername), //对应控制器
		HttpType:       ginp.HttpPost,                         //http请求类型
		NeedLogin:      false,                                 //是否需要登录
		NeedPermission: false,                                 //是否需要鉴权
		PermissionName: "User.login_by_username",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "login_by_username",
			Description: "",
			RequestDto:  RequestLoginByUsername{},
		},
	})
}
