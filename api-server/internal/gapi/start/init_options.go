package start

import (
	"bytes"
	"fmt"
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

const enumConstPrefix = "Enum"
const enumFilePrefix = "enum_"

func initOptions() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		panic(fmt.Errorf("生成枚举常量失败: %w", err))
	}

	enumDir := filepath.Join(projectRoot, "internal", "gapi", "enums")
	if err := os.MkdirAll(enumDir, 0755); err != nil {
		panic(fmt.Errorf("创建枚举常量目录失败: %w", err))
	}

	for _, item := range EntityGenerationList {
		entityItem, ok := item.(typ.IEntity)
		if !ok {
			continue
		}

		entityName := getEntityName(entityItem)
		if entityName == "" {
			continue
		}

		filePath := filepath.Join(enumDir, enumFilePrefix+gen.NameToLine(entityName)+"_consts.go")
		enumItems := normalizeEnumItems(entityItem.GenEnumOptions())
		if len(enumItems) == 0 {
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				panic(fmt.Errorf("删除空枚举文件 %s 失败: %w", filePath, err))
			}
			continue
		}

		content, err := buildEnumConstFile(entityName, enumItems)
		if err != nil {
			panic(fmt.Errorf("生成 %s 枚举常量失败: %w", entityName, err))
		}

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			panic(fmt.Errorf("写入 %s 失败: %w", filePath, err))
		}
	}
}

func buildEnumConstFile(entityName string, enumItems []typ.EntityEnumOption) ([]byte, error) {
	var builder bytes.Buffer
	builder.WriteString("// 该文件为自动生成(覆盖模式)，请勿在此文件上做修改\n\n")
	builder.WriteString("package enums\n\n")

	enumItems = normalizeEnumItems(enumItems)
	if len(enumItems) > 0 {
		builder.WriteString("const (\n")
		for _, enumItem := range enumItems {
			groupName := enumConstPrefix + entityName + gen.NameToCameBig(enumItem.FieldName)
			builder.WriteString("\t//------------------------ " + groupName + "  ------------------\n")

			for _, value := range sortedOptionValues(enumItem.Options) {
				label := enumItem.Options[value]
				constName := groupName + toConstNamePart(value)
				builder.WriteString(fmt.Sprintf("\t%s = %q // %s\n", constName, value, label))
			}
		}
		builder.WriteString(")\n")
	}

	for _, enumItem := range enumItems {
		groupName := enumConstPrefix + entityName + gen.NameToCameBig(enumItem.FieldName)
		builder.WriteString("\n")
		builder.WriteString("var " + groupName + "LabelMap = map[string]string{\n")
		for _, value := range sortedOptionValues(enumItem.Options) {
			label := enumItem.Options[value]
			constName := groupName + toConstNamePart(value)
			builder.WriteString(fmt.Sprintf("\t%s: %q,\n", constName, label))
		}
		builder.WriteString("}\n")
	}

	formatted, err := format.Source(builder.Bytes())
	if err != nil {
		return nil, err
	}
	return formatted, nil
}

func normalizeEnumItems(enumItems []typ.EntityEnumOption) []typ.EntityEnumOption {
	result := make([]typ.EntityEnumOption, 0, len(enumItems))
	for _, enumItem := range enumItems {
		if enumItem.FieldName == "" || len(enumItem.Options) == 0 {
			continue
		}
		result = append(result, enumItem)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].FieldName < result[j].FieldName
	})
	return result
}

func sortedOptionValues(options map[string]string) []string {
	values := make([]string, 0, len(options))
	for value := range options {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func getEntityName(entityItem typ.IEntity) string {
	entityConfig := entityItem.GenConfig()
	if entityConfig != nil && entityConfig.AsName != "" {
		return entityConfig.AsName
	}

	entityType := reflect.TypeOf(entityItem)
	if entityType == nil {
		return ""
	}
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	return entityType.Name()
}

func toConstNamePart(value string) string {
	var parts []string
	var builder strings.Builder

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		parts = append(parts, builder.String())
		builder.Reset()
	}

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		flush()
	}
	flush()

	if len(parts) == 0 {
		return "Value"
	}

	for i, part := range parts {
		if isAllUpperOrDigit(part) {
			parts[i] = part
			continue
		}
		parts[i] = gen.NameToCameBig(strings.ToLower(part))
	}
	return strings.Join(parts, "")
}

func isAllUpperOrDigit(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func findProjectRoot() (string, error) {
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		if root, err := searchProjectRoot(filepath.Dir(currentFile)); err == nil {
			return root, nil
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return searchProjectRoot(wd)
}

func searchProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到 go.mod")
		}
		dir = parent
	}
}
