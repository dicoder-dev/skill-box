// interceptors.js - 拦截器栈。
//
// 仿 axios 风格:
//   interceptors.request.use((config) => config, (err) => Promise.reject(err))
//   interceptors.response.use((resp) => resp, (err) => Promise.reject(err))
//
// 链式调用:后注册的拦截器包住先注册的(洋葱模型):
//   request:  fn1 → fn2 → fn3 → 真正请求
//   response: 真正请求 → fn3 → fn2 → fn1
//
// 用法:
//   interceptors.request.use((cfg) => { cfg.headers['X-Trace'] = '...'; return cfg })
//   interceptors.response.use((resp) => { return resp.data }, (err) => Promise.reject(err))

import { BusinessError, HttpError } from './errors.js'
import { getRuntime } from '../runtime.js'

function createManager() {
  const handlers = []
  return {
    use(onFulfilled, onRejected) {
      const id = handlers.length
      handlers.push({ onFulfilled, onRejected })
      return id
    },
    eject(id) {
      handlers[id] = null
    },
    /**
     * 串联运行所有 handler。
     * - 第一个 handler 接收 initial
     * - 每个 handler 返回值传给下一个
     * - 任一 handler 抛错或 reject,跳到对应 onRejected;后续 onFulfilled 跳过
     */
    async run(initial) {
      let value = initial
      for (const h of handlers) {
        if (!h) continue
        if (!h.onFulfilled) continue
        try {
          // eslint-disable-next-line no-await-in-loop
          value = await h.onFulfilled(value)
        } catch (e) {
          if (h.onRejected) {
            // eslint-disable-next-line no-await-in-loop
            value = await h.onRejected(e)
          } else {
            throw e
          }
        }
      }
      return value
    },
    /**
     * 错误链:从尾到头找 onRejected。
     */
    async runReject(err) {
      let value = err
      for (let i = handlers.length - 1; i >= 0; i -= 1) {
        const h = handlers[i]
        if (!h || !h.onRejected) continue
        try {
          // eslint-disable-next-line no-await-in-loop
          value = await h.onRejected(value)
        } catch (e) {
          // 继续往后找
          value = e
        }
      }
      throw value
    },
  }
}

const request = createManager()
const response = createManager()

export const interceptors = {
  request,
  response,
}

/**
 * 安装默认拦截器:
 *   1) request  - needAuth=true 时从 localStorage.token 注入 Authorization
 *   2) response - HTTP 401 兜底:needAuth=true 跳 /login;needAuth=false 仅清 token
 *   3) response - 业务码剥离:code === 1 返回 data,否则抛 BusinessError
 *
 * 行为受后端注入的 window.__APP_RUNTIME__.needAuth 控制:
 *   - 桌面端(needAuth=false):不注入 token、不自动跳 /login
 *   - Web 端 (needAuth=true) :按上面的常规行为
 *
 * 桌面端和 Web 端都用同一套;Webview 内部 localStorage 可用,location.href 也可用。
 */
export function installDefaultInterceptors() {
  // (1) token 注入 — needAuth=false 时跳过
  request.use((cfg) => {
    if (!getRuntime().needAuth) return cfg
    try {
      const token = typeof localStorage !== 'undefined' && localStorage.getItem('token')
      if (token) {
        cfg.headers = { ...(cfg.headers || {}), Authorization: `Bearer ${token}` }
      }
    } catch (_) {
      // SSR / 不存在 localStorage 时忽略
    }
    return cfg
  })

  // (2) HTTP 401 兜底:按 needAuth 分流
  response.use(
    null,
    async (err) => {
      if (!(err instanceof HttpError) || err.status !== 401) {
        throw err
      }
      try {
        localStorage.removeItem('token')
      } catch (_) {
        // ignore
      }
      // needAuth=true 才跳 /login;桌面端由业务自己决定如何处理 401
      if (getRuntime().needAuth) {
        try {
          if (!window.__SKILL_BOX_401_REDIRECTING__) {
            window.__SKILL_BOX_401_REDIRECTING__ = true
            window.location.href = '/login'
          }
        } catch (_) {
          // ignore
        }
      }
      throw err
    },
  )

  // (3) 业务码剥离
  response.use((resp) => {
    const data = resp && resp.data
    // /api/health 之类不走统一结构,data 是 {status, service, ts}——直接返回原 resp
    if (
      data &&
      typeof data === 'object' &&
      // 兼容字段:code / success / status
      ('code' in data || 'success' in data)
    ) {
      const code = data.code !== undefined ? data.code : data.success ? 1 : 0
      if (code !== 1) {
        throw new BusinessError({
          status: resp.status,
          code,
          msg: data.msg || data.message || '',
          data: data.data !== undefined ? data.data : null,
        })
      }
      return data.data !== undefined ? data.data : data
    }
    return resp
  })
}