package where

import (
	"fmt"
	"strings"
)

func Check(wheres []*Condition) error {
	for _, cond := range wheres {
		if cond == nil {
			return fmt.Errorf("condition is nil")
		}

		// 检查操作符是否合法
		if !isValidOperator(cond.Opt) {
			return fmt.Errorf("invalid operator: %s", cond.Opt)
		}
		// 检查值是否合法
		// if !isValidValue(cond.Value) {
		// 	return fmt.Errorf("invalid value: %v", cond.Value)
		// }
	}
	return nil
}

// isValidOperator 检查操作符是否合法
func isValidOperator(op string) bool {
	switch op {
	case OptLike, OptIn, OptBetween, Greater, GreaterEqual, Less, LessEqual, Equal:
		return true
	default:
		return false
	}
}

// 定义危险字符列表
var dangerousChars = []string{"'", ";", "--", "/*", "*/"}

// isValidValue 检查值是否合法，预防依赖注入
func isValidValue(value interface{}) bool {
	// 基本类型检查，直接返回true
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	case string:
		// 简单的SQL注入检查，检查是否包含危险字符
		strValue := value.(string)
		for _, char := range dangerousChars {
			if strings.Contains(strValue, char) {
				return false
			}
		}
		return true
	case []string:
		// 处理字符串切片
		strSlice := value.([]string)
		for _, str := range strSlice {
			for _, char := range dangerousChars {
				if strings.Contains(str, char) {
					return false
				}
			}
		}
		return true
	case []int:
		// 处理整数切片，直接返回true
		return true
	default:
		return false
	}
}
