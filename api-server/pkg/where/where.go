package where

// Format 组装多个条件，where使用Opt函数组装一个条件
func Format(wheres ...*Condition) []*Condition {
	res := append(
		make([]*Condition, 0),
		wheres...,
	)
	return res
}

// FormatOne 组装1个条件
func FormatOne(FieldName, opt string, val any) []*Condition {
	field := &Condition{
		Field:   FieldName,
		Value:   val,
		Opt:     opt,
		Connect: AND,
	}
	res := append(
		make([]*Condition, 0),
		field,
	)
	return res
}

// FormatOne 组装1个条件
func FormatOneOr(FieldName, opt string, val any) []*Condition {
	field := &Condition{
		Field:   FieldName,
		Value:   val,
		Opt:     opt,
		Connect: OR,
	}
	res := append(
		make([]*Condition, 0),
		field,
	)
	return res
}
