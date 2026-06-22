// platform/index.js - 平台能力抽象。
//
// Web 端:返回 web 实现(全部 no-op 或抛"不支持")。
// 桌面端:返回 desktop 实现,通过 window.go.* Wails 绑定调桌面能力。
//
// 业务代码统一 import { platform } from '@/platform' 使用,
// 永远不要直接读 window.go.* —— 这样以后从桌面切换到 Web、或反过来,不用改业务。

const isDesktop = typeof window !== 'undefined' && !!window?.go?.app?.AppService

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

function createWebPlatform() {
  return {
    isDesktop: false,
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
    app: {
      async getVersion() {
        return window.go.app.AppService.GetVersion()
      },
      async getServerPort() {
        return window.go.app.AppService.GetServerPort()
      },
      async health() {
        return window.go.app.AppService.Health()
      },
      async quit() {
        return window.go.app.AppService.Quit()
      },
    },
    window: {
      async toggleAlwaysOnTop() {
        return window.go.desktop.WindowService.ToggleAlwaysOnTop()
      },
      async show() {
        return window.go.desktop.WindowService.Show()
      },
      async toggleMaximise() {
        return window.go.desktop.WindowService.ToggleMaximise()
      },
    },
    platform: {
      os: () => window.go.platform.PlatformService.OS(),
      arch: () => window.go.platform.PlatformService.Arch(),
      async clipboardText() {
        return window.go.platform.PlatformService.ClipboardText()
      },
      async setClipboardText(text) {
        return window.go.platform.PlatformService.SetClipboardText(text)
      },
      async openExternal(url) {
        return window.go.platform.PlatformService.OpenExternal(url)
      },
    },
    notify: {
      async hasPermission() {
        return window.go.notify.NotifyService.HasPermission()
      },
      async requestPermission() {
        return window.go.notify.NotifyService.RequestAuthorization()
      },
      async show(id, title, body) {
        // Wails v3 alpha.60 NotifyService.Show(id, title, body)
        return window.go.notify.NotifyService.Show(id || '', title || '', body || '')
      },
      onResult(cb) {
        // 后端 emit("notify:clicked", id, actionID) → 推 actionID 给前端
        return events.subscribe('notify:clicked', (actionID, notifID) => {
          try { cb(actionID, notifID) } catch (e) { console.error('[notify:clicked]', e) }
        })
      },
    },
    shortcut: {
      async register(combo) {
        return window.go.shortcut.ShortcutService.Register(combo)
      },
      async unregister(combo) {
        return window.go.shortcut.ShortcutService.Unregister(combo)
      },
      async list() {
        return window.go.shortcut.ShortcutService.List()
      },
    },
    prefs: {
      async get(key) {
        // Go 返回 []any = [value, exists, err];前端拿到是普通对象,转成三元组
        const r = window.go.prefs.PrefsService.Get(key)
        return [r?.[0] ?? '', !!r?.[1], r?.[2] || null]
      },
      async set(key, value) {
        return window.go.prefs.PrefsService.Set(key, String(value))
      },
      async getAll() {
        return window.go.prefs.PrefsService.GetAll() || {}
      },
    },
  }
}

export const platform = isDesktop ? createDesktopPlatform() : createWebPlatform()
