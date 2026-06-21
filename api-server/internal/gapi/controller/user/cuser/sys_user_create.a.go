package cuser

import (
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

type RequestSysUserCreate struct {
	entity.User
}

type RespondSysUserCreate struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/create",
		Handler:        ginp.BindParamsHandler(SysUserCreate, RequestSysUserCreate{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "SysUse.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "create user",
			Description:   "",
			RequestParams: RequestSysUserCreate{},
		},
	})
}

func SysUserCreate(c *ginp.ContextPlus, requestParams *RequestSysUserCreate) {
	info, err := suser.Model().Create(&requestParams.User)
	if err != nil {
		c.Fail("创建失败" + err.Error())
		return
	}
	c.SuccessData(info)
}
