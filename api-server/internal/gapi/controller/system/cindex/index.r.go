package cindex

import (
	"ginp-api/pkg/ginp"
)

func init() {
	// this is view
	ginp.RouterAppend(ginp.RouterItem{
		Path:      "/",                             //api路径
		Handlers:  ginp.RegisterHandler(IndexView), //对应控制器
		HttpType:  ginp.HttpGet,                    //http请求类型
		NeedLogin: false,                           //是否需要登录
	})
}
