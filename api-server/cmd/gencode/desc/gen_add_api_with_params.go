package desc

import (
	"fmt"
	"ginp-api/internal/gen"
	"os"
	"path/filepath"
	"strings"
)

// GenAddApiWithParams 使用传入的API名称和目录路径生成API
// apiName传入大驼峰如 GetUserInfo
// dirPath传入如 cuser
func GenAddApiWithParams(apiName, dirPath string) {
	//获取controller目录路径
	controllerDir := GetDirController()

	// 验证目录是否存在，不存在则创建
	apiDir := filepath.Join(controllerDir, dirPath)
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		fmt.Printf("目录不存在，正在创建: %s\n", apiDir)
		err = os.MkdirAll(apiDir, 0755)
		if err != nil {
			fmt.Printf("创建目录失败: %s\n", err)
			return
		}
	}

	// 处理API名称
	apiName = gen.NameToCameBig(apiName)
	if apiName == "" {
		fmt.Println("API名称不能为空")
		return
	}
	apiNameLine := gen.NameToLine(apiName)

	// 获取包名和实体名称
	packageName := filepath.Base(dirPath)
	if strings.HasPrefix(packageName, "c") {
		packageName = strings.TrimPrefix(packageName, "c")
	}

	// 获取实体名称
	entityName := gen.NameToCameBig(packageName)
	entityLineName := gen.NameToLine(entityName)

	// 创建API文件路径
	apiFilePath := filepath.Join(apiDir, apiNameLine+".a.go")

	// 准备替换数据
	replaceData := map[string]string{
		ReplaceApiNameBig:  apiName,        // API名称 大驼峰
		ReplaceApiNameLine: apiNameLine,    // API名称 下划线
		ReplacePackageName: packageName,    // 包名 全小写
		ReplaceEntityName:  entityName,     // 实体名称 大驼峰
		ReplaceLineName:    entityLineName, // 实体名称 下划线
	}

	// 生成API文件
	gen.ReplaceAndWriteTemplate(TemplatePathAddApi(), apiFilePath, replaceData)

	fmt.Printf("成功生成API文件: %s\n", apiFilePath)
}

// 从目录名获取实体名称
func getEntityNameFromDir(apiDir string) string {
	// 从目录路径中提取实体名称
	dirName := filepath.Base(apiDir)
	if strings.HasPrefix(dirName, "c") {
		// 如果目录名以'c'开头，去掉'c'前缀
		return dirName[1:]
	}
	return dirName
}

// 获取控制器文件路径
func getControllerFilePath(apiDir string) string {
	// 查找以.c.go结尾的文件
	files, err := os.ReadDir(apiDir)
	if err != nil {
		fmt.Printf("读取目录失败: %s\n", err)
		return ""
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".c.go") {
			return filepath.Join(apiDir, file.Name())
		}
	}

	return ""
}

// 获取路由文件路径
func getRouterFilePath(apiDir string) string {
	// 查找以.r.go结尾的文件
	files, err := os.ReadDir(apiDir)
	if err != nil {
		fmt.Printf("读取目录失败: %s\n", err)
		return ""
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".r.go") {
			return filepath.Join(apiDir, file.Name())
		}
	}

	return ""
}

// 生成API函数
func generateApiFunction(filePath, apiName, entityName string) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("读取文件失败: %s\n", err)
		return
	}

	// 生成API函数代码
	apiFunction := fmt.Sprintf(`
// %s API
func %s(c *ginp.ContextPlus) {
	// TODO: 实现%s功能
	c.Success()
}
`, apiName, apiName, apiName)

	// 在文件末尾添加API函数
	newContent := string(content) + apiFunction

	// 写入文件
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		fmt.Printf("写入文件失败: %s\n", err)
	}
}

// 生成路由注册
func generateRouterRegistration(filePath, apiName, entityName string) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("读取文件失败: %s\n", err)
		return
	}

	// 生成API常量
	apiConstName := "Api" + apiName
	apiPath := "/api/" + entityName + "/" + gen.NameToLine(apiName)
	apiConst := fmt.Sprintf("\tconst %s = \"%s\"\n", apiConstName, apiPath)

	// 在常量定义部分添加API常量
	lines := strings.Split(string(content), "\n")
	constEndIndex := 0
	for i, line := range lines {
		if strings.Contains(line, ")") && strings.Contains(line, "const") {
			constEndIndex = i
			break
		}
	}

	if constEndIndex > 0 {
		// 在常量定义结束后插入新的API常量
		newLines := append(lines[:constEndIndex+1], append([]string{apiConst}, lines[constEndIndex+1:]...)...)
		content = []byte(strings.Join(newLines, "\n"))
	}

	// 生成路由注册代码
	routerRegistration := fmt.Sprintf(`
	// %s
	ginp.RouterAppend(ginp.RouterItem{
		Path:           %s,                    //api路径
		Handlers:       ginp.RegisterHandler(%s), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      false,                        //是否需要登录
		NeedPermission: false,                        //是否需要鉴权
		PermissionName: "%s.%s",           //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "%s %s",
			Description: "",
			RequestDto:  entity.%s{},
		},
	})
`, apiName, apiConstName, apiName, strings.Title(entityName), gen.NameToLine(apiName), apiName, entityName, strings.Title(entityName))

	// 在init函数末尾添加路由注册
	initEndIndex := 0
	for i, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "}") && strings.Contains(strings.Split(string(content), "\n")[i-1], ")") {
			initEndIndex = i
			break
		}
	}

	if initEndIndex > 0 {
		// 在init函数结束前插入新的路由注册
		newLines := strings.Split(string(content), "\n")
		newLines = append(newLines[:initEndIndex], append([]string{routerRegistration}, newLines[initEndIndex:]...)...)
		content = []byte(strings.Join(newLines, "\n"))
	}

	// 写入文件
	err = os.WriteFile(filePath, content, 0644)
	if err != nil {
		fmt.Printf("写入文件失败: %s\n", err)
	}
}
