// Package bootstrap 集中所有部署形态(Web / Desktop / 传统 CLI)共享的启动流程:
//
//  1. 加载配置(cfg.InitCfg)
//  2. 把数据库类型同步给 dbs 包
//  3. 启动日志(写到 logs/<date>.log)
//  4. 启动 DB + 自动迁移
//  5. 启动定时任务(goroutine)
//
// 启动 HTTP server 的方式由调用方控制:
//   - Run(opts)         = Boot(opts) + Serve(b) 阻塞到底,适合 CLI / Web
//   - Boot(opts)        = 跑完 1-5,返回 *Backend,server 还没起
//   - Serve(b)          = 阻塞跑 gin HTTP server,适合桌面端(主循环在 Wails)
//
// 之所以放在 cmd/ 而不是 pkg/,是因为:
//   - 这是业务启动编排(cfg/DB/Task/Logger/Server),不是通用工具
//   - skill-box(桌面端)是另一个 Go module,通过 go.work 跨 module import 它
//   - 未来如果有第三方 CLI / 嵌入式场景,也需要复用这套启动流程
package bootstrap

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/internal/settings"
	"ginp-api/internal/skillversion" // 2026-07-17 增:启动兜底初始化 git 仓库
	"ginp-api/pkg/cfg"
	"ginp-api/pkg/launchagent"
	"ginp-api/pkg/logger"
	sharefunc "ginp-api/share/func"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// DefaultConfigFile 是 cli 入口未显式指定配置时的默认路径。
const DefaultConfigFile = "configs.yaml"

// ConfigFile 全局配置路径,由 Run / Boot 在最早期注入。
var ConfigFile = DefaultConfigFile

// ServerOptionsBuilder 由调用方传入,负责把外部资源(embed.FS 等)拼成 ServerOptions。
// 设计为函数式是为了规避循环引用:本包被 cmd/web、cmd/gapi、skill-box/main 三处使用,
// 让调用方在自己的 main 闭包里构造 ServerOptions,避免 bootstrap 反向依赖这些包。
//
// runMode 由 bootstrap 在装配阶段从 BootOptions.RunMode 透传进来(单一权威),
// 避免配置文件 system.run_mode 与启动命令双源歧义。
type ServerOptionsBuilder func(runMode string) ServerOptions

// BootOptions 控制启动行为。
type BootOptions struct {
	// ConfigFile 配置文件路径,留空使用 DefaultConfigFile。
	ConfigFile string

	// ServerOptions 由调用方构造 ServerOptions 的函数。
	// 留空表示走磁盘静态(老 gapi 行为);非空表示由调用方注入 embed.FS / 覆盖端口。
	ServerOptions ServerOptionsBuilder

	// DisableLogger / DisableTask 允许桌面端在测试或嵌入式场景跳过日志文件/定时任务。
	DisableLogger bool
	DisableTask   bool

	// RunMode 由调用方显式声明运行形态("web" / "desktop"),用于决定是否走
	// 用户家目录下的数据目录(~/<AppName>/)。非空时覆盖 configs.System.RunMode
	// 并写回配置文件;为空时以配置文件为准(便于 dev / 测试场景通过 yaml 切换)。
	RunMode string
}

// Backend 是 Boot 阶段返回的"已就绪但 server 还没起"句柄。
// 调用方可以读 Port/URL、决定何时调 Serve 阻塞。
type Backend struct {
	srvOpts ServerOptions // Boot 时已装配好,Serve 直接用
	port    int           // 从 srvOpts.Addr 解析得到
	dbs     *dbsHolder    // DB 句柄,供桌面端 settings 等服务按需构造

	// DesktopHooks 桌面端特有的回调,Web 模式下全为 nil。
	// 由桌面端入口(skill-box/main.go)在 NewApp 阶段注入,cdesktop
	// controller 在 HTTP 请求时通过 hooks.Get() 调到真正的 OS 能力。
	//
	// 类型定义在 hooks 子包里(独立子包规避 bootstrap → cdesktop 的导入环),
	// 此处只是持有 + 转发,Serve 阶段桥接到 hooks.Set。
	desktopHooks hooks.BootstrapHooks
}

