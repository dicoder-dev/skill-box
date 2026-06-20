package cdemotable

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"

	"ginp-api/pkg/ginp"
)

// this is router define file
func init() {

	// Create
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/create",                      //api路径
		Handler:        ginp.BindParamsHandler(Create, entity.DemoTable{}), //对应控制器
		HttpType:       ginp.HttpPost,                                   //http请求类型
		NeedLogin:      false,                                           //是否需要登录
		NeedPermission: false,                                           //是否需要鉴权
		PermissionName: "DemoTable.create",                              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "create demo_table",
			Description:   "",
			RequestParams: entity.DemoTable{},
		},
	})

	// FindById
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/findById",                         //api路径
		Handler:        ginp.BindParamsHandler(FindByID, comdto.ReqFindById{}), //对应控制器
		HttpType:       ginp.HttpPost,                                    //http请求类型
		NeedLogin:      false,                                            //是否需要登录
		NeedPermission: false,                                            //是否需要鉴权
		PermissionName: "DemoTable.findById",                              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "find demo_table by id",
			Description:   "",
			RequestParams: entity.DemoTable{},
		},
	})

	// 修改
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/update",                      //api路径
		Handler:        ginp.BindParamsHandler(Update, entity.DemoTable{}), //对应控制器
		HttpType:       ginp.HttpPost,                                   //http请求类型
		NeedLogin:      true,                                            //是否需要登录
		NeedPermission: true,                                            //是否需要鉴权
		PermissionName: "DemoTable.update",                              //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "modify demo_table",
			Description:   "",
			RequestParams: entity.DemoTable{},
		},
	})

	// 删除
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/delete",                           //api路径
		Handler:        ginp.BindParamsHandler(Delete, comdto.ReqDelete{}), //对应控制器
		HttpType:       ginp.HttpPost,                                     //http请求类型
		NeedLogin:      true,                                              //是否需要登录
		NeedPermission: true,                                              //是否需要鉴权
		PermissionName: "DemoTable.delete",                                 //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "delet demo_table",
			Description:   "",
			RequestParams: comdto.ReqDelete{},
		},
	})

	// search 搜索
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/search",                         //api路径
		Handler:        ginp.BindParamsHandler(Search, comdto.ReqSearch{}), //对应控制器
		HttpType:       ginp.HttpPost,                                     //http请求类型
		NeedLogin:      true,                                              //是否需要登录
		NeedPermission: true,                                              //是否需要鉴权
		PermissionName: "DemoTable.search",                                 //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "search demo_table",
			Description:   "",
			RequestParams: comdto.ReqSearch{},
		},
	})

}