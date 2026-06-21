package swagen

import (
	"fmt"
	"ginp-api/pkg/ginp"
	"testing"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

func TestTypeParse(t *testing.T) {
	//infos := GetStructSchemaInfo(User{})
	//println(infos)
}

func TestGenSwagger(t *testing.T) {
	s := NewSwaGen(SwaggerInfo{
		Host:        "/",
		Title:       "api docs --2023",
		Version:     "2.0",
		Description: "测试",
	})

	// 创建模拟路由数据用于测试
	mockRouters := []ginp.RouterItem{
		{
			Path:     "/api/test",
			HttpType: ginp.HttpGet,
			Swagger: &ginp.SwaggerInfo{
				Title:       "测试接口",
				Description: "这是一个测试接口",
			},
		},
	}
	s.AddApis(mockRouters)

	err := s.Save("")
	if err != nil {
		fmt.Println("initSwagger error :" + err.Error())
		return
	}
}
