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
//   backend.SetDesktopHooks 注入;bootstrap 在 Serve 之前把 backend 指针
//   Bind 到本包;cdesktop 各 handler 通过 Get() 实时读 backend 的最新
//   desktopHooks —— 这样 backend.SetDesktopHooks 之后,后续所有 HTTP
//   请求立即拿到新值,不用重启 server。
//
// 为什么不在 Serve 启动时一次性 Set 到 current 变量:
//   时序问题:go Serve(backend) 在 goroutine 里立刻跑,此时 desktop.NewApp
//   还没执行 SetDesktopHooks,Serve 第一次 Set 的是空值。后续 NewApp 再注入
//   也不会传播到 current(只 Set 一次)。改成持有 backend 指针 + 实时读,
//   彻底解决时序。
package hooks

import "sync"

// Provider 让本包通过 Backend/任意持有者读取最新 BootstrapHooks。
// bootstrap 包实现这个接口(只读 getter),解耦本包与具体 backend 类型。
type Provider interface {
	// GetDesktopHooks 返回当前注入的 BootstrapHooks 快照(可能为零值)。
	GetDesktopHooks() BootstrapHooks
}

var (
	mu       sync.RWMutex
	provider Provider
)

// Bind 由 bootstrap.Serve 调用,把 backend(实现 Provider 接口)注入本包。
// 重复调用以最后一次为准;传 nil 清空(Web 部署关闭时)。
func Bind(p Provider) {
	mu.Lock()
	provider = p
	mu.Unlock()
}

// Get 返回当前 backend 中注入的 hooks(只读快照)。
//
// 实现方式:实时从 Provider.GetDesktopHooks 读,而不是读本包缓存。
// 原因:desktop.SetDesktopHooks 之后 backend 的 hooks 会更新,如果缓存
// 在 Bind 时一次性拷走,后续 Get 拿到的还是旧值;实时读则总是最新。
func Get() BootstrapHooks {
	mu.RLock()
	p := provider
	mu.RUnlock()
	if p == nil {
		return BootstrapHooks{}
	}
	return p.GetDesktopHooks()
}

// BootstrapHooks 桌面端 OS 能力钩子集合。
//
// 每个字段都是可选的(nil 表示该能力在当前部署形态不可用,HTTP 端点应返回 501)。
// 由 desktop 包在 NewApp 时通过 backend.SetDesktopHooks 注入。
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
	// FsReadText 读文件文本内容(给前端获取 SKILL.md 等),返回内容 + error。
	// 限制 1 MB(超过返 error),避免把巨大文件拉爆内存。
	FsReadText func(path string) (string, error)
	// FsReveal 在系统文件管理器中显示给定路径(若 path 是文件,定位到该文件;
	// 在 macOS 走 `open -R` / Windows 走 `explorer /select` / Linux 走 xdg-open
	// 父目录)。Web 端无桌面,hook 不会注入,前端走到降级(打开父目录 file://)。
	FsReveal func(path string) error
	// FsPickFolder 弹出系统文件夹选择对话框,用户选择后返回绝对路径;
	// 取消选择时返回空字符串且 error 为 nil。Web 端无桌面,hook 不会注入,
	// 前端需要降级到 input[type=file] webkitdirectory 等价方案。
	FsPickFolder func() (string, error)
	// FsPickFile 弹出系统文件选择对话框(2026-07-01 增),可选 accept 过滤
	// 后缀(如 []string{".zip"})。返回用户选中的绝对路径,取消时返空串。
	// 桌面端通过 wails3 v3 OpenFileDialog 绑定(待 wails3 alpha 稳定后补);
	// 当前未实现时,hook 不会被注入,前端走到降级(<input type="file">)。
	FsPickFile func(accept []string) (string, error)
	WindowShow                 func()
	WindowToggleAlwaysOnTop    func() bool
	WindowToggleMaximise       func()
	ShortcutRegister           func(combo string) error
	ShortcutUnregister         func(combo string) error
	ShortcutList               func() []string
	AppQuit                    func()
}
