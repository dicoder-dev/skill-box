// wails-runtime-shim.js - Web 模式下 @wailsio/runtime 的占位实现。
//
// 用途:
//   bindings/*.js 是 wails 自动生成的代码,固定 import "@wailsio/runtime" 的
//   { Call, CancellablePromise, Create }。在 web 模式下:
//
//     - 我们不会调用 bindings(platform/index.js 的 createWebPlatform 已经兜底了所有调用);
//     - 但 bindings 顶层 import 会被 vite 拉进 bundle,从而触发 @wailsio/runtime/dist/index.js
//       的副作用(System.invoke / console.warn)—— 在浏览器里没意义,反而污染控制台。
//
//   这个 shim 通过 vite alias 在 web 模式替换 "@wailsio/runtime",提供同名导出但走最小副作用:
//     - Call.ByID 永远 reject(不应被调到,因为 web 模式跑不到);
//     - CancellablePromise 是普通 Promise 的别名;
//     - Create.Any / Create.Array / Create.Map 是 identity。

// Web 模式下 window.runtime 永远不存在,这里用严格检查让任何意外的调用立刻暴露问题。
function callByID() {
  return Promise.reject(
    new Error(
      '[wails-runtime-shim] Call.ByID 在 web 模式下不应被调用;检查 platform/index.js 的 isDesktop 路径。',
    ),
  )
}

class CancellablePromise extends Promise {
  constructor(executor) {
    super(executor)
    this._cancel = null
  }
  cancel() {
    if (typeof this._cancel === 'function') {
      try { this._cancel() } catch (_) { /* ignore */ }
    }
  }
}

const identity = (v) => v
const Create = {
  Any: identity,
  Array: (elem) => identity,
  Map: (key, value) => identity,
}

export const Call = { ByID: callByID }
export { CancellablePromise, Create }
// 兼容 named + default 导出(以防 bindings 用 default 形式)
export default { Call, CancellablePromise, Create }