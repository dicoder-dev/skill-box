package ginp

import (
	"context"
	"fmt"
	"reflect"
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

func (c *ContextPlus) GetUserID() uint {
	claims := c.getJWTClaims()
	if claims == nil {
		return 0
	}
	return c.extractUserIDFromClaims(claims)
}

// GetAppKey 获取应用Key，从请求头中读取，默认值为 "common"
func (c *ContextPlus) GetAppKey() string {
	appKey := c.GetHeader("app_key")
	if appKey == "" {
		return "common"
	}
	return appKey
}

// GetAppVersion 获取应用版本，从请求头中读取，默认值为 "1.0.0"
func (c *ContextPlus) GetAppVersion() string {
	appVersion := c.GetHeader("app_version")
	if appVersion == "" {
		return "1.0.0"
	}
	return appVersion
}

// getJWTClaims 获取 JWT claims（根据不同类型自动怀旧）
func (c *ContextPlus) getJWTClaims() map[string]interface{} {
	if tokenInterface, exists := c.Get("jwt_user"); exists {
		// 优先核寸：新的 JWT token 对象
		if token, ok := tokenInterface.(jwt.Token); ok {
			if claims, err := token.AsMap(context.Background()); err == nil {
				return claims
			}
		}
		// 析受：旧版本 map 类型
		if claims, ok := tokenInterface.(map[string]interface{}); ok {
			return claims
		}
	}
	return nil
}

// extractUserIDFromClaims 从 claims 中提取用户 ID
func (c *ContextPlus) extractUserIDFromClaims(claims map[string]interface{}) uint {
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

	codeVal, msgVal := extractCodeAndMsg(data)
	fmt.Printf("path:%s status:%d code:%v msg:%s\n",
		c.Request.URL.Path,
		c.Writer.Status(),
		codeVal,
		msgVal,
	)
}

func (c *ContextPlus) GetApiList() []RouterItem {
	return routers
}

// extractCodeAndMsg best-effort extraction of code/msg from response payload.
func extractCodeAndMsg(data any) (any, string) {
	switch v := data.(type) {
	case map[string]any:
		return v["code"], fmt.Sprint(v["msg"])
	case gin.H:
		return v["code"], fmt.Sprint(v["msg"])
	}

	val := reflect.ValueOf(data)
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, ""
		}
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		codeField := val.FieldByName("Code")
		msgField := val.FieldByName("Msg")
		var code any
		if codeField.IsValid() && codeField.CanInterface() {
			code = codeField.Interface()
		}
		if msgField.IsValid() && msgField.CanInterface() {
			return code, fmt.Sprint(msgField.Interface())
		}
		return code, ""
	}

	return nil, fmt.Sprint(data)
}
