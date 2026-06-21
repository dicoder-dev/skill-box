package cuser

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/service/user/suser"
	"ginp-api/pkg/where"

	"ginp-api/pkg/ginp"
)

type RequestSysUserSearch struct {
	comdto.ReqSearch
}

type RespondSysUserSearch struct {
	List     interface{} `json:"list"`
	Total    uint        `json:"total"`
	PageNum  uint        `json:"page_num"`
	PageSize uint        `json:"page_size"`
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/search",
		Handler:        ginp.BindParamsHandler(SysUserSearch, RequestSysUserSearch{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "SysUse.search",
		Swagger: &ginp.SwaggerInfo{
			Title:         "search user",
			Description:   "",
			RequestParams: comdto.ReqSearch{},
		},
	})
}

func SysUserSearch(c *ginp.ContextPlus, requestParams *RequestSysUserSearch) {
	if where.Check(requestParams.Wheres) != nil {
		c.Fail(where.Check(requestParams.Wheres).Error())
		return
	}
	list, total, err := suser.Model().FindList(requestParams.Wheres, requestParams.Extra)
	if err != nil {
		c.Fail("查询失败" + err.Error())
		return
	}

	resp := &RespondSysUserSearch{
		List:     list,
		Total:    uint(total),
		PageNum:  uint(requestParams.Extra.PageNum),
		PageSize: uint(requestParams.Extra.PageSize),
	}
	c.SuccessData(resp)
}
