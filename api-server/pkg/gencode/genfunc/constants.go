package genfunc

import "ginp-api/pkg/gencode/gen"

const (
	ReplaceEntityName = "$ENTITY_NAME$"
	ReplaceLineName   = "$ENTITY_LINE$"
	//全小写命名
	ReplacePackageName      = "$PACKAGE_NAME$"
	ReplaceApiNameBig       = "$API_NAME_BIG$"
	ReplaceApiNameLine      = "$API_NAME_LINE$"
	PlaceholderRouterImport = "//{{placeholder_router_import}}//"
	RouterReplaceStr        = `_ "ginp-api/internal/app/gapi/controller/`
)

// 基础替换数据 传入大驼峰如 $ENTITY_NAME$Group
func getBaseReplaceMap(BigCameName string, parentDir ...string) map[string]string {
	BigCameName = gen.NameToCameBig(BigCameName)
	lineName := gen.NameToLine(BigCameName)
	var replaceData map[string]string = map[string]string{
		ReplaceEntityName:  BigCameName,
		ReplaceLineName:    lineName,
		ReplacePackageName: gen.NameToAllSmall(lineName),
		"$PARENT_DIR$":     "",
	}

	if len(parentDir) > 0 && parentDir[0] != "" {
		replaceData["$PARENT_DIR$"] = parentDir[0] + "/"
	}
	return replaceData
}

// GetBaseReplaceMap 公共接口，用于外部包调用
func GetBaseReplaceMap(BigCameName string) map[string]string {
	return getBaseReplaceMap(BigCameName)
}
