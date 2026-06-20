package ginp

import "net/http"

var codeOk any
var codeFail any
var codeNoLogin any

// HttpSuccess 成功的http状态码
var codeHttpSuccess int
var codeHttpFail int

var showLog = true
var successMsgDefault = "success"
var failMsgDefault = "fail"

// 初始化 code
func init() {
	SetFailCode(0)
	SetSuccessCode(1)
	SetNoLoginCode(401)
	codeHttpSuccess = http.StatusOK
	codeHttpFail = http.StatusOK
}

func SetSuccessMsg(msg string) {
	successMsgDefault = msg
}
func SetFailMsg(msg string) {
	failMsgDefault = msg
}
func SetSuccessCode(code any) {
	codeOk = code
}
func SetShowLog(show bool) {
	showLog = show
}

func SetSuccessHttpCode(code int) {
	codeHttpSuccess = code
}

func SetFailHttpCode(code int) {
	codeHttpFail = code
}

func SetFailCode(code any) {
	codeFail = code
}
func SetNoLoginCode(code any) {
	codeNoLogin = code
}
