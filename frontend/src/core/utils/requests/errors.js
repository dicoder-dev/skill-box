// errors.js - 业务异常类。
//
// 所有 HTTP/业务层异常都继承 ApiError,业务侧可以用 instanceof 区分:
//   err instanceof HttpError      → HTTP 层失败(网络/HTTP 非 2xx)
//   err instanceof BusinessError  → 业务码失败(后端 {code,msg,data} 中 code !== 1)
//   err instanceof TimeoutError   → 请求超时
// 统一字段:
//   status  HTTP 状态码(网络异常时为 0)
//   code    业务码(成功 1 / 失败 0 或自定义)
//   data    后端 data 字段

export class ApiError extends Error {
  constructor({ message, status = 0, code = null, data = null } = {}) {
    super(message || 'api request failed')
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.data = data
  }
}

// HTTP 层失败:网络断 / 超时 / HTTP 非 2xx
export class HttpError extends ApiError {
  constructor(opts = {}) {
    super({ ...opts, message: opts.message || `http error: ${opts.status || 'network'}` })
    this.name = 'HttpError'
  }
}

// 业务码失败:后端返回 {code: 0, msg, data} 这种结构时抛出
export class BusinessError extends ApiError {
  constructor(opts = {}) {
    super({
      ...opts,
      // 业务失败时 status 通常是 200,只把 msg 当 message
      message: opts.message || opts.msg || 'business error',
    })
    this.name = 'BusinessError'
    this.msg = opts.msg || opts.message || ''
  }
}

// 请求超时
export class TimeoutError extends HttpError {
  constructor(opts = {}) {
    super({ ...opts, status: opts.status || 0, message: opts.message || 'request timeout' })
    this.name = 'TimeoutError'
  }
}