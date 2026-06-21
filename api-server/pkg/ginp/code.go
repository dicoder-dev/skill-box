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

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelOff LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

var currentLogLevel = LogLevelInfo

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

// SetLogLevel 设置日志级别
func SetLogLevel(level LogLevel) {
	currentLogLevel = level
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
