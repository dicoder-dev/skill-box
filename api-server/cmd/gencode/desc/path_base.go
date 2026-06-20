package desc

import (
	"os"
	"path/filepath"
	"strings"
)

func GetPwd() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// println("pwd:", pwd)
	return pwd
}

func GetDirProject() string {
	pwd := GetPwd()
	if strings.Contains(pwd, "/cmd") {
		// /Users/xiaod/projects_new/my/ginpapi/cmd/gencode
		arr := strings.Split(pwd, "/cmd")
		return arr[0]
	}
	return pwd
}

func GetDirAPP() string {
	projetDir := GetDirProject()
	return filepath.Join(projetDir, "internal", "app", "gapi")
}
func GetDirGencode() string {
	projetDir := GetDirProject()
	return filepath.Join(projetDir, "cmd", "gencode")
}

func GetDirTemplate() string {
	return filepath.Join(GetDirGencode(), "templates")
}
func GetDirRouter() string {
	return filepath.Join(GetDirAPP(), "router")
}

func GetDirEntidy() string {
	return filepath.Join(GetDirAPP(), "entity")
}

func GetDirController() string {
	return filepath.Join(GetDirAPP(), "controller")
}
func GetDirService() string {
	return filepath.Join(GetDirAPP(), "service")
}
func GetDirModel() string {
	return filepath.Join(GetDirAPP(), "model")
}
