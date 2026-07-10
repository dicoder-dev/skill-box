// core/utils/markdown_view.js
//
// 查看态用 markdown 渲染器(只用于"展示",不用于编辑态)。
// 编辑态还是用 core/utils/markdown.js + RichTextEditor 那条路)。
//
// 与自研 renderMarkdown 的区别:
//   - 用成熟的 markdown-it,支持 GFM(表格 / 任务列表 / 删除线 / 自动链接)
//   - 代码块接 highlight.js 做语法高亮(支持的语言自动检测)
//   - 输出 HTML 类名保持稳定,样式在 SkillsView 的 .md-body 块里集中控制
//
// 安全性:markdown-it 默认对源文本做 escape,只通过白名单标签;
// html 选项关闭(true → 关闭)以避免用户内容里嵌入危险脚本。
//
// 2026-06-27 改:不再 import 'highlight.js/lib/languages/xxx' 然后 registerLanguage,
// 因为 Vite/Rollup 的 tree-shaking 会认为这种"只用于副作用"的 import 是死代码,
// 把语言定义 JSON 整段摇掉,导致 highlight 输出里完全没有 token 染色。
// 改用全量 'highlight.js'(自带 register),代价是 +~50KB,换取正确的语法高亮。

import MarkdownIt from 'markdown-it'
import taskLists from 'markdown-it-task-lists'
import hljs from 'highlight.js'

