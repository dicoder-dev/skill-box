// platform/index.js - 平台能力抽象。
//
// 识别规则(权威):
//   由后端在 index.html 注入 window.__APP_RUNTIME__.runMode,
//   由启动命令决定:
//     - wails3 dev / 桌面二进制 → "desktop"(后端会注入 Wails 绑定 window.go.*)
//     - go run ./cmd/web / Web 单进程二进制 → "web"
//   业务代码统一 import { platform } from '@/platform' 使用;
//   永远不要直接读 window.go.* —— 这样以后从桌面切换到 Web、或反过来,不用改业务。
//
// 退化路径:
//   1) 拿不到 __APP_RUNTIME__(如 SSR / 早期报错)→ 兜底按"web"处理。
//   2) runMode="desktop" 但 window.go 缺失(说明 wails 绑定还没生成 / 注入失败)
//      → 仍按 desktop 暴露,但所有能力方法在调用时再做"能力缺失"兜底;
//        这样 store / UI 的"是否桌面端"判断始终跟后端一致,不会误判为 web。

import { getRuntime } from '@/core/utils/runtime.js'

const runMode = (typeof window !== 'undefined'
  ? (window.__APP_RUNTIME__?.runMode || getRuntime().runMode)
  : 'web')

// runMode 是单一权威;只有 runMode==="desktop" 才认为是桌面端。
// 是否真正存在 window.go 绑定作为"能力健全检查",不影响 isDesktop 判定。
const isDesktop = runMode === 'desktop'
const hasWailsBinding = typeof window !== 'undefined' && !!window?.go?.app?.AppService

// 桌面端 Wails event 总线适配器(window.runtime.EventsOn)。
// 浏览器端 no-op 返回一个 dispose 函数。
function createEventSubscriber() {
  if (typeof window === 'undefined' || !window?.runtime?.EventsOn) {
    return { subscribe: () => () => {} }
  }
  return {
    subscribe(name, cb) {
      window.runtime.EventsOn(name, cb)
      return () => {
        try { window.runtime.EventsOff(name) } catch (_) { /* ignore */ }
      }
    },
  }
}

const events = createEventSubscriber()

// 桌面能力的安全包装:在 wails 绑定缺失时直接抛"不支持"错误,
// 而不是触发 window.go.foo is not a function 的隐式崩溃。
function guard(name, fn) {
  return async (...args) => {
    if (!hasWailsBinding || !window?.go?.app?.AppService) {
      throw new Error(`desktop capability "${name}" unavailable: wails bindings missing`)
    }
    return fn(...args)
  }
}

function createWebPlatform() {
  return {
    isDesktop: false,
    runMode: 'web',
    app: {
      async getVersion() { return 'web' },
      async getServerPort() { return 0 },
      async health() { return 'web' },
      async quit() { /* no-op */ },
    },
    window: {
      async toggleAlwaysOnTop() { return false },
      async show() { /* no-op */ },
      async toggleMaximise() { /* no-op */ },
    },
    platform: {
      os: () => 'web',
      arch: () => 'web',
      async clipboardText() { return '' },
      async setClipboardText() { return false },
      async openExternal(url) {
        // Web 端打开外链直接用 window.open
        window.open(url, '_blank', 'noopener')
      },
    },
    notify: {
      async hasPermission() { return false },
      async requestPermission() { return false },
      async show() { return false },
      onResult() { return () => {} },
    },
    shortcut: {
      async register() { return false },
      async unregister() { return false },
      async list() { return [] },
    },
    prefs: {
      async get() { return ['', false, null] },
      async set() { return false },
      async getAll() { return {} },
    },
  }
}

function createDesktopPlatform() {
  return {
    isDesktop: true,
    runMode: 'desktop',
    app: {
      getVersion: guard('app.getVersion', () =>
        window.go.app.AppService.GetVersion()),
      getServerPort: guard('app.getServerPort', () =>
        window.go.app.AppService.GetServerPort()),
      health: guard('app.health', () =>
        window.go.app.AppService.Health()),
      quit: guard('app.quit', () =>
        window.go.app.AppService.Quit()),
    },
    window: {
      toggleAlwaysOnTop: guard('window.toggleAlwaysOnTop', () =>
        window.go.desktop.WindowService.ToggleAlwaysOnTop()),
      show: guard('window.show', () =>
        window.go.desktop.WindowService.Show()),
      toggleMaximise: guard('window.toggleMaximise', () =>
        window.go.desktop.WindowService.ToggleMaximise()),
    },
    platform: {
      os: () => window.go.platform.PlatformService.OS(),
      arch: () => window.go.platform.PlatformService.Arch(),
      clipboardText: guard('platform.clipboardText', () =>
        window.go.platform.PlatformService.ClipboardText()),
      setClipboardText: guard('platform.setClipboardText', (text) =>
        window.go.platform.PlatformService.SetClipboardText(text)),
      openExternal: guard('platform.openExternal', (url) =>
        window.go.platform.PlatformService.OpenExternal(url)),
    },
    notify: {
      hasPermission: guard('notify.hasPermission', () =>
        window.go.notify.NotifyService.HasPermission()),
      requestPermission: guard('notify.requestPermission', () =>
        window.go.notify.NotifyService.RequestAuthorization()),
      // Wails v3 alpha.60 NotifyService.Show(id, title, body)
      show: guard('notify.show', (id, title, body) =>
        window.go.notify.NotifyService.Show(id || '', title || '', body || '')),
      onResult(cb) {
        // 后端 emit("notify:clicked", id, actionID) → 推 actionID 给前端
        return events.subscribe('notify:clicked', (actionID, notifID) => {
          try { cb(actionID, notifID) } catch (e) { console.error('[notify:clicked]', e) }
        })
      },
    },
    shortcut: {
      register: guard('shortcut.register', (combo) =>
        window.go.shortcut.ShortcutService.Register(combo)),
      unregister: guard('shortcut.unregister', (combo) =>
        window.go.shortcut.ShortcutService.Unregister(combo)),
      list: guard('shortcut.list', () =>
        window.go.shortcut.ShortcutService.List()),
    },
    prefs: {
      // Go 返回 []any = [value, exists, err];前端拿到是普通对象,转成三元组
      get: guard('prefs.get', (key) => {
        const r = window.go.prefs.PrefsService.Get(key)
        return [r?.[0] ?? '', !!r?.[1], r?.[2] || null]
      }),
      set: guard('prefs.set', (key, value) =>
        window.go.prefs.PrefsService.Set(key, String(value))),
      getAll: guard('prefs.getAll', () =>
        window.go.prefs.PrefsService.GetAll() || {}),
    },
  }
}

export const platform = isDesktop ? createDesktopPlatform() : createWebPlatform()
