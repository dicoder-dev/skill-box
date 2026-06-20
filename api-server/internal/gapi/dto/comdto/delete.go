package comdto

// ReqDelete 删除参数
type ReqDelete struct {
	ID uint `json:"id" validate:"required"`
}
