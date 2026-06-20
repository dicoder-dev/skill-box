package bootstrap

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"ginp-api/configs"
	"ginp-api/internal/gapi/router"

	"github.com/gin-gonic/gin"
)

// ServerOptions 描述如何装配一个 HTTP 服务。
//
// 桌面端与 Web 端共用同一套业务路由（由 routers_import.go 通过 blank import
// 收集），仅在监听地址、静态资源来源与模板加载上有所差异。
type ServerOptions struct {
	// Addr 监听地址，例如 ":8080" 或 "127.0.0.1:0"。
	// 留空时使用 configs.ServerPort()，并附加 ":" 前缀。
	Addr string

	// ViewGlob 是模板加载 glob（gin.LoadHTMLGlob），桌面端一般留空。
	ViewGlob string

	// StaticDir 指向磁盘上的静态资源目录；不为空时同时挂载 /static 与 /static/assets。
	StaticDir string

	// StaticFS 用 embed.FS 替代磁盘目录；不为空时优先于 StaticDir。
	// 仍以 /static、/assets 暴露，保留兼容旧版 View 模板。
	// 双部署形态下推荐只传 FrontRootFS(自动覆盖 /static、/assets、/ 三个根),不再单独使用 StaticFS。
	StaticFS fs.FS

	// FrontRootFS 单页前端根目录，挂在 "/" 并接管 NoRoute fallback 到 index.html。
	// Web 端与桌面端都用同一个 embed.FS，确保两种部署形态前端产物一致。
	FrontRootFS fs.FS

	// SPAFallback 启用 NoRoute 时回落 index.html。默认 true（前端为 SPA）。
	SPAFallback *bool
}

// New 构造一个装配好业务路由的 *http.Server，调用方负责 ListenAndServe / Shutdown。
func New(opts ServerOptions) *http.Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 1) 旧式 /static、/assets 兼容(磁盘或 embed.FS)
	mountStatic(r, opts)

	// 2) View 模板
	if opts.ViewGlob != "" {
		r.LoadHTMLGlob(opts.ViewGlob)
	}

	// 3) 业务路由
	router.Register(r)

	// 4) 前端 SPA 入口与 fallback(放在最后,确保不抢业务路由)
	mountFrontRoot(r, opts)

	addr := opts.Addr
	if addr == "" {
		addr = ":" + configs.ServerPort()
	}
	return &http.Server{Addr: addr, Handler: r}
}

// mountStatic 把 FrontRootFS 或 StaticFS 挂到 /static、/assets。
//
// gin 的 r.StaticFS 内部用 http.StripPrefix + http.FileServer 二次包装,
// 会导致 Sub 过的 fs 出现路径错位。这里改用自定义 handler 避免重复 strip。
func mountStatic(r *gin.Engine, opts ServerOptions) {
	fs := opts.StaticFS
	if fs == nil {
		fs = opts.FrontRootFS
	}
	if fs == nil {
		return
	}
	if sub, err := subFS(fs, "assets"); err == nil {
		mountFileServer(r, "/assets", sub)
	}
	mountFileServer(r, "/static", fs)
	if opts.StaticDir == "" {
		return
	}
	r.Static("/static", opts.StaticDir)
	r.Static("/assets", filepath.Join(opts.StaticDir, "assets"))
}

// mountFileServer 自定义挂载:
//
//	gin 路由 /assets/*filepath 把 filepath 段提取为 c.Param("filepath")
//	(注意:filepath 可能带或不带 leading slash,取决于请求路径)。
//	FileServer(http.FS(root)) 期望 URL.Path 是相对 fs 根的路径(以 "/" 开头)。
func mountFileServer(r *gin.Engine, prefix string, root fs.FS) {
	fileServer := http.FileServer(http.FS(root))
	r.GET(prefix+"/*filepath", func(c *gin.Context) {
		file := strings.TrimPrefix(c.Param("filepath"), "/")
		if file == "" {
			file = "index.html"
		}
		r2 := c.Request.Clone(c.Request.Context())
		r2.URL.Path = "/" + file
		fileServer.ServeHTTP(c.Writer, r2)
	})
	r.HEAD(prefix+"/*filepath", func(c *gin.Context) {
		file := strings.TrimPrefix(c.Param("filepath"), "/")
		if file == "" {
			file = "index.html"
		}
		r2 := c.Request.Clone(c.Request.Context())
		r2.URL.Path = "/" + file
		fileServer.ServeHTTP(c.Writer, r2)
	})
}

// mountFrontRoot 把前端 SPA 入口挂在 "/" 并把 NoRoute 接管为 SPA fallback。
//
// 重要：必须放在 router.Register(r) 之后注册,否则会与业务路由冲突。
// 这里不用 r.StaticFS("/") 那个 catch-all 形式,因为它会和 /static、/api 等具体路由冲突。
// 拆成 r.GET("/", ...) + r.NoRoute(...) 后,/api/xxx 永远命中业务路由,/assets/xxx 走 StaticFS。
func mountFrontRoot(r *gin.Engine, opts ServerOptions) {
	if opts.FrontRootFS == nil {
		return
	}
	fileServer := http.FileServer(http.FS(opts.FrontRootFS))
	r.GET("/", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		if spaFallback(opts.SPAFallback) {
			serveSpaFallback(c, opts.FrontRootFS, fileServer)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// serveSpaFallback 提供静态文件,文件不存在时回落到 index.html,实现 SPA 路由能力。
func serveSpaFallback(c *gin.Context, root fs.FS, fileServer http.Handler) {
	upath := c.Request.URL.Path
	if upath == "" || upath == "/" {
		fileServer.ServeHTTP(c.Writer, c.Request)
		return
	}
	clean := strings.TrimPrefix(upath, "/")
	if clean == "" {
		fileServer.ServeHTTP(c.Writer, c.Request)
		return
	}
	if f, err := root.Open(clean); err == nil {
		_ = f.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
		return
	}
	indexBytes, err := fs.ReadFile(root, "index.html")
	if err != nil || len(indexBytes) == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(indexBytes)
}

func spaFallback(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// subFS 包装 io/fs.Sub,便于错误处理集中。
func subFS(root fs.FS, dir string) (fs.FS, error) {
	return fs.Sub(root, dir)
}
