package desc

import (
	"ginp-api/internal/gen"
)

// entityName传入大驼峰如 UserGroup
func GenEntity() {

	entityName := gen.Input(Select1, nil)
	entityName = gen.NameToCameBig(entityName)
	if entityName == "" {
		println("实体名称不能为空")
		return
	}
	lineName := gen.NameToLine(entityName)
	println("lineName:" + lineName + ",entityName" + entityName)
	replaceData := getBaseReplaceMap(entityName)

	//1.开始生成：entity文件
	tPathEntity := TemplatePathEntity()
	oPathEntity := PathEntity(lineName)
	gen.ReplaceAndWriteTemplate(tPathEntity, oPathEntity, replaceData)

	//2.开始生成：router文件
	tPathRouter := TemplatePathRouter()
	oPathRouter := PathRouter(lineName)
	gen.ReplaceAndWriteTemplate(tPathRouter, oPathRouter, replaceData)
	AddImportRouterPackage(lineName) //添加router导入包

	//3.开始生成：controller文件
	tPathController := TemplatePathController()
	oPathController := PathController(lineName)
	gen.ReplaceAndWriteTemplate(tPathController, oPathController, replaceData)

	//4.开始生成：service文件
	tPathService := TemplatePathService()
	oPathService := PathService(lineName)
	gen.ReplaceAndWriteTemplate(tPathService, oPathService, replaceData)

	//5.开始生成：model文件
	tPathRepository := TemplatePathModel()
	oPathRepository := PathModel(lineName)
	gen.ReplaceAndWriteTemplate(tPathRepository, oPathRepository, replaceData)

	//6.开始生成：fields文件（先生成默认字段 后续修改实体字段需要重新生成字段常量）
	tPathFields := TemplatePathFields()
	oPathFields := PathFields(lineName)
	gen.ReplaceAndWriteTemplate(tPathFields, oPathFields, replaceData)
}
