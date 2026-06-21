package router

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {

	//1.------------中间件---------------
	//跨域设置
	r.Use(CORSMiddleware())

	//请求日志记录:控制台打印一行,异步落盘到 ./logs/YYYY-MM/MM-DD-*.txt 并刷新 stats_*.csv
	r.Use(ginp.LoggingMiddleware())

	//2.-----------------路由注册---------------
	// InitRouters()          //路由定义
	ginp.RegisterRouter(r) //注册路由

	//注册公共视图路由
	//registerRouter(r, PublicViewRoutes)

}
