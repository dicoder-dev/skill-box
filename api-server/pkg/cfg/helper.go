// Package cfg
// @Author: zhangdi
// @File: helper
// @Version: 1.0.0
// @Date: 2023/11/22 12:21
package cfg

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

var instance *Config

// yamlPath 保存配置文件的路径，用于热加载
var yamlPath string

// fieldConfigKeyMap 存储字段路径到配置键的映射，用于反向写入
var fieldConfigKeyMap = make(map[string]string)
var mapMutex sync.RWMutex

const DefaultYamlPath = "configs.yaml"

// InitCfg 初始化
func InitCfg(path string) error {
	yamlPath = path
	if yamlPath == "" {
		yamlPath = DefaultYamlPath
	}
	var err error
	instance, err = NewConfig()
	if err != nil {
		return err
	}
	err = instance.LoadConfig(yamlPath)
	if err != nil {
		return err
	}
	return nil
}

func checkInstance() {
	if instance == nil {
		err := InitCfg(filepath.Join(DefaultYamlPath))
		if err != nil {
			println(err.Error())
			return
		}
	}
}

// Reload 重新加载配置文件
func Reload() error {
	if yamlPath == "" {
		yamlPath = DefaultYamlPath
	}
	// 清空字段映射
	mapMutex.Lock()
	fieldConfigKeyMap = make(map[string]string)
	mapMutex.Unlock()

	// 重新加载配置
	return instance.LoadConfig(yamlPath)
}

// Set 设置配置值（写入配置文件）
func Set(k string, v any) error {
	checkInstance()
	err := instance.Set(k, v)
	return err
}

// SetDefault 设置初始值
func SetDefault(k string, v any) error {
	checkInstance()
	val := Get(k)
	if val == nil || val == "" {
		err := instance.Set(k, v)
		return err
	}
	return nil
}

// setDefaultWithoutWrite 设置默认值但不写入文件（用于 ParseConfigStruct）
func setDefaultWithoutWrite(k string, v any) {
	checkInstance()
	val := Get(k)
	if val == nil || val == "" {
		instance.setWithoutWrite(k, v)
	}
}

// InitDefaults 将配置结构体的默认值写入配置文件
// 如果配置文件中没有值，则写入 default tag 指定的默认值
func InitDefaults(ptr any) error {
	checkInstance()
	return initDefaultsStruct(ptr, "")
}

