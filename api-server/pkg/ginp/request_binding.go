package ginp

import (
	"fmt"
)

// BindResult 绑定结果
type BindResult struct {
	Success bool        // 是否成功
	Data    interface{} // 绑定后的数据
	Error   error       // 错误信息
	Message string      // 用户友好的错误消息
}

// BindJSON 从 JSON 请求体中绑定参数
// 如果绑定失败，返回 BindResult.Success=false 和错误信息
// 调用者应该检查 Success 字段并据此返回错误响应
//
// 使用示例：
//   result := ginp.BindJSON(ctx, &params)
//   if !result.Success {
//       ctx.Fail(result.Message)
//       return
//   }
func BindJSON(ctx *ContextPlus, data interface{}) *BindResult {
	if err := ctx.ShouldBindJSON(&data); err != nil {
		return &BindResult{
			Success: false,
			Error:   err,
			Message: "请求参数有误: " + err.Error(),
		}
	}
	return &BindResult{
		Success: true,
		Data:    data,
	}
}

// BindQuery 从 URL Query 参数中绑定参数
func BindQuery(ctx *ContextPlus, data interface{}) *BindResult {
	if err := ctx.ShouldBindQuery(&data); err != nil {
		return &BindResult{
			Success: false,
			Error:   err,
			Message: "查询参数有误: " + err.Error(),
		}
	}
	return &BindResult{
		Success: true,
		Data:    data,
	}
}

// BindURI 从 URL 路径参数中绑定参数
func BindURI(ctx *ContextPlus, data interface{}) *BindResult {
	if err := ctx.ShouldBindUri(&data); err != nil {
		return &BindResult{
			Success: false,
			Error:   err,
			Message: "路径参数有误: " + err.Error(),
		}
	}
	return &BindResult{
		Success: true,
		Data:    data,
	}
}

// MustBindJSON 强制从 JSON 绑定，如果失败则自动返回错误响应
// 使用此函数会更简洁，但失去了对错误的控制权
//
// 使用示例：
//   params := &CreateRequest{}
//   if !ginp.MustBindJSON(ctx, params) {
//       return  // 错误响应已自动发送
//   }
//   // 继续处理业务逻辑
func MustBindJSON(ctx *ContextPlus, data interface{}) bool {
	result := BindJSON(ctx, data)
	if !result.Success {
		ctx.Fail(result.Message)
		return false
	}
	return true
}

// MustBindQuery 强制从 Query 绑定，如果失败则自动返回错误响应
func MustBindQuery(ctx *ContextPlus, data interface{}) bool {
	result := BindQuery(ctx, data)
	if !result.Success {
		ctx.Fail(result.Message)
		return false
	}
	return true
}

// MustBindURI 强制从 URI 绑定，如果失败则自动返回错误响应
func MustBindURI(ctx *ContextPlus, data interface{}) bool {
	result := BindURI(ctx, data)
	if !result.Success {
		ctx.Fail(result.Message)
		return false
	}
	return true
}

// ValidateStruct 验证结构体（需要在 struct tag 中定义 validate 标签）
// 需要引入 github.com/go-playground/validator/v10
//
// 使用示例：
//   type CreateRequest struct {
//       Name string `json:"name" validate:"required,min=1,max=100"`
//       Age  int    `json:"age" validate:"required,min=0,max=150"`
//   }
//   errors := ginp.ValidateStruct(params)
//   if len(errors) > 0 {
//       ctx.Fail("参数验证失败: " + errors[0])
//       return
//   }
func ValidateStruct(data interface{}) []string {
	// 注意：这是一个占位符实现
	// 实际使用需要引入 validator 库
	// 可以在后续的版本中完善这个功能
	return []string{}
}

// BindAndValidate 绑定并验证参数
func BindAndValidate(ctx *ContextPlus, data interface{}) (*BindResult, []string) {
	result := BindJSON(ctx, data)
	if !result.Success {
		return result, nil
	}
	
	// TODO: 集成验证逻辑
	validationErrors := ValidateStruct(data)
	if len(validationErrors) > 0 {
		return &BindResult{
			Success: false,
			Message: "参数验证失败: " + validationErrors[0],
		}, validationErrors
	}
	
	return result, nil
}

// OperationHandler 操作处理器类型
// 定义了一个通用的处理器接口，可以根据操作类型进行不同的处理
type OperationHandler func(ctx *ContextPlus, operationType OperationType, params interface{}) error

// OperationRegistry 操作处理器注册表
type OperationRegistry struct {
	handlers map[OperationType]OperationHandler
}

// NewOperationRegistry 创建新的操作注册表
func NewOperationRegistry() *OperationRegistry {
	return &OperationRegistry{
		handlers: make(map[OperationType]OperationHandler),
	}
}

// Register 注册操作处理器
func (r *OperationRegistry) Register(opType OperationType, handler OperationHandler) {
	r.handlers[opType] = handler
}

// Execute 执行操作处理器
func (r *OperationRegistry) Execute(ctx *ContextPlus, opType OperationType, params interface{}) error {
	handler, exists := r.handlers[opType]
	if !exists {
		return fmt.Errorf("operation %s not registered", opType)
	}
	return handler(ctx, opType, params)
}
