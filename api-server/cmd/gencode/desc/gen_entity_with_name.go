package desc

import (
	"ginp-api/internal/gen"
)

// GenEntityWithName 使用传入的实体名称生成实体
// entityName传入大驼峰如 UserGroup
func GenEntityWithName(entityName string) {
	GenEntityWithNameAndParent(entityName, "")
}

// GenEntityWithNameAndParent 使用传入的实体名称和父级目录生成实体
// entityName传入大驼峰如 UserGroup
// parentDir传入父级目录名称如 user
func GenEntityWithNameAndParent(entityName string, parentDir string) {
	entityName = gen.NameToCameBig(entityName)
	if entityName == "" {
		println("实体名称不能为空")
		return
	}
	lineName := gen.NameToLine(entityName)
	println("lineName:" + lineName + ",entityName" + entityName)
	if parentDir != "" {
		println("父级目录:" + parentDir)
	}
	replaceData := getBaseReplaceMap(entityName, parentDir)

	//1.开始生成：entity文件
	tPathEntity := TemplatePathEntity()
	oPathEntity := PathEntity(lineName)
	gen.ReplaceAndWriteTemplate(tPathEntity, oPathEntity, replaceData)

	//2.开始生成：router文件
	tPathRouter := TemplatePathRouter()
	oPathRouter := PathRouter(lineName, parentDir)
	gen.ReplaceAndWriteTemplate(tPathRouter, oPathRouter, replaceData)
	AddImportRouterPackage(lineName, parentDir) //添加router导入包

	//3.开始生成：controller文件
	tPathController := TemplatePathController()
	oPathController := PathController(lineName, parentDir)
	gen.ReplaceAndWriteTemplate(tPathController, oPathController, replaceData)

	//4.开始生成：service文件
	tPathService := TemplatePathService()
	oPathService := PathService(lineName, parentDir)
	gen.ReplaceAndWriteTemplate(tPathService, oPathService, replaceData)

	//5.开始生成：model文件
	tPathModel := TemplatePathModel()
	oPathModel := PathModel(lineName, parentDir)
	gen.ReplaceAndWriteTemplate(tPathModel, oPathModel, replaceData)

	//6.开始生成：fields文件（先生成默认字段 后续修改实体字段需要重新生成字段常量）
	tPathFields := TemplatePathFields()
	oPathFields := PathFields(lineName, parentDir)
	gen.ReplaceAndWriteTemplate(tPathFields, oPathFields, replaceData)
}
