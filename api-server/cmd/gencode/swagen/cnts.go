// Package swagen
// @Author: zhangdi
// @File: cnts
// @Version: 1.0.0
// @Date: 2023/10/30 16:32
package swagen

const (
	infoHost        = "host"
	infoTitle       = "info.title"
	infoVersion     = "info.version"
	infoDescription = "info.description"
)

const (
	apiPathMethod                            = "consumes"
	apiPathConsumes                          = "consumes" //指定请求的数据类型
	apiPathSummary                           = "summary"  //标题
	apiPathDescription                       = "description"
	apiPathProduces                          = "produces" //指定了API返回的数据类型为application/json
	apiPathResponsesOKDescription            = "responses.200.description"
	apiPathResponsesOKSchema                 = "responses.200.schema"
	apiPathResponsesNotLoginDescription      = "responses.401.description"
	apiPathResponsesNotPermissionDescription = "responses.403.description"
	//apiPathResponsesOKSchema      = "responses.200.schema"

	apiPathTags                           = "tags"                            //分组，取实体名
	apiAPathParameterBodyIn               = "parameters[0].in"                //参数列表
	apiAPathParameterBodyName             = "parameters[0].name"              //参数列表
	apiAPathParameterBodySchemaType       = "parameters[0].schema.type"       //参数列表
	apiAPathParameterBodySchemaProperties = "parameters[0].schema.properties" //参数列表
)

//前缀都是：{path}.{method}.
const (
	paramInBody       = "body"     //表示参数位于HTTP请求的请求体中（通常用于POST请求）
	paramInQuery      = "query"    //如：/users?id=123
	paramInPath       = "path"     //它定义了一个路径 /users/{id}
	paramInHeader     = "header"   //Authorization: Bearer xyz123
	paramInFormData   = "formData" //表示参数位于HTTP请求的表单数据中（通常用于表单提交）
	paramInFormCookie = "cookie"   //表示参数位于HTTP请求的cookie中，例如 Cookie: session_id=abc123
)
