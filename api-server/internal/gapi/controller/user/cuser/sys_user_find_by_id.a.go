package cuser

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/ginp"
)

type RequestSysUserFindById struct {
	ID uint `json:"id"`
}

type RespondSysUserFindById struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/findById",
		Handler:        ginp.BindParamsHandler(SysUserFindById, RequestSysUserFindById{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "SysUse.findById",
		Swagger: &ginp.SwaggerInfo{
			Title:         "find user by id",
			Description:   "",
			RequestParams: comdto.ReqFindById{},
		},
	})
}

func SysUserFindById(c *ginp.ContextPlus, requestParams *RequestSysUserFindById) {
	info, err := suser.Model().FindOneById(requestParams.ID)
	if err != nil {
		c.Fail(err.Error())
		return
	}
	c.SuccessData(info)
}
