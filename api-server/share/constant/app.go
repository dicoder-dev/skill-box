// Package constant 集中维护跨模块的固定值。
//
// 命名约定:业务默认值(应用名、目录名等)统一放这里,避免散落到 configs /
// 各处 init() 里。任何需要"绝对路径"或"目录名"的代码都从这里读。
package constant

import (
	"os"
	"path/filepath"
)

// AppName 应用目录名(~/.<AppName>/)。
//
// 同时作为 sqlite 文件名、配置文件名、logs/ 子目录的统一前缀。
// 改值要慎重 — 已有数据目录会被遗留,不会自动迁移。
//
// 业务侧展示名仍由 configs.System.AppName 控制(允许不同,如"dianji"
// 是项目代号,展示名可以叫别的)。
const AppName = "skill-box"

// dataDirInternal 内部实现,允许在 home 解析失败时返回 "",由调用方决定兜底策略。
func dataDirInternal() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "."+AppName)
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
