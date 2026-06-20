package desc

import (
	"fmt"
	"ginp-api/internal/gen"
	"strings"
)

// GenBatchCrud 批量生成多个实体的CRUD代码
// entities 传入实体名称数组，每个实体名称应为大驼峰命名，如 ["UserGroup", "UserRole"]
func GenBatchCrud(entities []string) {
	GenBatchCrudWithParent(entities, "")
}

// GenBatchCrudWithParent 批量生成多个实体的CRUD代码，支持指定父级目录
// entities 传入实体名称数组，每个实体名称应为大驼峰命名，如 ["UserGroup", "UserRole"]
// parentDir 父级目录名称，如 "user"
func GenBatchCrudWithParent(entities []string, parentDir string) {
	// 初始化工作目录
	GetPwd()

	// 记录成功和失败的实体
	successEntities := []string{}
	failedEntities := []string{}

	// 为每个实体生成CRUD代码
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
		fmt.Printf("正在生成实体: %s (lineName: %s)\n", entityNameBig, lineName)
		if parentDir != "" {
			fmt.Printf("父级目录: %s\n", parentDir)
		}

		// 获取替换数据
		replaceData := getBaseReplaceMap(entityNameBig, parentDir)

		// 使用 try-catch 风格处理错误
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("生成实体 %s 时发生错误: %v\n", entityNameBig, r)
					failedEntities = append(failedEntities, entityNameBig)
				}
			}()

			// 1. 生成entity文件
			tPathEntity := TemplatePathEntity()
			oPathEntity := PathEntity(lineName)
			gen.ReplaceAndWriteTemplate(tPathEntity, oPathEntity, replaceData)

			// 2. 生成router文件
			tPathRouter := TemplatePathRouter()
			oPathRouter := PathRouter(lineName, parentDir)
			gen.ReplaceAndWriteTemplate(tPathRouter, oPathRouter, replaceData)
			AddImportRouterPackage(lineName, parentDir) // 添加router导入包

			// 3. 生成controller文件
			tPathController := TemplatePathController()
			oPathController := PathController(lineName, parentDir)
			gen.ReplaceAndWriteTemplate(tPathController, oPathController, replaceData)

			// 4. 生成service文件
			tPathService := TemplatePathService()
			oPathService := PathService(lineName, parentDir)
			gen.ReplaceAndWriteTemplate(tPathService, oPathService, replaceData)

			// 5. 生成model文件
			tPathRepository := TemplatePathModel()
			oPathRepository := PathModel(lineName, parentDir)
			gen.ReplaceAndWriteTemplate(tPathRepository, oPathRepository, replaceData)

			// 6. 生成fields文件
			tPathFields := TemplatePathFields()
			oPathFields := PathFields(lineName, parentDir)
			gen.ReplaceAndWriteTemplate(tPathFields, oPathFields, replaceData)

			// 添加到成功列表
			successEntities = append(successEntities, entityNameBig)
			fmt.Printf("实体 %s 的CRUD代码生成完成\n", entityNameBig)
		}()
	}

	// 输出生成结果统计
	fmt.Println("\n批量生成CRUD代码完成")
	fmt.Printf("成功: %d 个实体 (%s)\n", len(successEntities), strings.Join(successEntities, ", "))
	if len(failedEntities) > 0 {
		fmt.Printf("失败: %d 个实体 (%s)\n", len(failedEntities), strings.Join(failedEntities, ", "))
	}
}

// GenBatchCrudInteractive 交互式批量生成多个实体的CRUD代码
func GenBatchCrudInteractive() {
	// 初始化工作目录
	GetPwd()

	// 获取实体名称列表
	entitiesInput := gen.Input("请输入要生成CRUD代码的实体名称列表，多个实体用逗号分隔（例如：UserGroup,UserRole）：", nil)

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

	// 调用批量生成函数
	GenBatchCrud(cleanedEntities)
}
