package ginp

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestXxx(t *testing.T) {
	r := gin.Default()
	SetSuccessCode(100)
	r.GET("/", RegisterHandler(index))
	r.Run(":8082")
}

func index(ctx *ContextPlus) {
	ctx.Success()
	// result:
	// {
	// 	"code": 100,
	// 	"msg": "ok"
	// }
}
