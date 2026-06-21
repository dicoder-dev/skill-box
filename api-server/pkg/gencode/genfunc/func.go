package genfunc

import (
	"fmt"
	"ginp-api/pkg/filehelper"
	"ginp-api/pkg/gencode/gen"
	"strings"
)

func AddImportRouterPackage(lineName string, parentDir ...string) {
	allSmallName := gen.NameToAllSmall(lineName)
	// packgeName := "r" + allSmallName
	content, err := filehelper.ReadContent(PathRouterEntry())
	if err != nil {
		println(err.Error())
		return
	}

	// 构建导入路径
	importPath := ""
	if len(parentDir) > 0 && parentDir[0] != "" {
		// Remove 'c' prefix from parent directory to fix import path
		importPath = parentDir[0] + "/c" + allSmallName
	} else {
		importPath = "c" + allSmallName
	}

	if strings.Contains(content, importPath) {
		println("outers_entry.go 已经存在", importPath, "不再添加")
		return
	}

	placeHolder := PlaceholderRouterImport
	importStr := ""
	if len(parentDir) > 0 && parentDir[0] != "" {
		// Fix the import path by ensuring we don't have 'c' prefix in parent directory
		// This ensures imports like 'admin/ctestentity8' instead of 'cadmin/ctestentity8'
		importStr = RouterReplaceStr + parentDir[0] + "/c" + allSmallName + `"`
	} else {
		importStr = RouterReplaceStr + "c" + allSmallName + `"`
	}

	newContent := strings.Replace(content, placeHolder, importStr+"\n\t"+placeHolder, -1)
	err = filehelper.WriteContent(PathRouterEntry(), newContent)
	if err != nil {
		println("outers_entry 写入失败" + err.Error())
		return
	} else {
		println("outers_entry import写入成功")
	}
}

// RemoveImportRouterPackage 从路由导入文件中删除指定实体的导入语句
func RemoveImportRouterPackage(lineName string, parentDir ...string) {
	allSmallName := gen.NameToAllSmall(lineName)
	content, err := filehelper.ReadContent(PathRouterEntry())
	if err != nil {
		println(err.Error())
		return
	}

	// 构建要删除的导入语句
	importStr := ""
	if len(parentDir) > 0 && parentDir[0] != "" {
		importStr = RouterReplaceStr + parentDir[0] + "/c" + allSmallName + `"`
	} else {
		importStr = RouterReplaceStr + "c" + allSmallName + `"`
	}

	// 检查导入语句是否存在
	if !strings.Contains(content, importStr) {
		println("路由导入不存在，跳过删除:", importStr)
		return
	}

	// 删除导入语句（包括换行符和制表符）
	newContent := strings.Replace(content, "\t"+importStr+"\n", "", -1)
	// 如果上面的替换没有成功，尝试其他可能的格式
	if newContent == content {
		newContent = strings.Replace(content, importStr+"\n", "", -1)
	}
	if newContent == content {
		newContent = strings.Replace(content, importStr, "", -1)
	}

	err = filehelper.WriteContent(PathRouterEntry(), newContent)
	if err != nil {
		println("路由导入删除失败:" + err.Error())
		return
	} else {
		println("路由导入删除成功:", importStr)
	}
}

// RegenerateRouterImports 完全重写router导入列表
// importPaths: 导入路径列表，格式为 ["system/ccommon", "system/cuserprofile", "cuser"] 等
// 前缀会自动添加为 ginp-api/internal/app/gapi/controller/
func RegenerateRouterImports(importPaths []string) error {
	// 读取当前文件内容
	content, err := filehelper.ReadContent(PathRouterEntry())
	if err != nil {
		return fmt.Errorf("读取router导入文件失败: %v", err)
	}

	// 找到 import 块的位置
	importStart := strings.Index(content, "import (")
	if importStart == -1 {
		return fmt.Errorf("未找到import块")
	}

	// 找到 import 块的结束位置（匹配的 )）
	// 从 importStart + len("import (") 开始查找，找到匹配的 )
	searchStart := importStart + len("import (")
	importEnd := strings.LastIndex(content[searchStart:], ")")
	if importEnd == -1 {
		return fmt.Errorf("未找到import块的结束位置")
	}
	importEnd += searchStart // 转换为在整个内容中的位置

	// 构建新的导入列表
	var importLines []string
	for _, importPath := range importPaths {
		// 添加导入语句，格式: _ "ginp-api/internal/app/gapi/controller/路径"
		importLine := "\t" + RouterReplaceStr + importPath + `"`
		importLines = append(importLines, importLine)
	}

	// 组装新的导入块内容
	newImportContent := "import (\n"
	newImportContent += strings.Join(importLines, "\n")
	newImportContent += "\n\t" + PlaceholderRouterImport + "\n"
	newImportContent += "\t// 上面的占位符请不要动动，否则会导致生成工具无法自动替换\n"
	newImportContent += "\t//Please do not move the placeholders above, otherwise it will cause the generation tool to fail to replace them automatically\n"
	newImportContent += ")"

	// 替换整个import块
	beforeImport := content[:importStart]
	afterImport := content[importEnd+1:]
	newContent := beforeImport + newImportContent + afterImport

	// 写入文件
	routerEntryPath := PathRouterEntry()
	fmt.Printf("准备写入文件: %s\n", routerEntryPath)
	fmt.Printf("新内容长度: %d 字符\n", len(newContent))
	
	err = filehelper.WriteContent(routerEntryPath, newContent)
	if err != nil {
		return fmt.Errorf("写入router导入文件失败 [%s]: %v", routerEntryPath, err)
	}

	fmt.Printf("成功重写router导入列表，共 %d 个导入\n", len(importPaths))
	return nil
}
