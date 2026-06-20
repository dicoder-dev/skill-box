package swagen

// SwaggerInfo 结构体用于保存Swagger中的Info部分
// 与 ginp.SwaggerInfo 兼容
type SwaggerInfo struct {
	Host        string `yaml:"host"`
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// ParamInfo 参数信息
type ParamInfo struct {
	In     string
	Name   string
	Schema map[string]string
}

// Schema Swagger Schema 定义
type Schema struct {
	Type       string
	Properties map[string]PropertiedInfo
}

// PropertiedInfo 属性信息
type PropertiedInfo struct {
	Description string
	Example     any
	Type        string
}

// RespondData 默认返回的数据格式
// 与 ginp-api 的响应格式兼容
type RespondData struct {
	Code uint   `json:"code" swa:"desc:状态码1正常,0异常,401未登录,403无操作权限;"`
	Msg  string `json:"msg"  swa:"desc:提示消息;"`
	Data any    `json:"data,omitempty"  swa:"desc:返回的数据;"`
}
