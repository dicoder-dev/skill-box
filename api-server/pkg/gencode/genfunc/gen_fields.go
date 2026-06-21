package genfunc

import (
	"fmt"
	"ginp-api/pkg/gencode/gen"
	"io/ioutil"
	"reflect"
)

// GenFields 生成实体字段常量文件
// entities 传入需要生成字段常量的实体列表，每个元素应为指向结构体的指针（如 new(entity.User)）
// 使用方法：调用方在自己的 setting 包中维护实体列表，然后传入本函数
func GenFields(entities []any) {
	if len(entities) == 0 {
		fmt.Println("警告：实体列表为空，未生成任何字段常量")
		return
	}
	for _, entity_ := range entities {
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
