package ginp

import (
	"log"

	"github.com/gin-gonic/gin"
)

// ContextPlus ContextPlus PLUS
type ContextPlus struct {
	*gin.Context
}

// Success 返回OK,形式为JSON
func (c *ContextPlus) Success(messages ...string) {
	c.R(codeHttpSuccess, gin.H{
		"code": codeOk,
		"msg":  formatSuccessMsg(messages...),
	})
}

// Fail 返回ERROR,形式为JSON
func (c *ContextPlus) Fail(strs ...string) {
	c.R(codeHttpFail, gin.H{
		"code": codeFail,
		"msg":  formatFailMsg(strs...),
	})
}

// FailData 返回OK,形式为JSON
func (c *ContextPlus) FailData(data any, messages ...string) {
	c.R(codeHttpFail, gin.H{
		"code": codeFail,
		"msg":  formatFailMsg(messages...),
		"data": data,
	})
}

// SuccessData 返回OK,形式为JSONextra为任意类型数据。
// extra使用场景：data是固定结构体形式，无法再添加字段时可以将其他信息传到extra中，
// 如直接传map,嫌map麻烦也可以是第一个传key，第二个参数val，
// 前端自己处理业务逻辑（前段收到的extra字段是数组形式）
func (c *ContextPlus) SuccessData(data any, messages ...string) {
	c.R(codeHttpSuccess, gin.H{
		"code": codeOk,
		"msg":  formatSuccessMsg(messages...),
		"data": data,
	})
}

func (c *ContextPlus) SuccessHtml(path string) {
	c.HTML(codeHttpSuccess, path, gin.H{})
}

// R RespondJson 返回JSON,形式为JSON
func (c *ContextPlus) R(code int, obj any) {
	c.Log(obj)
	c.JSON(code, obj)
}

func (c *ContextPlus) Log(data any) {
	if showLog == false {
		return
	}

	// 生成日志格式并记录
	log.Printf("%s %s %s %d  user_id:%v request:%+v respond:%+v",
		c.ClientIP(),
		c.Request.Method,
		c.Request.URL.Path,
		c.Writer.Status(),
		0,
		c.Request.Form,
		data,
	)
}
