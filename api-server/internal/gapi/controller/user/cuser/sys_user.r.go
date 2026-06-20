package cuser

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"

	"ginp-api/pkg/ginp"
)

const (
	ApiUserCreate   = "/api/sys_user/create"
	ApiUserFindById = "/api/sys_user/findById"
	ApiUserSearch   = "/api/sys_user/search"
	ApiUserUpdate   = "/api/sys_user/update"
	ApiUserDelete   = "/api/sys_user/delete"
)

// this is router define file
func init() {

	// Create
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUserCreate,                //api路径
		Handlers:       ginp.RegisterHandler(Create), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      false,                        //是否需要登录
		NeedPermission: false,                        //是否需要鉴权
		PermissionName: "SysUse.create",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "create user",
			Description: "",
			RequestDto:  entity.User{},
		},
	})

	// FindById
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUserFindById,                //api路径
		Handlers:       ginp.RegisterHandler(FindByID), //对应控制器
		HttpType:       ginp.HttpPost,                  //http请求类型
		NeedLogin:      false,                          //是否需要登录
		NeedPermission: false,                          //是否需要鉴权
		PermissionName: "SysUse.findById",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "find user by id",
			Description: "",
			RequestDto:  entity.User{},
		},
	})

	// 修改
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUserUpdate,                //api路径
		Handlers:       ginp.RegisterHandler(Update), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "SysUse.update",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "modify user",
			Description: "",
			RequestDto:  entity.User{},
		},
	})

	// 删除
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUserDelete,                //api路径
		Handlers:       ginp.RegisterHandler(Delete), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "SysUse.delete",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "delet user",
			Description: "",
			RequestDto:  entity.User{},
		},
	})

	// search 搜索
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUserSearch,                //api路径
		Handlers:       ginp.RegisterHandler(Search), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      false,                        //是否需要登录
		NeedPermission: false,                        //是否需要鉴权
		PermissionName: "SysUse.search",              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "search user",
			Description: "",
			RequestDto:  comdto.ReqSearch{},
		},
	})

}
