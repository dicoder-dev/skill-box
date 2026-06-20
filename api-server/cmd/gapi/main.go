// Package main 传统 gapi 入口:磁盘静态资源 + 配置在同级 configs.yaml。
//
// 用于本地开发和不依赖前端 dist embed 的部署形态。
// 复用 bootstrap.Run() 共享 DB / Task / Logger 启动逻辑。
package main

import (
	"flag"
	"fmt"
	"os"

	"ginp-api/cmd/bootstrap"
)

func main() {
	configPath := flag.String("config", bootstrap.DefaultConfigFile, "配置文件路径(yaml)")
	flag.Parse()
	// 允许 CONFIG 环境变量覆盖(便于 systemd / docker)。
	if env := os.Getenv("CONFIG"); env != "" {
		*configPath = env
	}
	fmt.Printf("gapi: starting with config %s\n", *configPath)
	bootstrap.Run(bootstrap.BootOptions{ConfigFile: *configPath})
}
