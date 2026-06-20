// Package swagen
// @Author: zhangdi
// @File: struct
// @Version: 1.0.0
// @Date: 2023/10/30 17:05
package swagen

// SwaggerInfo 结构体用于保存Swagger中的Info部分
type SwaggerInfo struct {
	Host        string `yaml:"host"`
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type ParamInfo struct {
	In     string
	Name   string
	Schema map[string]string
}

type Schema struct {
	Type       string
	Properties map[string]PropertiedInfo
}

type PropertiedInfo struct {
	Description string
	Example     any
	Type        string
}

// RespondData 默认返回的数据格式
type RespondData struct {
	Code uint   `json:"code" swa:"desc:状态码1正常,0异常,401未登录,403无操作权限;"`
	Msg  string `json:"msg"  swa:"desc:提示消息;"`
	Data any    `json:"data,omitempty"  swa:"desc:返回的数据;"`
}
