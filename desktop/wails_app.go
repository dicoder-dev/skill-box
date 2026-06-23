package desktop

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"ginp-api/cmd/bootstrap"
	"skill-box/desktop/services"
	"skill-box/pkg/fsutil"

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
	// FrontendURL 可选:自定义前端入口 URL。非空时 Webview 加载此 URL,
	// 而不是 backend.URL()。
	//
	// 典型用途:`wails3 dev` 时由 wails3 CLI 注入 WAILS_VITE_PORT,搭配
	// `wails3 task common:dev:frontend` 起 Vite dev server,让桌面端 Webview
	// 加载 http://localhost:<port>/,享受 Vite HMR 热更新,改前端代码无需重启 Go 进程。
	// 不传则走 backend 内置 gin + embed.FS 的生产路径。
	FrontendURL string
}

// App 包装 *application.App,提供 Quit 优雅退出。
//
// 注意:本结构体只关心 Wails UI 相关的状态,后端 server 由调用方在 NewApp 之前
// 通过 bootstrap.Boot 启动并 Serve 阻塞。后端生命周期跟 App 解耦 ——
// App 只在退出时通知 Wails,后端 server 由 main 的 Serve 阻塞在另一个 goroutine。
type App struct {
	app       *application.App
	backend   *bootstrap.Backend
	notifier  *Notifier
	shortcut  *ShortcutManager
}

