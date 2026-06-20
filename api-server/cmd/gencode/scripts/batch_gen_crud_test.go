package script

import (
	"ginp-api/cmd/gencode/desc"
	"ginp-api/internal/gen"
	"testing"
)

func TestBatchGenCrud(t *testing.T) {
	// 初始化工作目录
	desc.GetPwd()

	// 测试实体列表
	entities := []string{
		// "XlTrip",
		"XlTripItem",
		"XlTripRoute",
		"XlTripSay",
		"XlTripSayImage",
		"XlTripBill",
		"XlLocation",
	}

	// 为每个实体生成CRUD代码
	for _, entityName := range entities {
		t.Logf("正在生成实体: %s", entityName)

		// 转换为大驼峰命名
		entityNameBig := gen.NameToCameBig(entityName)
		if entityNameBig == "" {
			t.Errorf("实体名称不能为空: %s", entityName)
			continue
		}

		// 转换为下划线命名
		lineName := gen.NameToLine(entityNameBig)
		t.Logf("lineName: %s, entityName: %s", lineName, entityNameBig)

		// 获取替换数据
		replaceData := desc.GetBaseReplaceMap(entityNameBig)

		// 1. 生成entity文件
		tPathEntity := desc.TemplatePathEntity()
		oPathEntity := desc.PathEntity(lineName)
		gen.ReplaceAndWriteTemplate(tPathEntity, oPathEntity, replaceData)

		// 2. 生成router文件
		tPathRouter := desc.TemplatePathRouter()
		oPathRouter := desc.PathRouter(lineName)
		gen.ReplaceAndWriteTemplate(tPathRouter, oPathRouter, replaceData)
		desc.AddImportRouterPackage(lineName) // 添加router导入包

		// 3. 生成controller文件
		tPathController := desc.TemplatePathController()
		oPathController := desc.PathController(lineName)
		gen.ReplaceAndWriteTemplate(tPathController, oPathController, replaceData)

		// 4. 生成service文件
		tPathService := desc.TemplatePathService()
		oPathService := desc.PathService(lineName)
		gen.ReplaceAndWriteTemplate(tPathService, oPathService, replaceData)

		// 5. 生成model文件
		tPathRepository := desc.TemplatePathModel()
		oPathRepository := desc.PathModel(lineName)
		gen.ReplaceAndWriteTemplate(tPathRepository, oPathRepository, replaceData)

		// 6. 生成fields文件
		tPathFields := desc.TemplatePathFields()
		oPathFields := desc.PathFields(lineName)
		gen.ReplaceAndWriteTemplate(tPathFields, oPathFields, replaceData)

		t.Logf("实体 %s 的CRUD代码生成完成", entityName)
	}

	t.Log("所有实体的CRUD代码生成完成")
}