// initDefaultsStruct 递归将默认值写入配置文件
func initDefaultsStruct(ptr any, prefix string) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("ptr must be a pointer to a struct")
	}

	structVal := v.Elem()
	structType := structVal.Type()

	// 获取结构体名称
	structName := structType.Name()
	if prefix == "" {
		prefix = toLowerSnakeCase(structName)
	} else {
		prefix = prefix + "." + toLowerSnakeCase(structName)
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		// 如果是嵌套结构体，递归处理
		if field.Type.Kind() == reflect.Struct {
			err := initDefaultsStruct(fieldVal.Addr().Interface(), prefix)
			if err != nil {
				return err
			}
			continue
		}

		// 获取配置键
		configKey := field.Tag.Get(TagConfigKey)
		if configKey == "" {
			configKey = prefix + "." + toLowerSnakeCase(field.Name)
		}

		// 使用 Lookup 检查 default 标签是否存在（区分"标签不存在"和"标签存在但为空"）
		defaultValue, hasDefault := field.Tag.Lookup(TagDefault)

		// 如果配置文件中没有值，则写入默认值（无论默认值是否为空字符串）
		if hasDefault {
			val := Get(configKey)
			if val == nil || val == "" {
				err := instance.Set(configKey, defaultValue)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func Get(k string) any {
	checkInstance()
	val, err := instance.Get(k)
	if err != nil {
		return nil
	}
	return val
}

func GetString(k string) string {
	checkInstance()
	val, err := instance.GetString(k)
	if err != nil {
		return ""
	}
	return val
}
func GetStringDefault(k string, dv string) string {
	checkInstance()
	val, err := instance.GetString(k, dv)
	if err != nil {
		return ""
	}
	return val
}

func GetBool(k string) bool {
	checkInstance()
	val, _ := instance.GetString(k)
	if val == "yes" || val == "ok" || val == "1" {
		return true
	}
	if val == "no" || val == "ng" || val == "0" {
		return false
	}

	return false
}

func GetInt(k string) int {
	checkInstance()
	val, err := instance.GetInt(k)
	if err != nil {
		return 0
	}
	return val
}

// ConfigStruct 配置结构体标签
// 使用方法:
// type ServerConfig struct {
//     Port string `default:"8082"` // 自动生成 configkey: server.port
//     Host string `configkey:"server.host" default:"localhost"` // 手动指定
// }
const (
	TagConfigKey = "configkey" // 配置键标签（可选，不指定则自动生成）
	TagDefault   = "default"   // 默认值标签
)

// ToLowerSnakeCase 将驼峰字符串转为小写下划线
// 例如: EmailConfig -> email, ClientPwd -> client_pwd, APIURL -> apiurl, LLM -> llm
// 自动去除 Config 后缀
// 规则：
// 1. 驼峰词（如 ClientPwd, MiniMax）在每个大写字母前加下划线
// 2. 连续大写字母：
//    - 如果最后一个字母是大写，整个转小写（如 APIURL -> apiurl, LLM -> llm）
//    - 如果最后一个字母后面是小写，以该大写字母作为分界线（如 LLMData -> llm_data, APIKey -> api_key）
func toLowerSnakeCase(s string) string {
	if s == "" {
		return s
	}

	// 先去除 Config 后缀
	s = strings.TrimSuffix(s, "Config")
	s = strings.TrimSuffix(s, "CONFIG")

	var result strings.Builder
	length := len(s)

	for i := 0; i < length; i++ {
		c := s[i]

		if c >= 'A' && c <= 'Z' {
			// 当前是大写字母
			if i == 0 {
				// 第一个字符，直接写入
				result.WriteByte(c)
			} else {
				prev := s[i-1]
				if prev >= 'a' && prev <= 'z' {
					// 前一个是小写，在大写前加下划线
					result.WriteRune('_')
					result.WriteByte(c)
				} else if prev >= 'A' && prev <= 'Z' {
					// 前一个也是大写，需要判断是否要加分隔
					// 找到前面连续大写的开始位置
					j := i - 1
					for j > 0 && s[j] >= 'A' && s[j] <= 'Z' {
						j--
					}
					if s[j] >= 'A' && s[j] <= 'Z' {
						j = 0 // 全是连续大写
					}

					// 看当前字符后面是否还有字母
					hasNext := false
					hasNextLower := false
					for k := i + 1; k < length; k++ {
						if (s[k] >= 'a' && s[k] <= 'z') || (s[k] >= 'A' && s[k] <= 'Z') {
							hasNext = true
							if s[k] >= 'a' && s[k] <= 'z' {
								hasNextLower = true
							}
							break
						}
					}

					if !hasNext {
						// 后面没有字母了，整个连续大写转小写
						// 不加分隔
					} else if hasNextLower {
						// 后面是小写，在当前位置加分隔
						result.WriteRune('_')
					}
					// 后面还是大写，不加分隔
					result.WriteByte(c)
				} else {
					// 前一个不是字母
					result.WriteByte(c)
				}
			}
		} else {
			// 小写字母直接写入
			result.WriteByte(c)
		}
	}

	return strings.ToLower(result.String())
}

// ParseConfigStruct 解析配置结构体
// 自动根据结构体字段的 tag 设置默认值并读取配置值
// 支持嵌套结构体，自动生成 configkey
// ptr: 指向结构体的指针
func ParseConfigStruct(ptr any) {
	checkInstance()
	fieldConfigKeyMap = make(map[string]string)
	parseStruct(ptr, "")
}

// parseStruct 递归解析结构体
// prefix: 配置键的前缀
func parseStruct(ptr any, prefix string) {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic("ptr must be a pointer to a struct")
	}

	structVal := v.Elem()
	structType := structVal.Type()

	// 获取结构体名称作为前缀的一部分
	structName := structType.Name()
	if prefix == "" {
		prefix = toLowerSnakeCase(structName)
	} else {
		prefix = prefix + "." + toLowerSnakeCase(structName)
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		// 检查是否指定了 configkey 标签
		configKey := field.Tag.Get(TagConfigKey)
		// 使用 Lookup 检查 default 标签是否存在（区分"标签不存在"和"标签存在但为空"）
		defaultValue, hasDefault := field.Tag.Lookup(TagDefault)

		// 如果是嵌套结构体，递归处理
		if field.Type.Kind() == reflect.Struct {
			// 创建字段路径用于反向映射
			fieldPath := prefix + "." + toLowerSnakeCase(field.Name)
			// 递归解析嵌套结构体
			parseStruct(fieldVal.Addr().Interface(), prefix)
			// 记录嵌套结构体的字段路径（用于后续可能的操作）
			mapMutex.Lock()
			fieldConfigKeyMap[fieldPath] = fieldPath
			mapMutex.Unlock()
			continue
		}

		// 如果没有指定 configkey，则自动生成
		if configKey == "" {
			configKey = prefix + "." + toLowerSnakeCase(field.Name)
		}

		// 存储字段路径到配置键的映射
		fieldPath := prefix + "." + toLowerSnakeCase(field.Name)
		mapMutex.Lock()
		fieldConfigKeyMap[fieldPath] = configKey
		mapMutex.Unlock()

		// 设置默认值（如果标签存在，无论值是否为空都写入）
		if hasDefault {
			val := Get(configKey)
			if val == nil || val == "" {
				instance.Set(configKey, defaultValue)
			}
		}

		// 根据字段类型设置值
		setFieldValue(fieldVal, configKey)
	}
}

// setFieldValue 根据字段类型从配置中获取值并设置到字段
func setFieldValue(fieldVal reflect.Value, configKey string) {
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(GetString(configKey))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldVal.SetInt(int64(GetInt(configKey)))
	case reflect.Bool:
		fieldVal.SetBool(GetBool(configKey))
	case reflect.Float32, reflect.Float64:
		fieldVal.SetFloat(GetFloat(configKey))
	}
}

// GetFloat 获取浮点数配置
func GetFloat(k string) float64 {
	checkInstance()
	val, err := instance.Get(k)
	if err != nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	}
	return 0
}

