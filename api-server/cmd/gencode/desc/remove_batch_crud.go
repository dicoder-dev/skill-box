package desc

import (
	"fmt"
	"ginp-api/internal/gen"
	"os"
	"strings"
)

// RemoveBatchCrud 批量删除多个实体的CRUD代码
// entities 传入实体名称数组，每个实体名称应为大驼峰命名，如 ["UserGroup", "UserRole"]
func RemoveBatchCrud(entities []string) {
	RemoveBatchCrudWithParent(entities, "")
}

// RemoveBatchCrudWithParent 批量删除多个实体的CRUD代码，支持指定父级目录
// entities 传入实体名称数组，每个实体名称应为大驼峰命名，如 ["UserGroup", "UserRole"]
// parentDir 父级目录名称，如 "user"
func RemoveBatchCrudWithParent(entities []string, parentDir string) {
	// 初始化工作目录
	GetPwd()

	// 记录成功和失败的实体
	successEntities := []string{}
	failedEntities := []string{}

	// 为每个实体删除CRUD代码
	for _, entityName := range entities {
		// 转换为大驼峰命名
		entityNameBig := gen.NameToCameBig(entityName)
		if entityNameBig == "" {
			fmt.Printf("实体名称不能为空: %s\n", entityName)
			failedEntities = append(failedEntities, entityName)
			continue
		}

		// 转换为下划线命名
		lineName := gen.NameToLine(entityNameBig)
		fmt.Printf("正在删除实体: %s (lineName: %s)\n", entityNameBig, lineName)
		if parentDir != "" {
			fmt.Printf("父级目录: %s\n", parentDir)
		}

		// 使用 try-catch 风格处理错误
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("删除实体 %s 时发生错误: %v\n", entityNameBig, r)
					failedEntities = append(failedEntities, entityNameBig)
				}
			}()

			// 删除文件计数
			deletedCount := 0

			// 1. 删除entity文件
			oPathEntity := PathEntity(lineName)
			if removeFileIfExists(oPathEntity) {
				deletedCount++
			}

			// 2. 删除router文件
			oPathRouter := PathRouter(lineName, parentDir)
			if removeFileIfExists(oPathRouter) {
				deletedCount++
			}

			// 3. 删除controller文件
			oPathController := PathController(lineName, parentDir)
			if removeFileIfExists(oPathController) {
				deletedCount++
			}

			// 4. 删除service文件
			oPathService := PathService(lineName, parentDir)
			if removeFileIfExists(oPathService) {
				deletedCount++
			}

			// 5. 删除model文件
			oPathModel := PathModel(lineName, parentDir)
			if removeFileIfExists(oPathModel) {
				deletedCount++
			}

			// 6. 删除fields文件
			oPathFields := PathFields(lineName, parentDir)
			if removeFileIfExists(oPathFields) {
				deletedCount++
			}

			// 7. 删除路由导入
			RemoveImportRouterPackage(lineName, parentDir)

			// 删除空目录
			removeEmptyDirectories(lineName, parentDir)

			// 添加到成功列表
			successEntities = append(successEntities, entityNameBig)
			fmt.Printf("实体 %s 的CRUD代码删除完成，共删除 %d 个文件\n", entityNameBig, deletedCount)
		}()
	}

	// 输出删除结果统计
	fmt.Println("\n批量删除CRUD代码完成")
	fmt.Printf("成功: %d 个实体 (%s)\n", len(successEntities), strings.Join(successEntities, ", "))
	if len(failedEntities) > 0 {
		fmt.Printf("失败: %d 个实体 (%s)\n", len(failedEntities), strings.Join(failedEntities, ", "))
	}
}

// RemoveBatchCrudInteractive 交互式批量删除多个实体的CRUD代码
func RemoveBatchCrudInteractive() {
	// 初始化工作目录
	GetPwd()

	// 获取实体名称列表
	entitiesInput := gen.Input("请输入要删除CRUD代码的实体名称列表，多个实体用逗号分隔（例如：UserGroup,UserRole）：", nil)

	// 分割实体名称列表
	entities := strings.Split(entitiesInput, ",")

	// 清理实体名称（去除空格）
	cleanedEntities := []string{}
	for _, entity := range entities {
		entity = strings.TrimSpace(entity)
		if entity != "" {
			cleanedEntities = append(cleanedEntities, entity)
		}
	}

	// 确认删除操作
	confirm := gen.Input(fmt.Sprintf("确认删除以下实体的CRUD代码吗？(%s) [y/N]: ", strings.Join(cleanedEntities, ", ")), nil)
	if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
		fmt.Println("操作已取消")
		return
	}

	// 调用批量删除函数
	RemoveBatchCrud(cleanedEntities)
}

// removeFileIfExists 删除文件如果存在，返回是否成功删除
func removeFileIfExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		// 文件存在，删除它
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("删除文件失败: %s, 错误: %v\n", filePath, err)
			return false
		}
		fmt.Printf("已删除文件: %s\n", filePath)
		return true
	} else if os.IsNotExist(err) {
		// 文件不存在
		fmt.Printf("文件不存在，跳过: %s\n", filePath)
		return false
	} else {
		// 其他错误
		fmt.Printf("检查文件状态失败: %s, 错误: %v\n", filePath, err)
		return false
	}
}

// removeEmptyDirectories 删除空目录
func removeEmptyDirectories(lineName string, parentDir string) {
	allSmallName := gen.NameToAllSmall(lineName)

	// 尝试删除controller目录
	if parentDir != "" {
		controllerDir := GetDirController() + "/" + parentDir + "/c" + allSmallName
		removeEmptyDir(controllerDir)

		// 尝试删除service目录
		serviceDir := GetDirService() + "/" + parentDir + "/s" + allSmallName
		removeEmptyDir(serviceDir)

		// 尝试删除model目录
		modelDir := GetDirModel() + "/" + parentDir + "/m" + allSmallName
		removeEmptyDir(modelDir)
	} else {
		controllerDir := GetDirController() + "/c" + allSmallName
		removeEmptyDir(controllerDir)

		// 尝试删除service目录
		serviceDir := GetDirService() + "/s" + allSmallName
		removeEmptyDir(serviceDir)

		// 尝试删除model目录
		modelDir := GetDirModel() + "/m" + allSmallName
		removeEmptyDir(modelDir)
	}
}

// removeEmptyDir 删除空目录
func removeEmptyDir(dirPath string) {
	if entries, err := os.ReadDir(dirPath); err == nil {
		if len(entries) == 0 {
			if err := os.Remove(dirPath); err == nil {
				fmt.Printf("已删除空目录: %s\n", dirPath)
			} else {
				fmt.Printf("删除空目录失败: %s, 错误: %v\n", dirPath, err)
			}
		}
	}
}
