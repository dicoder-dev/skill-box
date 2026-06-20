package where

type Extra struct {
	OrderByColumn string `json:"order_by_column,omitempty"`
	OrderByDesc   bool   `json:"order_by_desc,omitempty"`
	PageSize      int    `json:"page_size,omitempty"`
	PageNum       int    `json:"page_num,omitempty"`
}

// 创建一个空的额外参数
func NewExtra() *Extra {
	return &Extra{}
}

// orderByDesc是否倒序
func NewExtraParam(pageSize int, orderByDesc bool) *Extra {
	return &Extra{
		PageSize:      pageSize,
		OrderByDesc:   orderByDesc,
		OrderByColumn: "created_at",
		PageNum:       1,
	}
}

func (s *Extra) PSize(size int) *Extra {
	s.PageSize = size
	return s
}
func (s *Extra) PNum(num int) *Extra {
	s.PageNum = num
	return s
}
func (s *Extra) OrderBy(column string, desc bool) *Extra {
	s.OrderByColumn = column
	s.OrderByDesc = desc
	return s
}
