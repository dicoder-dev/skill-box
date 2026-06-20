package templates

import (
	"ginp-api/pkg/gencode/gen"
	"ginp-api/pkg/gencode/genfunc"
	"testing"
)

// 快速将现有的文件做成模板文件
func TestReplace(t *testing.T) {

	demoEntityLineName := "demo_table"
	demoEntityBigName := "DemoTable"
	demoPackageName := "demotable"

	demoApiBigName := "DemoAddApi"
	demoApiLineName := "demo_add_api"

	replaceData := map[string]string{
		demoEntityLineName: genfunc.ReplaceLineName,
		demoEntityBigName:  genfunc.ReplaceEntityName,
		demoPackageName:    genfunc.ReplacePackageName,
		demoApiBigName:     genfunc.ReplaceApiNameBig,
		demoApiLineName:    genfunc.ReplaceApiNameLine,
	}

	//1.开始生成：entity文件
	tPathEntity := genfunc.TemplatePathEntity()
	oPathEntity := genfunc.PathEntity(demoEntityLineName)
	gen.ReplaceAndWriteTemplate(oPathEntity, tPathEntity, replaceData)

	//2.开始生成：router文件
	tPathRouter := genfunc.TemplatePathRouter()
	oPathRouter := genfunc.PathRouter(demoEntityLineName)
	gen.ReplaceAndWriteTemplate(oPathRouter, tPathRouter, replaceData)

	//3.开始生成：controller文件
	tPathController := genfunc.TemplatePathController()
	oPathController := genfunc.PathController(demoEntityLineName)
	gen.ReplaceAndWriteTemplate(oPathController, tPathController, replaceData)

	//4.开始生成：service文件
	tPathService := genfunc.TemplatePathService()
	oPathService := genfunc.PathService(demoEntityLineName)
	gen.ReplaceAndWriteTemplate(oPathService, tPathService, replaceData)

	//5.开始生成：model文件
	tPathRepository := genfunc.TemplatePathModel()
	oPathRepository := genfunc.PathModel(demoEntityLineName)
	gen.ReplaceAndWriteTemplate(oPathRepository, tPathRepository, replaceData)

	//6.开始生成：api文件
	// tPathApi := genfunc.TemplatePathAddApi()
	// oPathApi := genfunc.PathAddApi(demoEntityLineName, demoApiLineName)
	// println(oPathApi + "\n" + tPathApi)
	// gen.ReplaceAndWriteTemplate(oPathApi, tPathApi, replaceData)
}
