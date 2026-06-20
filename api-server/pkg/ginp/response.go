package ginp

// ApiResponse 标准 API 响应体结构
// 前端和 SDK 可基于此结构生成类型定义
type ApiResponse struct {
	Code    interface{} `json:"code"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
	Payload interface{} `json:"payload,omitempty"` // 用于额外信息
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}, messages ...string) *ApiResponse {
	msg := successMsgDefault
	if len(messages) > 0 {
		msg = ""
		for _, message := range messages {
			msg += message
		}
	}
	return &ApiResponse{
		Code: codeOk,
		Msg:  msg,
		Data: data,
	}
}

// NewFailResponse 创建失败响应
func NewFailResponse(messages ...string) *ApiResponse {
	msg := failMsgDefault
	if len(messages) > 0 {
		msg = ""
		for _, message := range messages {
			msg += message
		}
	}
	return &ApiResponse{
		Code: codeFail,
		Msg:  msg,
	}
}

// NewFailResponseWithData 创建带数据的失败响应
func NewFailResponseWithData(data interface{}, messages ...string) *ApiResponse {
	msg := failMsgDefault
	if len(messages) > 0 {
		msg = ""
		for _, message := range messages {
			msg += message
		}
	}
	return &ApiResponse{
		Code: codeFail,
		Msg:  msg,
		Data: data,
	}
}
