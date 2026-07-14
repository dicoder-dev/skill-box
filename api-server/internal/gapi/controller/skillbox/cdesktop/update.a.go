// Package cdesktop 的 update.a.go 提供"检测升级 / 下载 / 替身脚本"端点。
//
// 设计动机:
//   Wails v3 alpha.60 没有官方自升级 API,因此本端点对外暴露简单的"check +
//   state + download"三件套,真正的"Quit → 替身脚本接管二进制替换"由桌面端
//   desktop/wails_app.go 通过 BootstrapHooks.UpdaterSpawnHelper / UpdaterDownloadPath 注入。
//   本 controller 不直接 import desktop 包,只通过 hooks.Get() 调用,避免 import cycle。
//
// 路由:
//   GET  /api/desktop/update/check    全形态接;Web 端亦不需 501,行为一致。
//   GET  /api/desktop/update/state    全形态接;进度查询。
//   POST /api/desktop/update/download 桌面端专属;Web 端返 501。
//
// 不变量(I-1):所有路由通过 ginp.RouterAppend 注册,严格遵守 cdesktop 其他端点风格。
package cdesktop

import (
	"os"
	"path/filepath"
	"runtime"

	"ginp-api/internal/gapi/service/updater/ssupdater"
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// ===== /api/desktop/update/check =====

// RespondUpdateCheck 是 check 的统一响应(桌面 + Web 共用)。
type RespondUpdateCheck struct {
	LocalVersion string                  `json:"local_version"`
	RemoteVersion string                `json:"remote_version"`
	Channel      string                  `json:"channel"`
	Status       string                  `json:"status"` // upToDate / available / mustUpdate / incomparable
	HasUpdate    bool                    `json:"has_update"`
	MustUpdate   bool                    `json:"must_update"`
	Notes        map[string]string       `json:"notes,omitempty"`
	Assets       []ssupdater.Asset       `json:"assets,omitempty"`
	Source       string                  `json:"source,omitempty"`
}

// RequestUpdateCheck 当前无入参,占位。
type RequestUpdateCheck struct{}

// GetUpdateCheck GET /api/desktop/update/check
//
// 流程:
//   1) 桌面端通过 BootstrapHooks.UpdaterDownloadPath 拿到 manifest url
//      (manifest.example.json 见 build/updater/manifest.example.json);
//   2) Web 端也走同一路径,前端期望值跟 desktop 一致;
//   3) 当前 OS/Arch 由 runtime.GOOS / GOARCH 取得;
//   4) 本地版本 desktop 走 services.Version 由 ldflags 注入;Web 端
//      兜底 "0.0.0+web-<git short sha>",后续由 A4 注入 __APP_RUNTIME__.version。
func GetUpdateCheck(c *ginp.ContextPlus, _ *RequestUpdateCheck) {
	h := hooks.Get()
	urls := defaultManifestURLs(h)
	if len(urls) == 0 {
		c.JSON(503, gin.H{"error": "updater: no manifest urls configured"})
		return
	}
	cli := ssupdater.NewClient()
	m, err := cli.FetchManifest(c.Request.Context(), urls, "stable")
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	localVer := localVersion(h)
	remoteVer := m.Version
	status := ssupdater.Compare(localVer, remoteVer, m.MinSupport)
	resp := RespondUpdateCheck{
		LocalVersion:  localVer,
		RemoteVersion: remoteVer,
		Channel:       m.Channel,
		Status:        status,
		HasUpdate:     status == ssupdater.StatusAvailable || status == ssupdater.StatusMustUpdate,
		MustUpdate:    status == ssupdater.StatusMustUpdate,
		Notes:         m.Notes,
		Assets:        m.Assets,
	}
	c.JSON(200, resp)
}

// ===== /api/desktop/update/state =====

// RespondUpdateState 是 state 的统一响应。
type RespondUpdateState struct {
	Phase          string `json:"phase"`           // idle / checking / downloading / verifying / pendingRestart / failed
	Progress       int    `json:"progress"`        // 0~100
	Error          string `json:"error,omitempty"`
	DownloadedPath string `json:"downloaded_path,omitempty"`
}

// RequestUpdateState 无入参。
type RequestUpdateState struct{}

// GetUpdateState GET /api/desktop/update/state
func GetUpdateState(c *ginp.ContextPlus, _ *RequestUpdateState) {
	s := ssupdater.StateOf()
	c.JSON(200, RespondUpdateState{
		Phase:          s.Phase,
		Progress:       s.Progress,
		Error:          s.Err,
		DownloadedPath: s.Path,
	})
}

// ===== /api/desktop/update/download =====

// RequestUpdateDownload 触发下载,body 里只带 manifest 的 asset key(暂用 sha256 作 id 兜底)。
type RequestUpdateDownload struct {
	// AssetIndex 选 assets[AssetIndex] 的具体一条;缺省值 0(通常 OS/Arch 已唯一)。
	AssetIndex int `json:"asset_index"`
	// OldVersion 由前端 __APP_RUNTIME__.version 提供,用来给新进程 env SKILLBOX_UPDATER_FROM。
	OldVersion string `json:"old_version"`
}

// PostUpdateDownload POST /api/desktop/update/download
//
// 硬规则(参考 ssupdater.SpawnOrder):
//   1) dbs.IsDesktop() == false(Web) → 501
//   2) 调 hooks.Get().UpdaterSpawnHelper,desktop 实现里 exec.Cmd.Start() **不 Wait**
//   3) helper Start() 失败 → 500,**不调 AppQuit**
//   4) helper Start() 成功 → 调 hooks.Get().AppQuit() 退出 wails 主循环
//
// **Web 端完全不走这条**,因为替身脚本只能 fork 同主机进程,而 web 单进程
// 是 server-less 性质的,不在用户本机。
func PostUpdateDownload(c *ginp.ContextPlus, req *RequestUpdateDownload) {
	h := hooks.Get()
	if h.UpdaterSpawnHelper == nil || h.UpdaterDownloadPath == nil {
		c.JSON(501, gin.H{"error": "updater: only available on desktop"})
		return
	}
	urls := defaultManifestURLs(h)
	cli := ssupdater.NewClient()
	m, err := cli.FetchManifest(c.Request.Context(), urls, "stable")
	if err != nil {
		c.JSON(502, gin.H{"error": err.Error()})
		return
	}
	localVer := localVersion(h)
	status := ssupdater.Compare(localVer, m.Version, m.MinSupport)
	if status == ssupdater.StatusIncomparable || status == ssupdater.StatusUpToDate {
		c.JSON(409, gin.H{"error": "updater: no update available"})
		return
	}
	pick := pickAssetForRuntime(m)
	if pick == nil {
		c.JSON(409, gin.H{"error": "updater: no asset matches current platform"})
		return
	}

	targetDir := h.UpdaterDownloadPath()
	dl := &ssupdater.Downloader{TargetDir: targetDir}
	ssupdater.StartTracking(ssupdater.PhaseDownloading)
	dest, err := dl.Download(c.Request.Context(), pick)
	if err != nil {
		ssupdater.FinishTracking(ssupdater.PhaseFailed, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ssupdater.FinishTracking(ssupdater.PhasePendingRestart, nil)

	// 触发替身脚本(必须在 AppQuit 前 Start)。
	bundle := &ssupdater.HelperBundle{
		DestPath:         dest,
		TargetInstallDir: h.UpdaterInstallDir(),
		OldVersion:       req.OldVersion,
		OS:               pick.OS,
		Arch:             pick.Arch,
	}
	if err := h.UpdaterSpawnHelper(bundle.Args()); err != nil {
		ssupdater.FinishTracking(ssupdater.PhaseFailed, err)
		c.JSON(500, gin.H{"error": "updater: spawn helper failed: " + err.Error()})
		return
	}
	// helper Start 成功 → 退出 wails 主循环。
	if h.AppQuit != nil {
		h.AppQuit()
	}
	c.JSON(200, gin.H{"ok": true})
}

// ===== 工具方法 =====

// defaultUpdateURLs 桌面端可选注入自定义 urls;未注入走默认 GitHub + Gitea mirror 样例,
// 真正上线后由运营在 BootstrapHooks.UpdaterManifestURLs 注入。
//
// 返回值不会被持久化,纯运行时拼接。
func defaultUpdateURLs(h hooks.BootstrapHooks) []string {
	return ssupdater.DefaultManifestURLs(h)
}

// 保留 defaultManifestURLs 作为 alias,保证外部引用兼容。
func defaultManifestURLs(h hooks.BootstrapHooks) []string { return defaultUpdateURLs(h) }

// pickAssetForRuntime 按 runtime.GOOS/GOARCH 选当前平台的 asset。
func pickAssetForRuntime(m *ssupdater.Manifest) *ssupdater.Asset {
	if m == nil {
		return nil
	}
	return m.PickAsset(runtime.GOOS, runtime.GOARCH)
}

// localVersion 桌面端用 services.Version,Web 端兜底 "web"+short commit。
// 由于 ginp-api / skill-box 双模块,这里用包名 `services` 在 desktop 上下文里
// 路径为 `skill-box/desktop/services`。Web 端 main.go 也会通过
// __APP_RUNTIME__ 注入,故 controller 这层统一以 "0.0.0+web" 兜底。
func localVersion(h hooks.BootstrapHooks) string {
	if h.LocalVersion != nil {
		if v := h.LocalVersion(); v != "" {
			return v
		}
	}
	// 兜底:Web 端从 env 里取 sha
	if commit := os.Getenv("SKILLBOX_WEB_COMMIT"); commit != "" {
		return "web-" + commit
	}
	return "0.0.0+web"
}

// cacheDir 给下载目标文件夹兜底(桌面 + Web 都能写)。
func cacheDir(h hooks.BootstrapHooks) string {
	if h.UpdaterDownloadPath != nil {
		return h.UpdaterDownloadPath()
	}
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "skill-box", "updater")
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/update/check", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetUpdateCheck, &RequestUpdateCheck{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.update.check", Description: "查询最新版本与本地对比", RequestParams: RequestUpdateCheck{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/update/state", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetUpdateState, &RequestUpdateState{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.update.state", Description: "当前升级阶段与进度", RequestParams: RequestUpdateState{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/update/download", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostUpdateDownload, &RequestUpdateDownload{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.update.download", Description: "下载 + 触发桌面端替身脚本 + Quit", RequestParams: RequestUpdateDownload{}},
	})
}
