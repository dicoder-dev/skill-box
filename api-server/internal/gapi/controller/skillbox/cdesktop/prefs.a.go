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

	"ginp-api/configs"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
)

// ===== /api/desktop/app/* =====

// RequestAppHealth / RequestAppVersion / RequestAppQuit 都是无入参。
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
	c.JSON(200, ginp.H{"status": "ok", "go_version": runtime.Version()})
}

// PostAppQuit POST /api/desktop/app/quit
// 当前仅返回 200 占位;真正退出由前端通过 menu 触发 Wails app.Quit(),
// 这个端点保留以便未来通过后端命令式退出(避免前端与 wails 主循环耦合)。
func PostAppQuit(c *ginp.ContextPlus, _ *RequestAppQuit) {
	c.JSON(200, ginp.H{"ok": true})
}

// ===== /api/desktop/prefs =====

// RequestPrefGet 单 key 查询。
// 走 query string:GET /api/desktop/prefs?key=xxx
// 这里 BindParams 不读 query,而是 handler 内自己读 c.Query("key")。
// 但 BindParams 仍需要个空 struct 占位,才能注册 handler。
type RequestPrefGet struct{}

// RequestPrefSet 写单条偏好。
type RequestPrefSet struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RequestPrefGetAll 无入参。
type RequestPrefGetAll struct{}

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

// GetPref GET /api/desktop/prefs?key=xxx
func GetPref(c *ginp.ContextPlus, _ *RequestPrefGet) {
	key := c.Query("key")
	if key == "" {
		c.JSON(400, ginp.H{"error": "missing key"})
		return
	}
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	v, ok, err := st.Get(key)
	if err != nil {
		c.JSON(500, ginp.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondPrefGet{Value: v, Exists: ok})
}

// PutPref PUT /api/desktop/prefs { key, value }
func PutPref(c *ginp.ContextPlus, req *RequestPrefSet) {
	if req.Key == "" {
		c.JSON(400, ginp.H{"error": "missing key"})
		return
	}
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := st.Set(req.Key, req.Value); err != nil {
		c.JSON(500, ginp.H{"error": err.Error()})
		return
	}
	c.JSON(200, ginp.H{"ok": true})
}

// GetPrefAll GET /api/desktop/prefs (无 query)
func GetPrefAll(c *ginp.ContextPlus, _ *RequestPrefGetAll) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	snap, err := st.GetAll()
	if err != nil {
		c.JSON(500, ginp.H{"error": err.Error()})
		return
	}
	items := map[string]string{}
	if snap != nil {
		items = snap.Items
	}
	c.JSON(200, RespondPrefGetAll{Items: items})
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
		Handler: ginp.BindParamsHandler(GetPrefAll, &RequestPrefGetAll{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.prefs.all", Description: "取全部桌面偏好(无 query 时)", RequestParams: RequestPrefGetAll{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/prefs", HttpType: ginp.HttpPut,
		Handler: ginp.BindParamsHandler(PutPref, &RequestPrefSet{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.prefs.set", Description: "写一条桌面偏好", RequestParams: RequestPrefSet{}},
	})
}