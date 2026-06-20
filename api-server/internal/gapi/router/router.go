package router

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {

	//1.------------中间件---------------
	//跨域设置
	r.Use(CORSMiddleware())

	//登录鉴权检验
	// r.Use(ginp.RegisterHandler(AuthorizationCheck))

	//权限验证
	// r.Use(ginp.ConvHandler(permissionCheck))

	//2.-----------------路由注册---------------
	// InitRouters()          //路由定义
	ginp.RegisterRouter(r) //注册路由

	//注册公共视图路由
	//registerRouter(r, PublicViewRoutes)

}
