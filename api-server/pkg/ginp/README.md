
## ContextPlus API

<div align="right">
  <a href="https://github.com/DicoderCn/ginp">English</a>
  |
  <a href="https://github.com/DicoderCn/ginp/blob/master/README_zh.md">中文</a>
</div>

<a id="english"></a>
### introduction
`ContextPlus` is an extension of `gin.Context` that provides more convenient response methods.

### install
```bash
go get github.com/DicoderCn/ginp
```
### usage
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
### Context Method List

#### Success
```go
func (c *ContextPlus) Success(messages ...string)
```
Returns success JSON response
- `messages`: Optional success messages

#### Fail
```go
func (c *ContextPlus) Fail(strs ...string)
```
Returns failure JSON response
- `strs`: Optional error messages

#### SuccessData
```go
func (c *ContextPlus) SuccessData(data any, extra any, messages ...string)
```
Returns success JSON response with data
- `data`: Main data
- `extra`: Additional data
- `messages`: Optional success messages

#### FailData
```go
func (c *ContextPlus) FailData(data any, extra any, messages ...string)
```
Returns failure JSON response with data
- `data`: Main data
- `extra`: Additional data
- `messages`: Optional error messages

#### SuccessHtml
```go
func (c *ContextPlus) SuccessHtml(path string)
```
Returns HTML response
- `path`: HTML template path

#### R
```go
func (c *ContextPlus) R(code int, obj any)
```
General JSON response method
- `code`: HTTP status code
- `obj`: Response object

### Setting Methods 
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

  