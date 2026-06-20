import { createApp } from 'vue'
import App from './App.vue'
import { resolveBaseURL } from './api/http.js'
import { http } from './core/utils/requests'
import { platform } from './platform/index.js'

// 双部署入口:
// 1) 解析后端 baseURL(Web 走相对路径,桌面走 http://127.0.0.1:<port>)
// 2) 探测一次健康检查,确认后端真的在跑(桌面端尤其重要:Webview 加载时后端可能还在初始化)
// 3) 再 mount Vue,确保首次业务请求能找到后端
async function bootstrap() {
  const base = await resolveBaseURL()
  // 暴露到全局,方便调试
  window.__APP_CONFIG__ = { baseURL: base, isDesktop: platform.isDesktop }

  try {
    await http.get('/api/health')
  } catch (e) {
    console.warn('health check failed,业务接口可能暂时不可用:', e.message)
  }

  createApp(App).mount('#app')
}

bootstrap()
