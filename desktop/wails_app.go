package desktop

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"ginp-api/cmd/bootstrap"
	"skill-box/desktop/services"
	"skill-box/pkg/fsutil"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 主窗口自适应屏幕比例的默认值。
// 启动后 Window.GetScreen() 返回 *Screen,里面 Size/Bounds 都是 DIP 宽度,
// alpha.60 的 Window option.Width/Height 也是 DIP,无需再手动除 ScaleFactor。
//
// 选 90% × 90% 的理由：用户反馈之前的 80% × 16:10 不够"大",这次宽高都按 90%。
// 注意 100% 会把任务栏覆盖(且 SetSize 计算会偏 1px);0.9 在主流屏上视觉接近最大化
// 但仍保留边距。
const (
	// defaultPrimaryWidthRatio 主窗口初始宽度占屏幕 DIP 宽度的比例(90%)。
	defaultPrimaryWidthRatio = 0.75
	// defaultPrimaryHeightRatio 主窗口初始高度占屏幕 DIP 高度的比例(90%)。
	defaultPrimaryHeightRatio = 0.8
	// minPrimaryWidthRatio 主窗口最小宽度占屏幕 DIP 宽度的下限比例(60%)。
	minPrimaryWidthRatio = 0.6
	// fallbackPrimaryWidth / Height 当屏幕尺寸获取失败时,降级到固定的初始值。
	fallbackPrimaryWidth  = 1280
	fallbackPrimaryHeight = 800
	// minPrimarySizeFloor 最小的兜底 MinWidth/MinHeight,避免小屏幕比例算出 0/太小。
	minPrimarySizeFloorWidth  = 960
	minPrimarySizeFloorHeight = 600
)

// ParseAspectRatio 解析 "W:H" 形式的宽高比字符串(2026-07-02 增),返回 (w, h)。
// 不合法返 (0, 0)。允许写法:"16:9"、"4:3"、"21:9"、" 16 : 9 "。
//
// 用途:主窗口初始尺寸按比例算宽高(widthRatio × screenW, heightRatio × screenH)
// 时,不能保证落在某个固定宽高比上(4K 屏按 0.9 算出来是 16.5:9,可能不是用户期望)。
// 给调用方一个"锁比例"选项:窗口宽按 widthRatio 算,高按"宽 × H/W"反推,
// 这样无论屏幕怎么变,窗口始终是 16:9(配置后)。同时 MinSize 也按同一比例推。
func ParseAspectRatio(s string) (int, int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return 0, 0
	}
	return w, h
}

// screenResolutionRE 匹配 system_profiler SPDisplaysDataType 输出里的
// "Resolution: 1920 x 1080 (...)" 一行,捕获宽高两个数字。
// 例:"Resolution: 1920 x 1080 (1080p FHD)" → matches=["1920", "1080"]
var screenResolutionRE = regexp.MustCompile(`Resolution:\s+(\d+)\s+x\s+(\d+)`)

