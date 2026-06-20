// Package swagen
// @Author: zhangdi
// @File: core1
// @Version: 1.0.0
// @Date: 2023/10/30 16:33
package swagen

import (
	"fmt"
	"ginp-api/pkg/filehelper"
	"ginp-api/pkg/ginp"
	"path/filepath"
	"reflect"

	"github.com/spf13/viper"
)

type SwaGen struct {
	SwaggerInfo SwaggerInfo `json:"swagger_info"`
	viper       *viper.Viper
}

func NewSwaGen(swaggerInfo SwaggerInfo) *SwaGen {
	v := viper.New()

	v.Set("swagger", "2.0")
	//v.Set(infoHost, swaggerInfo.Host)
	v.Set(infoDescription, swaggerInfo.Description)
	v.Set(infoTitle, swaggerInfo.Title)
	v.Set(infoVersion, swaggerInfo.Version)

	return &SwaGen{
		SwaggerInfo: swaggerInfo,
		viper:       v,
	}
}

func (s *SwaGen) AddApis(routers []ginp.RouterItem) {
	for i := 0; i < len(routers); i++ {
		r := routers[i]
		if r.Swagger != nil && r.Swagger.IsIgnore {
			//跳过该接口生成
			continue
		}
		preKey := fmt.Sprintf("paths.%s.%s.", r.Path, r.HttpType)
		s.viper.Set(preKey+apiPathMethod, r.HttpType)                                                                            //请求方式
		s.viper.Set(preKey+apiPathConsumes, getConsumes(r.Swagger))                                                              //指定【请求】发送的数据类型
		s.viper.Set(preKey+apiPathSummary, getTitle(r))                                                                          //接口标题
		s.viper.Set(preKey+apiPathDescription, getDescription(r))                                                                //接口描述
		s.viper.Set(preKey+apiPathProduces, getProduces(r.Swagger))                                                              //指定请求[返回]的数据类型
		s.viper.Set(preKey+apiPathResponsesOKDescription, "OK，返回正常")                                                             //状态码200的描述
		s.viper.Set(preKey+apiPathResponsesOKSchema, map[string]string{"$ref": fmt.Sprintf("#/definitions/%s", "respond_data")}) //状态码200的描述
		s.viper.Set(preKey+apiPathResponsesNotLoginDescription, "需要登录，当前未登录！")                                                   //状态码401的描述
		s.viper.Set(preKey+apiPathResponsesNotPermissionDescription, "无操作权限！")                                                   //状态码403的描述
		s.viper.Set(preKey+apiPathTags, getTags(r))                                                                              //获取分组

		//请求参数
		if r.Swagger != nil && r.Swagger.RequestDto != nil {
			s.createDefinitions(r.Swagger.RequestDto) //创建一个schema
			entityName := reflect.TypeOf(r.Swagger.RequestDto).Name()
			params := []ParamInfo{
				{
					In:     "body",
					Name:   entityName,
					Schema: map[string]string{"$ref": fmt.Sprintf("#/definitions/%s", NameToLine(entityName))},
				},
			}
			s.viper.Set(preKey+"parameters", params)
		}

	}
}

// 添加一个schema
func (s *SwaGen) createDefinitions(obj any) {
	schema := GetStructSchemaInfo(obj)
	entityName := reflect.TypeOf(obj).Name()
	s.viper.Set(fmt.Sprintf("definitions.%s", NameToLine(entityName)), schema)
}

func (s *SwaGen) Save(saveDir string) error {
	//生成默认respond结构体
	s.createDefinitions(RespondData{
		Code: 1,
		Msg:  "OK",
		Data: nil,
	})
	dir := "./static/docs"
	if saveDir != "" {
		dir = saveDir
	}
	if !filehelper.FileExists(dir) {
		filehelper.CreateDir(dir)
	}
	if !filehelper.FileExists(filepath.Join(dir, "swagger.yaml")) {
		filehelper.CreateFile(filepath.Join(dir, "swagger.yaml"))
	}
	s.viper.AddConfigPath(dir)       // 路径(当前路径下的conf文件夹)
	s.viper.SetConfigName("swagger") // 名称
	s.viper.SetConfigType("yaml")    // 类型

	err := s.viper.WriteConfig()
	return err
}
