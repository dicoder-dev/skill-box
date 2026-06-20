// Package where
// @Author: zhangdi
// @File: common
// @Version: 1.0.0
// @Date: 2023/5/18 11:58
package where

import (
	"errors"
	"fmt"

	"reflect"

	"strconv"
)

// Condition 字段名-字段值结构体
type Condition struct {
	Field string      `gorm:"not null" json:"field"`
	Value interface{} `gorm:"not null" json:"value"`
	//Opt 可选条件：= > < >=  <=  LIKE  BETWEEN IN
	Opt     string `gorm:"not null" json:"opt,omitempty"`
	ErrMsg  string `json:"err_msg,omitempty"`
	Connect string `json:"connect,omitempty"` // 【后接的】连接符，默认为 "AND"  ，如果是最后一个条件 该连接符无效
}

// StrToUint64 将字符串转uint64
func (s *Condition) StrToUint64() *Condition {
	NumberInt, err := strconv.ParseUint(s.Value.(string), 10, 64)
	if err != nil {
		s.ErrMsg = "类型转成uint64失败"
		return s
	}
	s.Value = NumberInt
	return s
}

// OptEqual 创建一个FieldWhere对象，操作符默认为 =
func OptEqual(fieldName string, fieldValue any) *Condition {
	return &Condition{Field: fieldName, Value: fieldValue, Opt: "=", ErrMsg: ""}
}

// Opt 创建一个带操作符opt的FieldWhere对象
func Opt(fieldName string, opt string, fieldValue interface{}) *Condition {
	optList := []string{"=", ">", "<", ">=", "<=", OptLike, OptIn, OptBetween}
	if !arrContains(optList, opt) {
		return &Condition{ErrMsg: "操作符不符合，可选操作符: = > < >=  <=  LIKE  BETWEEN IN"}
	}

	if opt == "IN" {
		if reflect.TypeOf(fieldValue).Kind() != reflect.Slice && reflect.TypeOf(fieldValue).Kind() != reflect.Array {
			return &Condition{ErrMsg: "IN操作需要传入一个列表（数组或者切片）"}
		}
	} else if opt == "BETWEEN" {
		if reflect.TypeOf(fieldValue).Kind() != reflect.Slice && reflect.TypeOf(fieldValue).Kind() != reflect.Array {
			return &Condition{ErrMsg: "BETWEEN操作需要传入一个长度为2的列表"}
		} else {
			sliceValue := reflect.ValueOf(fieldValue)
			if sliceValue.Len() != 2 {
				return &Condition{ErrMsg: "BETWEEN操作需要传入一个长度为2的列表"}
			}
		}
	}
	return &Condition{Field: fieldName, Value: fieldValue, Opt: opt, ErrMsg: ""}
}

// ConvertGormRepeatStr 找出所有的opt = REPEAT 的条件，并组装出Having和Group对应字符串,返回空则表示无重复选项
// demo:查询同时满足两个字段重号的记录:调用示例 Opt("num",OptRepeat,1) 表示num字段重复数量大于1的所有数据
// db.Group("email, phone_number")
// .Having("count(email) > 1 AND count(phone_number) > 1")
// .Find(&users)
//func ConvertGormRepeatStr(fieldWheres []*Field) (groupStr string, Having string) {
//	for _, fw := range fieldWheres {
//		switch fw.Opt {
//		case OptRepeat:
//			groupStr += fmt.Sprintf("%s,", fw.Field)
//			Having += fmt.Sprintf("count(%s) > %v AND ", fw.Field, fw.Value)
//		}
//	}
//	return strings.TrimRight(groupStr, ","), strings.TrimRight(Having, "AND ")
//}

// ConvertToGormWhere 将切片FieldWhere转换为符合gorm库的Where条件语句
func ConvertToGormWhere(fieldWheres []*Condition) (string, []interface{}, error) {
	var whereStr string
	var whereValues []interface{}
	var lastConnect string

	for _, fw := range fieldWheres {
		if fw.ErrMsg != "" {
			return "", nil, errors.New("异常错误：" + fw.Field + fw.Opt + fw.ErrMsg)
		}
		if lastConnect != "" {
			whereStr += " " + lastConnect + " "
		}
		lastConnect = fw.Connect
		//默认连击符为AND
		if lastConnect == "" {
			lastConnect = AND
		}

		switch fw.Opt {
		case "=":
			whereStr += fmt.Sprintf("(%s = ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case ">":
			whereStr += fmt.Sprintf("(%s > ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case "<":
			whereStr += fmt.Sprintf("(%s < ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case ">=":
			whereStr += fmt.Sprintf("(%s >= ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case "<=":
			whereStr += fmt.Sprintf("(%s <= ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case "LIKE":
			whereStr += fmt.Sprintf("(%s LIKE ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case "IN":
			whereStr += fmt.Sprintf("(%s IN ?)", fw.Field)
			whereValues = append(whereValues, fw.Value)
		case "BETWEEN":
			val := reflect.ValueOf(fw.Value)
			inValues := make([]interface{}, val.Len())
			for i := 0; i < val.Len(); i++ {
				inValues[i] = val.Index(i).Convert(reflect.TypeOf((*interface{})(nil)).Elem()).Interface()
			}
			if len(inValues) != 2 {
				return "", nil, errors.New("BETWEEN操作需要传入一个长度为2的列表")
			}
			whereStr += fmt.Sprintf("(%s BETWEEN ? AND ?)", fw.Field)
			whereValues = append(whereValues, inValues[0], inValues[1])
		default:
			return "", nil, errors.New("未知的操作符: " + fw.Opt)
		}
	}

	return whereStr, whereValues, nil
}

// 供关联查询的时候使用
func ConvertToGormWhere2(fieldWheres []*Condition) ([]interface{}, error) {
	set, values, err := ConvertToGormWhere(fieldWheres)
	if err != nil {
		return nil, err
	}
	var whereValues []interface{}
	whereValues = append(whereValues, set)
	whereValues = append(whereValues, values...)
	return whereValues, nil
}
