package where

import "log"

type whereManager struct {
	wheres []*Condition
}

const AND = "AND"
const OR = "OR"

// New 新建实例 连接方式为AND/ 新建实例，接受可选的 connect 参数
func New(fieldName, opt string, value any, connect ...string) *whereManager {
	conn := AND
	if len(connect) > 0 {
		conn = connect[0]
	}
	if conn != AND && conn != OR {
		log.Fatal("New() connect参数错误")
		return nil
	}
	conditions := FormatOne(fieldName, opt, value)
	if conn == OR {
		conditions = FormatOneOr(fieldName, opt, value)
	}

	for _, c := range conditions {
		c.Connect = conn
	}
	return &whereManager{wheres: conditions}
}

// 新增一个 AND 查询条件
func (w *whereManager) And(fieldName, opt string, value any) *whereManager {
	conditions := FormatOne(fieldName, opt, value)
	for _, c := range conditions {
		c.Connect = AND
	}
	w.wheres = append(w.wheres, conditions...)
	return w
}

// 新增一个 OR 查询条件
func (w *whereManager) Or(fieldName, opt string, value any) *whereManager {
	conditions := FormatOne(fieldName, opt, value)
	for _, c := range conditions {
		c.Connect = AND
	}
	w.wheres = append(w.wheres, conditions...)
	return w
}

// Conditions 输出查询条件
func (w *whereManager) Conditions() []*Condition {
	return w.wheres
}
