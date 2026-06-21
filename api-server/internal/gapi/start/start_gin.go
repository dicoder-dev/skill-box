package start

import (
	"ginp-api/configs"
	"ginp-api/internal/gapi/router"

	"github.com/gin-gonic/gin"
)

func startGinServer() {
	//配置路径已定死在 cmd/gapi/configs.yaml ,要修改只能去改cfg包
	//由于cionfigs包调用了init()初始化 因此使用cfg.initCfg函数可能会无效
	r := gin.Default()

	// 设置模板路径
	r.LoadHTMLGlob("view/*")

	// r.LoadHTMLFiles("view/*")
	//我们注册了 "/static" 路径，并指定其对应的静态文件目录为 "./static"
	r.Static("/static", "./static")
	//我们注册了 "/assets" 路径，并指定其对应的静态文件目录为 "/templates/index/assets"
	r.Static("/assets", "./static/assets")

	router.Register(r)
	println("start server on port: " + configs.Server.Port)
	r.Run(":" + configs.Server.Port)
}
