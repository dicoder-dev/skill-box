package genfunc

import (
	"fmt"
	"ginp-api/pkg/gencode/gen"
	"io/ioutil"
	"reflect"
)

// 注意：此功能依赖于 ginp-api/internal/gapi/setting 包
// 该包包含 EntityGenerationList，需要在业务项目中实现
// 如果不需要此功能，可以注释掉以下代码

// EntityGenerationList 是一个可选的实体列表
// 在业务项目中实现此变量即可使用 GenFields 功能
var EntityGenerationList []any

// GenFields 生成实体字段常量
// 使用方法：在业务项目的 setting 包中定义 EntityGenerationList 变量
func GenFields() {
	if len(EntityGenerationList) == 0 {
		fmt.Println("警告：EntityGenerationList 为空，请先在 setting 包中定义")
		return
	}

	for _, entity_ := range EntityGenerationList {
		t := reflect.TypeOf(entity_).Elem()
		// fileName := strings.ToLower(t.Name()) + ".go"
		packageName := "m" + gen.NameToAllSmall(t.Name())
		content := "package " + packageName + " \n\n"
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			constName := "Field" + field.Name
			fieldName := gen.NameToLine(field.Name)
			if fieldName == "model" { //gorm.model
				content += fmt.Sprintf("const %s = \"%s\"\n\n", t.Name()+"ID", "id")
				content += fmt.Sprintf("const %s = \"%s\"\n\n", t.Name()+"Created", "created_at")
				content += fmt.Sprintf("const %s = \"%s\"\n\n", t.Name()+"Updated", "updated_at")
			} else {
				content += fmt.Sprintf("const %s = \"%s\"\n\n", constName, fieldName)
			}
		}

		filePath := PathFields(t.Name())
		err := ioutil.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			continue
		}
		fmt.Println("Data written to file.")
	}
}
