// index.js - 桶导出。
//
// 业务侧统一从这里 import:
//   import { http, ApiError, BusinessError, interceptors } from '@/core/utils/requests'
//
// 也可以单独 import 子模块:
//   import { http } from '@/core/utils/requests/http.js'
//
// 默认拦截器(http.js 模块加载时自动安装):
//   - request:  从 localStorage.token 注入 Authorization
//   - response: HTTP 401 清 token + 跳 /login
//   - response: 业务码剥离,code !== 1 抛 BusinessError

import { http } from './http.js'
import {
  ApiError,
  HttpError,
  BusinessError,
  TimeoutError,
} from './errors.js'
import { interceptors, installDefaultInterceptors } from './interceptors.js'

// 模块加载时安装默认拦截器(idempotent:重复注册会重复执行,但都是幂等操作,只多走一次)
installDefaultInterceptors()

export { http }
export { ApiError, HttpError, BusinessError, TimeoutError }
export { interceptors, installDefaultInterceptors }