package desc

import (
	"fmt"
	"ginp-api/internal/gen"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GenAddApi 生成API相关代码
// 主要功能：扫描controller目录下的所有子目录，并打印目录名和路径
func GenAddApi() {
	//获取controller目录路径
	controllerDir := GetDirController()

	//获取目录列表
	folderMap := getControllerDirs(controllerDir)

	//选择Api创建目录
	selectDirName := InputApiDir(folderMap)

	apiDir := filepath.Join(controllerDir, selectDirName)

	//获取实体名称
	// entityLineName := getEntityLineName(apiDir)
	// if entityLineName == "" {
	// 	return
	// }

	//提示用户输入api名称
	apiName := gen.Input(Select3ApiName, nil)
	if apiName == "" || strings.Contains(apiName, " ") {
		fmt.Println(`api名称不能为空,且不能包含空格（API name cannot be empty and cannot contain spaces)`)
		return
	}
	apiNameBig := gen.NameToCameBig(apiName)
	apiNameLine := gen.NameToLine(apiNameBig)
	//扫描apiDir下的所有文件匹配到c.go结尾的文件
	apiPath := filepath.Join(apiDir, apiNameLine+".a.go")
	replaceData := getBaseReplaceMap(selectDirName)
	replaceData[ReplaceApiNameBig] = apiNameBig   //api名称 大驼峰
	replaceData[ReplaceApiNameLine] = apiNameLine //api名称 下划线

	//如果selectDirName以c开头,则去掉c
	packname := selectDirName
	if len(selectDirName) > 0 && selectDirName[0] == 'c' {
		packname = strings.TrimLeft(selectDirName, "c")
	}
	replaceData[ReplacePackageName] = packname         //包名 全小写
	replaceData[ReplaceEntityName] = selectDirName[1:] //实体名称 大驼峰
	println("selectDirName:", gen.NameToAllSmall(selectDirName))
	gen.ReplaceAndWriteTemplate(TemplatePathAddApi(), apiPath, replaceData)

}

func getControllerDirs(controllerDir string) []string {
	// 创建map用于存储目录名和路径的映射
	folderList := make([]string, 0)

	// 读取controller目录下的所有文件和子目录
	dirs, err := os.ReadDir(controllerDir)
	if err != nil {
		panic(err)
	}

	// 遍历目录项，只处理子目录
	// index := 1
	for _, dir := range dirs {
		if dir.IsDir() {
			// 将目录名和完整路径存入map
			folderList = append(folderList, dir.Name())
			// folderMap[fmt.Sprintf("%v", index)] = dir.Name()
		}
	}

	// 打印所有目录名和路径
	for i, name := range folderList {
		fmt.Printf("%d.%s\n", i+1, name)
	}

	return folderList
}

func InputApiDir(folderList []string) string {

	// 重新打印排序后的目录列表
	// for i, k := range keys {
	// 	fmt.Printf("%d.%s\n", i+1, k)
	// }

	inputCode := gen.Input(Select3, nil)
	//转成int
	code, err := strconv.ParseInt(inputCode, 10, 64)
	if err != nil {
		fmt.Println("输入的代码不是数字，请重新输入")
		return ""
	}

	//判断code可取
	if code < 1 || code > int64(len(folderList)) {
		fmt.Println("输入的代码不在范围内，请重新输入")
		return ""
	}

	println("select dir:", folderList[code-1])
	return folderList[code-1]
}

func getEntityLineName(apiDir string) string {
	apiFiles, err := os.ReadDir(apiDir)
	if err != nil {
		println("getEntityLineName error :" + err.Error())
		return ""
	}
	entityLineName := ""
	for _, apiFile := range apiFiles {
		if apiFile.IsDir() {
			continue
		}
		if strings.HasSuffix(apiFile.Name(), ".c.go") {
			//替换apiFile.Name()中的c.go为a.go
			entityLineName = strings.Replace(apiFile.Name(), ".c.go", "", -1)
			println("entityLineName:", entityLineName)
			break
		}
	}
	if entityLineName == "" {
		fmt.Println(apiDir + "下没有找到.c.go结尾的文件！")
		return ""
	}
	return entityLineName
}
