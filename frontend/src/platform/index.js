// platform/index.js - 平台能力抽象。
//
// 识别规则(权威):
//   由后端在 index.html 注入 window.__APP_RUNTIME__.runMode,
//   由启动命令决定:
//     - wails3 dev / 桌面二进制 → "desktop"(webview 加载后端 HTTP server,
//       桌面能力统一走 /api/desktop/* HTTP 端点)
//     - go run ./cmd/web / Web 单进程二进制 → "web"
//   业务代码统一 import { platform } from '@/platform' 使用。
//
// 退化路径:
//   1) 拿不到 __APP_RUNTIME__(如 SSR / 早期报错)→ 兜底按"web"处理。
//   2) runMode="desktop" 但 HTTP 调用失败(后端未启动)→ 桌面能力方法抛
//      "desktop capability X unavailable",业务可降级提示用户。
//
// 注意:Wails v3 alpha.60 不再像 v2 那样把 Go service 注入到 window.go.*;
// 自动生成的 bindings/* 用 $Call.ByID(methodID, ...) 走 fetch /wails/runtime,
// 而我们的 webview 是由后端 Gin 服务的,没有 /wails/runtime 路由。
// 因此 platform 层不再走 wails bindings,统一改用后端 HTTP 端点。

import { getRuntime } from '@/core/utils/runtime.js'
import { http } from '@/api/http.js'

const runMode = (typeof window !== 'undefined'
  ? (window.__APP_RUNTIME__?.runMode || getRuntime().runMode)
  : 'web')

// runMode 是单一权威;只有 runMode==="desktop" 才认为是桌面端。
const isDesktop = runMode === 'desktop'

// 桌面能力的安全包装:在调用失败时直接抛"不支持"错误,
// 而不是触发 undefined is not a function 的隐式崩溃。
function guard(name, fn) {
  return async (...args) => {
    if (!isDesktop) {
      throw new Error(`desktop capability "${name}" unavailable: not running in desktop mode`)
    }
    return fn(...args)
  }
}

// 桌面端 Wails event 总线适配器:
// 当前项目没有需要从后端推任意事件到前端的桌面事件(通知点击等已走 notify onResult),
// 这里保留 no-op subscribe 通道,保持 platform 接口形状稳定。
function createEventSubscriber() {
  return { subscribe: () => () => {} }
}

const events = createEventSubscriber()

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
      // web 端 prefs 用 /api/user/prefs 之类的业务路由(暂未实现),此处返回空
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
      // 桌面 webview 直接 load 后端 URL,不需要单独拿 port;返回 0 让 baseURL 解析走相对路径
      async getVersion() {
        try { return await http.get('/api/desktop/app/version') } catch (_) { return 'desktop' }
      },
      async getServerPort() {
        // 桌面模式下 baseURL 由后端 boot 阶段在 index.html 注入;
        // 这里返回 0,http.js 会用 fetch /api/desktop/app/health 探测或兜底走 origin
        return 0
      },
      async health() {
        try { return await http.get('/api/desktop/app/health') } catch (_) { return 'unavailable' }
      },
      async quit() {
        try { await http.post('/api/desktop/app/quit', {}) } catch (_) { /* ignore */ }
      },
    },
    window: {
      toggleAlwaysOnTop: guard('window.toggleAlwaysOnTop', () =>
        http.post('/api/desktop/window/toggle-always-on-top', {})),
      show: guard('window.show', () => http.post('/api/desktop/window/show', {})),
      toggleMaximise: guard('window.toggleMaximise', () =>
        http.post('/api/desktop/window/toggle-maximise', {})),
    },
    platform: {
      os: () => {
        try { return window.__APP_RUNTIME__?.os || 'desktop' } catch (_) { return 'desktop' }
      },
      arch: () => {
        try { return window.__APP_RUNTIME__?.arch || 'desktop' } catch (_) { return 'desktop' }
      },
      clipboardText: guard('platform.clipboardText', () => http.get('/api/desktop/clipboard/text')),
      setClipboardText: guard('platform.setClipboardText', (text) =>
        http.put('/api/desktop/clipboard/text', { text })),
      openExternal: guard('platform.openExternal', (url) =>
        http.post('/api/desktop/open-external', { url })),
    },
    notify: {
      hasPermission: guard('notify.hasPermission', () => http.get('/api/desktop/notify/permission')),
      requestPermission: guard('notify.requestPermission', () =>
        http.post('/api/desktop/notify/permission/request', {})),
      show: guard('notify.show', (id, title, body) =>
        http.post('/api/desktop/notify/show', { id: id || '', title: title || '', body: body || '' })),
      onResult(cb) {
        // 后端通过 SSE / WebSocket / fetch long-poll 推 notify:clicked 事件;
        // 当前项目暂未实现,先返回 no-op dispose,接口形状稳定,后续接 SSE 再补。
        return events.subscribe('notify:clicked', (actionID, notifID) => {
          try { cb(actionID, notifID) } catch (e) { console.error('[notify:clicked]', e) }
        })
      },
    },
    shortcut: {
      register: guard('shortcut.register', (combo) => http.post('/api/desktop/shortcut/register', { combo })),
      unregister: guard('shortcut.unregister', (combo) => http.post('/api/desktop/shortcut/unregister', { combo })),
      list: guard('shortcut.list', () => http.get('/api/desktop/shortcut/list')),
    },
    prefs: {
      // desktop 模式 prefs 走 HTTP,与业务 API 一致;
      // settings KV 由后端 bootstrap.Backend.NewSettings() 工厂方法构造,数据落 entity.Setting 表。
      // 返回值约定(对齐 wails v3 自动生成签名):
      //   get(key)      → [value: string, exists: boolean]
      //   getAll()      → { [key]: value }
      get: guard('prefs.get', async (key) => {
        const r = await http.get(`/api/desktop/prefs?key=${encodeURIComponent(key)}`)
        // 后端约定 { value, exists }
        return [r?.value ?? '', !!r?.exists]
      }),
      set: guard('prefs.set', (key, value) =>
        http.put('/api/desktop/prefs', { key, value: String(value) })),
      getAll: guard('prefs.getAll', async () => {
        const r = await http.get('/api/desktop/prefs')
        // 后端约定 { items: { [key]: value } }
        return r?.items || {}
      }),
    },
  }
}

export const platform = isDesktop ? createDesktopPlatform() : createWebPlatform()