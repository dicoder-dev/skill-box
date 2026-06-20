package ifthen

// Number 模拟三元运算符，支持数字类型的泛型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Number[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64](condition bool, trueValue, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// Bool 模拟三元运算符，支持布尔类型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Bool(condition bool, trueValue, falseValue bool) bool {
	if condition {
		return trueValue
	}
	return falseValue
}

// String 模拟三元运算符，支持字符串类型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func String(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

// Any 模拟三元运算符，支持任意类型的泛型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Any[T any](condition bool, trueValue, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// Ptr 模拟三元运算符，支持指针类型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Ptr[T any](condition bool, trueValue, falseValue *T) *T {
	if condition {
		return trueValue
	}
	return falseValue
}

// Slice 模拟三元运算符，支持切片类型
// condition: 条件判断
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Slice[T any](condition bool, trueValue, falseValue []T) []T {
	if condition {
		return trueValue
	}
	return falseValue
}

// Func 模拟三元运算符，支持函数类型
// condition: 条件判断
// trueValue: 条件为真时返回的函数
// falseValue: 条件为假时返回的函数
func Func[T any](condition bool, trueValue, falseValue func() T) func() T {
	if condition {
		return trueValue
	}
	return falseValue
}

// FuncWithArgs 模拟三元运算符，支持带参数的函数类型
// condition: 条件判断
// trueValue: 条件为真时返回的函数
// falseValue: 条件为假时返回的函数
func FuncWithArgs[T, R any](condition bool, trueValue, falseValue func(T) R) func(T) R {
	if condition {
		return trueValue
	}
	return falseValue
}

// Handler 模拟三元运算符，支持处理器函数类型（无返回值）
// condition: 条件判断
// trueValue: 条件为真时返回的处理器
// falseValue: 条件为假时返回的处理器
func Handler(condition bool, trueValue, falseValue func()) func() {
	if condition {
		return trueValue
	}
	return falseValue
}

// HandlerWithArgs 模拟三元运算符，支持带参数的处理器函数类型（无返回值）
// condition: 条件判断
// trueValue: 条件为真时返回的处理器
// falseValue: 条件为假时返回的处理器
func HandlerWithArgs[T any](condition bool, trueValue, falseValue func(T)) func(T) {
	if condition {
		return trueValue
	}
	return falseValue
}