// SyncConfig 同步配置值到配置文件
// 遍历全局配置变量，将内存中的值写回配置文件
func SyncConfig(ptr any) error {
	checkInstance()
	return syncStruct(ptr, "")
}

// syncStruct 递归同步结构体到配置文件
func syncStruct(ptr any, prefix string) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("ptr must be a pointer to a struct")
	}

	structVal := v.Elem()
	structType := structVal.Type()

	// 获取结构体名称
	structName := structType.Name()
	if prefix == "" {
		prefix = toLowerSnakeCase(structName)
	} else {
		prefix = prefix + "." + toLowerSnakeCase(structName)
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		// 如果是嵌套结构体，递归处理
		if field.Type.Kind() == reflect.Struct {
			err := syncStruct(fieldVal.Addr().Interface(), prefix)
			if err != nil {
				return err
			}
			continue
		}

		// 获取配置键
		configKey := field.Tag.Get(TagConfigKey)
		if configKey == "" {
			configKey = prefix + "." + toLowerSnakeCase(field.Name)
		}

		// 获取字段值并写入配置
		var configValue any
		switch fieldVal.Kind() {
		case reflect.String:
			configValue = fieldVal.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			configValue = int(fieldVal.Int())
		case reflect.Bool:
			configValue = fieldVal.Bool()
		case reflect.Float32, reflect.Float64:
			configValue = fieldVal.Float()
		}

		err := instance.Set(configKey, configValue)
		if err != nil {
			return err
		}
	}

	return nil
}