// BootstrapHooks 是 hooks.BootstrapHooks 的对外类型别名。
//
// hooks 子包位于 ginp-api/internal/ 路径下,skill-box/desktop 这种外部
// 模块无法直接 import。这里在 bootstrap 包做一层转发,既保留了
// hooks 子包规避导入环的初衷,又让桌面端代码能用
// `backend.SetDesktopHooks(bootstrap.BootstrapHooks{...})` 这种方式注入。
//
// 字段集与 hooks.BootstrapHooks 完全一致(同一个底层类型),赋值与传参
// 都按结构体值语义处理,不会产生额外的运行时开销。
type BootstrapHooks = hooks.BootstrapHooks

// dbsHolder 持有 write/read 两个 *gorm.DB,供桌面端 settings 等服务按需构造。
// 不导出指针,只通过 NewSettings 工厂方法构造,避免外部误持 db 句柄。
type dbsHolder struct {
	write *gorm.DB
	read  *gorm.DB
}

// SetDesktopHooks 注入桌面端回调。多次调用以最后一次为准;nil 全部清空。
//
// 调用时机:Boot 返回 *Backend 之后、Serve 启动 HTTP server 之前的窗口期。
// 桌面端入口(skill-box/main.go)在 desktop.NewApp 阶段调用。
// 这里只持有值,Serve 阶段才会桥接到 hooks.Set(那时 router 已注册)。
func (b *Backend) SetDesktopHooks(h BootstrapHooks) {
	if b == nil {
		return
	}
	b.desktopHooks = h
}

// GetDesktopHooks 返回当前注入的桌面端回调(只读快照,调用方不允许改)。
// 在桌面端启动链路里,Serve 会用它来桥接 hooks.Set;controller 本身不
// 直接走这个 getter,而是通过 hooks.Get() 拿到全局值。
func (b *Backend) GetDesktopHooks() BootstrapHooks {
	if b == nil {
		return BootstrapHooks{}
	}
	return b.desktopHooks
}

// NewSettings 构造一个新的 settings.Service,供桌面端 PrefsService 等场景使用。
// 每次调用都 new 一个 Service 实例,db 句柄是共享的;Service 本身无状态,
// 多实例并发安全(settings 用的是 entity.Setting 表 + 简单 CRUD)。
func (b *Backend) NewSettings() *settings.Service {
	if b == nil || b.dbs == nil {
		return nil
	}
	return settings.New(b.dbs.write, b.dbs.read)
}

// Run 阻塞入口:完成 cfg/DB/Task 初始化,然后跑 server 直到关闭。
// 适合 cmd/gapi、cmd/web 这种"server 就是主程序"的场景。
func Run(opts BootOptions) {
	b, err := Boot(opts)
	if err != nil {
		log.Fatalf("bootstrap: Boot failed: %v", err)
	}
	Serve(b)
}

