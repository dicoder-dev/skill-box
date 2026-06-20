// Package swagen
// @Author: zhangdi
// @File: gen_swa_test
// @Version: 1.0.0
// @Date: 2023/10/30 16:18
package swagen

import (
	"fmt"
	"ginp-api/internal/gapi/router"
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

	s.AddApis(router.GetAllRouter())

	err := s.Save("")
	if err != nil {
		fmt.Println("initSwagger error :" + err.Error())
		return
	}
}
