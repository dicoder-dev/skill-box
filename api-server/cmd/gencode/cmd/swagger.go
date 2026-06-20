package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ginp-api/cmd/gencode/swagen"
	"ginp-api/internal/gapi/router"

	"github.com/spf13/cobra"
)

// swaggerCmd 表示 swagger 命令
var swaggerCmd = &cobra.Command{
	Use:   "swagger",
	Short: "生成 Swagger API 文档",
	Long: `生成 Swagger API 文档命令。
该命令会扫描所有的路由并生成对应的 Swagger YAML 文档。
默认保存到 server/cmd/gapi/static/docs/swagger.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		generateSwagger()
	},
}

func init() {
	// 添加 swagger 命令到根命令
	rootCmd.AddCommand(swaggerCmd)
}

// generateSwagger 生成 Swagger 文档
func generateSwagger() {
	fmt.Println("开始生成 Swagger API 文档...")

	// 创建 Swagger 生成器
	s := swagen.NewSwaGen(swagen.SwaggerInfo{
		Host:        "/",
		Title:       "API 文档",
		Version:     "2.0",
		Description: "API 文档",
	})

	// 获取所有路由
	routers := router.GetAllRouter()
	fmt.Printf("找到 %d 个路由\n", len(routers))

	// 添加 API 到 Swagger
	s.AddApis(routers)

	// 设置保存路径
	saveDir := "../gapi/static/docs"

	// 确保目录存在
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}

	// 生成swagger文档
	if err := s.Save(saveDir); err != nil {
		fmt.Printf("生成 Swagger 文档失败: %v\n", err)
		return
	}

	// 输出成功信息
	swaggerPath := filepath.Join(saveDir, "swagger.yaml")
	fmt.Printf("✅ Swagger 文档生成成功！\n")
	fmt.Printf("📄 文档路径: %s\n", swaggerPath)
	fmt.Printf("🌐 可以通过 Swagger UI 查看文档\n")
}