// Boot 非阻塞:完成 cfg/DB/Task 初始化,装配 ServerOptions,返回 *Backend。
// server 还没启动 —— 调用方拿到 Backend 后可以读 Port/URL,再决定何时 Serve。
// 桌面端用此模式:它在 Boot 后起 Wails,等 Wails 主循环就绪后再 Serve。
func Boot(opts BootOptions) (*Backend, error) {
	if opts.ConfigFile != "" {
		ConfigFile = opts.ConfigFile
	}
	if err := cfg.InitCfg(ConfigFile); err != nil {
		return nil, err
	}
	// 关键:configs 各包的 init() 在程序启动时就跑了 ParseConfigStruct，把 struct 字段填上
	// 了 tag default；那时候 viper 还没加载配置。InitCfg 会把 viper 实例换成新加载的，
	// 但 struct 字段不会自动重读，所以这里必须显式再 parse 一遍把 viper 里的值刷进去。
	// 不加这段，configs.Db.UseType 永远是 tag default "sqlite"，所有路径都会 panic。
	cfg.ParseConfigStruct(configs.Db)
	cfg.ParseConfigStruct(configs.Server)
	cfg.ParseConfigStruct(configs.System)
	cfg.ParseConfigStruct(configs.Email)
	cfg.ParseConfigStruct(configs.Tencent)

	// 桌面端:把 configs/data.db/logs 锚到 ~/.<AppName>/,并把 cfg 重指到该目录下的 configs.yaml。
	// - 首次运行:从原始配置(默认 ./configs.yaml,或 -config 传入)种子到数据目录。
	// - 部署形态由 opts.RunMode 单源决定,不再回写到配置文件。
	applyDataDir(ConfigFile, opts.RunMode)

	// 桌面端 LaunchAgent 自注册(详见 pkg/launchagent 注释):
	// macOS 26 Tahoe 对 ad-hoc 签名的 .app 在 Finder 双击时 amfi Code=-423
	// 静默杀进程,GUI 兜底完全走不到;只有 launchd 直派发(launchedByLS=0)
	// 才能起 binary。所以桌面端首次启动时:
	//   1. 检测不是 launchd 子进程(没 LaunchAgentLabel env)
	//   2. 检测 plist 未安装
	//   3. 写 plist + launchctl bootstrap → launchd fork 一份新实例
	//   4. 当前进程 os.Exit(0) 让位,launchd 那份接管
	//
	// 注意:这一步必须在 StartDB / StartTask / Serve 之前,否则双进程会撞
	// sqlite / 8082 端口。
	if opts.RunMode == "desktop" {
		if err := maybeBootstrapLaunchAgent(); err != nil {
			// 自注册失败不阻断启动,继续用当前进程跑(用户没走 launchd 但
			// 至少能起来;下次再尝试)。日志暴露即可。
			log.Printf("bootstrap: LaunchAgent bootstrap skipped: %v", err)
		}
	}

	// 关键:dbs 包的 useDbType 需要在 cfg 加载完后同步过来。
	dbs.SetDbType(configs.Db.UseType)
	// 关键:sqlite 数据路径在桌面端要重定向到 ~/.<AppName>/data.db,
	// 由 dbs 包内的 IsDesktop() 判断;这里把 RunMode 注入到 dbs。
	dbs.SetRunMode(opts.RunMode)

	if !opts.DisableLogger {
		StartGinLogger()
	}
	StartDB()
	// SeedBundledSkills 必须在 StartTask / Serve 之前:被 seed 的 skill 是
	// 用户在 UI 上能看到 / 能 Apply 的内容,启动期一次性灌入。
	SeedBundledSkills()
	// 2026-07-17 增:skillversion 启动兜底 — 初始化本地 git 仓库 + 首次入库现有 skills。
	// 必须在 SeedBundledSkills 之后(否则 init 时扫描不到 seed 出来的 skill),
	// 在 Serve 之前(否则 HTTP 请求进来 store.Save hook 会先于 init commit)。
	skillversion.BootstrapInit()
	if !opts.DisableTask {
		StartTask()
	}
	srvOpts, err := buildServerOptions(opts.ServerOptions, opts.RunMode)
	if err != nil {
		return nil, err
	}
	return &Backend{
		srvOpts: srvOpts,
		port:    parsePortFromAddr(srvOpts.Addr),
		dbs:     &dbsHolder{write: dbs.GetWriteDb(), read: dbs.GetReadDb()},
	}, nil
}

