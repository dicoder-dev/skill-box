// client.js - 底层 fetch 封装。
//
// 职责:
//   1) 接受标准化 config { url, method, headers, body, timeout, signal }
//   2) 用 AbortController 串接 timeout + 外部 signal
//   3) 非 2xx 抛 HttpError(业务码剥离留给 response 拦截器)
//   4) 解析响应体:JSON 优先,失败回退到 text
//   5) 失败时附带 response.data 方便业务排查
//
// 不依赖拦截器,可以独立使用。

import { HttpError, TimeoutError } from './errors.js'

const DEFAULT_TIMEOUT_MS = 15000

/**
 * 解析响应体。优先按 JSON 解,失败回退成 text。
 */
async function readBody(resp) {
  const text = await resp.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch (_) {
    return text
  }
}

/**
 * 执行一次 fetch。
 *
 * @param {object} config
 * @param {string} config.url     完整 URL
 * @param {string} config.method  GET/POST/PUT/DELETE
 * @param {object} [config.headers]
 * @param {any}    [config.body]   已是序列化好的字符串 / FormData / Blob
 * @param {number} [config.timeout] 毫秒,默认 15000,0 表示不超时
 * @param {AbortSignal} [config.signal] 外部取消信号
 * @returns {Promise<{ok:boolean,status:number,statusText:string,headers:Headers,data:any}>}
 */
export async function request(config) {
  const {
    url,
    method = 'GET',
    headers = {},
    body,
    timeout = DEFAULT_TIMEOUT_MS,
    signal,
  } = config || {}

  // 串接:外部 signal + 我们自己的 timeout controller
  const controller = new AbortController()
  const onExternalAbort = () => controller.abort(signal?.reason)
  if (signal) {
    if (signal.aborted) {
      controller.abort(signal.reason)
    } else {
      signal.addEventListener('abort', onExternalAbort, { once: true })
    }
  }
  let timer = null
  if (timeout > 0) {
    timer = setTimeout(() => controller.abort(new DOMException('Timeout', 'TimeoutError')), timeout)
  }

  try {
    const resp = await fetch(url, {
      method,
      headers,
      body,
      signal: controller.signal,
      // 不让浏览器自动 follow redirect 时丢 headers;这里显式保留 credentials 关闭(走 same-origin)
      credentials: 'same-origin',
    })
    const data = await readBody(resp)
    const result = {
      ok: resp.ok,
      status: resp.status,
      statusText: resp.statusText,
      headers: resp.headers,
      data,
    }
    if (!resp.ok) {
      // HTTP 层失败:抛 HttpError,附 data 方便定位后端 message
      const message =
        (data && typeof data === 'object' && (data.error || data.message || data.msg)) ||
        `HTTP ${resp.status} ${resp.statusText}`
      throw new HttpError({ message, status: resp.status, data })
    }
    return result
  } catch (err) {
    // AbortController 因 timeout 触发:归一为 TimeoutError
    if (err && err.name === 'AbortError' && !signal?.aborted) {
      throw new TimeoutError({ message: `request timeout after ${timeout}ms` })
    }
    // fetch 自身的 TypeError(网络断/DNS 失败/CORS)被归一为 HttpError
    if (err instanceof HttpError) throw err
    if (err && err.name === 'TimeoutError') {
      throw new TimeoutError({ message: err.message || 'request timeout' })
    }
    throw new HttpError({ message: err?.message || 'network error', status: 0 })
  } finally {
    if (timer) clearTimeout(timer)
    if (signal) signal.removeEventListener('abort', onExternalAbort)
  }
}