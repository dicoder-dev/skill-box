package cmd

import (
	"encoding/json"
	"fmt"
	"ginp-api/internal/app/gapi/service/system/sgen"
	"os"

	"github.com/spf13/cobra"
)

// apiCmd API相关命令
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "API管理命令",
	Long:  `管理API接口的创建和查询操作`,
}

// apiListCmd 列出所有API
var apiListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有API",
	Long:  `列出系统中所有已注册的API接口信息`,
	Run: func(cmd *cobra.Command, args []string) {
		// 调用 service 层获取API列表
		apiList := sgen.GetApiList()

		if len(apiList) == 0 {
			fmt.Println("没有找到任何API")
			return
		}

		// 格式化输出
		fmt.Printf("共找到 %d 个API:\n\n", len(apiList))
		for i, api := range apiList {
			fmt.Printf("%d. %s\n", i+1, api["api_path"])
			if title, ok := api["title"].(string); ok && title != "" {
				fmt.Printf("   标题: %s\n", title)
			}
			if method, ok := api["method"].(string); ok {
				fmt.Printf("   方法: %s\n", method)
			}
			if permName, ok := api["permission_name"].(string); ok {
				fmt.Printf("   权限: %s\n", permName)
			}
			if needLogin, ok := api["need_login"].(bool); ok {
				fmt.Printf("   需要登录: %v\n", needLogin)
			}
			if needPerm, ok := api["need_permission"].(bool); ok {
				fmt.Printf("   需要权限: %v\n", needPerm)
			}
			fmt.Println()
		}
	},
}

// apiAddCmd 添加新API
var apiAddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加新API",
	Long:  `通过交互式方式或JSON文件添加新的API接口`,
	Run: func(cmd *cobra.Command, args []string) {
		// 检查是否提供了JSON文件
		jsonFile, _ := cmd.Flags().GetString("file")
		if jsonFile != "" {
			// 从JSON文件读取
			addApiFromFile(jsonFile)
			return
		}

		// 交互式添加
		addApiInteractive()
	},
}

// addApiFromFile 从JSON文件添加API
func addApiFromFile(filePath string) {
	// 读取JSON文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		os.Exit(1)
	}

	// 解析JSON
	var apiInfo sgen.ApiInfo
	if err := json.Unmarshal(data, &apiInfo); err != nil {
		fmt.Printf("解析JSON失败: %v\n", err)
		os.Exit(1)
	}

	// 调用 service 层添加API
	if err := sgen.AddApiInfo(&apiInfo); err != nil {
		fmt.Printf("添加API失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ API %s 添加成功！\n", apiInfo.ApiName)
}

// addApiInteractive 交互式添加API
func addApiInteractive() {
	var apiInfo sgen.ApiInfo

	fmt.Println("=== 添加新API ===")
	fmt.Println()

	// API名称（大驼峰）
	fmt.Print("API名称（大驼峰，如 GetUserInfo）: ")
	fmt.Scanln(&apiInfo.ApiName)
	if apiInfo.ApiName == "" {
		fmt.Println("API名称不能为空")
		os.Exit(1)
	}

	// API标题
	fmt.Print("API标题（如：获取用户信息）: ")
	fmt.Scanln(&apiInfo.Title)

	// API描述
	fmt.Print("API描述（可选）: ")
	fmt.Scanln(&apiInfo.Description)

	// 一级目录
	fmt.Print("一级目录名称（如 system 或 center）: ")
	fmt.Scanln(&apiInfo.FirstDirName)
	if apiInfo.FirstDirName == "" {
		fmt.Println("一级目录名称不能为空")
		os.Exit(1)
	}

	// 二级目录
	fmt.Print("二级目录名称（可选，如 cuser）: ")
	fmt.Scanln(&apiInfo.SecondDirName)

	// HTTP方法
	fmt.Print("HTTP方法（GET/POST/PUT/DELETE，默认POST）: ")
	var method string
	fmt.Scanln(&method)
	if method == "" {
		method = "POST"
	}
	apiInfo.Method = method

	// 是否需要登录
	fmt.Print("是否需要登录？(y/N): ")
	var needLogin string
	fmt.Scanln(&needLogin)
	apiInfo.NeedLogin = needLogin == "y" || needLogin == "Y"

	// 是否需要权限
	fmt.Print("是否需要权限？(y/N): ")
	var needPerm string
	fmt.Scanln(&needPerm)
	apiInfo.NeedPermission = needPerm == "y" || needPerm == "Y"

	// API路径（可选）
	fmt.Print("API路径（可选，留空自动生成）: ")
	fmt.Scanln(&apiInfo.ApiPath)

	fmt.Println()
	fmt.Println("正在生成API文件...")

	// 调用 service 层添加API
	if err := sgen.AddApiInfo(&apiInfo); err != nil {
		fmt.Printf("添加API失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ API %s 添加成功！\n", apiInfo.ApiName)
	fmt.Println()
	fmt.Println("提示：请手动编辑生成的文件以实现具体的业务逻辑")
}

// apiTemplateCmd 生成API模板JSON
var apiTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "生成API模板JSON",
	Long:  `生成一个API配置的JSON模板文件，可以编辑后使用 add --file 命令添加`,
	Run: func(cmd *cobra.Command, args []string) {
		// 创建模板
		template := sgen.ApiInfo{
			ApiName:        "ExampleApi",
			Title:          "示例API",
			Description:    "这是一个示例API",
			FirstDirName:   "system",
			SecondDirName:  "cexample",
			Method:         "POST",
			NeedLogin:      true,
			NeedPermission: true,
			ApiPath:        "/api/system/example/example_api",
			Params: []sgen.ParamItem{
				{
					ParamName:     "UserId",
					ParamJsonTag:  "user_id",
					ParamType:     "int64",
					ParamLocation: "body",
					ParamComment:  "用户ID",
					IsRequired:    true,
				},
				{
					ParamName:     "UserName",
					ParamJsonTag:  "user_name",
					ParamType:     "string",
					ParamLocation: "body",
					ParamComment:  "用户名称",
					IsRequired:    false,
				},
			},
			ResponseFields: []sgen.ResponseFieldItem{
				{
					FieldName:    "Success",
					FieldJsonTag: "success",
					FieldType:    "bool",
					FieldComment: "是否成功",
				},
				{
					FieldName:    "Message",
					FieldJsonTag: "message",
					FieldType:    "string",
					FieldComment: "返回消息",
				},
			},
		}

		// 序列化为JSON
		data, err := json.MarshalIndent(template, "", "  ")
		if err != nil {
			fmt.Printf("生成模板失败: %v\n", err)
			os.Exit(1)
		}

		// 输出文件名
		outputFile, _ := cmd.Flags().GetString("output")
		if outputFile == "" {
			outputFile = "api_template.json"
		}

		// 写入文件
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			fmt.Printf("写入文件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ API模板已生成: %s\n", outputFile)
		fmt.Println()
		fmt.Println("提示：编辑此文件后，使用以下命令添加API：")
		fmt.Printf("  gencode api add --file %s\n", outputFile)
	},
}

func init() {
	// 添加子命令
	apiCmd.AddCommand(apiListCmd)
	apiCmd.AddCommand(apiAddCmd)
	apiCmd.AddCommand(apiTemplateCmd)

	// 为 add 命令添加 file 标志
	apiAddCmd.Flags().StringP("file", "f", "", "从JSON文件读取API配置")

	// 为 template 命令添加 output 标志
	apiTemplateCmd.Flags().StringP("output", "o", "api_template.json", "输出文件名")

	// 注册到根命令
	rootCmd.AddCommand(apiCmd)
}