// detectScreenDIPSize 通过 macOS 原生的 system_profiler 拿主屏物理像素分辨率,
// 作为 wails v3 alpha.60 Window.GetScreen() 在启动时序拿不到值时的兜底来源。
//
// 为什么不用 wails 自己的 GetScreen:
//   - alpha.60 的 ScreenManager.primaryScreen 由 native 回调填充,
//     在 application.New 阶段是空的。
//   - 在 startupAsync 协程里 sleep 后调 GetScreen(),实测在用户机器上得到的尺寸不可靠
//     (用户报告:窗口看起来远小于预期,SetSize 没生效)。
//   - macOS 的 system_profiler 是同步输出,可以放心在 NewApp 阻塞阶段调一次,
//     并把结果直接灌进 WebviewWindowOptions.Width/Height,完全绕开 SetSize 路径。
//
// 输出文本格式约定:多屏时 system_profiler 会列多个 Resolution,只取第一个
// (主屏在最前)。非 darwin 平台或 system_profiler 失败时返回 (0, 0)。
//
// Retina 缩放说明:这条 Resolution 行给出的是"UI Looks like"等同的 DIP 值
// (例如 MBP 14" 内屏原生 3024×1965,缩放后是 1512×982),
// 跟 wails WebviewWindowOptions.Width 期望的 DIP 单位一致,直接用即可。
func detectScreenDIPSize() (int, int) {
	if runtime.GOOS != "darwin" {
		return 0, 0
	}
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		log.Printf("desktop: system_profiler failed: %v", err)
		return 0, 0
	}
	matches := screenResolutionRE.FindStringSubmatch(string(out))
	if len(matches) < 3 {
		log.Printf("desktop: screen resolution not found in system_profiler output")
		return 0, 0
	}
	w, errW := strconv.Atoi(strings.TrimSpace(matches[1]))
	h, errH := strconv.Atoi(strings.TrimSpace(matches[2]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		log.Printf("desktop: invalid screen resolution W=%q H=%q", matches[1], matches[2])
		return 0, 0
	}
	return w, h
}

// applyWindowSizeConfig 按 cfg.Size 显式配置计算 WindowSize。
//
// 计算出的最终值会写入 cfg.Width/cfg.Height/cfg.MinWidth/cfg.MinHeight,
// 后续 NewApp 沿用 cfg.Width/Height 灌给 WebviewWindowOptions。
//
// 两种模式:
//   - "fixed":沿用 Size.Width/Size.Height,屏幕尺寸不参与计算。
//   - "ratio" 或空(老默认值):沿用 Size.WidthRatio/Size.HeightRatio,
//     留 0 时用 const 默认 0.9 × 0.9。可选 AspectRatio 锁宽高比。
//   - 其他值:log warning,降级到 ratio 模式 + const 默认。
//
// detectSw/detectSh 是 system_profiler 拿到的屏幕 DIP 分辨率,0 表示拿不到,
// 此时 ratio 模式降级到 const 默认值。
func applyWindowSizeConfig(cfg *AppConfig, detectSw, detectSh int) {
	// 把可能写在 cfg.AspectRatio 的也镜像到 Size.AspectRatio,便于单一来源读取
	if cfg.Size.AspectRatio == "" && cfg.AspectRatio != "" {
		cfg.Size.AspectRatio = cfg.AspectRatio
	}

	switch cfg.Size.Mode {
	case "", "ratio":
		// 比例模式
		wr := cfg.Size.WidthRatio
		if wr <= 0 {
			wr = defaultPrimaryWidthRatio
		}
		hr := cfg.Size.HeightRatio
		if hr <= 0 {
			hr = defaultPrimaryHeightRatio
		}
		if detectSw > 0 {
			cfg.Width = int(math.Round(float64(detectSw) * wr))
		} else {
			cfg.Width = fallbackPrimaryWidth
		}
		aspectW, aspectH := ParseAspectRatio(cfg.Size.AspectRatio)
		if aspectW > 0 && aspectH > 0 && cfg.Width > 0 {
			derived := int(math.Round(float64(cfg.Width) * float64(aspectH) / float64(aspectW)))
			if derived > 0 {
				cfg.Height = derived
			} else {
				cfg.Height = fallbackPrimaryHeight
			}
		} else if detectSh > 0 {
			cfg.Height = int(math.Round(float64(detectSh) * hr))
		} else {
			cfg.Height = fallbackPrimaryHeight
		}

	case "fixed":
		// 固定尺寸模式:沿用 Size.Width/Size.Height,缺失降级到 fallback
		if cfg.Size.Width <= 0 {
			cfg.Width = fallbackPrimaryWidth
		} else {
			cfg.Width = cfg.Size.Width
		}
		if cfg.Size.Height <= 0 {
			cfg.Height = fallbackPrimaryHeight
		} else {
			cfg.Height = cfg.Size.Height
		}

	default:
		log.Printf("desktop: unknown Size.Mode=%q, falling back to ratio mode", cfg.Size.Mode)
		cfg.Width = fallbackPrimaryWidth
		cfg.Height = fallbackPrimaryHeight
	}

	// MinWidth/MinHeight 共用:留 0 走 const 兜底
	if cfg.Size.MinWidth > 0 {
		cfg.MinWidth = cfg.Size.MinWidth
	} else if detectSw > 0 {
		cfg.MinWidth = int(math.Round(float64(detectSw) * minPrimaryWidthRatio))
		if cfg.MinWidth < minPrimarySizeFloorWidth {
			cfg.MinWidth = minPrimarySizeFloorWidth
		}
	} else {
		cfg.MinWidth = minPrimarySizeFloorWidth
	}

	if cfg.Size.MinHeight > 0 {
		cfg.MinHeight = cfg.Size.MinHeight
	} else if detectSh > 0 {
		derivedFromMinW := int(math.Round(float64(cfg.MinWidth) * defaultPrimaryHeightRatio))
		if derivedFromMinW > 0 {
			cfg.MinHeight = derivedFromMinW
		} else {
			cfg.MinHeight = int(math.Round(float64(detectSh) * minPrimaryWidthRatio))
		}
		if cfg.MinHeight < minPrimarySizeFloorHeight {
			cfg.MinHeight = minPrimarySizeFloorHeight
		}
	} else {
		cfg.MinHeight = minPrimarySizeFloorHeight
	}
}