// NewApp 构造并完整组装桌面端 Wails 应用:
//   - 注册 Wails Bind 服务(AppService / WindowService / PlatformService /
//     NotifyService / ShortcutService / PrefsService)
//   - 创建主窗口,加载 backend.URL
//   - 装载应用菜单和系统托盘
//   - OnShutdown 时解绑全局快捷键
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
	_ = cfg.BackgroundColour

	windowMgr := NewWindowManager()
	appSvc := services.NewAppService(backend)
	windowSvc := services.NewWindowService(windowMgr) // windowMgr 满足 services.WindowManager 接口

	// 桌面端偏好 settings(由 bootstrap.Backend.NewSettings 工厂方法构造)
	prefsStore := settingsAdapter{backend: backend}

	app := application.New(application.Options{
		Name:        cfg.Name,
		Description: cfg.Description,
		Services: []application.Service{
			application.NewService(appSvc),
			application.NewService(windowSvc),
			application.NewService(services.NewPlatformService(nil)),
			application.NewService(services.NewPrefsService(prefsStore)),
		},
		// 关窗≠退出:macOS 关掉所有窗口后进程继续在托盘跑;
		// Windows 走 DisableQuitOnLastWindowClosed 同样语义。
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		OnShutdown: func() {
			// 进程退出时解绑全局快捷键(Carbon 端绑的事件回调会自然失效,
			// 这里主要打日志便于排查)。
			log.Printf("desktop: shutdown")
		},
	})

	// 通知 + 快捷键:在 NewApp 阶段就构造好,Startup 钩子里调系统 API。
	notifier := NewNotifier(app)
	shortcut := NewShortcutManager()
	notifySvc := services.NewNotifyService(notifier)
	shortcutSvc := services.NewShortcutService(shortcut)
	// 把 NotifyService / ShortcutService 也挂进 Services(独立 New,instance 不同)
	app.RegisterService(application.NewService(notifySvc))
	app.RegisterService(application.NewService(shortcutSvc))

	// 主窗口:加载前端 URL。
	//   - cfg.FrontendURL 非空:桌面端 Webview 直接加载此 URL(典型场景 = wails3 dev,
	//     URL 指向 Vite dev server,享受浏览器层 HMR,改前端代码无需重启 Go 进程)。
	//   - cfg.FrontendURL 为空:走生产路径,加载 backend 自带 gin + embed.FS,
	//     由桌面端 in-process 后端直接出 dist 静态资源。
	frontendURL := cfg.FrontendURL
	if frontendURL == "" {
		frontendURL = backend.URL() + "/"
	} else {
		log.Printf("desktop: Webview using custom frontend URL %q (dev/HMR mode)", frontendURL)
	}
	primary := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: cfg.Name,
		Width: cfg.Width, Height: cfg.Height,
		MinWidth: cfg.MinWidth, MinHeight: cfg.MinHeight,
		BackgroundColour: application.NewRGB(
			cfg.BackgroundColour[0],
			cfg.BackgroundColour[1],
			cfg.BackgroundColour[2],
		),
		URL: frontendURL,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})
	windowMgr.RegisterPrimary(primary)

	// 菜单 + 托盘
	showPrimary := func() { windowMgr.ShowPrimary() }
	quitApp := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = ctx
		app.Quit()
	}
	// 偏好设置:跳到前端 /settings/desktop 路由(SettingsView 末尾的桌面端 section)
	openSettings := func() {
		windowMgr.ShowPrimary()
		if w := windowMgr.Primary(); w != nil {
			w.SetURL(backend.URL() + "/settings/desktop")
		}
	}
	app.Menu.SetApplicationMenu(NewAppMenu(app, showPrimary, quitApp))
	_ = NewTrayManager(app, TrayCallbacks{
		OnShow:         showPrimary,
		OnQuit:         quitApp,
		OnOpenSettings: openSettings,
	}, notifier)

	// 注入桌面端 OS 能力钩子,让 cdesktop 各个 HTTP 端点能调到真能力。
	// 注入时机:Serve 之前;bootstrap.Serve 启动 gin server 时会再
	// 通过 hooks.Set 同步到 cdesktop controller。
	//
	// 缺失字段(如 clipboard / openExternal)在桌面端都已有对应实现,
	// 全部填齐,Web 部署下 backend 不会被注入,钩子保持 nil 自然降级到 501。
	if backend != nil {
		hooks := bootstrap.BootstrapHooks{
			Notify:                     notifier.Notify,
			NotifyHasPermission:        notifier.HasPermission,
			NotifyRequestAuthorization: notifier.RequestAuthorization,
			ClipboardText: func() (string, error) {
				if app.Clipboard == nil {
					return "", fmt.Errorf("clipboard not available")
				}
				text, ok := app.Clipboard.Text()
				if !ok {
					return "", fmt.Errorf("clipboard read failed")
				}
				return text, nil
			},
			SetClipboardText: func(text string) error {
				if app.Clipboard == nil {
					return fmt.Errorf("clipboard not available")
				}
				if !app.Clipboard.SetText(text) {
					return fmt.Errorf("clipboard write failed")
				}
				return nil
			},
			OpenExternal: func(url string) error {
				if app.Browser == nil {
					return fmt.Errorf("browser not available")
				}
				return app.Browser.OpenURL(url)
			},
			// 本地文件能力(fsutil)与桌面 UI 解耦,直接复用 api-server 内的实现,
			// 桌面端和 Web 端读文件/reveal 行为完全一致。
			FsReadText: fsutil.ReadText,
			FsReveal:   fsutil.Reveal,
			WindowShow:              showPrimary,
			WindowToggleAlwaysOnTop: windowMgr.ToggleAlwaysOnTop,
			WindowToggleMaximise:    windowMgr.ToggleMaximise,
			ShortcutRegister: func(combo string) error {
				return shortcut.Register(combo, func() {
					if w := windowMgr.Primary(); w != nil {
						w.Show()
						w.Focus()
					}
				})
			},
			ShortcutUnregister: shortcut.Unregister,
			ShortcutList:       shortcut.List,
			AppQuit:            quitApp,
		}
		backend.SetDesktopHooks(hooks)
		log.Printf("desktop: SetDesktopHooks installed (Notify=%v, ClipboardText=%v, OpenExternal=%v)",
			hooks.Notify != nil, hooks.ClipboardText != nil, hooks.OpenExternal != nil)
	} else {
		log.Printf("desktop: backend is nil, skipping SetDesktopHooks (all cdesktop endpoints will 501)")
	}

	return &App{
		app:      app,
		backend:  backend,
		notifier: notifier,
		shortcut: shortcut,
	}
}

// Run 阻塞运行 Wails 应用,直到 app.Quit / 关闭窗口被触发。
// 返回值为 Wails 内部退出码。
func (a *App) Run() error {
	if a == nil || a.app == nil {
		return nil
	}
	// Startup 钩子里:通知授权 + 启用全局快捷键 + 应用 start_minimized。
	// wails v3 alpha.60 没有 OnStartup 字段,改成在 Run() 之前开 goroutine
	// 异步跑(等 Wails 主循环 ready 后再调系统 API;最差情况是头几次点通知没反应)。
	a.startupAsync()
	return a.app.Run()
}

