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
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/pkg/cfg"
)

// DefaultConfigFile 是 cli 入口未显式指定配置时的默认路径。
const DefaultConfigFile = "configs.yaml"

// ConfigFile 全局配置路径,由 Run / Boot 在最早期注入。
var ConfigFile = DefaultConfigFile

// ServerOptionsBuilder 由调用方传入,负责把外部资源(embed.FS 等)拼成 ServerOptions。
// 设计为函数式是为了规避循环引用:本包被 cmd/web、cmd/gapi、skill-box/main 三处使用,
// 让调用方在自己的 main 闭包里构造 ServerOptions,避免 bootstrap 反向依赖这些包。
type ServerOptionsBuilder func() ServerOptions

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
}

// Backend 是 Boot 阶段返回的"已就绪但 server 还没起"句柄。
// 调用方可以读 Port/URL、决定何时调 Serve 阻塞。
type Backend struct {
	srvOpts ServerOptions // Boot 时已装配好,Serve 直接用
	port    int           // 从 srvOpts.Addr 解析得到
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
	// 不加这段，configs.Db.UseType 永远是 tag default "mysql"，所有路径都会 panic。
	cfg.ParseConfigStruct(configs.Db)
	cfg.ParseConfigStruct(configs.Server)
	cfg.ParseConfigStruct(configs.System)
	cfg.ParseConfigStruct(configs.Email)
	cfg.ParseConfigStruct(configs.Tencent)
	// 关键:dbs 包的 useDbType 需要在 cfg 加载完后同步过来。
	dbs.SetDbType(configs.Db.UseType)

	if !opts.DisableLogger {
		StartGinLogger()
	}
	StartDB()
	if !opts.DisableTask {
		StartTask()
	}
	srvOpts, err := buildServerOptions(opts.ServerOptions)
	if err != nil {
		return nil, err
	}
	return &Backend{
		srvOpts: srvOpts,
		port:    parsePortFromAddr(srvOpts.Addr),
	}, nil
}

// Serve 阻塞跑 gin HTTP server,直到 server 关闭。
// 通常在 main 的最后调用。
func Serve(b *Backend) {
	if b == nil {
		log.Fatal("bootstrap: Serve called with nil Backend")
	}
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
func buildServerOptions(builder ServerOptionsBuilder) (ServerOptions, error) {
	var srvOpts ServerOptions
	if builder != nil {
		srvOpts = builder()
	} else {
		srvOpts = ServerOptions{
			ViewGlob:  "view/*",
			StaticDir: "./static",
		}
	}
	if srvOpts.Addr == "" {
		srvOpts.Addr = "127.0.0.1:" + configs.Server.Port
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

// parsePortFromAddr 从 "127.0.0.1:8082" / ":8082" / "[::]:8082" 提取端口。
func parsePortFromAddr(addr string) int {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0
	}
	p, _ := strconv.Atoi(port)
	return p
}
