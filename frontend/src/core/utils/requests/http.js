// http.js - 业务层 API 入口。
//
// 暴露 http.get/post/put/delete,内部走:
//   1) 解析 baseURL(复用 @/api/http 的 resolveBaseURL)
//   2) 串接 query string(GET / DELETE 支持 params)
//   3) 跑 request 拦截器链 → client.request → response 拦截器链
//   4) 业务码剥离由默认 response 拦截器负责,这里只透传
//
// 调用风格与原 @/api/http 保持一致,便于平迁:
//   await http.get('/api/health')
//   await http.post('/api/sys_user/login', { username, password })
//   await http.get('/api/sys_user/search', { page: 1, size: 5 })   // 自动转 query

import { resolveBaseURL } from '@/api/http.js'
import { request as clientRequest } from './client.js'
import { interceptors } from './interceptors.js'

/**
 * 把 params 对象拼成 query string。空值跳过。
 */
function buildQuery(params) {
  if (!params || typeof params !== 'object') return ''
  const usp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null) continue
    usp.set(k, typeof v === 'object' ? JSON.stringify(v) : String(v))
  }
  const qs = usp.toString()
  return qs ? `?${qs}` : ''
}

/**
 * 执行一次请求(走拦截器链 + 底层 client)。
 * @returns {Promise<any>} 默认响应拦截器已剥离 {code,msg,data},这里直接拿到 data
 */
async function doRequest(method, path, bodyOrParams, options = {}) {
  const {
    timeout,
    signal,
    headers,
    raw, // raw=true 时不拼接 query(由调用方自己处理 path),也不走业务码剥离
  } = options

  // query string:GET/DELETE 走 params,POST/PUT 走 body
  let realPath = path
  let body
  if (method === 'GET' || method === 'DELETE') {
    realPath = `${path}${buildQuery(bodyOrParams)}`
  } else {
    body = bodyOrParams
  }

  // baseURL 复用 @/api/http 的解析逻辑
  const base = await resolveBaseURL()
  const url = `${base}${realPath}`

  // 初始 config
  let cfg = {
    url,
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(headers || {}),
    },
    body: body !== undefined ? (typeof body === 'string' ? body : JSON.stringify(body)) : undefined,
    timeout,
    signal,
    raw: !!raw,
  }

  // request 拦截器链
  cfg = await interceptors.request.run(cfg)

  // 底层请求
  const resp = await clientRequest(cfg)

  // response 拦截器链(raw 模式跳过业务码剥离)
  if (cfg.raw) return resp
  return interceptors.response.run(resp)
}

export const http = {
  get(path, params, options) {
    return doRequest('GET', path, params, options)
  },
  post(path, body, options) {
    return doRequest('POST', path, body, options)
  },
  put(path, body, options) {
    return doRequest('PUT', path, body, options)
  },
  delete(path, options) {
    return doRequest('DELETE', path, undefined, options)
  },
  request(method, path, bodyOrParams, options) {
    return doRequest(method.toUpperCase(), path, bodyOrParams, options)
  },
}