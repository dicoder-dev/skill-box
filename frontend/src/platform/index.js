// platform/index.js - 平台能力抽象。
//
// 识别规则(权威):
//   1) 优先 import.meta.env.VITE_RUN_MODE(由 vite.config.js 在 dev 模式
//      下从 VITE_DEPLOY_MODE 环境变量注入,wails3 dev 启动 Vite 时传
//      VITE_DEPLOY_MODE=desktop 即可让前端在 Vite dev server 上也识别为桌面端)。
//   2) 退回 window.__APP_RUNTIME__.runMode(由后端 gin 在生产 index.html
//      注入,桌面端 release binary 走这条路径)。
//   3) 都拿不到时兑底 "web"。
//
//   决定能力可见性:
//     - runMode === "desktop" → 桌面端能力可见(通知、剪贴板、托盘、偏好等)
//     - runMode === "web"     → 仅 Web 能力,桌面调用 guard() 抛 "unavailable"
//
//   启动命令对照:
//     - wails3 dev(VITE_DEPLOY_MODE=desktop) → 桌面形态,Webview 走 Vite
//     - ./bin/skill-box(release)             → 桌面形态,Webview 走后端 gin
//     - go run ./cmd/web                     → Web 单进程
import { getRuntime } from '@/core/utils/runtime.js'
import { http } from '@/api/http.js'

// 优先级:import.meta.env(Vite dev 注入) > __APP_RUNTIME__(后端注入) > 兑底 web
function resolveRunMode() {
  if (typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_RUN_MODE) {
    return import.meta.env.VITE_RUN_MODE
  }
  if (typeof window !== 'undefined' && window.__APP_RUNTIME__?.runMode) {
    return window.__APP_RUNTIME__.runMode
  }
  try {
    return getRuntime().runMode
  } catch (_) {
    return 'web'
  }
}

const runMode = resolveRunMode()

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
    fs: {
      // 读本地文件文本(Web 端走后端 HTTP,fsutil 兜底处理;失败抛错给调用方)
      async readText(path) {
        try {
          const r = await http.post('/api/desktop/fs/read-text', { path })
          return r?.content || ''
        } catch (e) {
          throw new Error(`readText(${path}) failed: ${e?.message || e}`)
        }
      },
      // reveal 在系统文件管理器显示该路径。
      // Web 端桌面 hook 不存在 → 501 带回退 URL(父目录 file://),用 openExternal 打开。
      async reveal(path) {
        try {
          await http.post('/api/desktop/fs/reveal', { path })
          return true
        } catch (e) {
          const fb = e?.data?.fallback_url || e?.response?.data?.fallback_url
          if (fb) {
            window.open(fb, '_blank', 'noopener')
            return true
          }
          throw new Error(`reveal(${path}) failed: ${e?.message || e}`)
        }
      },
      // pickFolder 弹系统文件夹选择对话框,用户取消时返空串(不抛错)。
      // Web 端无桌面 hook,后端返 501 → 这里降级抛错给调用方处理。
      async pickFolder() {
        try {
          const r = await http.post('/api/desktop/fs/pick-folder', {})
          return r?.path || ''
        } catch (e) {
          throw new Error(`pickFolder failed: ${e?.message || e}`)
        }
      },
      // 2026-07-01 增:pickFile 弹系统文件选择对话框。
      // Web 端无桌面 hook,后端返 501 → 这里抛"不支持",由调用方降级到
      // <input type="file"> 走 /api/skillbox/onboarding/import-zip-bytes。
      async pickFile() {
        try {
          const r = await http.post('/api/desktop/fs/pick-file', {})
          return r?.path || ''
        } catch (e) {
          throw new Error(`pickFile failed: ${e?.message || e}`)
        }
      },
      // inspectProject 从目录路径推断 name / alias,供"导入项目"预填表单。
      async inspectProject(path) {
        try {
          const r = await http.post('/api/desktop/fs/inspect-project', { path })
          return { name: r?.name || '', alias: r?.alias || '' }
        } catch (e) {
          throw new Error(`inspectProject(${path}) failed: ${e?.message || e}`)
        }
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
    fs: {
      async readText(path) {
        const r = await http.post('/api/desktop/fs/read-text', { path })
        return r?.content || ''
      },
      async reveal(path) {
        try {
          await http.post('/api/desktop/fs/reveal', { path })
          return true
        } catch (e) {
          // 兜底:桌面端没装 hook(Web 部署)时,fs.reveal 端点返 501 + fallback_url
          // 这里不再二次 openExternal(避免循环),由调用方决定后续动作
          const fb = e?.data?.fallback_url || e?.response?.data?.fallback_url
          if (fb) return { ok: false, fallbackUrl: fb }
          throw e
        }
      },
      async pickFolder() {
        const r = await http.post('/api/desktop/fs/pick-folder', {})
        return r?.path || ''
      },
      // 2026-07-01 增:桌面端 pickFile,后端走 wails3 OpenFileDialog 绑定
      // (wails3 v3 alpha.60 暂无该绑定,fsutil 端点先返 501,后续补齐)。
      async pickFile() {
        const r = await http.post('/api/desktop/fs/pick-file', {})
        return r?.path || ''
      },
      async inspectProject(path) {
        const r = await http.post('/api/desktop/fs/inspect-project', { path })
        return { name: r?.name || '', alias: r?.alias || '' }
      },
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