package cuser

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"

	"ginp-api/pkg/ginp"
)

// this is router define file
func init() {

	// Create
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/create",                       //api路径
		Handler:        ginp.BindParamsHandler(Create, entity.User{}), //对应控制器
		HttpType:       ginp.HttpPost,                                //http请求类型
		NeedLogin:      false,                                        //是否需要登录
		NeedPermission: false,                                        //是否需要鉴权
		PermissionName: "SysUse.create",                              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "create user",
			Description:   "",
			RequestParams: entity.User{},
		},
	})

	// FindById
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/findById",                         //api路径
		Handler:        ginp.BindParamsHandler(FindByID, comdto.ReqFindById{}), //对应控制器
		HttpType:       ginp.HttpPost,                                    //http请求类型
		NeedLogin:      false,                                            //是否需要登录
		NeedPermission: false,                                            //是否需要鉴权
		PermissionName: "SysUse.findById",                                //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "find user by id",
			Description:   "",
			RequestParams: comdto.ReqFindById{},
		},
	})

	// 修改
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/update",                       //api路径
		Handler:        ginp.BindParamsHandler(Update, entity.User{}), //对应控制器
		HttpType:       ginp.HttpPost,                                //http请求类型
		NeedLogin:      true,                                         //是否需要登录
		NeedPermission: true,                                         //是否需要鉴权
		PermissionName: "SysUse.update",                              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "modify user",
			Description:   "",
			RequestParams: entity.User{},
		},
	})

	// 删除
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/delete",                            //api路径
		Handler:        ginp.BindParamsHandler(Delete, comdto.ReqDelete{}), //对应控制器
		HttpType:       ginp.HttpPost,                                     //http请求类型
		NeedLogin:      true,                                              //是否需要登录
		NeedPermission: true,                                              //是否需要鉴权
		PermissionName: "SysUse.delete",                                   //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "delet user",
			Description:   "",
			RequestParams: comdto.ReqDelete{},
		},
	})

	// search 搜索
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/sys_user/search",                          //api路径
		Handler:        ginp.BindParamsHandler(Search, comdto.ReqSearch{}), //对应控制器
		HttpType:       ginp.HttpPost,                                     //http请求类型
		NeedLogin:      false,                                             //是否需要登录
		NeedPermission: false,                                             //是否需要鉴权
		PermissionName: "SysUse.search",                                   //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "search user",
			Description:   "",
			RequestParams: comdto.ReqSearch{},
		},
	})

}