package bootstrap

import (
	"bytes"
	"encoding/json"
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
		addr = ":" + configs.Server.Port
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
//
// index.html 会在写回响应体之前被注入一段 <script>window.__APP_RUNTIME__={...}</script>,
// 把当前运行模式 / 是否启用鉴权 / 应用名告知前端。注入失败时静默放行,前端兜底走默认值。
func mountFrontRoot(r *gin.Engine, opts ServerOptions) {
	if opts.FrontRootFS == nil {
		return
	}
	runtimeScript := buildRuntimeScript()
	fileServer := http.FileServer(http.FS(opts.FrontRootFS))
	r.GET("/", func(c *gin.Context) {
		serveIndexWithRuntime(c, opts.FrontRootFS, runtimeScript)
	})
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		if spaFallback(opts.SPAFallback) {
			serveSpaFallback(c, opts.FrontRootFS, fileServer, runtimeScript)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// serveIndexWithRuntime 返回被注入运行时配置的 index.html。
// 路径 "/" 直接命中首页,文件存在则读出来注入 script,否则走 fileServer 兜底。
func serveIndexWithRuntime(c *gin.Context, root fs.FS, runtimeScript []byte) {
	indexBytes, err := fs.ReadFile(root, "index.html")
	if err != nil || len(indexBytes) == 0 {
		// 没找到就让 fileServer 自己处理(404 / 别的 fallback)
		http.FileServer(http.FS(root)).ServeHTTP(c.Writer, c.Request)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(injectRuntimeScript(indexBytes, runtimeScript))
}

// serveSpaFallback 提供静态文件,文件不存在时回落到 index.html,实现 SPA 路由能力。
// fallback 到 index.html 时同样会注入运行时配置 script。
func serveSpaFallback(c *gin.Context, root fs.FS, fileServer http.Handler, runtimeScript []byte) {
	upath := c.Request.URL.Path
	if upath == "" || upath == "/" {
		serveIndexWithRuntime(c, root, runtimeScript)
		return
	}
	clean := strings.TrimPrefix(upath, "/")
	if clean == "" {
		serveIndexWithRuntime(c, root, runtimeScript)
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
	_, _ = c.Writer.Write(injectRuntimeScript(indexBytes, runtimeScript))
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

// runtimePayload 描述下发给前端的运行时配置。
type runtimePayload struct {
	RunMode  string `json:"runMode"`
	NeedAuth bool   `json:"needAuth"`
	AppName  string `json:"appName"`
}

// buildRuntimeScript 从当前 system 配置生成要注入的 <script>...</script> 片段。
//
// 只读一次 configs.System,在 mountFrontRoot 启动时调用,后续请求复用。
// 字段缺失时给出安全默认值(runMode=web、needAuth=true)。
func buildRuntimeScript() []byte {
	cfg := configs.System
	if cfg == nil {
		// bootstrap 早期出错时可能为 nil,这里兜底
		cfg = &configs.SystemConfig{RunMode: "web", NeedAuth: true, AppName: "skill-box"}
	}
	payload := runtimePayload{
		RunMode:  defaultStr(cfg.RunMode, "web"),
		NeedAuth: cfg.NeedAuth,
		AppName:  defaultStr(cfg.AppName, "skill-box"),
	}
	js, err := json.Marshal(payload)
	if err != nil {
		// 序列化失败兜底
		js = []byte(`{"runMode":"web","needAuth":true,"appName":"skill-box"}`)
	}
	return []byte(`<script>window.__APP_RUNTIME__=` + string(js) + `;</script>`)
}

// injectRuntimeScript 把 runtime script 注入 index.html 的 </head> 之前。
//
// 找不到 </head> 时退回到 body 起始位置之前;两者都找不到时直接拼接在头部。
// 注入只发生在内存中,FS 不被修改。
func injectRuntimeScript(html, script []byte) []byte {
	if len(script) == 0 {
		return html
	}
	// 优先 </head>
	if idx := bytes.Index(html, []byte("</head>")); idx >= 0 {
		out := make([]byte, 0, len(html)+len(script))
		out = append(out, html[:idx]...)
		out = append(out, script...)
		out = append(out, html[idx:]...)
		return out
	}
	// 退而求其次 <body>
	if idx := bytes.Index(html, []byte("<body")); idx >= 0 {
		out := make([]byte, 0, len(html)+len(script))
		out = append(out, html[:idx]...)
		out = append(out, script...)
		out = append(out, html[idx:]...)
		return out
	}
	// 都找不到就拼在头部
	out := make([]byte, 0, len(script)+len(html))
	out = append(out, script...)
	out = append(out, html...)
	return out
}

func defaultStr(s, dv string) string {
	if s == "" {
		return dv
	}
	return s
}
