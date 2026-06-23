// Package hooks 桌面端 OS 能力钩子的注册中心。
//
// 设计动机:
//   bootstrap 包依赖 router,router 依赖 cdesktop 各 controller。
//   如果 cdesktop 直接依赖 bootstrap,就会形成 import cycle(bootstrap →
//   router → cdesktop → bootstrap)。把"钩子注册中心"(Set/Get)放到独立子包,
//   让 bootstrap 和 cdesktop 都依赖这个子包,反向链就只剩:
//     bootstrap → hooks
//     cdesktop → hooks
//   互不依赖,无循环。
//
// 用法:
//   桌面端 main 启动时由 desktop.NewApp 构造 BootstrapHooks 并通过
//   backend.SetDesktopHooks 注入;bootstrap.Serve 再把它同步到本包的
//   currentHooks;cdesktop 各 handler 通过 Get() 读取后调到真 OS 能力。
package hooks

import "sync"

// BootstrapHooks 桌面端 OS 能力钩子集合。
//
// 每个字段都是可选的(nil 表示该能力在当前部署形态不可用,HTTP 端点应返回 501)。
// 由 desktop 包在 NewApp 时通过 backend.SetBootstrapHooks 注入。
//
// 类型定义在本包里,because both bootstrap 和 cdesktop 都需要它,放在任意
// 一边都会引发循环依赖。bootstrap 包通过类型别名 `BootstrapHooks =
// hooks.BootstrapHooks` 把这个类型透出给跨 module 的桌面端使用。
type BootstrapHooks struct {
	Notify                     func(id, title, body string) error
	NotifyHasPermission        func() bool
	NotifyRequestAuthorization func() (bool, error)
	ClipboardText              func() (string, error)
	SetClipboardText           func(text string) error
	OpenExternal               func(url string) error
	WindowShow                 func()
	WindowToggleAlwaysOnTop    func() bool
	WindowToggleMaximise       func()
	ShortcutRegister           func(combo string) error
	ShortcutUnregister         func(combo string) error
	ShortcutList               func() []string
	AppQuit                    func()
}

var (
	mu      sync.RWMutex
	current BootstrapHooks
)

// Set 由 bootstrap.Serve 调用,把 backend 已注入的 hooks 同步到本包。
// 重复调用以最后一次为准;传零值清空(Web 部署关闭时)。
func Set(h BootstrapHooks) {
	mu.Lock()
	current = h
	mu.Unlock()
}

// Get 返回当前注入的 hooks(只读快照)。handler 通过这个读到 func 字段。
func Get() BootstrapHooks {
	mu.RLock()
	defer mu.RUnlock()
	return current
}
