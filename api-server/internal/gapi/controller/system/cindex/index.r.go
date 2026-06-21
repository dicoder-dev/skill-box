package cindex

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

type RequestIndex struct{}

type RespondIndex struct{}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/",
		Handler:        ginp.BindParamsHandler(IndexView, RequestIndex{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
	})
}

func IndexView(c *ginp.ContextPlus, requestParams *RequestIndex) {
	c.HTML(200, "index.html", gin.H{})
}
