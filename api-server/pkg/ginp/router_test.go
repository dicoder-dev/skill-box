package ginp

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRouter(t *testing.T) {
	// 测试基本路由注册
	r := gin.Default()

	RouterAppend(RouterItem{
		Path:     "/test",
		Handlers: RegisterHandler(mockHandler),
		HttpType: HttpGet,
	})

	RouterAppend(RouterItem{
		Path:     "/test1",
		Handlers: RegisterHandler(mockHandler),
		HttpType: HttpGet,
	})

	// 注册路由
	RegisterRouter(r)

	// 测试路由是否注册成功
	// w := httptest.NewRecorder()
	// req, _ := http.NewRequest("GET", "/test", nil)
	// r.ServeHTTP(w, req)

}

func mockHandler(c *ContextPlus) {
	c.String(http.StatusOK, "OK")
}
