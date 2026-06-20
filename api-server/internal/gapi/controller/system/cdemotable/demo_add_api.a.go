package cdemotable

import (
	"ginp-api/pkg/ginp"
)

func DemoAddApi(c *ginp.ContextPlus) {
	var requestParams *RequestDemoAddApi
	if err := c.ShouldBindJSON(&requestParams); err != nil {
		c.FailData("request param error:" + err.Error())
		return
	}

	//TODO:
	//sdemotable.Model().Create(&entity.DemoTable{})
	//c.SuccessData(&RespondDemoAddApi{})
}

const ApiDemoAddApi = "/api/demo_table/demo_add_api" //API Path

type RequestDemoAddApi struct {
}

type RespondDemoAddApi struct {
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiDemoAddApi,                //api路径
		Handlers:       ginp.RegisterHandler(Create), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      false,                        //是否需要登录
		NeedPermission: false,                        //是否需要鉴权
		PermissionName: "DemoTable.demo_add_api",     //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "demo_add_api",
			Description: "",
			RequestDto:  RequestDemoAddApi{},
		},
	})
}
