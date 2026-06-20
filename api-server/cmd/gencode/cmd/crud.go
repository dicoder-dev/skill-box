package cmd

import (
	"ginp-api/cmd/gencode/desc"
	"strings"

	"github.com/spf13/cobra"
)

// crudCmd 表示批量生成CRUD代码的命令
var crudCmd = &cobra.Command{
	Use:   "crud",
	Short: "批量生成多个实体的CRUD代码",
	Long:  `为 ginpapi 框架批量生成多个实体的CRUD代码，可以一次性生成多个实体的所有相关文件。`,
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

			// 设置实体名称列表并生成代码
			desc.GetPwd()
			desc.GenBatchCrudWithParent(cleanedEntities, parentDir)
		} else {
			// 如果没有提供实体名称列表，使用交互模式
			desc.GetPwd()
			desc.GenBatchCrudInteractive()
		}
	},
}

func init() {
	genCmd.AddCommand(crudCmd)

	// 为crud命令添加标志
	crudCmd.Flags().StringP("entities", "e", "", "要生成CRUD代码的实体名称列表，多个实体用逗号分隔（例如：UserGroup,UserRole）")
	crudCmd.Flags().StringP("parent", "p", "", "父级目录名称（例如：user）")
}
