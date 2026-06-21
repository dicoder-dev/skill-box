package where

// RelationSearch 关联搜索配置
type RelationSearch struct {
	RelationName   string              `json:"relation_name"`   // 关联表名
	RelationFidName string             `json:"relation_fid_name"` // 外键字段名
	Where          []*Condition `json:"where"`           // 搜索条件
}

