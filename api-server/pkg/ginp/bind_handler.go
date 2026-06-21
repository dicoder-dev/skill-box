package ginp

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

// 将其转换成我们自定义的扩展
func BindHandler(handler func(c *ContextPlus)) func(c *gin.Context) {
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




// BindParamsHandler 自动参数绑定处理器包装函数
// 将原始的 handler 包装起来，自动执行参数绑定
// 如果绑定失败，直接返回错误响应，不执行原始 handler
//
// 使用示例：
//   func Create(ctx *ginp.ContextPlus, params *entity.User) error {
//       // 直接处理业务逻辑，params 已经从请求中绑定
//       return service.Create(params)
//   }
//
//   ginp.RouterAppend(ginp.RouterItem{
//       Path:     "/api/user/create",
//       Handler:  ginp.BindParamsHandler(Create, &entity.User{}),
//       ParamTypes: []interface{}{&entity.User{}}, // 自动提取到SwaggerInfo.RequestParams
//       // ...
//   })
func BindParamsHandler(handler interface{}, paramTypes ...interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &ContextPlus{Context: c}
		
		// 绑定参数
		// params 数组长度 = 1 (ctx) + len(paramTypes)
		params := make([]reflect.Value, 1+len(paramTypes))
		params[0] = reflect.ValueOf(ctx)
		
		// 绑定每个参数类型
		for i, paramType := range paramTypes {
			// 创建参数实例
			paramInstance := reflect.New(reflect.TypeOf(paramType).Elem()).Interface()
			
			// 绑定参数
			var err error
			switch c.Request.Method {
			case "POST", "PUT", "PATCH":
				err = c.ShouldBindJSON(paramInstance)
			case "GET":
				err = c.ShouldBindQuery(paramInstance)
			default:
				err = c.ShouldBindJSON(paramInstance)
			}
			
			// 参数绑定失败，直接返回错误
			if err != nil {
				ctx.Fail("请求参数有误: " + err.Error())
				return
			}
			
			// 注意：params[0] 是 ctx，所以 paramTypes[i] 对应 params[i+1]
			params[i+1] = reflect.ValueOf(paramInstance)
		}
		
		// 调用原始 handler
		handlerFunc := reflect.ValueOf(handler)
		results := handlerFunc.Call(params)
		
		// 处理返回值
		if len(results) > 0 {
			if errVal := results[len(results)-1]; errVal.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !errVal.IsNil() {
					err := errVal.Interface().(error)
					ctx.Fail(err.Error())
					return
				}
			}
		}
		
		// 检查响应是否已发送（handler 中可能已经调用了 ctx.Success/SuccessData/Fail 等）
		// 如果已发送响应，不再自动调用 ctx.Success()
		if c.Writer.Written() {
			return
		}
		
		// 默认返回成功
		ctx.Success()
	}
}
