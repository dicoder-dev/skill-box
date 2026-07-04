// core/utils/external_link.js
//
// 统一外链点击拦截。
// 背景:markdown-it 默认会把 <a> 渲染成 target="_blank",桌面 webview 会"在 webview 内打开",
// 体感很糟糕(用户期望走系统浏览器)。我们在 markdown_view.js 把 link_open 改成输出
// class="md-external-link" data-url="<href>"(不带 target),然后由调用方在容器
// 上挂 @click="handleExternalClick" → handleExternalClick 拦下来后走
// platform.platform.openExternal(url)。
//
// Web 端走 window.open(url, '_blank')。
// 桌面端走 /api/desktop/open-external → wails Browser.OpenURL(系统默认浏览器)。
//
// 2026-07-04 增:首页技能文件浏览器(Commit 2),把 SkillsView 主区 markdown
// 与 SkillFileDrawer 内 markdown 走同一份外链逻辑,保证行为一致。

import { platform } from '@/platform'
import { useToastStore } from '@/core/store/toast'

/**
 * 处理容器内 .md-external-link 类的点击。
 * 在模板里写:
 *   <div class="md-body" @click="onMdClick" v-html="renderedHtml" />
 *   function onMdClick(e) { handleExternalClick(e) }
 *
 * @param {Event} e - click 事件
 * @returns {boolean} true 表示拦下了链接点击
 */
export function handleExternalClick(e) {
  // 兼容 e.target 可能是子元素,closest 找最近的 a 标签
  const a = e.target && e.target.closest && e.target.closest('a.md-external-link')
  if (!a) return false
  e.preventDefault()
  const url = a.getAttribute('data-url') || a.getAttribute('href') || ''
  if (!url) return false
  // 异步 openExternal,失败弹 toast(Web 端 window.open 一般不会失败,兜底)
  try {
    const r = platform.platform.openExternal(url)
    if (r && typeof r.catch === 'function') {
      r.catch((err) => {
        try {
          useToastStore().error(`打开外链失败: ${err?.message || err}`)
        } catch (_) { /* toast 不可用就静默 */ }
      })
    }
  } catch (err) {
    try {
      useToastStore().error(`打开外链失败: ${err?.message || err}`)
    } catch (_) { /* 静默 */ }
  }
  return true
}
