package ginp

import "github.com/gin-gonic/gin"

// routers 存放所有的路由定义
var routers = []RouterItem{}

// RouterGroup 路由分组结构
type RouterGroup struct {
	Prefix      string
	Items       []RouterItem
	Middlewares []gin.HandlerFunc //分组级别中间件
}

const (
	HttpPost  = "POST"
	HttpGet   = "GET"
	HttpPut   = "PUT"
	HttpPatch = "PATCH"
	HttpAny   = "ANY"
)

type RouterItem struct {
	Path           string //api路径
	Handlers       gin.HandlerFunc
	Middlewares    []gin.HandlerFunc //中间件链
	HttpType       string
	NeedLogin      bool         //是否需要登录才能访问
	NeedPermission bool         //是否需要进行权限验证(如果不需要登录，则不会进行权限验证)
	PermissionName string       //权限名称
	Swagger        *SwaggerInfo //swagger信息
}

// SwaggerInfo swagger info
type SwaggerInfo struct {
	Title       string   //接口标题
	Description string   //接口描述
	RequestDto  any      // post [body部分]请求请求结构体，不要传入指针！！！示例:dto.userGet{}
	Consumes    []string //指定【请求】发送的数据类型,默认["application/json"]
	Produces    []string //指定【返回】的数据类型,默认["application/json"]
	IsIgnore    bool     //默认false,是否忽略该接口的生成，true则不会生成
}

// 注册路由
func RegisterRouter(r *gin.Engine) {
	for _, item := range routers {
		handlers := make([]gin.HandlerFunc, 0)
		if len(item.Middlewares) > 0 {
			handlers = append(handlers, item.Middlewares...)
		}
		handlers = append(handlers, item.Handlers)
		switch item.HttpType {
		case HttpPost:
			r.POST(item.Path, handlers...)
		case HttpGet:
			r.GET(item.Path, handlers...)
		case HttpPut:
			r.PUT(item.Path, handlers...)
		case HttpPatch:
			r.PATCH(item.Path, handlers...)
		case HttpAny: //任意请求方式都行
			r.Any(item.Path, handlers...)
		}
	}
}

// AppendRouter 向现有的路由新增
func RouterAppend(item RouterItem) {
	routers = append(routers, item)
}

func RouterFormatCreate(item RouterItem) {
	routers = append(routers, item)
}

// GetAllRouter 获取所有的路由
func GetAllRouter() []RouterItem {
	return routers
}
