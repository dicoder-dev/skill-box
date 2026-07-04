// core/composables/useMonaco.js
//
// Monaco Editor 懒加载 + 单例管理。
//
// 设计要点:
//   1. 首次调用 loadMonaco() 时动态 import,加载期间显示 spinner
//   2. 全局单例缓存,后续切文件不再重新加载
//   3. Worker 走 Vite ?worker 语法,自动拆 chunk
//   4. 主题跟随站点(html.dark 类切换),定义两套 skillbox-light / skillbox-dark
//
// 2026-07-04 增:首页技能文件浏览器(Commit 3)。

import { ref, watchEffect } from 'vue'

let monacoRef = null
let loadingPromise = null
let monacoInstance = null
let themeObserver = null

// 站点是否深色模式
function isDark() {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}

// 应用主题(light / dark 跟随站点)
function applyTheme(monaco) {
  if (typeof monaco === 'undefined') return
  monaco.editor.defineTheme('skillbox-light', {
    base: 'vs',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': '#ffffff',
      'editor.foreground': '#171717',
    },
  })
  monaco.editor.defineTheme('skillbox-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': '#171717',
      'editor.foreground': '#e5e5e5',
    },
  })
  monaco.editor.setTheme(isDark() ? 'skillbox-dark' : 'skillbox-light')
}

// 监听 html.dark 类变化,自动重设主题
function watchTheme(monaco) {
  if (typeof document === 'undefined' || monacoInstance) return
  monacoInstance = monaco
  if (themeObserver) return
  themeObserver = new MutationObserver(() => {
    if (!monacoInstance) return
    monacoInstance.editor.setTheme(isDark() ? 'skillbox-dark' : 'skillbox-light')
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
}

// 加载 monaco + 注册 worker
export async function loadMonaco() {
  if (monacoRef) return monacoRef
  if (loadingPromise) return loadingPromise
  loadingPromise = (async () => {
    // 动态 import,首次加载会下载 ~700KB gzip(按需加载)
    const monaco = await import('monaco-editor/esm/vs/editor/editor.api.js')
    // 设置 worker(Vite 单独 chunk)
    if (typeof self !== 'undefined') {
      self.MonacoEnvironment = {
        getWorkerUrl(_moduleId, _label) {
          // 走 Vite ?worker,直接用 editor.worker
          // 语言相关 worker(typescript / json / css / html)也走同一个 editor.worker
          // (简化处理,功能上仍可语法高亮 + 折叠,只是无高级补全)
          return new URL('monaco-editor/esm/vs/editor/editor.worker?worker', import.meta.url).toString()
        },
      }
    }
    applyTheme(monaco)
    watchTheme(monaco)
    monacoRef = { monaco }
    return monacoRef
  })()
  return loadingPromise
}

// 当前是否已加载
export function isMonacoLoaded() {
  return !!monacoRef
}
