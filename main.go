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
//
// macOS 26 Tahoe 启动问题诊断辅助:
//   桌面端 dmg 装到 /Applications 后双击闪退(2026-07-16 实测:runningboard 报
//   'termination reported by launchd (0, 0, 256)',LSExitStatus=1),目前根因在
//   maybeBootstrapLaunchAgent 的 plist 路径漂移(commit c52ab1a 已修)。
//   但 launchd 派发链上 panic / os.Exit 仍可能发生,本 main.go 在入口处
//   捕获所有 panic、记录 exit 原因、把日志写到 ~/.skill-box/logs/ 下,方便
//   后续 syspolicyd / amfi / runningboard 看不到的内部错误也能从用户家目录抓出。
package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"ginp-api/cmd/bootstrap"
	sharefunc "ginp-api/share/func"
	"skill-box/desktop"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

// startupLogFile 全局,用于 defer 写"退出原因"。
var startupLogFile *os.File

func main() {
	// 1. 第一件事:开启动期日志文件,所有 stdout/stderr 都写一份到
	//    ~/.skill-box/logs/startup-YYYYMMDD-HHMMSS-<pid>.log。
	//    即使进程在 flag.Parse / 后续任意位置 panic / os.Exit,日志都已经写到磁盘。
	//
	//    关键点:macOS 26 Tahoe 双击 .app 闪退时,Finder / Console / log show 都
	//    看不到 binary 内部输出(launchd 派发链把 binary stdout 关掉了)。写到
	//    家目录独立文件,事后 `ls -lat ~/.skill-box/logs/` 找最新 startup-*.log
	//    即可看完整 trace。
	if err := setupStartupLog(); err != nil {
		// setupStartupLog 失败不能 fatal — 这是辅助日志,失败也得起进程。
		log.Printf("startup: setupStartupLog failed: %v (continue without file log)", err)
	} else {
		// 把启动时间、pid、命令行、launchd 环境全打一份,便于排查双击启动链。
		logStartupContext()
	}

	// 2. panic recover:任何 panic 都会被捕获,写 stack trace 到启动日志文件。
	//    包装成 deferred func,后续 main 流程都能 catch。
	defer func() {
		if r := recover(); r != nil {
			writeStartupLine(fmt.Sprintf("PANIC: %v\n%s", r, debug.Stack()))
			// 重抛一次让 Go runtime 按惯例 exit 2,但 stack trace 已经留底。
			panic(r)
		}
	}()

	// 3. main 退出原因记录:
	//   - app.Run() 返回:正常退出(用户关窗 / Quit),exit code 0
	//   - app.Run() 返回 err:异常退出,记到日志
	//   - main 函数 panic:上面 recover 已捕获,这里再补一行 "exit reason"
	defer func() {
		writeStartupLine(fmt.Sprintf("main: exit at %s", time.Now().Format(time.RFC3339)))
	}()

	// 桌面端优先用项目根的 configs.yaml(便于开发期覆盖配置);
	// 真正的"数据目录"由 bootstrap.applyDataDir 在 RunMode=desktop 时接管。
	//
	// 2026-07-16 实测关键 bug:launchd 派发 dmg .app 时(binary 装在 /Applications),
	// 进程 working directory 被 launchd 设为 /,如果 flag 没显式传 -config,
	// 默认值 "./configs.yaml" 解析为 "/configs.yaml" → cfg.InitCfg 试图
	// 在 / 下创建文件,macOS 根目录 read-only,Boot 失败 exit(1)。
	// 修复:默认值优先用 sharefunc.ConfigPath() = ~/skill-box/configs.yaml,
	// 跟数据目录锚定,不依赖 working directory。
	defaultConfig := sharefunc.ConfigPath()
	if defaultConfig == "" {
		defaultConfig = bootstrap.DefaultConfigFile
	}
	configPath := flag.String("config", defaultConfig, "配置文件路径(yaml)")
	flag.Parse()

	// embed 路径 "frontend/dist" 在 fs 里保留了目录前缀,先 Sub 出 dist 子 FS。
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		writeStartupLine(fmt.Sprintf("ERROR: sub frontend/dist failed: %v", err))
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
		writeStartupLine(fmt.Sprintf("ERROR: bootstrap.Boot failed: %v", err))
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
	// 2026-07-02 增:窗口尺寸配置改用 WindowSizeConfig,显式两模式选择:
	//   - Mode=desktop.WindowSizeModeRatio(默认):窗口按屏幕比例,WidthRatio/HeightRatio 生效(0 用 const 兜底 0.9 × 0.9)。
	//   - Mode=desktop.WindowSizeModeFixed:窗口固定 W×H,不随屏幕变(打包/调试场景)。
	//   - AspectRatio 可选,锁宽高比,如 "16:9"。
	// 上一次"两次不生效"的原因已经定位为 alpha.60 Window.SetSize 不可靠;
	// 现在用 system_profiler 同步拿屏在 NewApp 阶段直接灌大尺寸,完全不依赖 SetSize。
	app := desktop.NewApp(desktop.AppConfig{
		Name: "Skill-Box",
		Size: desktop.WindowSizeConfig{
			Mode:        desktop.WindowSizeModeFixed,
			WidthRatio:  0.9,
			HeightRatio: 0.9,
			AspectRatio: "16:9", // 宽高比锁 16:9,无论屏幕如何窗口都是 16:9
			MinWidth:    1440,
			MinHeight:   820,
		},
		// 老顶层字段(Width/Height/AutoSizeByScreen)继续兼容,但 Size 配置过时被忽略。
		FrontendURL: desktop.NewFrontendURLFromEnv("", 0),
	}, backend)

	// 3) 运行 Wails 主循环(阻塞)
	if err := app.Run(); err != nil {
		writeStartupLine(fmt.Sprintf("ERROR: app.Run() returned err: %v", err))
		log.Printf("app run error: %v", err)
	}
	writeStartupLine("main: app.Run() returned normally (window closed by user or Quit called)")
}

