package server

import (
	"io/fs"
	"net/http"
	"path/filepath"

	"ginp-api/configs"
	"ginp-api/internal/gapi/router"

	"github.com/gin-gonic/gin"
)

// Options 描述如何装配一个 HTTP 服务。
//
// 桌面端与 Web 端共用同一套业务路由（由 routers_import.go 通过 blank import
// 收集），仅在监听地址、静态资源来源与模板加载上有所差异。
type Options struct {
	// Addr 监听地址，例如 ":8080" 或 "127.0.0.1:0"。
	// 留空时使用 configs.ServerPort()，并附加 ":" 前缀。
	Addr string

	// ViewGlob 是模板加载 glob（gin.LoadHTMLGlob），桌面端一般留空。
	ViewGlob string

	// StaticDir 指向磁盘上的静态资源目录；不为空时同时挂载 /static 与 /static/assets。
	StaticDir string

	// StaticFS 用 embed.FS 替代磁盘目录；不为空时优先于 StaticDir。
	// 仍以 /static、/assets 暴露，保留兼容旧版 View 模板。
	StaticFS fs.FS

	// StaticSubdir embed.FS 中静态资源的子目录前缀，默认 "dist"。
	StaticSubdir string

	// FrontRootFS 单页前端根目录，挂在 "/" 并接管 NoRoute fallback 到 index.html。
	// Web 端与桌面端都用同一个 embed.FS，确保两种部署形态前端产物一致。
	// 不为空时优先于 StaticFS 提供根路径访问；StaticFS 仍负责 /static、/assets。
	FrontRootFS fs.FS

	// SPAFallback 启用 NoRoute 时回落 index.html。默认 true（前端为 SPA）。
	SPAFallback *bool
}

// New 构造一个装配好业务路由的 *http.Server，调用方负责 ListenAndServe / Shutdown。
func New(opts Options) *http.Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	mountStatic(r, opts)
	mountFrontRoot(r, opts)
	if opts.ViewGlob != "" {
		r.LoadHTMLGlob(opts.ViewGlob)
	}

	// 业务路由（CORS、登录鉴权等都由 router.Register 内部装配）
	router.Register(r)

	addr := opts.Addr
	if addr == "" {
		addr = ":" + configs.ServerPort()
	}
	return &http.Server{Addr: addr, Handler: r}
}

// mountStatic 兼容旧版 /static、/assets 路径，便于现有 View 模板继续工作。
func mountStatic(r *gin.Engine, opts Options) {
	if opts.StaticFS != nil {
		subdir := opts.StaticSubdir
		if subdir == "" {
			subdir = "dist"
		}
		if sub, err := fs.Sub(opts.StaticFS, subdir); err == nil {
			r.StaticFS("/static", http.FS(sub))
			r.StaticFS("/assets", http.FS(sub))
		}
		return
	}
	if opts.StaticDir == "" {
		return
	}
	r.Static("/static", opts.StaticDir)
	r.Static("/assets", filepath.Join(opts.StaticDir, "assets"))
}

// mountFrontRoot 把前端 SPA 挂在 "/" 并接管 NoRoute 回落到 index.html。
// 这样桌面端 Webview 加载 http://127.0.0.1:<port>/ 与 Web 端走同一份 dist。
func mountFrontRoot(r *gin.Engine, opts Options) {
	if opts.FrontRootFS == nil {
		return
	}
	r.StaticFS("/", http.FS(opts.FrontRootFS))
	if !spaFallback(opts.SPAFallback) {
		return
	}
	// SPA 路由回落到 index.html，避免刷新深层路径 404
	indexBytes, err := fs.ReadFile(opts.FrontRootFS, "index.html")
	if err != nil || len(indexBytes) == 0 {
		return
	}
	r.NoRoute(func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(200)
		_, _ = c.Writer.Write(indexBytes)
	})
}

func spaFallback(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}
