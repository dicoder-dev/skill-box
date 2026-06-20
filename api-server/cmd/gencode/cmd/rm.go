package cmd

import (
	"ginp-api/cmd/gencode/desc"
	"strings"

	"github.com/spf13/cobra"
)

// rmCmd 表示删除代码的命令
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "删除 ginpapi 生成的代码文件",
	Long: `删除 ginpapi 框架生成的代码文件。

此命令可以删除实体的CRUD代码文件。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有提供子命令，显示帮助信息
		cmd.Help()
	},
}

// crudRmCmd 表示批量删除CRUD代码的命令
var crudRmCmd = &cobra.Command{
	Use:   "crud",
	Short: "批量删除多个实体的CRUD代码",
	Long:  `删除 ginpapi 框架批量生成的多个实体的CRUD代码，可以一次性删除多个实体的所有相关文件。`,
	Run: func(cmd *cobra.Command, args []string) {
		entities, _ := cmd.Flags().GetString("entities")
		parentDir, _ := cmd.Flags().GetString("parent")
		if entities != "" {
			// 分割实体名称列表
			entityList := strings.Split(entities, ",")

			// 清理实体名称（去除空格）
			cleanedEntities := []string{}
			for _, entity := range entityList {
				entity = strings.TrimSpace(entity)
				if entity != "" {
					cleanedEntities = append(cleanedEntities, entity)
				}
			}

			// 设置实体名称列表并删除代码
			desc.GetPwd()
			desc.RemoveBatchCrudWithParent(cleanedEntities, parentDir)
		} else {
			// 如果没有提供实体名称列表，使用交互模式
			desc.GetPwd()
			desc.RemoveBatchCrudInteractive()
		}
	},
}

func init() {
	// 将rm命令添加到gen命令
	genCmd.AddCommand(rmCmd)

	// 将crud子命令添加到rm命令
	rmCmd.AddCommand(crudRmCmd)

	// 为crud删除命令添加标志
	crudRmCmd.Flags().StringP("entities", "e", "", "要删除CRUD代码的实体名称列表，多个实体用逗号分隔（例如：UserGroup,UserRole）")
	crudRmCmd.Flags().StringP("parent", "p", "", "父级目录名称（例如：user）")
}
