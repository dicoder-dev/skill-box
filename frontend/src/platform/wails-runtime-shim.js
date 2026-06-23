// wails-runtime-shim.js - 浏览器侧 @wailsio/runtime 的轻量替身。
//
// 设计动机:
//   bindings/skill-box/desktop/services/*.js 是 wails v3 自动生成的代码,
//   固定 import { Call, CancellablePromise, Create } from "@wailsio/runtime"。
//   整个 bundle 里只有 platform/index.js 在 isDesktop 时才调它们,
//   web 模式(createWebPlatform)已用空实现兜底,不会走到 Call.ByID。
//
//   真正的 wails 运行时由桌面 webview 注入到 window.runtime(system.js 里
//   检测 window.chrome.webview / window.webkit.messageHandlers / window.wails.invoke),
//   那个运行时只在桌面二进制启动的 webview 里才存在;浏览器里没有。
//
//   这个 shim 通过 vite alias 把 "@wailsio/runtime" 替换成:
//     - Call.ByID: 转发到 window.runtime?.Call?.ByID(若存在),
//                  否则 reject 给出明确错误;
//     - CancellablePromise: 一个原生 Promise 的薄包装;
//     - Create.*: identity,够 bindings 反序列化用。
//
// 副作用:
//   shim 自身不 import 任何东西、不注册全局副作用,因此 web 端 console
//   不会被 wails runtime 污染,fetch /wails/runtime 也不会被打。

function ensureRuntime() {
  if (typeof window === 'undefined') return null
  return window.runtime || window._wails || null
}

export function ByID(methodID, ...args) {
  const rt = ensureRuntime()
  if (rt && rt.Call && typeof rt.Call.ByID === 'function') {
    return rt.Call.ByID(methodID, ...args)
  }
  return Promise.reject(
    new Error(
      `[wails-runtime-shim] Call.ByID(${methodID}) 在当前环境不可用: ` +
      'window.runtime 不存在(预期:仅桌面 webview 才会调用)。',
    ),
  )
}

export function ByName(methodName, ...args) {
  const rt = ensureRuntime()
  if (rt && rt.Call && typeof rt.Call.ByName === 'function') {
    return rt.Call.ByName(methodName, ...args)
  }
  return Promise.reject(
    new Error(
      `[wails-runtime-shim] Call.ByName("${methodName}") 在当前环境不可用`,
    ),
  )
}

export function Call(options) {
  const rt = ensureRuntime()
  if (rt && rt.Call && typeof rt.Call.Call === 'function') {
    return rt.Call.Call(options)
  }
  return Promise.reject(
    new Error('[wails-runtime-shim] Call() 在当前环境不可用'),
  )
}

// CancellablePromise 是 wails runtime 给 binding 返回的"可取消 promise"。
// shim 用一个原生 Promise 子类,提供 .cancel() 钩子但取消时是 no-op。
export class CancellablePromise extends Promise {
  constructor(executor) {
    super(executor)
    this._onCancel = null
  }
  cancel() {
    if (typeof this._onCancel === 'function') {
      try { this._onCancel() } catch (_) { /* ignore */ }
    }
  }
}

const identity = (v) => v
const Create = {
  Any: identity,
  Array: (_elem) => identity,
  Map: (_key, _value) => identity,
}

export { Create }
// 兼容 default 导出(以防生成器切到 default 形式)
export default { Call: { ByID, ByName, Call }, CancellablePromise, Create }