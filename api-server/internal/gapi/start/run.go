// Package start 是 gapi 历史代码使用的入口包,
// 实际逻辑已迁到 cmd/bootstrap,这里保留为薄壳,保证:
//   - cmd/gapi/main.go 的 import 路径不变
//   - start.ConfigFile 兼容旧引用
//   - DefaultConfigFile / EntityAutoMigrateList 仍可被 internal 其它包访问
//
// 新代码请直接 import ginp-api/cmd/bootstrap。
package start

import "ginp-api/cmd/bootstrap"

// Re-exports for backwards compatibility.
var (
	DefaultConfigFile     = bootstrap.DefaultConfigFile
	EntityAutoMigrateList = bootstrap.EntityAutoMigrateList
	EntityGenerationList  = bootstrap.EntityGenerationList
)

// ConfigFile 兼容旧引用;读 / 写都转发到 bootstrap.ConfigFile。
var ConfigFile = bootstrap.ConfigFile

// Options 类型别名,允许旧代码用 start.Options{...}。
type Options = bootstrap.BootOptions

// Run 转发到 bootstrap.Run。
func Run(opts Options) {
	bootstrap.Run(opts)
}
