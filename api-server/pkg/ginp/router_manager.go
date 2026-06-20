package ginp

import (
	"sync"

	"github.com/gin-gonic/gin"
)

// routers 存放所有的路由定义
var (
	routers = []RouterItem{}
	routerMutex sync.RWMutex
)

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
	AliasePaths        []string        //路径别名列表，共同指向同一个Handlers，可用于兼容旧接口路径
	Handler       gin.HandlerFunc
	Middlewares    []gin.HandlerFunc //中间件链
	HttpType       string
	NeedLogin      bool         //是否需要登录才能访问
	NeedPermission bool         //是否需要进行权限验证(如果不需要登录，则不会进行权限验证)
	PermissionName string       //权限名称，推荐父级目录.接口名称.功能.api 如 system.permission.sync_api_permission.api
	OperationType  OperationType //操作类型(CREATE, READ, UPDATE, DELETE, SEARCH 等)
	Swagger        *SwaggerInfo //swagger信息
	ParamTypes     []interface{} // 存储handler的参数类型元数据（用于自动提取到RequestParams）
}

// SwaggerInfo swagger info
type SwaggerInfo struct {
	Title       string   //接口标题
	Description string   //接口描述
	RequestParams  any      // post [body部分]请求请求结构体，不要传入指针！！！示例:dto.userGet{}
	Consumes    []string //指定【请求】发送的数据类型,默认["application/json"]
	Produces    []string //指定【返回】的数据类型,默认["application/json"]
	IsIgnore    bool     //默认false,是否忽略该接口的生成，true则不会生成
}

// 注册路由
func RegisterRouter(r *gin.Engine) {
	for _, item := range GetAllRouter() {
		handlers := make([]gin.HandlerFunc, 0)
		if len(item.Middlewares) > 0 {
			handlers = append(handlers, item.Middlewares...)
		}
		handlers = append(handlers, item.Handler)
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

		// 处理别名路由，按相同的 HttpType 进行注册
		if len(item.AliasePaths) > 0 {
			for _, alias := range item.AliasePaths {
				switch item.HttpType {
				case HttpPost:
					r.POST(alias, handlers...)
				case HttpGet:
					r.GET(alias, handlers...)
				case HttpPut:
					r.PUT(alias, handlers...)
				case HttpPatch:
					r.PATCH(alias, handlers...)
				case HttpAny:
					r.Any(alias, handlers...)
				}
			}
		}
	}
}

// HandlerWithMetadata 用于存储handler和其参数类型元数据
type HandlerWithMetadata struct {
	Handler     gin.HandlerFunc
	ParamTypes  []interface{} // 存储handler的参数类型（除了ContextPlus）
}

// AppendRouter 向现有的路由新增，支持并发安全的路由注册
func RouterAppend(item RouterItem) {
	// 自动提取handler的参数类型到SwaggerInfo.RequestParams
	if item.Swagger != nil && item.Swagger.RequestParams == nil && len(item.ParamTypes) > 0 {
		// 使用第一个参数类型作为RequestParams
		item.Swagger.RequestParams = item.ParamTypes[0]
	}
	
	routerMutex.Lock()
	defer routerMutex.Unlock()
	routers = append(routers, item)
}

// GetAllRouter 获取所有的路由（线程安全）
func GetAllRouter() []RouterItem {
	routerMutex.RLock()
	defer routerMutex.RUnlock()
	// 返回副本避免并发修改
	routes := make([]RouterItem, len(routers))
	copy(routes, routers)
	return routes
}
