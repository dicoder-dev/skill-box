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
//
// 2026-07-04 改:增加 tokenColors + editorColors 美化。
// 旧版只设 background / foreground,代码高亮走 VS 默认(在浅色背景上对比度不够)。
// 新版:
//   - 浅色(skillbox-light):浅米背景 + 高对比关键字(蓝/紫/绿/橙/红)
//   - 暗色(skillbox-dark):深蓝黑背景 + 高对比关键字(青/紫/绿/黄/红)
//   - 编辑器:当前行高亮、选区、行号、缩进参考线、光标配色都精细化
// 颜色基于项目 UI 调色板(蓝/绿/橙/红,主色禁用紫),不引入紫色 token。
function applyTheme(monaco) {
  if (typeof monaco === 'undefined') return

  // ====== 浅色主题 ======
  monaco.editor.defineTheme('skillbox-light', {
    base: 'vs',
    inherit: true,
    rules: [
      // 关键字:def / class / return / if / import 等
      { token: 'keyword', foreground: '2563eb', fontStyle: 'bold' },         // 蓝
      { token: 'keyword.control', foreground: '2563eb', fontStyle: 'bold' },
      { token: 'keyword.operator', foreground: '334155' },
      // 字符串
      { token: 'string', foreground: '16a34a' },                             // 绿
      { token: 'string.quoted', foreground: '16a34a' },
      { token: 'string.escape', foreground: 'ea580c' },                       // 橙
      // 注释
      { token: 'comment', foreground: '94a3b8', fontStyle: 'italic' },       // 灰
      { token: 'comment.line', foreground: '94a3b8', fontStyle: 'italic' },
      { token: 'comment.block', foreground: '94a3b8', fontStyle: 'italic' },
      // 数字 / 常量
      { token: 'number', foreground: 'ea580c' },                              // 橙
      { token: 'number.hex', foreground: 'ea580c' },
      { token: 'constant', foreground: 'ea580c' },
      { token: 'constant.language', foreground: 'ea580c' },
      // 类型 / 类名
      { token: 'type', foreground: '0891b2' },                                // 青
      { token: 'type.identifier', foreground: '0891b2' },
      { token: 'entity.name.type', foreground: '0891b2' },
      { token: 'entity.name.class', foreground: '0891b2' },
      // 函数名
      { token: 'entity.name.function', foreground: '2563eb' },
      { token: 'support.function', foreground: '2563eb' },
      // 变量
      { token: 'variable', foreground: '334155' },
      { token: 'variable.parameter', foreground: '7c3aed' },                  // 紫(仅变量,token 用)
      { token: 'variable.other', foreground: '334155' },
      // 标签(HTML/JSX)
      { token: 'tag', foreground: 'dc2626' },                                 // 红
      { token: 'tag.class', foreground: '0891b2' },
      { token: 'tag.id', foreground: 'ea580c' },
      // 属性(HTML/CSS)
      { token: 'attribute.name', foreground: 'ea580c' },
      { token: 'attribute.value', foreground: '16a34a' },
      // JSON key
      { token: 'key', foreground: '0891b2' },
      // 操作符 / 分隔符
      { token: 'delimiter', foreground: '475569' },
      { token: 'delimiter.bracket', foreground: '475569' },
    ],
    colors: {
      'editor.background': '#fafafa',
      'editor.foreground': '#0f172a',
      'editor.lineHighlightBackground': '#e0e7ff',
      'editor.lineHighlightBorder': '#c7d2fe',
      'editor.selectionBackground': '#bfdbfe',
      'editor.inactiveSelectionBackground': '#dbeafe',
      'editorCursor.foreground': '#2563eb',
      'editor.lineNumbers.foreground': '#94a3b8',
      'editor.lineNumbers.activeForeground': '#475569',
      'editor.lineNumbers.background': '#fafafa',
      'editor.indentGuides': '#e2e8f0',
      'editor.activeIndentGuide': '#94a3b8',
      'editorBracketMatch.background': '#fef3c7',
      'editorBracketMatch.border': '#fbbf24',
      'editor.findMatchBackground': '#fde68a',
      'editor.findMatchHighlightBackground': '#fef3c7',
      'editorWidget.background': '#ffffff',
      'editorWidget.border': '#e2e8f0',
      'editorSuggestWidget.background': '#ffffff',
      'editorSuggestWidget.border': '#e2e8f0',
      'editorSuggestWidget.selectedBackground': '#e0e7ff',
      'scrollbarSlider.background': '#cbd5e1',
      'scrollbarSlider.hoverBackground': '#94a3b8',
      'scrollbarSlider.activeBackground': '#64748b',
    },
  })

  // ====== 暗色主题 ======
  monaco.editor.defineTheme('skillbox-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: '60a5fa', fontStyle: 'bold' },         // 亮蓝
      { token: 'keyword.control', foreground: '60a5fa', fontStyle: 'bold' },
      { token: 'keyword.operator', foreground: 'cbd5e1' },
      { token: 'string', foreground: '4ade80' },                             // 亮绿
      { token: 'string.quoted', foreground: '4ade80' },
      { token: 'string.escape', foreground: 'fb923c' },                      // 亮橙
      { token: 'comment', foreground: '64748b', fontStyle: 'italic' },
      { token: 'comment.line', foreground: '64748b', fontStyle: 'italic' },
      { token: 'comment.block', foreground: '64748b', fontStyle: 'italic' },
      { token: 'number', foreground: 'fb923c' },
      { token: 'number.hex', foreground: 'fb923c' },
      { token: 'constant', foreground: 'fb923c' },
      { token: 'constant.language', foreground: 'fb923c' },
      { token: 'type', foreground: '22d3ee' },                               // 亮青
      { token: 'type.identifier', foreground: '22d3ee' },
      { token: 'entity.name.type', foreground: '22d3ee' },
      { token: 'entity.name.class', foreground: '22d3ee' },
      { token: 'entity.name.function', foreground: '60a5fa' },
      { token: 'support.function', foreground: '60a5fa' },
      { token: 'variable', foreground: 'e2e8f0' },
      { token: 'variable.parameter', foreground: 'c4b5fd' },                 // 亮紫(仅变量)
      { token: 'variable.other', foreground: 'e2e8f0' },
      { token: 'tag', foreground: 'f87171' },                                // 亮红
      { token: 'tag.class', foreground: '22d3ee' },
      { token: 'tag.id', foreground: 'fb923c' },
      { token: 'attribute.name', foreground: 'fb923c' },
      { token: 'attribute.value', foreground: '4ade80' },
      { token: 'key', foreground: '22d3ee' },
      { token: 'delimiter', foreground: '94a3b8' },
      { token: 'delimiter.bracket', foreground: '94a3b8' },
    ],
    colors: {
      'editor.background': '#0f172a',                                        // 深蓝黑
      'editor.foreground': '#e2e8f0',
      'editor.lineHighlightBackground': '#1e293b',
      'editor.lineHighlightBorder': '#334155',
      'editor.selectionBackground': '#1e40af',
      'editor.inactiveSelectionBackground': '#1e3a8a',
      'editorCursor.foreground': '#60a5fa',
      'editor.lineNumbers.foreground': '#475569',
      'editor.lineNumbers.activeForeground': '#94a3b8',
      'editor.lineNumbers.background': '#0f172a',
      'editor.indentGuides': '#1e293b',
      'editor.activeIndentGuide': '#475569',
      'editorBracketMatch.background': '#422006',
      'editorBracketMatch.border': '#fbbf24',
      'editor.findMatchBackground': '#854d0e',
      'editor.findMatchHighlightBackground': '#713f12',
      'editorWidget.background': '#1e293b',
      'editorWidget.border': '#334155',
      'editorSuggestWidget.background': '#1e293b',
      'editorSuggestWidget.border': '#334155',
      'editorSuggestWidget.selectedBackground': '#1e3a8a',
      'scrollbarSlider.background': '#334155',
      'scrollbarSlider.hoverBackground': '#475569',
      'scrollbarSlider.activeBackground': '#64748b',
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
//
// 2026-07-04 改:import 路径从 'editor.api.js' 换成 'editor.main.js'。
// 旧版只 import API stub,monaco.editor.defineTheme / setTheme / 实际主题实现
// 都在 'standalone/browser/standaloneEditor.js' 里(不在 editor.api.js),
// 结果 setTheme 静默失败,tokenColors 全部不生效。
// 'editor.main.js' 通过 basic-languages + language/* 自动引入 standalone 主题注册。
export async function loadMonaco() {
  if (monacoRef) return monacoRef
  if (loadingPromise) return loadingPromise
  loadingPromise = (async () => {
    // 动态 import 完整入口(包含所有 language + theme 注册)
    const monaco = await import('monaco-editor/esm/vs/editor/editor.main.js')
    // 设置 worker(Vite 单独 chunk)
    if (typeof self !== 'undefined') {
      self.MonacoEnvironment = {
        getWorkerUrl(_moduleId, _label) {
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

// 导出 isDark 供外部使用(CodeViewer 创建 editor 时需要判断 theme)
// 2026-07-04 改:之前 CodeViewer 没传 theme,Monaco 用默认 'vs',
// useMonaco 里 setTheme 的全局主题不会自动应用。
export { isDark }
