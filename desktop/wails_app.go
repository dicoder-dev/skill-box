package desktop

import (
	"context"
	"io/fs"
	"time"

	"ginp-api/cmd/bootstrap"
	"skill-box/desktop/services"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// AppConfig 描述桌面端 Wails 应用的全部配置。
// 调用方在 main.go 构造并传给 NewApp,NewApp 内部完成 Wails 全部组装并返回 *App。
type AppConfig struct {
	// Name 应用名,显示在菜单栏 / 标题栏。
	Name string
	// Description 应用描述,部分系统会用到。
	Description string
	// Width / Height 主窗口初始尺寸。
	Width, Height int
	// MinWidth / MinHeight 主窗口最小尺寸。
	MinWidth, MinHeight int
	// BackgroundColour 主窗口背景色(R,G,B),各分量 0-255。
	BackgroundColour [3]uint8
}

// App 包装 *application.App,提供 Quit 优雅退出。
//
// 注意:本结构体只关心 Wails UI 相关的状态,后端 server 由调用方在 NewApp 之前
// 通过 bootstrap.Boot 启动并 Serve 阻塞。后端生命周期跟 App 解耦 ——
// App 只在退出时通知 Wails,后端 server 由 main 的 Serve 阻塞在另一个 goroutine。
type App struct {
	app     *application.App
	backend *bootstrap.Backend
}

// NewApp 构造并完整组装桌面端 Wails 应用:
//   - 注册 Wails Bind 服务(AppService / WindowService / PlatformService)
//   - 创建主窗口,加载 backend.URL
//   - 装载应用菜单和系统托盘
//
// 注意:此函数不会阻塞。Run() 才会阻塞直到应用退出。
// 调用方应保证 backend 已经在 NewApp 之前通过 bootstrap.Boot 启动。
func NewApp(cfg AppConfig, backend *bootstrap.Backend) *App {
	if cfg.Name == "" {
		cfg.Name = "skill-box"
	}
	if cfg.Description == "" {
		cfg.Description = "桌面端 + Web 端双部署"
	}
	if cfg.Width == 0 {
		cfg.Width = 1280
	}
	if cfg.Height == 0 {
		cfg.Height = 800
	}
	if cfg.MinWidth == 0 {
		cfg.MinWidth = 960
	}
	if cfg.MinHeight == 0 {
		cfg.MinHeight = 600
	}
	if cfg.BackgroundColour == [3]uint8{} {
		cfg.BackgroundColour = [3]uint8{27, 38, 54}
	}
	_ = cfg.BackgroundColour // 上面比较后再赋,这里仅消除 unused 警告

	windowMgr := NewWindowManager()
	appSvc := services.NewAppService(backend)
	windowSvc := services.NewWindowService(windowMgr) // windowMgr 满足 services.WindowManager 接口

	app := application.New(application.Options{
		Name:        cfg.Name,
		Description: cfg.Description,
		Services: []application.Service{
			application.NewService(appSvc),
			application.NewService(windowSvc),
			application.NewService(services.NewPlatformService(nil)),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 主窗口:加载后端 URL(注意 Webview 走 HTTP,不走 Wails AssetServer)
	primary := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: cfg.Name,
		Width: cfg.Width, Height: cfg.Height,
		MinWidth: cfg.MinWidth, MinHeight: cfg.MinHeight,
		BackgroundColour: application.NewRGB(
			cfg.BackgroundColour[0],
			cfg.BackgroundColour[1],
			cfg.BackgroundColour[2],
		),
		URL: backend.URL() + "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})
	windowMgr.RegisterPrimary(primary)

	// 菜单 + 托盘
	// quitApp 直接退出 Wails,后端 server 在 main 协程的 Serve 中由 OS 强制关闭。
	// 这是 Wails 桌面端最常见的退出方式 —— OS 杀进程比 graceful Shutdown 简单且足够。
	showPrimary := func() { windowMgr.ShowPrimary() }
	quitApp := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = ctx
		app.Quit()
	}
	app.Menu.SetApplicationMenu(NewAppMenu(app, showPrimary, quitApp))
	_ = NewTrayManager(app, showPrimary, quitApp)

	return &App{app: app, backend: backend}
}

// Run 阻塞运行 Wails 应用,直到 app.Quit / 关闭窗口被触发。
// 返回值为 Wails 内部退出码。
func (a *App) Run() error {
	if a == nil || a.app == nil {
		return nil
	}
	return a.app.Run()
}

// AppFSEmbed 把 embed.FS 适配成 server.New 需要的 io/fs.FS。
// 这里主要是为了调用方少写一行 import,实际就是 fs.FS 类型别名。
type AppFSEmbed = fs.FS
