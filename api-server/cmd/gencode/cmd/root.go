package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.2"

// rootCmd 表示在不调用任何子命令时的基本命令
var rootCmd = &cobra.Command{
	Use:   "gapi",
	Short: "GAPI - ginpapi的代码生成工具",
	Long: `GAPI 是一个用于生成 ginpapi 框架代码的命令行工具。

它可以生成实体、字段常量和API控制器。`,
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