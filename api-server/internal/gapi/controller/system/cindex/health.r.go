package cindex

import (
	"ginp-api/pkg/ginp"
)

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:      "/api/health",
		Handlers:  ginp.RegisterHandler(Health),
		HttpType:  ginp.HttpGet,
		NeedLogin: false,
	})
}
