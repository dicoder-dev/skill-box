package cuser

import (
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/user/suser"
	"ginp-api/pkg/where"

	"ginp-api/pkg/ginp"
)

type RequestSysUserUpdate struct {
	entity.User
}

type RespondSysUserUpdate struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/update",
		Handler:        ginp.BindParamsHandler(SysUserUpdate, RequestSysUserUpdate{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      true,
		NeedPermission: true,
		PermissionName: "SysUse.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "modify user",
			Description:   "",
			RequestParams: RequestSysUserUpdate{},
		},
	})
}

func SysUserUpdate(c *ginp.ContextPlus, requestParams *RequestSysUserUpdate) {
	wheres := where.Format(where.OptEqual("id", requestParams.User.ID))
	err := suser.Model().Update(wheres, &requestParams.User)
	if err != nil {
		c.Fail("修改失败" + err.Error())
		return
	}
	c.Success()
}