// applyLegacySizeDefaults 老路径(顶层 Width/Height + AutoSizeByScreen)兜底,
// 当 cfg.Size 未显式配置时调用,行为与改前完全一致。
func applyLegacySizeDefaults(cfg *AppConfig, detectSw, detectSh int) {
	aspectW, aspectH := ParseAspectRatio(cfg.AspectRatio)
	if cfg.Width == 0 {
		if detectSw > 0 {
			cfg.Width = int(math.Round(float64(detectSw) * defaultPrimaryWidthRatio))
		} else {
			cfg.Width = fallbackPrimaryWidth
		}
	}
	if cfg.Height == 0 {
		if aspectW > 0 && aspectH > 0 && cfg.Width > 0 {
			derived := int(math.Round(float64(cfg.Width) * float64(aspectH) / float64(aspectW)))
			if derived > 0 {
				cfg.Height = derived
			} else {
				cfg.Height = fallbackPrimaryHeight
			}
		} else if detectSh > 0 {
			cfg.Height = int(math.Round(float64(detectSh) * defaultPrimaryHeightRatio))
		} else {
			cfg.Height = fallbackPrimaryHeight
		}
	}
	if cfg.MinWidth == 0 {
		if detectSw > 0 {
			cfg.MinWidth = int(math.Round(float64(detectSw) * minPrimaryWidthRatio))
			if cfg.MinWidth < minPrimarySizeFloorWidth {
				cfg.MinWidth = minPrimarySizeFloorWidth
			}
		} else {
			cfg.MinWidth = minPrimarySizeFloorWidth
		}
	}
	if cfg.MinHeight == 0 {
		if aspectW > 0 && aspectH > 0 && cfg.MinWidth > 0 {
			derived := int(math.Round(float64(cfg.MinWidth) * float64(aspectH) / float64(aspectW)))
			if derived >= minPrimarySizeFloorHeight {
				cfg.MinHeight = derived
			} else {
				cfg.MinHeight = minPrimarySizeFloorHeight
			}
		} else if detectSh > 0 {
			cfg.MinHeight = int(math.Round(float64(detectSh) * minPrimaryWidthRatio))
			if cfg.MinHeight < minPrimarySizeFloorHeight {
				cfg.MinHeight = minPrimarySizeFloorHeight
			}
		} else {
			cfg.MinHeight = minPrimarySizeFloorHeight
		}
	}
}

