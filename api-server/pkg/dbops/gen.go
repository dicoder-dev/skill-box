// Package dbgen
// @Author: zhangdi
// @File: conf
// @Version: 1.0.0
// @Date: 2023/9/4 12:04
package dbops

// EntityConfig 实体配置，可在定义实体时定义GenConfig()方法返回该结构体
type EntityConfig struct {
	Title     string //中文标题
	TableName string
	//别名，建议大驼峰命名法,起名的实体相关的名称都会用这个名称
	AsName string
	//父级目录相对路径，传入system,或者system/
	//比如system,则实体相关文件会在system文件夹下创建
	ViewsFatherFolder string
	//是否仅生成后台代码
	OnlyGenBackendCode bool
}
