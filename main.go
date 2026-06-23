// Package main 桌面端入口。
//
// 双部署形态:
//   - Web 端: 编译 api-server/cmd/web,一份二进制 = 静态前端 + 业务接口。
//   - 桌面端: 编译本 main.go,启动 in-process api-server + Wails Webview 加载它。
//
// 启动流程:
//  1. 调 bootstrap.Boot(在另一个 goroutine)→ 跑 cfg→DB→Task→Logger,返回 *Backend
//  2. 调 bootstrap.Serve(在另一个 goroutine)→ 阻塞跑 gin HTTP server
//  3. 调 desktop.NewApp + App.Run 跑 Wails 主循环
//
// 客户端(Wails 窗口)是可选的——可以只跑 backend(供 CLI / 测试 / 第三方前端用)。
// 客户端和后端的边界很清晰:
//
//	bootstrap.Boot + bootstrap.Serve  ←  进程内起后端,必启动
//	desktop.NewApp + App.Run           ←  构造 Wails 窗口/菜单/托盘,可选
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"

	"ginp-api/cmd/bootstrap"
	"skill-box/desktop"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	// 桌面端优先用项目根的 configs.yaml(便于开发期覆盖配置);
	// 真正的"数据目录"由 bootstrap.applyDataDir 在 RunMode=desktop 时接管。
	configPath := flag.String("config", bootstrap.DefaultConfigFile, "配置文件路径(yaml)")
	flag.Parse()

	// embed 路径 "frontend/dist" 在 fs 里保留了目录前缀,先 Sub 出 dist 子 FS。
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("sub frontend/dist failed: %v", err)
	}

	// 1) 后端:直接调 bootstrap.Boot + bootstrap.Serve(和 web/gapi 同一份启动流程)。
	//    Serve 是阻塞的,放 goroutine 里跑;Wails 主循环在另一个 goroutine。
	backend, err := bootstrap.Boot(bootstrap.BootOptions{
		ConfigFile: *configPath,
		RunMode:    "desktop",
		ServerOptions: func(runMode string) bootstrap.ServerOptions {
			return bootstrap.ServerOptions{
				StaticFS:    distFS,
				FrontRootFS: distFS,
				RunMode:     runMode,
			}
		},
	})
	if err != nil {
		log.Fatalf("bootstrap: Boot failed: %v", err)
	}
	log.Printf("desktop: backend ready at %s", backend.URL())
	go bootstrap.Serve(backend)

	// 2) 客户端:启动 Wails。如果以后要做"只跑后端 + 第三方前端"模式,
	// 把这一段替换成 select{} 阻塞即可。
	//
	// dev 模式:wails3 dev 启动前会自动注入 WAILS_VITE_PORT(默认 9245)。
	// 这里读出来后把 Webview 切到 Vite dev server,前端代码改动由 Vite HMR
	// 直接热替换,Go 进程不需要重启。否则按原逻辑加载 backend 内置 gin + embed.FS。
	app := desktop.NewApp(desktop.AppConfig{
		Name:        "Skill Box",
		FrontendURL: desktop.NewFrontendURLFromEnv("", 0),
	}, backend)

	// 3) 运行 Wails 主循环(阻塞)
	if err := app.Run(); err != nil {
		log.Printf("app run error: %v", err)
	}
}
