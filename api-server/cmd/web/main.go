// Package main Web 端单进程入口:把前端 dist 通过 embed.FS 注入 api-server,
// 一份二进制同时承担静态资源服务 + 业务接口。
//
// 桌面端不在这里:桌面端通过 skill-box/main.go 在进程内起 ginp-api/cmd/bootstrap,
// 走 in-process 模式。
//
// 启动流程复用 ginp-api/cmd/bootstrap.Run,确保 Web 和传统 gapi 走的
// 配置/DB/Task/Logger 初始化逻辑一致,避免双部署出现行为分裂。
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"ginp-api/cmd/bootstrap"
	"ginp-api/configs"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	configPath := flag.String("config", bootstrap.DefaultConfigFile, "配置文件路径(yaml)")
	flag.Parse()
	if env := os.Getenv("CONFIG"); env != "" {
		*configPath = env
	}

	// 2026-07-09 调试:打印进程内 HTTPS_PROXY(看 init() 是否真设了)
	fmt.Printf("[DEBUG web main] HTTPS_PROXY=%q HTTP_PROXY=%q https_proxy=%q\n",
		os.Getenv("HTTPS_PROXY"), os.Getenv("HTTP_PROXY"), os.Getenv("https_proxy"))

	// embed 路径 "frontend/dist" 在 fs 里保留了目录前缀,先 Sub 出 dist 子 FS,
	// 让 server 拿到的 FrontRootFS/StaticFS 已经是 dist 根。
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("sub frontend/dist failed: %v", err)
	}

	fmt.Printf("web: starting with config %s\n", *configPath)
	// 注意:ServerOptions 是函数,在 bootstrap 内部(已加载 cfg)才调用,
	// 所以这里读 configs.ServerPort() 是安全的。
	bootstrap.Run(bootstrap.BootOptions{
		ConfigFile: *configPath,
		ServerOptions: func(runMode string) bootstrap.ServerOptions {
			return bootstrap.ServerOptions{
				Addr:        "0.0.0.0:" + configs.Server.Port,
				StaticFS:    distFS,
				FrontRootFS: distFS,
				RunMode:     runMode,
			}
		},
	})
}