// markdown-it 实例(单例)
const md = new MarkdownIt({
  html: false,         // 关闭源 HTML,避免用户嵌入 <script>
  xhtmlOut: false,     // 不输出自闭合 <br />
  breaks: true,        // 软换行 \n 转 <br>(贴近自研渲染器行为)
  linkify: true,       // 自动把裸链接转 <a>(类似 GFM autolink)
  typographer: false,  // 不做排版替换(避免中文标点被改)
  // 代码块高亮:有 lang 时高亮,无 lang 时不自动检测(避免把普通文本误识别为 CSS / 其他语言)
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code class="language-${lang}">${
          hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
        }</code></pre>`
      } catch (_) { /* fallthrough */ }
    }
    return `<pre class="hljs"><code class="hljs">${md.utils.escapeHtml(str)}</code></pre>`
  },
})
// GFM 任务列表:把 "- [x] xxx" 转成 <input type="checkbox" checked>
.use(taskLists, { enabled: true, label: true })

// 2026-07-10 增:给每个 heading 输出 id,用于大纲导航定位。
// 重写 heading_open rule:h1-h6 在输出时追加 id="md-h-{slug}",
// slug 是把标题文本做转小写 + 替换非字母数字为 - + 末尾截断的产物,
// 同名标题追加 -1 / -2 区分(虽然大纲里同名也只显示一次,但 id 必须唯一)。
// CodeViewer 渲染时会从 tokens 列表里扫描 heading,生成大纲树传给右侧导航。
const _defaultHeadingOpen = md.renderer.rules.heading_open
  || function (tokens, idx, options, env, self) { return self.renderToken(tokens, idx, options) }
md.renderer.rules.heading_open = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  // tag 形如 'h1' / 'h2' ...
  const tag = token.tag
  if (tag && /^h[1-6]$/.test(tag)) {
    // 下一个 token 通常是 inline,包含标题文本。优先用 children 拼出纯文本
    // (去掉 * _ ` [ 等 markdown 标记),不要直接用 content(content 是源文本,
    // 会带 "*italic*" 这种残留,影响 slug 美观)。
    const next = tokens[idx + 1]
    let text = ''
    if (next && next.type === 'inline' && next.children && Array.isArray(next.children)) {
      text = next.children
        .map((c) => (c && typeof c.content === 'string') ? c.content : '')
        .join('')
    } else if (next && next.type === 'inline' && typeof next.content === 'string') {
      text = next.content
    } else if (next && Array.isArray(next.content)) {
      text = next.content.join('')
    } else if (next && typeof next.content === 'string') {
      text = next.content
    }
    const slug = slugifyHeading(text)
    if (slug) {
      // 用 env._headingIdCounts 跟踪同名标题,自动追加 -1 / -2
      env._headingIdCounts = env._headingIdCounts || {}
      const base = `md-h-${slug}`
      const cnt = (env._headingIdCounts[base] || 0) + 1
      env._headingIdCounts[base] = cnt
      const id = cnt === 1 ? base : `${base}-${cnt}`
      token.attrSet('id', id)
    }
  }
  return _defaultHeadingOpen(tokens, idx, options, env, self)
}

// 简单的 slug 化:小写 + 去标点 + 空格转 -;中文等 CJK 字符直接保留
// (浏览器的 scrollIntoView 支持任意 id 字符串,无需 ASCII-only)。
function slugifyHeading(text) {
  if (!text) return ''
  return String(text)
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '-')
    // 移除 markdown-it inline 残存的标记符号(如 * / _ / ` / [ / ])
    .replace(/[*_`~<>\[\]()#]/g, '')
    // 截断过长 slug,避免 id 过长
    .slice(0, 80)
    .replace(/^-+|-+$/g, '')
}

// 2026-07-04 改:外链统一走 platform.openExternal,不走 webview 自带 target=_blank。
// 重写 link_open rule,把链接改写成 class="md-external-link" data-url="<href>"(不带 target),
// 由容器上的 @click="handleExternalClick" 拦截,统一调 openExternal。
// 这样桌面端 webview 不会在内部打开外链,Web 端 / 桌面端行为一致。
const _defaultLinkOpen = md.renderer.rules.link_open
  || function (tokens, idx, options, env, self) { return self.renderToken(tokens, idx, options) }
md.renderer.rules.link_open = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  const hrefIdx = token.attrIndex('href')
  const href = hrefIdx >= 0 ? token.attrs[hrefIdx][1] : ''
  // 锚点(以 # 开头)保留默认行为,不强制外链
  if (href && !href.startsWith('#')) {
    token.attrSet('class', 'md-external-link')
    token.attrSet('data-url', href)
    // 显式把 target="_blank" 去掉
    token.attrSet('target', '_self')
  }
  return _defaultLinkOpen(tokens, idx, options, env, self)
}

/**
 * 把 markdown 字符串渲染成 HTML,用于详情预览区。
 * @param {string} src markdown 源码
 * @returns {string} HTML 字符串
 */
export function renderMarkdownView(src) {
  if (!src) return ''
  return md.render(src)
}

// 2026-07-10 增:从 markdown 源码提取大纲(标题列表),供右侧大纲导航使用。
// 跟 renderMarkdownView 共享 heading_open 重写,id 生成规则一致
// (同名标题追加 -1 / -2),保证大纲 id 跟渲染后 DOM id 严格对应。
// 走 md.parse 拿 tokens(不 render 拿 html),用同一个 slugify 跟
// 计数规则自己算 id,避免 render 一次 html 浪费 CPU。
export function extractHeadings(src) {
  if (!src) return []
  const env = {}
  const tokens = md.parse(src, env)
  const idCounts = {}
  const out = []
  for (let i = 0; i < tokens.length; i++) {
    const t = tokens[i]
    if (t.type !== 'heading_open') continue
    const tag = t.tag
    if (!tag || !/^h[1-6]$/.test(tag)) continue
    const level = Number(tag.slice(1))
    // 同一段 heading_open + inline + heading_close 顺序,tokens 平铺
    const inline = tokens[i + 1]
    let text = ''
    if (inline && inline.type === 'inline') {
      // 优先用 children(解析后的 token 列表,content 是纯文本不含 * _ 等标记)
      if (inline.children && Array.isArray(inline.children)) {
        text = inline.children
          .map((c) => (c && typeof c.content === 'string') ? c.content : '')
          .join('')
      } else if (typeof inline.content === 'string') {
        text = inline.content
      } else if (Array.isArray(inline.content)) {
        text = inline.content.join('')
      }
    }
    const slug = slugifyHeading(text)
    if (!slug) continue
    const base = `md-h-${slug}`
    const cnt = (idCounts[base] || 0) + 1
    idCounts[base] = cnt
    const id = cnt === 1 ? base : `${base}-${cnt}`
    out.push({ level, text: text.trim(), id })
  }
  return out
}