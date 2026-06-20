package cdemotable

import (
	"ginp-api/pkg/ginp"
)

// RequestDemoAddApi demo 自定义接口的请求参数
type RequestDemoAddApi struct {
}

// RespondDemoAddApi demo 自定义接口的响应参数
type RespondDemoAddApi struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/demo_table/demo_add_api",                  //api路径
		Handler:        ginp.BindParamsHandler(DemoAddApi, RequestDemoAddApi{}), //对应控制器
		HttpType:       ginp.HttpPost,                                     //http请求类型
		NeedLogin:      false,                                             //是否需要登录
		NeedPermission: false,                                             //是否需要鉴权
		PermissionName: "DemoTable.demo_add_api",                          //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:         "demo_add_api",
			Description:   "",
			RequestParams: RequestDemoAddApi{},
		},
	})
}

func DemoAddApi(c *ginp.ContextPlus, requestParams *RequestDemoAddApi) {
	//TODO:
	//sdemotable.Model().Create(&entity.DemoTable{})
	//c.SuccessData(&RespondDemoAddApi{})
}