// AppConfig 描述桌面端 Wails 应用的全部配置。
// 调用方在 main.go 构造并传给 NewApp,NewApp 内部完成 Wails 全部组装并返回 *App。
type AppConfig struct {
	// Name 应用名,显示在菜单栏 / 标题栏。
	Name string
	// Description 应用描述,部分系统会用到。
	Description string
	// Width / Height 主窗口初始尺寸。
	// 留空时优先按 AutoSizeByScreen 走"屏幕宽度 90% + 高度 90%",失败再降级到 1280×800。
	// 显式给定时,AutoSizeByScreen 自动关掉,本次启动固定用这个尺寸。
	Width, Height int
	// MinWidth / MinHeight 主窗口最小尺寸。
	// AutoSizeByScreen=true 时,默认按"屏幕宽高 60%"算(且不低于 960×600 兜底)。
	MinWidth, MinHeight int
	// AutoSizeByScreen 是否把主窗口初始尺寸按当前屏幕 DIP 宽高各 90% 自动算。
	// 默认 true(Widht 与 Height 都是 0 时),main.go 显式设置 Width/Height 时会被自动改成 false。
	// 关掉后保持调用方传的固定尺寸,常用于打包时写死统一窗口规格。
	AutoSizeByScreen bool
	// AspectRatio 锁宽高比(2026-07-02 增),格式 "W:H",如 "16:9"、"4:3"、"21:9"。
	//   - 非空:窗口宽按 widthRatio × screenW 算,高按"宽 × H/W"反推;MinSize 同理。
	//   - 空:保持原行为,宽高各自按 widthRatio / heightRatio 独立算(可能不是整数比)。
	// 不传或格式非法都被视为空;调用方应在 NewApp 之前用 ParseAspectRatio 预校验。
	// 注意:仅在 AutoSizeByScreen=true 时生效;显式指定 Width/Height 时忽略。
	AspectRatio string
	// Size 主窗口尺寸配置(2026-07-02 增,显式两模式选择)。
	// **新代码推荐使用本字段,语义比顶层 Width/Height + AutoSizeByScreen 更清晰。**
	// 通过 WindowSizeConfig.configured() 判断是否被显式配置:
	//   - 已配置:走 Size.Mode 决定 ratio / fixed;
	//   - 未配置:回落到顶层 Width/Height + AutoSizeByScreen 行为(向后兼容)。
	// 详见 WindowSizeConfig 注释。
	Size WindowSizeConfig
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

// WindowSizeConfig 主窗口尺寸配置(2026-07-02 增,显式模式选择)。
//
// 旧接口(顶层 Width/Height/MinWidth/MinHeight/AutoSizeByScreen)继续保留,行为不变;
// **新代码推荐使用本结构,语义更清晰**。
//
// 两种模式:
//   - "ratio"(默认):窗口 = 屏幕宽 × WidthRatio、屏幕高 × HeightRatio;
//     Width/Height 被忽略。适用桌面端日常使用,跟屏幕尺寸走。
//   - "fixed":窗口 = 固定 Width × Height,不随屏幕变化;
//     WidthRatio/HeightRatio 被忽略。适用打包统一规格。
//
// 字段语义:
//   - Mode: 选哪种算法。"ratio" 或 "fixed",空字符串等同 "ratio" 兼容旧行为。
//   - Width/Height:    Mode=="fixed" 时使用;Mode=="ratio" 时无效。
//   - WidthRatio/HeightRatio: Mode=="ratio" 时使用;Mode=="fixed" 时无效。
//     留空(0)时 NewApp 内部用 const 默认值兜底(0.9 × 0.9)。
//   - MinWidth/MinHeight: 共用,留 0 时按 minPrimaryWidthRatio(0.6)+Floor 兜底。
//   - AspectRatio: 可选,锁宽高比("16:9" 等),仅 Mode=="ratio" 时生效;
//     非空时 Height 按"Width × H/W"反推。
//
// 使用示例(main.go):
//
//	desktop.NewApp(desktop.AppConfig{
//	    Size: desktop.WindowSizeConfig{
//	        Mode:        "ratio",
//	        WidthRatio:  0.9,
//	        HeightRatio: 0.9,
//	    },
//	}, backend)
//
//	// 或固定尺寸:
//	desktop.NewApp(desktop.AppConfig{
//	    Size: desktop.WindowSizeConfig{
//	        Mode:   "fixed",
//	        Width:  1280,
//	        Height: 800,
//	    },
//	}, backend)
type WindowSizeConfig struct {
	// Mode 选哪种算法。可选 "ratio" 或 "fixed"。空字符串等同 "ratio"。
	Mode string
	// Width / Height Mode=="fixed" 时使用,Mode=="ratio" 时无效。
	Width, Height int
	// WidthRatio / HeightRatio Mode=="ratio" 时使用(0~1);Mode=="fixed" 时无效。
	// 留空(0)NewApp 内部用 const 默认值兜底,推荐显式给值以免新人不清楚默认。
	WidthRatio, HeightRatio float64
	// MinWidth / MinHeight 留 0 时 NewApp 内部按 minPrimaryWidthRatio + Floor 兜底。
	MinWidth, MinHeight int
	// AspectRatio 可选,锁宽高比("16:9" 等),仅 Mode=="ratio" 时生效。
	AspectRatio string
}

// configured 标记 WindowSizeConfig 是否被显式配过(非零值)。
// 区别于空值与默认值,避免顶层字段完全没填时误判为 "fixed"。
// 当前实现:Mode 非空、Width/Height 任一非零、WidthRatio/HeightRatio 任一非零,
//
//	或 MinWidth/MinHeight 任一非零、AspectRatio 非空 → 都算被显式配置过。
func (s WindowSizeConfig) configured() bool {
	return s.Mode != "" ||
		s.Width > 0 || s.Height > 0 ||
		s.WidthRatio > 0 || s.HeightRatio > 0 ||
		s.MinWidth > 0 || s.MinHeight > 0 ||
		s.AspectRatio != ""
}

// App 包装 *application.App,提供 Quit 优雅退出。
//
// 注意:本结构体只关心 Wails UI 相关的状态,后端 server 由调用方在 NewApp 之前
// 通过 bootstrap.Boot 启动并 Serve 阻塞。后端生命周期跟 App 解耦 ——
// App 只在退出时通知 Wails,后端 server 由 main 的 Serve 阻塞在另一个 goroutine。
type App struct {
	app        *application.App
	backend    *bootstrap.Backend
	notifier   *Notifier
	shortcut   *ShortcutManager
	autoResize bool // startupAsync 里按屏幕 DIP 宽高各 90% 重置主窗口尺寸
	aspectW    int  // 2026-07-02 增:锁宽高比(W),autoResize=true 时生效
	aspectH    int  // 2026-07-02 增:锁宽高比(H),autoResize=true 时生效
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
	// 默认尺寸：在 NewApp 阻塞阶段就能拿屏幕尺寸的兜底来源，优先用 macOS 原生
	// system_profiler 拿主屏分辨率（同步可执行，不依赖 Wails 主循环 ready），
	// 然后按选定模式灌给 cfg.Width/Height,让窗口天生就是大尺寸。
	//
	// 模式分发(2026-07-02 增):
	//   - cfg.Size 被显式配过（Mode/W/H/Ratio 任一非零）→ 走 WindowSizeConfig 路径
	//   - 否则 → 走顶层 Width/Height + AutoSizeByScreen 老路径(向后兼容)
	//
	// 注意：调用方在 main.go 显式给 cfg.Width / cfg.Height 时，下面这段兜底会跳过，
	// 那时 startupAsync 阶段的 resizePrimaryToScreenRatio 也被 autoResize=false 关掉。
	detectSw, detectSh := detectScreenDIPSize()
	if cfg.Size.configured() {
		applyWindowSizeConfig(&cfg, detectSw, detectSh)
	} else {
		// 老路径(顶层 Width/Height + AutoSizeByScreen)
		applyLegacySizeDefaults(&cfg, detectSw, detectSh)
	}
	log.Printf("desktop: primary window initial size = %dx%d (mode=%s, detected screen %dx%d DIP from system_profiler), min = %dx%d",
		cfg.Width, cfg.Height, cfg.Size.Mode, detectSw, detectSh, cfg.MinWidth, cfg.MinHeight)
	if cfg.BackgroundColour == [3]uint8{} {
		cfg.BackgroundColour = [3]uint8{27, 38, 54}
	}
	_ = cfg.BackgroundColour

	// AutoSizeByScreen 旧字段同步路由(顶层级)——固定尺寸模式下保持 false,
	// ratio 模式下保持 true 给 startupAsync 异步校准兜底。
	if cfg.Width != 0 || cfg.Height != 0 {
		cfg.AutoSizeByScreen = false
	}

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
			// FsPickFolder 弹系统文件夹选择对话框,走 wails v3 的 OpenFileDialog +
			// CanChooseDirectories(true)。从 wails dialog 派生的结果是一个
			// 字符串,取消选择时为空串,与 Web 端降级协议一致。
			FsPickFolder: func() (string, error) {
				if app == nil {
					return "", fmt.Errorf("wails app not initialized")
				}
				return app.Dialog.OpenFile().
					CanChooseDirectories(true).
					CanChooseFiles(false).
					CanCreateDirectories(true).
					PromptForSingleSelection()
			},
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

	// 2026-07-02 增:透传宽高比,让 startupAsync 协程按 aspect 反推窗口高。
	// ParseAspectRatio 已经过滤了非法值,(0,0) 时 resizePrimaryToScreenRatio 走
	// 原独立 heightRatio 路径,行为完全向后兼容。
	aw, ah := ParseAspectRatio(cfg.AspectRatio)
	return &App{
		app:        app,
		backend:    backend,
		notifier:   notifier,
		shortcut:   shortcut,
		autoResize: cfg.AutoSizeByScreen,
		aspectW:    aw,
		aspectH:    ah,
	}
}

// Run 阻塞运行 Wails 应用，直到 app.Quit / 关闭窗口被触发。
// 返回值为 Wails 内部退出码。
func (a *App) Run() error {
	if a == nil || a.app == nil {
		return nil
	}
	// Startup 钩子里:通知授权 + 启用全局快捷键 + 应用 start_minimized + 按屏幕比例调整尺寸。
	// wails v3 alpha.60 没有 OnStartup 字段，改成在 Run() 之前开 goroutine
	// 异步跑（等 Wails 主循环 ready 后再调系统 API；最差情况是头几次点通知没反应）。
	// 2026-07-02 增:把锁宽高比的 aspectW/aspectH 一起透传给 startupAsync,
	// 让协程在按屏幕比例调整时按 aspect 反推窗口高(用户配 16:9 时,无论
	// 屏幕实际比例如何,窗口始终是 16:9)。
	a.startupAsync(a.autoResize, a.aspectW, a.aspectH)
	return a.app.Run()
}

// resizePrimaryToScreenRatio 按当前屏幕 DIP 宽高自适应主窗口尺寸。
//
// 调用时机:Wails 主循环 ready 后(startupAsync 协程 sleep 完再调),
// 此时 GetScreen() / SetSize() 才有意义。
//
// 算法:
//   - 屏宽 W = Screen.Size.Width(DIP)
//   - 屏高 H = Screen.Size.Height(DIP)
//   - 窗口宽 = round(W × widthRatio)
//   - 窗口高:aspectW/aspectH > 0 时按"宽 × H/W"反推,否则 round(H × heightRatio)
//   - MinWidth = round(W × minPrimaryWidthRatio),且不低于 minPrimarySizeFloorWidth
//   - MinHeight:aspectW/aspectH > 0 时按"MinWidth × H/W"反推,否则 round(H × minPrimaryWidthRatio),且不低于 minPrimarySizeFloorHeight
//
// 屏幕尺寸获取失败时(多发生在无 GUI 或启动太早)记 warning,不动窗口,
// 由 NewApp 的兜底 Width/Height 顶住。
func (a *App) resizePrimaryToScreenRatio(widthRatio, heightRatio float64, aspectW, aspectH int) {
	if a == nil || a.app == nil {
		return
	}
	w := a.app.Window.Current()
	if w == nil {
		log.Printf("desktop: resizePrimaryToScreenRatio skipped, no primary window")
		return
	}
	screen, err := w.GetScreen()
	if err != nil || screen == nil {
		log.Printf("desktop: GetScreen failed (%v), keep fallback window size", err)
		return
	}
	screenW := screen.Size.Width
	if screenW <= 0 {
		// PhysicalBounds 作为兜底(某些平台 Size 是 0)
		screenW = screen.Bounds.Width
	}
	screenH := screen.Size.Height
	if screenH <= 0 {
		screenH = screen.Bounds.Height
	}
	if screenW <= 0 || screenH <= 0 {
		log.Printf("desktop: screen size unavailable (Size=%dx%d, Bounds=%dx%d), keep fallback window size",
			screen.Size.Width, screen.Size.Height, screen.Bounds.Width, screen.Bounds.Height)
		return
	}

	newW := int(math.Round(float64(screenW) * widthRatio))
	if newW <= 0 {
		return
	}
	// 锁宽高比:高按"宽 × H/W"反推;非法 aspect 走原独立 heightRatio 路径。
	newH := 0
	if aspectW > 0 && aspectH > 0 {
		newH = int(math.Round(float64(newW) * float64(aspectH) / float64(aspectW)))
	}
	if newH <= 0 {
		newH = int(math.Round(float64(screenH) * heightRatio))
	}
	if newH <= 0 {
		return
	}
	minW := int(math.Round(float64(screenW) * minPrimaryWidthRatio))
	if minW < minPrimarySizeFloorWidth {
		minW = minPrimarySizeFloorWidth
	}
	minH := 0
	if aspectW > 0 && aspectH > 0 {
		minH = int(math.Round(float64(minW) * float64(aspectH) / float64(aspectW)))
	}
	if minH <= 0 {
		minH = int(math.Round(float64(screenH) * minPrimaryWidthRatio))
	}
	if minH < minPrimarySizeFloorHeight {
		minH = minPrimarySizeFloorHeight
	}

	w.SetSize(newW, newH)
	w.SetMinSize(minW, minH)
	aspectStr := "free"
	if aspectW > 0 && aspectH > 0 {
		aspectStr = fmt.Sprintf("%d:%d", aspectW, aspectH)
	}
	log.Printf("desktop: primary window resized by screen ratio: size=%dx%d (%d%% × %d%% of %dx%d DIP, aspect=%s), min=%dx%d",
		newW, newH, int(widthRatio*100), int(heightRatio*100), screenW, screenH, aspectStr, minW, minH)
}

// startupAsync 在 Run() 阻塞前异步跑启动期副作用。
// 用 goroutine + 小 sleep 错开 Wails 主循环初始化,避免和 macOS app delegate 抢线程。
//
// autoResize=true 时,等 Wails 主循环 ready 后(500ms 后)按屏幕 DIP 宽高 90% 重置尺寸。
// 注意:resize 不能放在 sleep 前 — alpha.60 的 GetScreen() 走 InvokeSync,
// 在主循环没起来时会卡死 / 拿不到值。
func (a *App) startupAsync(autoResize bool, aspectW, aspectH int) {
	go func() {
		time.Sleep(500 * time.Millisecond)

		// 0) 主窗口按屏幕比例调整（最优先,要在 start_minimized 之前做完）
		if autoResize {
			a.resizePrimaryToScreenRatio(defaultPrimaryWidthRatio, defaultPrimaryHeightRatio, aspectW, aspectH)
		}

		// 1) 读偏好
		var (
			notifyEnabled   = true
			shortcutEnabled = true
			startMinimized  = false
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