// setupStartupLog 在 main 入口开启动期日志文件,所有写入通过 writeStartupLine
// 同步落到磁盘 + log 包默认 writer(原 stderr 行为不变)。
//
// 文件路径:`~/.skill-box/logs/startup-YYYYMMDD-HHMMSS-<pid>.log`。
// 优先 sharefunc.LogsDir() 获取桌面端数据目录(~/skill-box/),失败兜底
// os.UserHomeDir()/skill-box/logs/。
func setupStartupLog() error {
	logsDir := func_LogsDir()
	if logsDir == "" {
		// 实在拿不到家目录(很罕见,几乎不会发生),跳过文件日志。
		return fmt.Errorf("cannot resolve logs dir")
	}
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", logsDir, err)
	}

	ts := time.Now().Format("20060102-150405")
	fname := filepath.Join(logsDir, fmt.Sprintf("startup-%s-%d.log", ts, os.Getpid()))
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", fname, err)
	}
	startupLogFile = f

	// 把日志同时写 stderr + 文件,这样 Console.app / log show 也能看一份。
	log.SetOutput(io.MultiWriter(os.Stderr, f))

	log.Printf("startup: log file opened at %s", fname)
	return nil
}

// writeStartupLine 同步写一行 + flush 到磁盘,避免 panic / os.Exit 来不及落盘。
func writeStartupLine(line string) {
	if startupLogFile == nil {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	// 写时间戳 + 行 + 换行
	if _, err := fmt.Fprintf(startupLogFile, "[%s] %s\n", ts, line); err == nil {
		_ = startupLogFile.Sync()
	}
}

// logStartupContext 记录启动期关键环境变量,便于排查 macOS 26 Tahoe 双击派发链问题。
// 打印项:
//   - LaunchAgentLabel env(launchd 派发特征)
//   - 当前 pid / ppid / uid
//   - 可执行文件绝对路径
//   - 工作目录
//   - 关键 env 子集
func logStartupContext() {
	exe, _ := os.Executable()
	wd, _ := os.Getwd()
	writeStartupLine(fmt.Sprintf("START: pid=%d ppid=%d uid=%d", os.Getpid(), os.Getppid(), os.Getuid()))
	writeStartupLine(fmt.Sprintf("START: exe=%s", exe))
	writeStartupLine(fmt.Sprintf("START: wd=%s", wd))
	writeStartupLine(fmt.Sprintf("START: LaunchAgentLabel=%q (空=非 launchd 派发,即 Finder 双击 / open / 终端启动)", os.Getenv("LaunchAgentLabel")))
	writeStartupLine(fmt.Sprintf("START: LaunchEvents=%q", os.Getenv("LaunchEvents")))
	// 透传关键 Go runtime 信息,有助于 stack trace 关联 build。
	writeStartupLine(fmt.Sprintf("START: GOOS=%s GOARCH=%s", os.Getenv("GOOS"), os.Getenv("GOARCH")))
	writeStartupLine(fmt.Sprintf("START: argv=%v", os.Args))
}

// func_LogsDir 是 sharefunc.LogsDir 的间接引用,放在这里避免 import cycle。
// sharefunc 是 ginp-api 的子包,本 main.go 在 skill-box 顶层,跟 sharefunc
// 不存在循环 import,但这里用变量保存返回值并兜底空字符串处理,让
// setupStartupLog 的失败路径更稳健。
func func_LogsDir() string {
	// sharefunc.LogsDir() 内部走 dataDirInternal → os.UserHomeDir() → ~/<AppName>/logs,
	// 跟 RunMode 无关,所以可以在 main 入口(bootstrap.Boot 之前)安全调用。
	return sharefunc.LogsDir()
}
