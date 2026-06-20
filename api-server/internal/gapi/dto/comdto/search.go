package comdto

import "ginp-api/pkg/where"

// ReqSearch 搜索参数
type ReqSearch struct {
	//字段条件列表
	Wheres []*where.Condition `json:"wheres,omitempty"`
	//分页和排序信息
	Extra *where.Extra `json:"extra"`
}

// RespSearch 搜索返回
type RespSearch struct {
	List     any  `json:"list"`
	Total    uint `json:"total"`
	PageNum  uint `json:"pageNum,omitempty"`
	PageSize uint `json:"pageSize,omitempty"`
}
