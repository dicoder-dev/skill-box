// Package appdir 提供应用数据目录的派生路径函数。
//
// 数据目录名 ~/.<AppName>/ 来自 share/constant.AppName。本包只做路径拼装,
// 不做 MkdirAll / 写文件等副作用,调用方按需自行落盘。
//
// 命名说明:本包放在 share/func/ 目录下,但 "func" 是 Go 关键字,不能作为
// package 名,所以用语义化的 appdir 作为包名(目录与包名解耦是 Go 允许的)。
package sharefunc

import (
	"os"
	"path/filepath"

	"ginp-api/share/constant"
)

// dataDirInternal 内部实现,允许在 home 解析失败时返回 "",由调用方决定兜底策略。
func dataDirInternal() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "."+constant.AppName)
}

// DataDir 返回应用数据目录的绝对路径(~/.<AppName>)。
// home 解析失败时返回 "";调用方按需 MkdirAll 后再使用。
func DataDir() string { return dataDirInternal() }

// LogsDir 返回日志目录的绝对路径,末尾带分隔符;不可用时返回 ""。
func LogsDir() string {
	d := dataDirInternal()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "logs") + string(filepath.Separator)
}

// DbPath 返回 sqlite 数据库文件绝对路径;不可用时返回 ""。
func DbPath() string {
	d := dataDirInternal()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "data.db")
}

// ConfigPath 返回配置文件绝对路径;不可用时返回 ""。
func ConfigPath() string {
	d := dataDirInternal()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "configs.yaml")
}
