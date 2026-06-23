// Package hooks 桌面端 OS 能力钩子的注册中心。
//
// 设计动机:
//   bootstrap 包依赖 router,router 依赖 cdesktop 各 controller。
//   如果 cdesktop 直接依赖 bootstrap,就会形成 import cycle(bootstrap →
//   router → cdesktop → bootstrap)。把"钩子注册中心"(Set/Get)提到独立子包,
//   让 bootstrap 和 cdesktop 都依赖这个子包,反向链就只剩:
//     bootstrap → hooks
//     cdesktop → hooks
//   互不依赖,无循环。
//
// 类型定义 BootstrapHooks 放在 bootstrap 包里(非 internal),便于桌面端
// (skill-box/desktop)直接复用同一个类型构造钩子值,避免在桌面端重复定义。
//
// 用法:
//   桌面端 main 启动时由 desktop.NewApp 构造 bootstrap.BootstrapHooks 并通过
//   backend.SetDesktopHooks 注入;bootstrap.Serve 再把它同步到本包的
//   currentHooks;cdesktop 各 handler 通过 Get() 读取后调到真 OS 能力。
package hooks

import (
	"sync"

	"ginp-api/cmd/bootstrap"
)

// current 持有当前生效的钩子集合。Web 部署下保持零值(所有 func 字段 nil),
// cdesktop 端点自然降级到 501。
var (
	mu      sync.RWMutex
	current bootstrap.BootstrapHooks
)

// Set 由 bootstrap.Serve 调用,把 backend 已注入的 hooks 同步到本包。
// 重复调用以最后一次为准;传零值清空(Web 部署关闭时)。
func Set(h bootstrap.BootstrapHooks) {
	mu.Lock()
	current = h
	mu.Unlock()
}

// Get 返回当前注入的 hooks(只读快照)。handler 通过这个读到 func 字段。
func Get() bootstrap.BootstrapHooks {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
