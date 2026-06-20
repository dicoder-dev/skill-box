package cdemotable

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"

	"ginp-api/pkg/ginp"
)

const (
	ApiCreate   = "/api/demo_table/create"
	ApiFindById = "/api/demo_table/findById"
	ApiSearch   = "/api/demo_table/search"
	ApiUpdate   = "/api/demo_table/update"
	ApiDelete   = "/api/demo_table/delete"
)

// this is router define file
func init() {

	// Create
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiCreate,                    //api路径
		Handlers:       ginp.RegisterHandler(Create), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      false,                        //是否需要登录
		NeedPermission: false,                        //是否需要鉴权
		PermissionName: "DemoTable.create",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "create demo_table",
			Description: "",
			RequestDto:  entity.DemoTable{},
		},
	})

	// FindById
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiFindById,                    //api路径
		Handlers:       ginp.RegisterHandler(FindByID), //对应控制器
		HttpType:       ginp.HttpPost,                  //http请求类型
		NeedLogin:      false,                          //是否需要登录
		NeedPermission: false,                          //是否需要鉴权
		PermissionName: "DemoTable.findById",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "find demo_table by id",
			Description: "",
			RequestDto:  entity.DemoTable{},
		},
	})

	// 修改
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUpdate,                    //api路径
		Handlers:       ginp.RegisterHandler(Update), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "DemoTable.update",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "modify demo_table",
			Description: "",
			RequestDto:  entity.DemoTable{},
		},
	})

	// 删除
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiDelete,                    //api路径
		Handlers:       ginp.RegisterHandler(Delete), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "DemoTable.delete",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "delet demo_table",
			Description: "",
			RequestDto:  entity.DemoTable{},
		},
	})

	// search 搜索
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiSearch,                    //api路径
		Handlers:       ginp.RegisterHandler(Search), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "DemoTable.search",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "search demo_table",
			Description: "",
			RequestDto:  comdto.ReqSearch{},
		},
	})

}