// Serve 阻塞跑 gin HTTP server,直到 server 关闭。
// 通常在 main 的最后调用。
func Serve(b *Backend) {
	if b == nil {
		log.Fatal("bootstrap: Serve called with nil Backend")
	}
	// 把 backend(实现了 hooks.Provider 接口)桥接到 cdesktop controller,
	// 这样 controller 在 HTTP 请求时能通过 hooks.Get() 实时拿到 backend
	// 最新的 desktopHooks 快照。
	//
	// 关键:这里只 Bind backend 指针,不在此时拷贝 hooks 值。原因是时序:
	//   - go Serve(backend) 在 main 启动后立刻跑,此时 desktop.NewApp
	//     还没执行 SetDesktopHooks,如果在这里 Set 当前值,以后 NewApp
	//     注入的 hooks 就拿不到(cdesktop 永远读到第一次的空值)。
	//   - 改成 Bind backend + Get 时实时读,无论 Serve 早跑还是晚跑,
	//     cdesktop 都能拿到 NewApp 注入后的真能力。
	//
	// Web 部署下 backend 不会被注入,hooks.Get() 始终返回零值,cdesktop
	// 自然降级到 501,前端 guard 捕获后给出友好提示。
	hooks.Bind(b)
	log.Printf("bootstrap: hooks provider bound, current Notify=%v (实时读,后续 SetDesktopHooks 后立即生效)",
		b.GetDesktopHooks().Notify != nil)
	srv := New(b.srvOpts)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// Port 返回后端监听端口。Boot 后即可读,不依赖 server 已 listen。
func (b *Backend) Port() int {
	if b == nil {
		return 0
	}
	return b.port
}

// URL 返回 http://<host>:<port>。host 部分从 Addr 解析得到。
func (b *Backend) URL() string {
	if b == nil {
		return ""
	}
	host := "127.0.0.1"
	if h, _, err := net.SplitHostPort(b.srvOpts.Addr); err == nil && h != "" && h != "::" {
		host = h
	}
	return "http://" + host + ":" + strconv.Itoa(b.port)
}

// buildServerOptions 装配 ServerOptions,Addr 留空时用 configs.ServerPort() 兜底。
// runMode 由调用方在 builder 内传(典型用法:Boot 阶段从 BootOptions.RunMode 拿),
// 不在这里硬编码,避免配置文件与启动命令双源歧义。
func buildServerOptions(builder ServerOptionsBuilder, runMode string) (ServerOptions, error) {
	var srvOpts ServerOptions
	if builder != nil {
		srvOpts = builder(runMode)
	} else {
		srvOpts = ServerOptions{
			ViewGlob:  "view/*",
			StaticDir: "./static",
		}
	}
	if srvOpts.Addr == "" {
		srvOpts.Addr = "127.0.0.1:" + configs.Server.Port
	}
	// builder 内部也可以不读 runMode,这里兜底填一份,保证 index.html 注入
	// 永远拿到值(空时 buildRuntimeScript 内部再兑底 "web")。
	if srvOpts.RunMode == "" {
		srvOpts.RunMode = runMode
	}
	return srvOpts, nil
}

// Shutdown 优雅关闭 server,供桌面端 Quit 时调用。
// 注意:Serve 是 ListenAndServe 阻塞,Shutdown 需要从另一 goroutine 调用,
// 否则 Shutdown 信号永远到不了 server(经典 net/http 死锁)。
func (b *Backend) Shutdown() {
	if b == nil {
		return
	}
	// 这里没有保留 *http.Server 引用(给 Serve 用),
	// 实际关闭走 signal/超时路径,桌面端一般靠 os.Exit(0)。
}

// maybeBootstrapLaunchAgent 桌面端首次启动时,如果不走 launchd 派发(Finder 双击
// 进来的),就把 binary 自己注册到 launchd,让 launchd 接管后续启动。
//
// 设计逻辑(2026-07-15 第三版,实测通过):
//
//   首次启动(无 plist):
//     1. 写 plist + launchctl bootstrap(注册到 launchd 域,不立刻跑)
//     2. 本进程继续跑 Serve,不退出 —— 这样用户双击后立刻能用,不用
//        第二次双击或等 kickstart
//     3. 用户退出程序后,launchctl list 里有记录,下次启动走 launchd 派发
//
//   首次启动(无 plist):
//     1. 写 plist + launchctl bootstrap(注册到 launchd 域,不立刻跑)
//     2. 本进程继续跑 Serve,不退出 —— 这样用户双击后立刻能用,不用
//        第二次双击或等 kickstart
//     3. 用户退出程序后,launchctl list 里有记录,下次启动走 launchd 派发
//
//   后续启动(plist 已装 + plist program 路径 != 当前 binary 路径,典型场景
//   是 dmg 装的 .app 首次双击,plist 还是 dev 期 build 留下的路径):
//     1. Install 重写 plist 指向当前 binary 路径(Install 内部 bootout + 写 + bootstrap)
//     2. 本进程继续 Serve,plist 已重写,下次启动路径一致
//
//   后续启动(plist 已装 + plist program 路径 == 当前 binary 路径):
//     1. 如果 8082 已被 Skill-Box 占用 → launchd 那份在跑,本进程退出避免撞端口
//     2. 如果 8082 没占用 → 本进程直接 Serve(不 kickstart,kickstart -k 5s 超时
//        在 macOS 26 Tahoe 下不稳定,launchd child 启动 + 跑 Serve 通常 > 5s,
//        5s 超时会让本进程退出后 launchd 那份还没起来 → 用户看到二次启动)
//
// 为什么首次启动和后续启动都不 kickstart:实测 2026-07-16,kickstart -k 后
// launchd child 启动到 8082 LISTEN 通常需要 8-12s(Go runtime + sqlite migrate
// + SeedBundledSkills + Gin 监听),我们写死的 5s 超时不够。改成"本进程 Serve
// + plist 仅作 launchctl asuser 直派发的标记",用户视觉上是"双击即开,无延迟"。
func maybeBootstrapLaunchAgent() error {
	if launchagent.IsLaunchdChild() {
		log.Printf("bootstrap: launchd child (LaunchAgentLabel=%s), skip bootstrap",
			os.Getenv("LaunchAgentLabel"))
		return nil
	}

	installed, err := launchagent.IsInstalled()
	if err != nil {
		return fmt.Errorf("check install: %w", err)
	}

	if installed {
		// plist 已装。第一件事:看 plist 里的 program 路径跟当前 binary 一不一致。
		// macOS 26 Tahoe 下 dmg 装的 .app 首次双击,如果 plist 还是 dev 期写的
		// 路径(/Volumes/MyDrive/.../bin/Skill-Box.dev.app/...),launchd 派发
		// 链拉的还是 dev 期 binary,那个 binary 也走 maybeBootstrap 又去
		// kickstart,5s 超时退出死循环。修复:用当前 binary 路径重写 plist。
		currentExe, _ := os.Executable()
		installedPath, _ := launchagent.InstalledBinaryPath()
		if currentExe != "" && installedPath != "" && currentExe != installedPath {
			log.Printf("bootstrap: plist program=%s, current=%s, rewriting plist",
				installedPath, currentExe)
			if err := launchagent.Install(currentExe, launchLogPath()); err != nil {
				log.Printf("bootstrap: rewrite plist failed: %v (will continue with current process)", err)
				return nil
			}
			log.Printf("bootstrap: plist rewritten to %s, continuing to serve", currentExe)
			return nil
		}

		// plist 已装 + program 路径对得上。如果 8082 已被 Skill-Box 占用,
		// 说明 launchd 那份已经在跑,当前是 Finder 双击起来的(被 amfi -423
		// 杀掉之前的 fork),本进程退出避免撞端口。
		if isPortOccupied(8082, "Skill-Box") {
			log.Printf("bootstrap: LaunchAgent installed and 8082 already LISTEN, exiting")
			os.Exit(0)
		}
		// plist 已装 + 路径对 + 8082 没占用:launchd 没拉 launchd 那份
		// (KeepAlive=false RunAtLoad=false,plist 仅作注册标记)。本进程
		// 直接 Serve 即可,plist 留着供下次 launchctl asuser 直派发时用。
		log.Printf("bootstrap: LaunchAgent installed and 8082 free, serving in current process")
		return nil
	}

	// plist 未装 → 首次启动
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	log.Printf("bootstrap: bootstrapping LaunchAgent (binary=%s, log=%s)", exe, launchLogPath())
	if err := launchagent.Install(exe, launchLogPath()); err != nil {
		return err
	}
	log.Printf("bootstrap: LaunchAgent bootstrapped successfully, continuing to serve")
	// 关键:不 os.Exit(0),让本进程继续 Serve。
	// 后续启动时,Finder 双击会被 amfi 杀,但 launchd 那份会接管。
	// 用户退出本进程后,plist 已注册,下次启动 kickstart 拉一份。
	return nil
}

// launchLogPath 返回 ~/.skill-box/logs/launchagent.log 的绝对路径,
// bootstrap 模块需要传这个路径给 launchagent.Install。
//
// 抽出来单独一个函数避免在 maybeBootstrapLaunchAgent 内部重复构造路径。
// 与 sharefunc.LogsDir() 同源,只是 sharefunc 在 dbs 等包依赖重,
// 这里手动拼保持 launchagent.Install 的参数语义清晰。
func launchLogPath() string {
	return filepath.Join(sharefunc.LogsDir(), "launchagent.log")
}

// isPortOccupied 检测指定端口是否被指定进程名占用。
//
// 用 lsof -nP -iTCP:<port> -sTCP:LISTEN 看 LISTEN 的进程,如果 command 包含
// processName 视为占用。processName 空字符串表示只要 LISTEN 就算占用。
func isPortOccupied(port int, processName string) bool {
	out, err := exec.Command("lsof", "-nP",
		fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN").CombinedOutput()
	if err != nil {
		return false // lsof exit 非 0 通常表示无匹配
	}
	if processName == "" {
		return len(out) > 0
	}
	return strings.Contains(string(out), processName)
}

// parsePortFromAddr 从 "127.0.0.1:8082" / ":8082" / "[::]:8082" 提取端口。
func parsePortFromAddr(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	p, _ := strconv.Atoi(port)
	return p
}

// applyDataDir 桌面端装配数据目录:~/.<AppName>/,把所有程序文件锚到该目录。
// Web 端(非 desktop)不做事,保留 ./logs、./data.db 行为。
//
// originalConfigPath:Boot 阶段 cfg.InitCfg 用过的路径(默认 ./configs.yaml,
//
//	或 -config 显式传入)。applyDataDir 用它做"首次种子"。
//
// overrideRunMode  :调用方显式声明的运行形态("desktop"),完全由启动命令
//	传入,不再回写到配置文件(避免双源歧义)。
func applyDataDir(originalConfigPath, overrideRunMode string) {
	runMode := overrideRunMode
	if runMode == "" {
		runMode = "web"
	}
	if runMode != "desktop" {
		return
	}
	dataDir := sharefunc.DataDir()
	logsDir := sharefunc.LogsDir()
	cfgPath := sharefunc.ConfigPath()
	if dataDir == "" {
		log.Printf("bootstrap: cannot resolve user home, skip data dir setup")
		return
	}
	for _, p := range []string{dataDir, logsDir} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			log.Printf("bootstrap: mkdir %s failed: %v", p, err)
			return
		}
	}

	// 确保数据目录里有"可用的" configs.yaml。
	// - 文件不存在 → 从 originalConfigPath / ./configs.yaml / ../configs.yaml 找一份有效的种子;
	//   全失败就写硬编码默认。
	// - 文件存在但 isConfigEffective()==false(cfg.InitCfg 刚创出来的空文件 / struct
	//   defaults 写出的 use_type: mysql 但 mysql.db=="" 等)→ 也走种子逻辑覆盖掉。
	// - 文件存在且有效 → 跳过,尊重用户已编辑的版本。
	if abs, _ := filepath.Abs(cfgPath); abs != originalConfigPath {
		needWrite := false
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			needWrite = true
		} else if data, err := os.ReadFile(cfgPath); err != nil || !isConfigEffective(data) {
			needWrite = true
		}
		if needWrite {
			seeded := false
			for _, src := range []string{originalConfigPath, "configs.yaml", "../configs.yaml"} {
				if data, err := os.ReadFile(src); err == nil && isConfigEffective(data) {
					if werr := os.WriteFile(cfgPath, data, 0o644); werr == nil {
						log.Printf("bootstrap: seeded %s from %s", cfgPath, src)
						seeded = true
					} else {
						log.Printf("bootstrap: write %s failed: %v", cfgPath, werr)
					}
					break
				}
			}
			if !seeded {
				// 不再硬编码默认 YAML:cfg.InitCfg 会创建空文件,随后
				// cfg.ParseConfigStruct 把 struct tag 上的 default 灌进 viper 并
				// 通过 cfg.Set -> viper.WriteConfig 写回磁盘,所有默认值自动落盘。
				log.Printf("bootstrap: no effective seed source; %s will be filled by struct tag defaults", cfgPath)
			}
		}
	}

	// 把 cfg 重指到数据目录下的 configs.yaml,并刷新各 struct(可能 seed 过也可能是新文件)。
	ConfigFile = cfgPath
	if err := cfg.InitCfg(cfgPath); err != nil {
		log.Printf("bootstrap: reload cfg from %s failed: %v", cfgPath, err)
	}
	cfg.ParseConfigStruct(configs.Db)
	cfg.ParseConfigStruct(configs.Server)
	cfg.ParseConfigStruct(configs.System)
	cfg.ParseConfigStruct(configs.Email)
	cfg.ParseConfigStruct(configs.Tencent)

	// 注意:此处不再回写 system.run_mode 到数据目录的 configs.yaml。
	// 部署形态由启动命令单一权威(BootOptions.RunMode),配置文件里
	// 也不再保留 run_mode 字段,避免双源歧义。

	// 锚定日志目录;StartGinLogger / middleware 会自动跟随。
	if logsDir != "" {
		logger.SetLogPath(logsDir)
	}
}

