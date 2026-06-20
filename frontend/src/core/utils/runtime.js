// runtime.js - 运行时配置。
//
// 后端在 index.html 的 <head> 末尾注入 <script>window.__APP_RUNTIME__={...}</script>,
// 我们在这里同步读取它,提供给拦截器 / 业务使用。
//
// 关键点:必须在 import 时立即读取(而不是在 await 后),
// 因为拦截器注册是同步的,首次请求时 runtime 必须就绪。
//
// 字段缺失时的兜底值,选择"安全默认":
//   runMode  = "web"
//   needAuth = true(没拿到配置 = 当成需要鉴权,避免安全洞)

let cached = null

function readFromWindow() {
  if (typeof window === 'undefined') return null
  return window.__APP_RUNTIME__ || null
}

export function getRuntime() {
  if (cached) return cached
  const raw = readFromWindow()
  cached = {
    runMode: raw?.runMode || 'web',
    // undefined / null 都视为 true,这是"安全默认"
    needAuth: raw?.needAuth !== false,
    appName: raw?.appName || '',
  }
  return cached
}

// 调试辅助:把 runtime 打到全局,方便 devtools 看
export function dumpRuntime() {
  if (typeof window !== 'undefined') {
    window.__APP_RUNTIME_DEBUG__ = getRuntime()
  }
  return getRuntime()
}