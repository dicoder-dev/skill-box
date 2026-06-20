// Package desktop 提供桌面端特有逻辑：窗口控制、托盘、菜单、本地 server 启停等。
// 严格不包含任何业务，业务统一走 ginp-api/pkg/server 提供的 HTTP 接口。
package desktop

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"ginp-api/pkg/server"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LocalServer 在桌面进程内启动一个 api-server,供 Webview 通过
// http://127.0.0.1:<port> 调用业务。零外部依赖、退出干净。
type LocalServer struct {
	srv    *http.Server
	port   atomic.Int32
	closed atomic.Bool
}

// StartLocalServer 监听 127.0.0.1:0(自动分配端口),返回 *LocalServer。
// 调用方在退出时调用 Stop() 优雅关闭。
//
// opts.StaticFS / opts.FrontRootFS 必须由调用方提供(本进程 embed 的 frontend/dist)。
func StartLocalServer(opts server.Options) (*LocalServer, error) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:0"
	}
	ls := &LocalServer{}
	// 先建一个临时 listener 拿端口,再让 server.New 绑定这个 listener,
	// 这样不用改 server.New 的签名就能拿到真实端口。
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return nil, err
	}
	realPort := ln.Addr().(*net.TCPAddr).Port
	ls.port.Store(int32(realPort))

	srv := server.New(opts)
	srv.Addr = ln.Addr().String() // 保持 server.Addr 与实际监听一致
	ls.srv = srv

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("desktop: local server stopped: %v", err)
		}
	}()
	return ls, nil
}

// Port 返回本地 server 的实际监听端口,前端用其拼出 http://127.0.0.1:<port>。
func (l *LocalServer) Port() int {
	return int(l.port.Load())
}

// Stop 优雅关闭本地 server,留给 main.go 在 app.Run 退出后调用。
func (l *LocalServer) Stop(ctx context.Context) {
	if l == nil || l.closed.Swap(true) {
		return
	}
	if ctx == nil {
		ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	}
	_ = l.srv.Shutdown(ctx)
}

// AttachToApp 把 server 关闭逻辑挂到 app 的 OnShutdown 钩子。
func (l *LocalServer) AttachToApp(app *application.App) {
	if app == nil {
		return
	}
	// application v3 alpha 暂未提供统一 shutdown hook,改为在 main.go 显式 Stop。
	// 保留此方法便于后续 v3 GA 或自定义事件时使用。
	_ = app
}
