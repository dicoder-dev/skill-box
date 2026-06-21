// Package utils
// @Author: zhangdi
// @File: dto
// @Version: 1.0.0
// @Date: 2023/5/18 10:15
package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// DtoToEntity 将 DTO 转换成实体对象
func DtoToEntity(dto interface{}, entity interface{}) {
	// 获取Dto对象和Entity对象的反射值
	dtoValue := reflect.ValueOf(dto).Elem()       // dto必须传入一个指针类型
	entityValue := reflect.ValueOf(entity).Elem() // entity必须传入一个指针类型

	// 遍历Dto对象的每个字段并转换
	for i := 0; i < dtoValue.NumField(); i++ { // NumField返回结构体中字段的数量
		// 获取Dto对象的字段值
		dtoField := dtoValue.Field(i)

		// 如果Dto对象的字段值不为空，则进行转换
		//如果字段必填则零值正常转换,非必填零值不转换
		if !dtoField.IsZero() {

			// 获取Dto对象的字段名
			fieldName := dtoValue.Type().Field(i).Name

			// 根据字段名获取Entity对象的同名字段
			entityField := entityValue.FieldByName(fieldName)

			// 若找到同名字段且类型相同，将Dto对象的值复制给Entity对象
			if entityField.IsValid() && entityField.Type() == dtoField.Type() {
				entityField.Set(dtoField)
			}
		}

	}
}

// DataToJson 将data转json字符
func DataToJson(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func AnyDataParse(anyData any, data interface{}) error {
	//转json
	jsonBytes, err := json.Marshal(anyData)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, data)
	if err != nil {
		return err
	}
	return nil
}

// UpdateDtoCompat 通用的 UpdateDto 兼容结构，用于兼容新旧两种请求格式
type UpdateDtoCompat struct {
	ID           int
	UpdateData   interface{}
	UpdateFields []string
}

// ParseUpdateRequest 解析更新请求，兼容新旧两种格式
// 老格式: {"id": 1, "update_data": {...}, "update_fields": [...]}
// 新格式: {"id": 1, "name": "xx", "description": "yy", ...} (整个 map 就是实体数据)
// entityPtr: 实体类型的指针实例，例如 &entity.DjCategory{}
// 返回: ID、UpdateData（实体指针）、UpdateFields、错误
func ParseUpdateRequest(requestParam map[string]interface{}, entityPtr interface{}) (*UpdateDtoCompat, error) {
	if len(requestParam) == 0 {
		return nil, fmt.Errorf("请求参数为空")
	}

	// 判断是否为新格式：检查是否有 update_data 字段
	// 如果有 update_data，说明是老格式；否则是新格式（整个 map 就是实体）
	if _, hasUpdateData := requestParam["update_data"]; hasUpdateData {
		// 老格式：直接解析
		rawID, ok := requestParam["id"]
		if !ok {
			return nil, fmt.Errorf("缺少 id 参数")
		}
		idInt, err := parseInt(rawID)
		if err != nil || idInt <= 0 {
			return nil, fmt.Errorf("id 参数无效: %v", err)
		}

		// 解析 update_data
		updateDataRaw, ok := requestParam["update_data"]
		if !ok {
			return nil, fmt.Errorf("缺少 update_data 参数")
		}

		// 将 update_data 转为实体
		entityBytes, err := json.Marshal(updateDataRaw)
		if err != nil {
			return nil, fmt.Errorf("update_data 序列化失败: %w", err)
		}

		// 创建新的实体实例
		entityType := reflect.TypeOf(entityPtr).Elem()
		newEntity := reflect.New(entityType).Interface()
		if err := json.Unmarshal(entityBytes, newEntity); err != nil {
			return nil, fmt.Errorf("update_data 反序列化失败: %w", err)
		}

		// 解析 update_fields
		var updateFields []string
		if updateFieldsRaw, ok := requestParam["update_fields"]; ok {
			if fields, ok := updateFieldsRaw.([]interface{}); ok {
				updateFields = make([]string, 0, len(fields))
				for _, field := range fields {
					if str, ok := field.(string); ok {
						updateFields = append(updateFields, str)
					}
				}
			} else if fields, ok := updateFieldsRaw.([]string); ok {
				updateFields = fields
			}
		}

		return &UpdateDtoCompat{
			ID:           idInt,
			UpdateData:   newEntity,
			UpdateFields: updateFields,
		}, nil
	} else {
		// 新格式：整个 requestParam 就是实体数据
		// 提取 id
		rawID, ok := requestParam["id"]
		if !ok {
			return nil, fmt.Errorf("缺少 id 参数")
		}
		idInt, err := parseInt(rawID)
		if err != nil || idInt <= 0 {
			return nil, fmt.Errorf("id 参数无效: %v", err)
		}

		// 将 map 转为实体
		entityBytes, err := json.Marshal(requestParam)
		if err != nil {
			return nil, fmt.Errorf("参数序列化失败: %w", err)
		}

		// 创建新的实体实例
		entityType := reflect.TypeOf(entityPtr).Elem()
		newEntity := reflect.New(entityType).Interface()
		if err := json.Unmarshal(entityBytes, newEntity); err != nil {
			return nil, fmt.Errorf("参数反序列化失败: %w", err)
		}

		// 解析 update_fields：如果客户端指定了就使用指定的，否则设置为空数组
		// 空数组时，GORM 会默认只更新非零值字段
		updateFields := []string{} // 默认为空数组
		if updateFieldsRaw, ok := requestParam["update_fields"]; ok {
			// 客户端明确指定了 update_fields，使用客户端指定的值
			if fields, ok := updateFieldsRaw.([]interface{}); ok {
				updateFields = make([]string, 0, len(fields))
				for _, field := range fields {
					if str, ok := field.(string); ok {
						updateFields = append(updateFields, str)
					}
				}
			} else if fields, ok := updateFieldsRaw.([]string); ok {
				updateFields = fields
			}
		}
		// 如果客户端没有指定 update_fields，updateFields 保持为空数组 []
		// 空数组时，GORM 会默认只更新非零值字段

		return &UpdateDtoCompat{
			ID:           idInt,
			UpdateData:   newEntity,
			UpdateFields: updateFields,
		}, nil
	}
}

// ParseInt 将各种数字类型转换为 int
func ParseInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		return strconv.Atoi(fmt.Sprint(val))
	}
}

// parseInt 将各种数字类型转换为 int
func parseInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		return strconv.Atoi(fmt.Sprint(val))
	}
}
