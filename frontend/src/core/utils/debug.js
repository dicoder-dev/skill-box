// debug.js - 调试日志开关。
//
// 默认关闭。开启方式(任一即可):
//   1) main.js 里调 enableDebug()
//   2) URL 带 ?debug / ?debug=req / ?debug=1
//   3) 浏览器控制台执行 __skillBoxDebug(true)
//
// 使用:
//   import { dlog } from '@/core/utils/debug'
//   dlog('-> GET /api/health')
//   dlog('<- 200', { data })
//   dlog('x failed', err)
//
// 关闭:
//   import { disableDebug } from '@/core/utils/debug'
//   disableDebug()

let enabled = false

export function enableDebug() {
  enabled = true
}

export function disableDebug() {
  enabled = false
}

export function isDebug() {
  return enabled
}

/**
 * 输出调试日志。前缀 [req] 方便过滤。
 * 关闭时是 no-op,业务侧放心调。
 */
export function dlog(...args) {
  if (!enabled) return
  // eslint-disable-next-line no-console
  console.log('[req]', ...args)
}

/**
 * 错误日志,独立 tag 便于区分(失败路径总是要看)。
 */
export function derr(...args) {
  if (!enabled) return
  // eslint-disable-next-line no-console
  console.warn('[req]', ...args)
}

// 浏览器侧:暴露一个全局开关,便于 console 直接开/关
if (typeof window !== 'undefined') {
  // @ts-ignore
  window.__skillBoxDebug = (on) => {
    if (on === undefined) on = !enabled
    if (on) enableDebug()
    else disableDebug()
    return isDebug()
  }
}