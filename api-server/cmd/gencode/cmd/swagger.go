package cmd

import (
	"fmt"
	"ginp-api/pkg/gencode/swagen"
	"ginp-api/pkg/ginp"
	"os"

	"github.com/spf13/cobra"
)

// swaggerCmd Swagger文档生成命令
var swaggerCmd = &cobra.Command{
	Use:   "swagger",
	Short: "生成Swagger文档",
	Long:  `根据系统路由自动生成Swagger API文档`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取命令行参数
		host, _ := cmd.Flags().GetString("host")
		title, _ := cmd.Flags().GetString("title")
		version, _ := cmd.Flags().GetString("version")
		description, _ := cmd.Flags().GetString("description")
		saveDir, _ := cmd.Flags().GetString("output")

		// 设置默认值
		if host == "" {
			host = "localhost:8082"
		}
		if title == "" {
			title = "GINP API 文档"
		}
		if version == "" {
			version = "1.0.0"
		}
		if description == "" {
			description = "GINP框架自动生成的API文档"
		}
		if saveDir == "" {
			saveDir = "./static/docs"
		}

		fmt.Println("=== 生成Swagger文档 ===")
		fmt.Printf("主机地址: %s\n", host)
		fmt.Printf("文档标题: %s\n", title)
		fmt.Printf("版本号: %s\n", version)
		fmt.Printf("保存目录: %s\n", saveDir)
		fmt.Println()

		// 创建Swagger生成器
		swaggerInfo := swagen.SwaggerInfo{
			Host:        host,
			Title:       title,
			Version:     version,
			Description: description,
		}

		swaGen := swagen.NewSwaGen(swaggerInfo)

		// 获取所有路由并添加到生成器
		routers := ginp.GetAllRouter()
		swaGen.AddApis(routers)

		fmt.Printf("正在生成文档，共 %d 个API接口...\n", len(routers))

		// 保存swagger文档
		err := swaGen.Save(saveDir)
		if err != nil {
			fmt.Printf("生成swagger文档失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		fmt.Printf("✓ Swagger文档生成成功！\n")
		fmt.Printf("文档位置: %s/swagger.json\n", saveDir)
		fmt.Println()
		fmt.Println("访问方式：")
		fmt.Printf("  后端服务地址/swagger/index.html\n")
		fmt.Printf("  例如: http://%s/swagger/index.html\n", host)
	},
}

func init() {
	// 添加命令行标志
	swaggerCmd.Flags().StringP("host", "H", "localhost:8082", "API主机地址")
	swaggerCmd.Flags().StringP("title", "t", "GINP API 文档", "API文档标题")
	swaggerCmd.Flags().StringP("version", "v", "1.0.0", "API版本号")
	swaggerCmd.Flags().StringP("description", "d", "GINP框架自动生成的API文档", "API文档描述")
	swaggerCmd.Flags().StringP("output", "o", "./static/docs", "文档保存目录")

	// 注册到根命令
	rootCmd.AddCommand(swaggerCmd)
}
