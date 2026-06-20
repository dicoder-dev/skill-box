package cuser

import (
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

const ApiUpdateUserInfo = "/api/user/update_user_info" //API Path

func UpdateUserInfo(c *ginp.ContextPlus) {
	var requestParams *RequestUpdateUserInfo
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.Fail("request param error:" + err.Error())
		return
	}
	if requestParams.NewPwd != "" {
		//因为UserInfo.Password的json标签是-，因此无法直接获取，需要单独获取
		requestParams.UserInfo.Password = requestParams.NewPwd
	}

	err := suser.UpdateUserInfo(requestParams.UserInfo, requestParams.EmailCode)
	if err != nil {
		c.Fail(err.Error())
		return
	}

	c.Success()
}

type RequestUpdateUserInfo struct {
	EmailCode string       `json:"email_code"`
	NewPwd    string       `json:"new_pwd"`
	UserInfo  *entity.User `json:"user_info"`
}

type RespondUpdateUserInfo struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUpdateUserInfo,                    //api路径
		Handlers:       ginp.RegisterHandler(UpdateUserInfo), //对应控制器
		HttpType:       ginp.HttpPost,                        //http请求类型
		NeedLogin:      false,                                //是否需要登录
		NeedPermission: false,                                //是否需要鉴权
		PermissionName: "user.update_user_info",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "update_user_info",
			Description: "",
			RequestDto:  RequestUpdateUserInfo{},
		},
	})
}
