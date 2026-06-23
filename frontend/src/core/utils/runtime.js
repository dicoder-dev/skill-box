// runtime.js - 运行时配置。
//
// 三段优先级读取 runMode:
//   1) import.meta.env.VITE_RUN_MODE —— Vite dev 模式注入(wails3 dev 路径)
//   2) window.__APP_RUNTIME__.runMode —— 后端 gin 在 index.html 注入(桌面 release 路径)
//   3) 兑底 "web"
//
// 之所以要支持 Vite dev 注入:
//   wails3 dev 启动的 webview 加载 Vite dev server(端口 9245),不走后端 gin,
//   所以后端 injectRuntimeScript 永远不会被调用,平台层会兑底成 web。
//   解决:wails3 dev 在启动 Vite 时传 VITE_DEPLOY_MODE=desktop,vite.config.js
//   把它写到 import.meta.env.VITE_RUN_MODE,前端就能识别为桌面端。
//
// 关键点:必须在 import 时立即读取(而不是在 await 后),
// 因为拦截器注册是同步的,首次请求时 runtime 必须就绪。
//
// 字段缺失时的兑底值,选择"安全默认":
//   runMode  = "web"
//   needAuth = true(没拿到配置 = 当成需要鉴权,避免安全洞)

let cached = null

function readRunMode() {
  // 1) Vite dev 注入(编译时常量,dev/build 都可用,build 时被 VITE_ 前缀剔除)
  if (typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_RUN_MODE) {
    return import.meta.env.VITE_RUN_MODE
  }
  // 2) 后端注入的 __APP_RUNTIME__
  if (typeof window !== 'undefined' && window.__APP_RUNTIME__?.runMode) {
    return window.__APP_RUNTIME__.runMode
  }
  return 'web'
}

function readNeedAuth() {
  // Vite 没有 needAuth 注入,走 window 或兑底 true
  if (typeof window !== 'undefined' && typeof window.__APP_RUNTIME__?.needAuth === 'boolean') {
    return window.__APP_RUNTIME__.needAuth
  }
  return true
}

function readAppName() {
  if (typeof window !== 'undefined' && window.__APP_RUNTIME__?.appName) {
    return window.__APP_RUNTIME__.appName
  }
  return ''
}

export function getRuntime() {
  if (cached) return cached
  cached = {
    runMode: readRunMode(),
    // undefined / null 都视为 true,这是"安全默认"
    needAuth: readNeedAuth(),
    appName: readAppName(),
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