// isConfigEffective 判断一段 yaml 字节流是否包含"可用的" db 配置。
//
// "可用" = use_type 是 mysql/sqlite/pgsql 之一,且该类型下的关键字段都已设置。
// 用来过滤 cfg.InitCfg 把空文件当成配置(viper 退回 struct defaults)的情况。
//
// 返回 false 时,applyDataDir 会用一份"更合适"的种子覆盖目标文件,而不是
// 把空文件 / mysql-without-db 这种必崩配置传播到数据目录。
func isConfigEffective(data []byte) bool {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return false
	}
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
		return false
	}
	switch v.GetString("db.use_type") {
	case "mysql":
		return v.GetString("db.mysql.ip") != "" &&
			v.GetString("db.mysql.port") != "" &&
			v.GetString("db.mysql.user") != "" &&
			v.GetString("db.mysql.db") != "" &&
			v.GetString("db.mysql.pwd") != ""
	case "sqlite":
		return v.GetString("db.sqlite.db_path") != ""
	case "pgsql", "postgresql":
		return v.GetString("db.pgsql.ip") != "" &&
			v.GetString("db.pgsql.port") != "" &&
			v.GetString("db.pgsql.user") != "" &&
			v.GetString("db.pgsql.db") != "" &&
			v.GetString("db.pgsql.pwd") != ""
	default:
		return false
	}
}
