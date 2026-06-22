// api/http.js - 统一 HTTP 客户端入口。
//
// Web 端:baseURL 为空字符串,走相对路径(同源)。
// 桌面端:baseURL 为 http://127.0.0.1:<port>。
//
// 桌面/Web 判定依据:window.__APP_RUNTIME__.runMode(由后端按启动命令注入)。
//   - wails3 dev / 桌面二进制 → runMode="desktop"
//   - go run ./cmd/web / Web 单进程二进制 → runMode="web"
// 不再用 window.go 是否存在作为判据,因为后端注入的 runMode 才是权威。
//
// 端口解析策略:
//   1) 优先用 Wails 绑定 window.go.app.AppService.GetServerPort() 拿端口
//   2) 拿不到时,fetch 当前 origin /api/health 探测,成功则信任当前 origin
//   3) 兜底直接走当前 origin(window.location.origin)——开发时由 vite 代理后端
//
// 业务代码应 import { http } from '@/api/http' 调接口,不要直接用 fetch 或 axios。

import { getRuntime } from '@/core/utils/runtime.js'

let resolvedBaseURL = null

/**
 * 探测桌面端后端端口。
 * 顺序:Wails 绑定 → 健康检查探测 → 当前 origin。
 */
async function detectDesktopBaseURL() {
  // 1) Wails 绑定路径(桌面端 main.go 通过 application.NewService 暴露)
  try {
    const port = window?.go?.app?.AppService?.GetServerPort?.()
    if (port && port > 0) {
      return `http://127.0.0.1:${port}`
    }
  } catch (e) {
    // 忽略,继续走探测
  }

  // 2) 健康检查探测(Web 端也走这个,只是同源而已)
  const origin = window.location.origin
  try {
    const resp = await fetch(`${origin}/api/health`, { method: 'GET' })
    if (resp.ok) {
      return origin
    }
  } catch (e) {
    // 忽略
  }

  // 3) 兜底用当前 origin(开发环境 vite 代理)
  return origin
}

/**
 * 解析最终 baseURL。同一会话内只解析一次,缓存到模块作用域。
 */
export async function resolveBaseURL() {
  if (resolvedBaseURL !== null) return resolvedBaseURL

  // Web 端:相对路径即可(同源)。通过 location.protocol 区分:
  //   http/https → 同源 Web,baseURL 留空
  //   file:     → 不可能(Wails 走 http),兜底空
  const proto = window.location.protocol
  // 桌面/Web 判据:由启动命令注入到 __APP_RUNTIME__.runMode,不再探测 window.go。
  const runMode = getRuntime().runMode
  if (proto === 'http:' || proto === 'https:') {
    if (runMode === 'desktop') {
      resolvedBaseURL = await detectDesktopBaseURL()
    } else {
      resolvedBaseURL = '' // Web 端相对路径
    }
  } else {
    resolvedBaseURL = ''
  }
  return resolvedBaseURL
}

/**
 * 轻量 fetch 封装,自动拼 baseURL。
 * 用法:const data = await http.get('/api/health')
 *       const data = await http.post('/api/sys_user/login', { username, password })
 */
export const http = {
  async request(method, path, body) {
    const base = await resolveBaseURL()
    const url = `${base}${path}`
    const opts = {
      method,
      headers: { 'Content-Type': 'application/json' },
    }
    if (body !== undefined) {
      opts.body = typeof body === 'string' ? body : JSON.stringify(body)
    }
    const resp = await fetch(url, opts)
    const text = await resp.text()
    let data
    try {
      data = text ? JSON.parse(text) : null
    } catch (e) {
      data = text
    }
    if (!resp.ok) {
      const err = new Error(`HTTP ${resp.status} ${resp.statusText}`)
      err.status = resp.status
      err.data = data
      throw err
    }
    return data
  },
  get(path) { return this.request('GET', path) },
  post(path, body) { return this.request('POST', path, body ?? {}) },
  put(path, body) { return this.request('PUT', path, body ?? {}) },
  delete(path) { return this.request('DELETE', path) },
}
