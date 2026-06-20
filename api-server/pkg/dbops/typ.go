package dbops

import (
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

type RelationItem struct {
	RelationName string
	//Preload 用于预加载符合条件的订单,无关主表查询，如果需要跟主表关联，请使用join
	Wheres []*where.Condition
}

type FindListConfig struct {
	Conditions     []*where.Condition
	Extra          *where.Extra
	NewEntityList  any
	GetSoftDelData bool
	Db             *gorm.DB
	// ReloationNum   int //关联表数量
	RelationList []*RelationItem
	Fields       []string
	// ReloationAttrName1 string //关联表1没有则填空
	// ReloationAttrName2 string //关联表2没有则填空
	// ReloationAttrName3 string //关联表3 没有则填空
}

type FindOneConfig struct {
	Fields         []string //要查询的字段，不穿则表示查询所有
	Wheres         []*where.Condition
	NewEntity      any
	Db             *gorm.DB
	getSoftDelData bool
	RelationList   []*RelationItem
	// ReloationAttrName1 string //关联表1没有则填空
	// ReloationAttrName2 string //关联表2没有则填空
	// ReloationAttrName3 string //关联表3 没有则填空
}

// 删除
type DeleteConfig struct {
	NewEntity    any
	Wheres       []*where.Condition
	SoftDelete   bool
	Db           *gorm.DB
	RelationList []string
}

// 更新
type UpdateConfNew struct {
	Db               *gorm.DB
	NewEntity        any
	DataUpdate       any
	Wheres           []*where.Condition
	UpdateColumnsCfg []string
}

// // 获取关联数量
// func (s *FindOneConfig) GetRelationTotal() int {
// 	total := 0
// 	if s.ReloationAttrName1 != "" {
// 		total += 1
// 	}

// 	if s.ReloationAttrName2 != "" {
// 		total += 1
// 	}

// 	if s.ReloationAttrName3 != "" {
// 		total += 1
// 	}

// 	return total
// }

// // 获取关联数量
// func (s *FindListConfig) GetRelationTotal() int {
// 	total := 0
// 	if s.ReloationAttrName1 != "" {
// 		total += 1
// 	}

// 	if s.ReloationAttrName2 != "" {
// 		total += 1
// 	}

// 	if s.ReloationAttrName3 != "" {
// 		total += 1
// 	}

// 	return total
// }
