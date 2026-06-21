package ginp

import (
	"context"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

// GetUserID 解析当前请求中由鉴权中间件写入的 jwt_user 上下文,返回 user id。
// 鉴权中间件(use_ auth_user_center.go)将 lestrrat-go/jwx 的 jwt.Token 放入
// gin.Context 的 "jwt_user" 键;这里兼容 jwt.Token 与旧版 map[string]any 两种形态。
// 解析失败(未登录 / token 异常)返回 0,调用方应自行处理未登录分支。
func (c *ContextPlus) GetUserID() uint {
	claims := c.getJWTClaims()
	if claims == nil {
		return 0
	}
	return extractUserIDFromClaims(claims)
}

// getJWTClaims 从 gin.Context 中取出 jwt_user 并尽可能反序列化为 map。
func (c *ContextPlus) getJWTClaims() map[string]interface{} {
	if tokenInterface, exists := c.Get("jwt_user"); exists {
		// 优先按 jwx 的 jwt.Token 处理
		if token, ok := tokenInterface.(jwt.Token); ok {
			if claims, err := token.AsMap(context.Background()); err == nil {
				return claims
			}
		}
		// 兼容旧版 map 形态
		if claims, ok := tokenInterface.(map[string]interface{}); ok {
			return claims
		}
	}
	return nil
}

// extractUserIDFromClaims 从 JWT claims 中提取 user id,字段名约定为 "id"。
func extractUserIDFromClaims(claims map[string]interface{}) uint {
	if uid, exists := claims["id"]; exists {
		switch v := uid.(type) {
		case float64:
			return uint(v)
		case int:
			return uint(v)
		case int64:
			return uint(v)
		case uint:
			return v
		case uint64:
			return uint(v)
		case string:
			if id, err := strconv.ParseUint(v, 10, 32); err == nil {
				return uint(id)
			}
		}
	}
	return 0
}
