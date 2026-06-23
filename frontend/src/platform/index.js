// platform/index.js - 平台能力抽象。
//
// 识别规则(权威):
//   由后端在 index.html 注入 window.__APP_RUNTIME__.runMode,
//   由启动命令决定:
//     - wails3 dev / 桌面二进制 → "desktop"(后端会注入 Wails 运行时,
//       业务通过自动生成的 bindings/* 模块 + @wailsio/runtime 的 Call.ByID 调到 Go)
//     - go run ./cmd/web / Web 单进程二进制 → "web"
//   业务代码统一 import { platform } from '@/platform' 使用;
//   永远不要直接读 window.go.* —— 这样以后从桌面切换到 Web、或反过来,不用改业务。
//
// 退化路径:
//   1) 拿不到 __APP_RUNTIME__(如 SSR / 早期报错)→ 兜底按"web"处理。
//   2) runMode="desktop" 但 wails 运行时缺失(说明 wails 还没 ready / 注入失败)
//      → 仍按 desktop 暴露,但所有能力方法在调用时再做"能力缺失"兜底;
//        这样 store / UI 的"是否桌面端"判断始终跟后端一致,不会误判为 web。

import { getRuntime } from '@/core/utils/runtime.js'

// 自动生成的 wails v3 绑定:每个 service 一个模块,内部用 Call.ByID(methodID, ...args) 调到 Go。
// 这些模块依赖 @wailsio/runtime 暴露的 window.runtime,在桌面 webview 里由 Wails 注入;
// 在 web 端不会走到这些调用路径(createWebPlatform 已兜底)。
// bindings 物理目录在 frontend/bindings/(与 src 平级),不走 @/ 别名。
import * as AppBindings from '../../../bindings/skill-box/desktop/services/appservice.js'
import * as WindowBindings from '../../../bindings/skill-box/desktop/services/windowservice.js'
import * as PlatformBindings from '../../../bindings/skill-box/desktop/services/platformservice.js'
import * as NotifyBindings from '../../../bindings/skill-box/desktop/services/notifyservice.js'
import * as ShortcutBindings from '../../../bindings/skill-box/desktop/services/shortcutservice.js'
import * as PrefsBindings from '../../../bindings/skill-box/desktop/services/prefsservice.js'

const runMode = (typeof window !== 'undefined'
  ? (window.__APP_RUNTIME__?.runMode || getRuntime().runMode)
  : 'web')

// runMode 是单一权威;只有 runMode==="desktop" 才认为是桌面端。
// 是否真正存在 wails 运行时作为"能力健全检查",不影响 isDesktop 判定。
const isDesktop = runMode === 'desktop'
// v3 alpha.60 把运行时挂在 window.runtime(由 @wailsio/runtime 注入),
// 同时 Wails 后端把所有 method 注册到 window.go.<package>.<Service>.* 上供 v2 风格代码使用;
// 这里只检查 v3 风格的 Call.ByID 入口是否存在,作为"运行时是否就绪"的最小判据。
const hasWailsBinding = typeof window !== 'undefined'
  && !!window?.runtime?.Call?.ByID

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

// 桌面能力的安全包装:在 wails 运行时缺失时直接抛"不支持"错误,
// 而不是触发 undefined is not a function 的隐式崩溃。
function guard(name, fn) {
  return async (...args) => {
    if (!hasWailsBinding) {
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
      // web 端三元组是 [value, exists](wails v3 生成器跳过 error 返回值)
      async get() { return ['', false] },
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
      getVersion: guard('app.getVersion', () => AppBindings.GetVersion()),
      getServerPort: guard('app.getServerPort', () => AppBindings.GetServerPort()),
      health: guard('app.health', () => AppBindings.Health()),
      quit: guard('app.quit', () => AppBindings.Quit()),
    },
    window: {
      toggleAlwaysOnTop: guard('window.toggleAlwaysOnTop', () =>
        WindowBindings.ToggleAlwaysOnTop()),
      show: guard('window.show', () => WindowBindings.Show()),
      toggleMaximise: guard('window.toggleMaximise', () =>
        WindowBindings.ToggleMaximise()),
    },
    platform: {
      os: () => PlatformBindings.OS(),
      arch: () => PlatformBindings.Arch(),
      clipboardText: guard('platform.clipboardText', () =>
        PlatformBindings.ClipboardText()),
      setClipboardText: guard('platform.setClipboardText', (text) =>
        PlatformBindings.SetClipboardText(text)),
      openExternal: guard('platform.openExternal', (url) =>
        PlatformBindings.OpenExternal(url)),
    },
    notify: {
      hasPermission: guard('notify.hasPermission', () => NotifyBindings.HasPermission()),
      requestPermission: guard('notify.requestPermission', () =>
        NotifyBindings.RequestAuthorization()),
      // Wails v3 alpha.60 NotifyService.Show(id, title, body)
      show: guard('notify.show', (id, title, body) =>
        NotifyBindings.Show(id || '', title || '', body || '')),
      onResult(cb) {
        // 后端 emit("notify:clicked", id, actionID) → 推 actionID 给前端
        return events.subscribe('notify:clicked', (actionID, notifID) => {
          try { cb(actionID, notifID) } catch (e) { console.error('[notify:clicked]', e) }
        })
      },
    },
    shortcut: {
      register: guard('shortcut.register', (combo) => ShortcutBindings.Register(combo)),
      unregister: guard('shortcut.unregister', (combo) =>
        ShortcutBindings.Unregister(combo)),
      list: guard('shortcut.list', () => ShortcutBindings.List()),
    },
    prefs: {
      // Go 端 PrefsService.Get(key) (string, bool, error) → wails v3 生成器跳过 error,
      // 生成的 prefsservice.js 返回 [string, boolean](见 prefsservice.js:19)
      get: guard('prefs.get', (key) => PrefsBindings.Get(key)),
      set: guard('prefs.set', (key, value) =>
        PrefsBindings.Set(key, String(value))),
      getAll: guard('prefs.getAll', () => PrefsBindings.GetAll() || {}),
    },
  }
}

export const platform = isDesktop ? createDesktopPlatform() : createWebPlatform()