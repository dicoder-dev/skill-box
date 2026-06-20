// Package services 提供给 Wails Webview 调用的桌面服务绑定。
//
// 命名空间约定:
//   window.go.app.AppService      → 通用 App 信息
//   window.go.desktop.WindowService→ 窗口控制
//   window.go.platform.PlatformService → 平台能力
//
// 业务调用请走 HTTP,不暴露在这里。
package services

import (
	"context"
	"time"

	"ginp-api/pkg/logger"
)

// Backend 描述桌面端后端的能力(端口查询)。
// 这里用接口定义,避免 services 反向依赖 desktop 或 bootstrap 包。
type Backend interface {
	Port() int
	URL() string
}

// Version 应用版本号,发布时通过 -ldflags 注入。
var Version = "0.0.0-dev"

// AppService 通用应用服务:版本、端口、健康、退出。
type AppService struct {
	local Backend
}

// NewAppService 构造 AppService。local 可以为空(仅 Web 端使用)。
func NewAppService(local Backend) *AppService {
	return &AppService{local: local}
}

// GetVersion 返回应用版本号。
func (s *AppService) GetVersion() string {
	return Version
}

// GetServerPort 返回本地 api-server 监听端口,前端用来拼 BASE_URL。
// Web 端模式下(无 local server)返回 0,前端应忽略该值走相对路径。
func (s *AppService) GetServerPort() int {
	if s.local == nil {
		return 0
	}
	return s.local.Port()
}

// Health 返回本地 server 健康状态;Web 端返回 "web" 表示非桌面部署。
func (s *AppService) Health() string {
	if s.local == nil {
		return "web"
	}
	return "ok"
}

// Quit 优雅退出应用(由前端触发,如用户主动退出按钮)。
// 这里通过 Wails 主循环退出,后端 server 由 main 协程的 Serve 阻塞、
// 进程退出时被 OS 强制关闭。
func (s *AppService) Quit() {
	if s.local != nil {
		// 给 logger 留出刷盘时间,然后让前端 Quit 触发 Wails 退出。
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = ctx
	}
	logger.Info("desktop: quit requested from frontend")
}
