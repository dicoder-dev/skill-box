package cindex

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

func IndexView(c *ginp.ContextPlus) {
	c.HTML(200, "index.html", gin.H{})
}
