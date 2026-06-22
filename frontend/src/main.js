import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import './style.css'
import { resolveBaseURL } from './api/http.js'
import { http } from './core/utils/requests'
import { getRuntime } from './core/utils/runtime.js'
import { enableDebug, dlog, isDebug } from './core/utils/debug.js'
import { useAppStore } from './core/store/app.js'
import i18n from './core/i18n/index.js'

// 双部署入口:
// 1) 注册 pinia 并一次性写 store(runtime + baseURL)
//    runMode 由后端在 index.html 注入到 window.__APP_RUNTIME__,是平台形态的唯一权威。
// 2) 探测一次健康检查,确认后端真的在跑(桌面端尤其重要:Webview 加载时后端可能还在初始化)
// 3) 再 mount Vue,确保首次业务请求能找到后端
async function bootstrap() {
  const pinia = createPinia()
  const app = createApp(App)
  app.use(pinia)
  // i18n 必须在 use(pinia) 之后,因为 App.vue 会通过 useI18n() 访问。
  // 注意:i18n 的 locale 已经在 createI18n 时基于 localStorage/navigator 解析;
  // 若想由后端 / runtime 强制覆盖,可在 setRuntime 之后调 setLocale(runtime.lang)。
  app.use(i18n)

  const store = useAppStore()

  // 1) 运行时配置(同步读 __APP_RUNTIME__)。
  //    runMode 决定后续一切"是否桌面端"判断,见 store/app.js。
  const runtime = getRuntime()
  store.setRuntime(runtime)
  // 2) 解析 baseURL(依据 runMode,不再探测 window.go)
  const base = await resolveBaseURL()
  store.setBaseURL(base)



  // 3) 调试模式:Vite dev 自动开,生产可由 ?debug=req 触发
  const wantDebug = import.meta.env.DEV ||
    (typeof location !== 'undefined' &&
      /(^|[?&])(debug|debug=req|debug=1)\b/.test(location.search))
  if (wantDebug) enableDebug()
  // 暴露到全局,方便调试
  window.__APP_CONFIG__ = { baseURL: base, runMode: runtime.runMode, isDesktop: runtime.runMode === 'desktop' }
  window.__APP_STORE__ = store
  dlog('bootstrap ready', {
    runtime,
    baseURL: base,
    debug: isDebug(),
  })

  // 5) 健康检查(走完整请求层,顺便验证拦截器)
  try {
    await http.get('/api/health')
    dlog('health ok')
  } catch (e) {
    console.warn('health check failed,业务接口可能暂时不可用:', e.message)
  }

  // 6) 挂载
  app.mount('#app')
}

bootstrap()
