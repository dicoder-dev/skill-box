# ginp

## ContextPlus API

<div align="right">
  <a href="#english">English</a> | 
  <a href="#chinese">中文</a>
</div>

<a id="chinese"></a>
### 简介
`ContextPlus` 是对 `gin.Context` 的扩展，提供了更便捷的响应方法。


### 安装
```bash
go get github.com/DicoderCn/ginp
```
### 快速开始
```go
package main

import (
	"github.com/DicoderCn/ginp"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
  // Set success code, default is 200
	ginp.SetSuccessCode(200) 
	r.GET("/", ginp.RegisterHandler(Index))
	r.Run(":8082")
}

func Index(c *ginp.ContextPlus) {
	c.Success()
}
```

### 方法列表

#### Success
```go
func (c *ContextPlus) Success(messages ...string)
```
返回成功JSON响应
- `messages`: 可选的成功消息

#### Fail
```go
func (c *ContextPlus) Fail(strs ...string)
```
返回失败JSON响应
- `strs`: 可选的错误消息

#### SuccessData
```go
func (c *ContextPlus) SuccessData(data any, extra any, messages ...string)
```
返回带数据的成功JSON响应
- `data`: 主要数据
- `extra`: 额外数据
- `messages`: 可选的成功消息

#### FailData
```go
func (c *ContextPlus) FailData(data any, extra any, messages ...string)
```
返回带数据的失败JSON响应
- `data`: 主要数据
- `extra`: 额外数据
- `messages`: 可选的错误消息

#### SuccessHtml
```go
func (c *ContextPlus) SuccessHtml(path string)
```
返回HTML响应
- `path`: HTML模板路径

#### R
```go
func (c *ContextPlus) R(code int, obj any)
```
通用JSON响应方法
- `code`: HTTP状态码
- `obj`: 响应对象

### 参数设置方法
#### SetSuccessCode
```go
func SetSuccessCode(code int)
```
Sets the success code, default is 200
- `code`: HTTP status code
#### SetFailCode
```go
func SetFailCode(code int)  
```
Sets the failure code, default is 400
- `code`: HTTP status code
#### SetSuccessMessage
```go
func SetSuccessMessage(message string)
```
Sets the success message, default is "success"
- `message`: Success message
#### SetFailMessage
```go
func SetFailMessage(message string)
```
Sets the failure message, default is "fail"
- `message`: Failure message  