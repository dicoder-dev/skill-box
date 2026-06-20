// Package swagen
// @Author: zhangdi
// @File: func
// @Version: 1.0.0
// @Date: 2023/10/30 17:07
package swagen

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"ginp-api/pkg/ginp"
)

// GetStructSchemaInfo 获取一个结构体的 Schema 同时返回实体名称
func GetStructSchemaInfo(struct_ any) Schema {
	infos := make(map[string]PropertiedInfo, 0)
	val := reflect.ValueOf(struct_)
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		tag := field.Tag.Get("json")
		description := getTagSwaggerKey(field.Tag.Get("swa"), "desc")

		tagArr := strings.Split(tag, ",")
		jsonName := field.Name
		//isRequire := "true"
		if len(tagArr) > 0 {
			jsonName = tagArr[0]
		}
		typeStr := toSwaggerTypeByString(field.Type.String())

		info := PropertiedInfo{
			Type:        typeStr,
			Description: description,
			Example:     getExampleValue(field, val.Field(i)),
		}
		infos[jsonName] = info
	}
	//获取类型

	schema := Schema{
		Type:       "object",
		Properties: infos,
	}
	return schema
}

// 获取 swa，如desc值
// `json:"msg"  swa:"desc:提示消息;"` kv使用:分割，多个kv使用分号;分割
func getTagSwaggerKey(tagSwagger string, k string) string {
	arr := strings.Split(tagSwagger, ";")
	for _, str := range arr {
		itemArr := strings.Split(str, ":")
		if len(itemArr) == 2 && itemArr[0] == k {
			return itemArr[1]
		}
	}
	return ""
}

// 转成swagger 的类型
func toSwaggerTypeByString(golangType string) string {
	switch golangType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "[]string":
		return "array"
	default:
		return "object"
	}
}

func getExampleValue(field reflect.StructField, val reflect.Value) any {
	switch field.Type.String() {
	case "string":
		return val.String()
	case
		"uint", "uint8", "uint16", "uint32", "uint64":
		return val.Uint()
	case "int", "int8", "int16", "int32", "int64":
		return val.Int()
	case "float32", "float64":
		return val.Float()
	case "bool":
		return val.Bool()
	case "[]string":
		return "array"
	default:
		return ""
	}
}

// 获取请求类型
func getConsumes(swagger *ginp.SwaggerInfo) []string {
	if swagger == nil || swagger.Consumes == nil {
		return []string{"application/json"}
	}
	return swagger.Consumes
}
func getProduces(swagger *ginp.SwaggerInfo) []string {
	if swagger == nil || swagger.Consumes == nil {
		return []string{"application/json"}
	}
	return swagger.Consumes
}

// 获取接口标题
func getTitle(r ginp.RouterItem) string {
	if r.Swagger == nil {
		arr := strings.Split(r.Path, "/")
		return fmt.Sprintf("%s %s", arr[len(arr)-1], arr[len(arr)-2])
	}
	return r.Swagger.Title
}

// 获取标签（分组）
func getTags(r ginp.RouterItem) []string {
	if r.Path == "" {
		return make([]string, 0)
	}
	arr := strings.Split(r.Path, "/")
	return []string{arr[len(arr)-2]}
}

// 获取接口描述
func getDescription(r ginp.RouterItem) string {
	if r.Swagger == nil {
		arr := strings.Split(r.Path, "/")
		return fmt.Sprintf("%s %s", arr[len(arr)-1], arr[len(arr)-2])
	}
	return r.Swagger.Description
}

// NameToLine 下划线命名
func NameToLine(camel string) string {
	var snake []rune
	for i, c := range camel {
		if unicode.IsUpper(c) {
			if i > 0 && (i+1 < len(camel) && unicode.IsLower(rune(camel[i+1])) || unicode.IsLower(rune(camel[i-1]))) {
				snake = append(snake, '_')
			}
			snake = append(snake, unicode.ToLower(c))
		} else {
			snake = append(snake, c)
		}
	}
	return string(snake)
}