// startupAsync 在 Run() 阻塞前异步跑启动期副作用。
// 用 goroutine + 小 sleep 错开 Wails 主循环初始化,避免和 macOS app delegate 抢线程。
func (a *App) startupAsync() {
	go func() {
		time.Sleep(500 * time.Millisecond)

		// 1) 读偏好
		var (
			notifyEnabled  = true
			shortcutEnabled = true
			startMinimized = false
		)
		if a.backend != nil {
			prefs := a.backend.NewSettings()
			if prefs != nil {
				if v, ok, _ := prefs.Get(PrefKeyNotifyEnabled); ok && v == "false" {
					notifyEnabled = false
				}
				if v, ok, _ := prefs.Get(PrefKeyShortcutEnabled); ok && v == "false" {
					shortcutEnabled = false
				}
				if v, ok, _ := prefs.Get(PrefKeyStartMinimized); ok && v == "true" {
					startMinimized = true
				}
			}
		}
		a.notifier.SetEnabled(notifyEnabled)
		a.shortcut.SetEnabled(shortcutEnabled)

		// 2) 通知授权
		if notifyEnabled && !a.notifier.HasPermission() {
			if ok, err := a.notifier.RequestAuthorization(); err != nil {
				log.Printf("desktop: notifier RequestAuth error: %v", err)
			} else if ok {
				log.Printf("desktop: notification authorized")
			} else {
				log.Printf("desktop: notification denied by user")
			}
		}

		// 3) 注册全局快捷键
		if shortcutEnabled {
			// 默认 combo = "Cmd+Shift+S";从 prefs 读用户改写值。
			combo := "Cmd+Shift+S"
			if a.backend != nil {
				prefs := a.backend.NewSettings()
				if prefs != nil {
					if v, ok, _ := prefs.Get(PrefKeyGlobalHotKey); ok && v != "" {
						combo = v
					}
				}
			}
			if err := a.shortcut.Register(combo, func() {
				if w := a.app.Window.Current(); w != nil {
					w.Show()
					w.Focus()
				}
			}); err != nil {
				log.Printf("desktop: shortcut register failed: %v (降级到菜单 accelerator)", err)
			} else {
				log.Printf("desktop: global shortcut registered: %s", combo)
			}
		}

		// 4) 启动最小化:隐藏主窗口,只露托盘
		if startMinimized {
			if w := a.app.Window.Current(); w != nil {
				w.Hide()
			}
		}
	}()
}

// settingsAdapter 把 *bootstrap.Backend 的 settings 工厂方法适配成
// services.PrefsStore 接口,避免 services 直接依赖 settings 包。
type settingsAdapter struct {
	backend *bootstrap.Backend
}

func (s settingsAdapter) Get(key string) (string, bool, error) {
	if s.backend == nil {
		return "", false, nil
	}
	st := s.backend.NewSettings()
	if st == nil {
		return "", false, nil
	}
	return st.Get(key)
}

func (s settingsAdapter) Set(key, value string) error {
	if s.backend == nil {
		return nil
	}
	st := s.backend.NewSettings()
	if st == nil {
		return nil
	}
	return st.Set(key, value)
}

func (s settingsAdapter) GetAll() (map[string]string, error) {
	if s.backend == nil {
		return map[string]string{}, nil
	}
	st := s.backend.NewSettings()
	if st == nil {
		return map[string]string{}, nil
	}
	snap, err := st.GetAll()
	if err != nil {
		return nil, err
	}
	return snap.Items, nil
}

// AppFSEmbed 把 embed.FS 适配成 server.New 需要的 io/fs.FS。
// 这里主要是为了调用方少写一行 import,实际就是 fs.FS 类型别名。
type AppFSEmbed = fs.FS

// NewFrontendURLFromEnv 根据 wails3 CLI 注入的环境变量构造 dev 模式下的前端 URL。
//
// wails3 dev 在启动子进程前会注入 WAILS_VITE_PORT(端口号,默认 9245)与
// WAILS_VITE_HOST(可选,默认 127.0.0.1);其他进程下这两个变量未设置时返回 ""。
// 调用方拿到非空结果时,把它赋给 AppConfig.FrontendURL,NewApp 就会让 Webview
// 加载 Vite dev server,从而享受 Vite HMR 热更新。
//
// host / port 也可以通过参数显式覆盖(便于单元测试或自定义场景)。
func NewFrontendURLFromEnv(host string, port int) string {
	if port <= 0 {
		if p := os.Getenv("WAILS_VITE_PORT"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				port = v
			}
		}
	}
	if port <= 0 {
		// 未检测到 Vite 端口,说明不在 wails3 dev 下,返回空 → NewApp 走生产路径。
		return ""
	}
	if host == "" {
		if h := os.Getenv("WAILS_VITE_HOST"); h != "" {
			host = h
		} else {
			host = "127.0.0.1"
		}
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}
