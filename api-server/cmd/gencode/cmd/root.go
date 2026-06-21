package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.2.0"

// rootCmd 表示在不调用任何子命令时的基本命令
var rootCmd = &cobra.Command{
	Use:   "gencode",
	Short: "GINP 代码生成工具",
	Long: `GINP 代码生成工具 - 基于 service 层的命令行工具

支持的功能：
  - entity: 实体管理（列表、生成、删除）
  - api: API接口管理（添加、列表）
  - swagger: 生成Swagger文档

使用示例：
  gencode entity list              # 列出所有实体
  gencode entity gen SysUser       # 生成实体CRUD代码
  gencode api add                  # 交互式添加API
  gencode swagger                  # 生成Swagger文档`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有提供子命令，显示帮助信息
		cmd.Help()
	},
}

// Execute 将所有子命令添加到根命令并适当设置标志。
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// 添加版本标志
	rootCmd.Flags().BoolP("version", "v", false, "打印版本号")
	
	// 重写使用 -v 标志时的默认行为
	rootCmd.SetVersionTemplate(`{{printf "GAPI 版本: %s" .Version}}`)
	rootCmd.Version = version
}