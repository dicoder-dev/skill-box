// Package cdesktop 暴露桌面端独有的 HTTP 端点,供 webview 中的前端调用。
//
// 设计动机:
//   Wails v3 alpha.60 不再像 v2 那样把 Go service 注入到 window.go.*;
//   自动生成的 bindings 用 $Call.ByID(methodID, ...) 走 fetch /wails/runtime,
//   而本项目的 webview 由后端 Gin server 提供,/wails/runtime 路由不存在。
//
//   因此桌面端能力(偏好、窗口、通知、剪贴板等)统一走 Gin HTTP 端点,
//   既复用已有的 HTTP 抽象(http.js),也免去在 webview 里塞入 runtime.js 的复杂度。
//
//   所有端点约定 NeedLogin=false / NeedPermission=false:这些是桌面本机能力,
//   不需要走用户鉴权(本机进程内调用);若以后需要区分用户,再补 auth。
package cdesktop

import (
	"runtime"
	"sync"

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// BackendHooks 桌面端 OS 能力钩子集合,由调用方(bootstrap)在初始化阶段注入。
//
// 设计要点:
//   - 每个字段都是可选的(nil 表示该能力在当前部署形态不可用);
//   - cdesktop 端点看到 nil hook 时返回 501 + 明确 error,前端 guard 捕获后
//     提示用户"该能力仅桌面端可用"。
//   - 接口定义留在 cdesktop 包内,避免 bootstrap 依赖 cdesktop 形成循环(bootstrap
//     装配 router → router imports cdesktop → 反向不应 import bootstrap)。
type BackendHooks struct {
	Notify                       func(id, title, body string) error
	NotifyHasPermission          func() bool
	NotifyRequestAuthorization   func() (bool, error)
	ClipboardText                func() (string, error)
	SetClipboardText             func(text string) error
	OpenExternal                 func(url string) error
	WindowShow                   func()
	WindowToggleAlwaysOnTop      func() bool
	WindowToggleMaximise         func()
	ShortcutRegister             func(combo string) error
	ShortcutUnregister           func(combo string) error
	ShortcutList                 func() []string
	AppQuit                      func()
}

// currentHooks 由 SetHooks 注入,所有端点 handler 通过 hooks() 读。
var (
	hooksMu      sync.RWMutex
	currentHooks BackendHooks
)

// SetHooks 由 bootstrap 在 Serve 启动前调用,注入当前进程可用的桌面端回调。
// 重复调用以最后一次为准;传空结构体清空所有 hook(Web 部署关闭时)。
func SetHooks(h BackendHooks) {
	hooksMu.Lock()
	currentHooks = h
	hooksMu.Unlock()
}

// hooks 返回当前注入的 hook 集合(只读快照)。
func hooks() BackendHooks {
	hooksMu.RLock()
	defer hooksMu.RUnlock()
	return currentHooks
}

// ===== /api/desktop/app/* =====

// RequestAppHealth / RequestAppVersion / RequestAppQuit 都是无入参占位。
type RequestAppHealth struct{}
type RequestAppVersion struct{}
type RequestAppQuit struct{}

// RespondAppVersion 应用版本(运行时读 configs.System.AppName 等)。
type RespondAppVersion struct {
	AppName string `json:"app_name"`
	RunMode string `json:"run_mode"`
	Version string `json:"version"`
}

// GetAppVersion GET /api/desktop/app/version
func GetAppVersion(c *ginp.ContextPlus, _ *RequestAppVersion) {
	appName := ""
	runMode := ""
	if configs.System != nil {
		appName = configs.System.AppName
		runMode = configs.System.RunMode
	}
	c.JSON(200, RespondAppVersion{
		AppName: appName,
		RunMode: runMode,
		Version: runtime.Version(),
	})
}

// GetAppHealth GET /api/desktop/app/health
func GetAppHealth(c *ginp.ContextPlus, _ *RequestAppHealth) {
	c.JSON(200, gin.H{"status": "ok", "go_version": runtime.Version()})
}

// PostAppQuit POST /api/desktop/app/quit
// 当前仅返回 200 占位;真正退出由前端通过 menu 触发 Wails app.Quit(),
// 这个端点保留以便未来通过后端命令式退出(避免前端与 wails 主循环耦合)。
func PostAppQuit(c *ginp.ContextPlus, _ *RequestAppQuit) {
	c.JSON(200, gin.H{"ok": true})
}

// ===== /api/desktop/prefs =====
//
// 单条 / 全量走同一 GET 端点,用 query key 区分:
//   GET /api/desktop/prefs         → RespondPrefGetAll
//   GET /api/desktop/prefs?key=xxx → RespondPrefGet
// 这样避免在 gin 上重复注册 GET /api/desktop/prefs 触发 panic。

// RequestPrefSet 写单条偏好。
type RequestPrefSet struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RequestPrefGet 无入参(handler 内自己读 query)。
type RequestPrefGet struct{}

// RespondPrefGet 返回单条偏好。
// 字段定义对齐前端 platform.prefs.get 的 [value, exists] 解构。
type RespondPrefGet struct {
	Value  string `json:"value"`
	Exists bool   `json:"exists"`
}

// RespondPrefGetAll 返回全部偏好快照。
type RespondPrefGetAll struct {
	Items map[string]string `json:"items"`
}

// GetPref GET /api/desktop/prefs(?key=xxx)
// 有 key → 单条;无 key → 全部。避免 gin 重复路由注册。
func GetPref(c *ginp.ContextPlus, _ *RequestPrefGet) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if key := c.Query("key"); key != "" {
		v, ok, err := st.Get(key)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, RespondPrefGet{Value: v, Exists: ok})
		return
	}
	snap, err := st.GetAll()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	items := map[string]string{}
	if snap != nil {
		items = snap.Items
	}
	c.JSON(200, RespondPrefGetAll{Items: items})
}

// PutPref PUT /api/desktop/prefs { key, value }
func PutPref(c *ginp.ContextPlus, req *RequestPrefSet) {
	if req.Key == "" {
		c.JSON(400, gin.H{"error": "missing key"})
		return
	}
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := st.Set(req.Key, req.Value); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	// app
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/app/version", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetAppVersion, &RequestAppVersion{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.app.version", Description: "桌面端应用版本、运行模式", RequestParams: RequestAppVersion{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/app/health", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetAppHealth, &RequestAppHealth{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.app.health", Description: "桌面端后端健康检查", RequestParams: RequestAppHealth{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/app/quit", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostAppQuit, &RequestAppQuit{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.app.quit", Description: "占位:由前端通过菜单/快捷键触发 Wails app.Quit", RequestParams: RequestAppQuit{}},
	})

	// prefs
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/prefs", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetPref, &RequestPrefGet{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.prefs.get", Description: "取单条偏好(?key=xxx)或全部(无 query)", RequestParams: RequestPrefGet{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/prefs", HttpType: ginp.HttpPut,
		Handler: ginp.BindParamsHandler(PutPref, &RequestPrefSet{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.prefs.set", Description: "写一条桌面偏好", RequestParams: RequestPrefSet{}},
	})
}