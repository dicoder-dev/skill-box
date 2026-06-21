package cmd

import (
	"encoding/json"
	"fmt"
	"ginp-api/internal/app/gapi/service/system/sgen"
	"os"

	"github.com/spf13/cobra"
)

// entityCmd 实体相关命令
var entityCmd = &cobra.Command{
	Use:   "entity",
	Short: "实体管理命令",
	Long:  `管理实体的创建、查询和删除操作`,
}

// entityListCmd 列出所有实体
var entityListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有实体",
	Long:  `列出系统中所有已注册的实体信息，包括实体名称、字段数、表名等`,
	Run: func(cmd *cobra.Command, args []string) {
		// 调用 service 层获取实体列表
		entityList := sgen.GetEntityList()

		if len(entityList) == 0 {
			fmt.Println("没有找到任何实体")
			return
		}

		// 格式化输出
		fmt.Printf("共找到 %d 个实体:\n\n", len(entityList))
		for i, entity := range entityList {
			fmt.Printf("%d. %s\n", i+1, entity.EntityName)
			fmt.Printf("   标题: %s\n", entity.Title)
			fmt.Printf("   表名: %s\n", entity.TableName)
			fmt.Printf("   字段数: %d\n", entity.FieldCount)
			if entity.FatherFolderName != "" {
				fmt.Printf("   父级目录: %s\n", entity.FatherFolderName)
			}
			if entity.OptionsLabelField != "" {
				fmt.Printf("   选项标签字段: %s\n", entity.OptionsLabelField)
			}
			fmt.Println()
		}
	},
}

// entityInfoCmd 查看实体详细信息
var entityInfoCmd = &cobra.Command{
	Use:   "info [实体名称]",
	Short: "查看实体详细信息",
	Long:  `查看指定实体的详细信息，包括所有字段配置`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entityName := args[0]

		// 获取所有实体
		entityList := sgen.GetEntityList()

		// 查找指定实体
		var targetEntity *sgen.EntityInfo
		for _, entity := range entityList {
			if entity.EntityName == entityName {
				targetEntity = &entity
				break
			}
		}

		if targetEntity == nil {
			fmt.Printf("未找到实体: %s\n", entityName)
			os.Exit(1)
		}

		// 输出JSON格式（便于查看详细信息）
		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			data, err := json.MarshalIndent(targetEntity, "", "  ")
			if err != nil {
				fmt.Printf("序列化实体信息失败: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(data))
		} else {
			// 格式化输出
			fmt.Printf("实体名称: %s\n", targetEntity.EntityName)
			fmt.Printf("标题: %s\n", targetEntity.Title)
			fmt.Printf("表名: %s\n", targetEntity.TableName)
			fmt.Printf("字段数: %d\n", targetEntity.FieldCount)
			if targetEntity.FatherFolderName != "" {
				fmt.Printf("父级目录: %s\n", targetEntity.FatherFolderName)
			}
			if targetEntity.OptionsLabelField != "" {
				fmt.Printf("选项标签字段: %s\n", targetEntity.OptionsLabelField)
			}

			fmt.Println("\n字段列表:")
			for i, field := range targetEntity.Fields {
				fmt.Printf("\n%d. %s (%s)\n", i+1, field.FieldName, field.GoType)
				fmt.Printf("   数据库字段: %s\n", field.DbName)
				if field.DbComment != "" {
					fmt.Printf("   注释: %s\n", field.DbComment)
				}
				if field.DbIsPrimaryKey {
					fmt.Printf("   主键: 是\n")
				}
			}
		}
	},
}

// entityGenCmd 生成实体CRUD代码
var entityGenCmd = &cobra.Command{
	Use:   "gen [实体名称]",
	Short: "生成实体CRUD代码",
	Long:  `为指定实体生成完整的CRUD代码，包括Controller、Service、Model等`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entityName := args[0]

		// 获取所有实体
		entityList := sgen.GetEntityList()

		// 查找指定实体
		var targetEntity *sgen.EntityInfo
		for _, entity := range entityList {
			if entity.EntityName == entityName {
				targetEntity = &entity
				break
			}
		}

		if targetEntity == nil {
			fmt.Printf("未找到实体: %s\n", entityName)
			os.Exit(1)
		}

		fmt.Printf("正在为实体 %s 生成CRUD代码...\n", entityName)

		// 调用 service 层生成完整的实体代码（包括实体文件、CRUD代码、前端代码、菜单权限等）
		if err := sgen.SaveEntityInfo(targetEntity); err != nil {
			fmt.Printf("生成代码失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ 实体 %s 的代码生成成功！\n", entityName)
	},
}

// entityDeleteCmd 删除实体CRUD代码
var entityDeleteCmd = &cobra.Command{
	Use:   "delete [实体名称]",
	Short: "删除实体CRUD代码",
	Long:  `删除指定实体的所有CRUD代码文件和目录`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entityName := args[0]

		// 确认删除
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("确定要删除实体 %s 的所有CRUD代码吗？(y/N): ", entityName)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("已取消删除操作")
				return
			}
		}

		fmt.Printf("正在删除实体 %s 的CRUD代码...\n", entityName)

		// 调用 service 层删除CRUD代码
		if err := sgen.DeleteCrudFolders(entityName); err != nil {
			fmt.Printf("删除CRUD代码失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ 实体 %s 的CRUD代码删除成功！\n", entityName)
	},
}

func init() {
	// 添加子命令
	entityCmd.AddCommand(entityListCmd)
	entityCmd.AddCommand(entityInfoCmd)
	entityCmd.AddCommand(entityGenCmd)
	entityCmd.AddCommand(entityDeleteCmd)

	// 为 info 命令添加 json 标志
	entityInfoCmd.Flags().BoolP("json", "j", false, "以JSON格式输出")

	// 为 delete 命令添加 force 标志
	entityDeleteCmd.Flags().BoolP("force", "f", false, "强制删除，不需要确认")

	// 注册到根命令
	rootCmd.AddCommand(entityCmd)
}
