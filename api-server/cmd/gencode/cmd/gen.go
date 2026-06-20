package cmd

import (
	"ginp-api/cmd/gencode/desc"

	"github.com/spf13/cobra"
)

// genCmd 表示生成代码的命令
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "为 ginpapi 生成代码",
	Long: `为 ginpapi 框架生成代码。

此命令可以生成实体、字段常量和API控制器。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有提供子命令，显示帮助信息
		cmd.Help()
	},
}

// entityCmd 表示实体命令
var entityCmd = &cobra.Command{
	Use:   "entity",
	Short: "创建实体并生成CRUD代码",
	Long:  `为 ginpapi 框架创建实体并生成CRUD代码。`,
	Run: func(cmd *cobra.Command, args []string) {
		entityName, _ := cmd.Flags().GetString("create")
		parentDir, _ := cmd.Flags().GetString("parent")
		if entityName != "" {
			// 设置实体名称并生成代码
			desc.GetPwd()
			desc.GenEntityWithNameAndParent(entityName, parentDir)
		} else {
			// 如果没有提供实体名称，使用交互模式
			desc.GetPwd()
			desc.GenEntity()
		}
	},
}

// constCmd 表示常量命令
var constCmd = &cobra.Command{
	Use:   "const",
	Short: "生成实体字段常量",
	Long:  `为 ginpapi 框架生成实体字段常量。`,
	Run: func(cmd *cobra.Command, args []string) {
		desc.GetPwd()
		desc.GenFields()
	},
}

// apiCmd 表示API命令
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "添加API接口控制器",
	Long:  `为 ginpapi 框架添加API接口控制器。`,
	Run: func(cmd *cobra.Command, args []string) {
		apiName, _ := cmd.Flags().GetString("add")
		dirPath, _ := cmd.Flags().GetString("dir")

		if apiName != "" && dirPath != "" {
			// 设置API名称和目录路径并生成代码
			desc.GetPwd()
			desc.GenAddApiWithParams(apiName, dirPath)
		} else {
			// 如果没有提供API名称或目录路径，使用交互模式
			desc.GetPwd()
			desc.GenAddApi()
		}
	},
}

func init() {
	rootCmd.AddCommand(genCmd)

	// 添加子命令到gen命令
	genCmd.AddCommand(entityCmd)
	genCmd.AddCommand(constCmd)
	genCmd.AddCommand(apiCmd)

	// 为entity命令添加标志
	entityCmd.Flags().StringP("create", "c", "", "要创建的实体名称（大驼峰命名，例如：UserGroup）")
	entityCmd.Flags().StringP("parent", "p", "", "父级目录名称（例如：user）")

	// 为api命令添加标志
	apiCmd.Flags().StringP("add", "a", "", "要添加的API名称（大驼峰命名，例如：GetUserInfo）")
	apiCmd.Flags().StringP("dir", "d", "", "API控制器的目录路径（例如：user/cuser）")
}
