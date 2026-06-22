package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// apiCmd API相关命令
// 注：API 列表/添加等功能依赖 ginp-api/internal/app/gapi/service/system/sgen 包，
// 该包当前未在仓库中实现。暂时禁用本组命令，避免编译失败。
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "API管理命令（暂未启用）",
	Long:  `API管理命令暂未启用，等待 sgen 服务层实现后开放。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("api 命令暂未启用")
		os.Exit(1)
	},
}

func init() {
	// 注册到根命令（占位，未挂载子命令）
	rootCmd.AddCommand(apiCmd)
}
