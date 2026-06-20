package router

import (
	"ginp-api/pkg/ginp"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func init() {

	url := ginSwagger.URL("/static/docs/swagger.yaml") // 设置Swagger接口文档的URL路径
	//访问方式：域名/swagger/index.html
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/swagger/*any",
		Handlers:       ginSwagger.WrapHandler(swaggerfiles.Handler, url),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "",
		Swagger: &ginp.SwaggerInfo{
			IsIgnore: true, //跳过该接口
		},
	})
}

func GetAllRouter() []ginp.RouterItem {
	return ginp.GetAllRouter()
}
