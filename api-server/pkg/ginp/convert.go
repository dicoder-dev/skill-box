package ginp

import "github.com/gin-gonic/gin"

// 将其转换成我们自定义的扩展
func RegisterHandler(handler func(c *ContextPlus)) func(c *gin.Context) {
	return func(c *gin.Context) {
		handler(&ContextPlus{
			Context: c,
		})
	}
}

//使用方式

//1.路由注册
// r.GET("/category/index", ginp.RegisterHandler(controller.CategoryIndex))

//2.控制器调用自定义扩展方法,也可以同时调用gin.context的所有方法
// func CategoryIndex(c *ginp.ContextPro) {
// 	c.OkJson()
// }

//3./category/index返回结果
//{"code":0,"msg":"ok"}
