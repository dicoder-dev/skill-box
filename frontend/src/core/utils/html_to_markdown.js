// core/utils/html_to_markdown.js
//
// 把 Tiptap 输出的 HTML 转回 markdown 字符串。
// 与 core/utils/markdown.js(renderMarkdown) 互为反向。
//
// 设计原则:
//   1. 仅依赖浏览器自带 DOMParser(运行在浏览器,不要 Node 环境)
//   2. 覆盖本项目实际需要的标签,够用即可
//   3. 行内格式顺序: text → a/strong/em/code/s 等包装
//
// 已知限制:
//   - 嵌套 ul/ol 会被拍扁成每层独立(自研 markdown 渲染器也只支持单层缩进)
//   - 表格不支持(Tiptap starter-kit 不带 Table 扩展)

// 浏览器环境全局有 Node 常量;为兼容测试桩(Node.js 跑 linkedom 时没挂 Node),
// 这里用数字字面量。Node.TEXT_NODE === 3 是 W3C DOM 标准。
const TEXT_NODE = 3
const ELEMENT_NODE = 1

/**
 * 行内序列化:把一个 DOM 子树转成单行 markdown(不含块级换行)。
 * 用于 <p> / <li> / <h1-6> / <blockquote> 等容器内。
 */
function serializeInline(node) {
  if (!node) return ''
  if (node.nodeType === TEXT_NODE) {
    return node.nodeValue || ''
  }
  if (node.nodeType !== ELEMENT_NODE) return ''

  const tag = node.tagName

  // 图片(虽然是块级元素,但单行表达):不递归子树
  if (tag === 'IMG') {
    const src = node.getAttribute('src') || ''
    const alt = node.getAttribute('alt') || ''
    if (!src) return ''
    return `![${alt}](${src})`
  }

  // 软换行 — 自研 markdown 渲染器不渲染 <br>,这里用空格
  if (tag === 'BR') {
    return '  \n'
  }

  // 拼接所有子节点
  let inner = ''
  for (const child of node.childNodes) {
    inner += serializeInline(child)
  }

  switch (tag) {
    case 'STRONG':
    case 'B':
      return inner ? `**${inner}**` : ''
    case 'EM':
    case 'I':
      return inner ? `*${inner}*` : ''
    case 'S':
    case 'DEL':
    case 'STRIKE':
      return inner ? `~~${inner}~~` : ''
    case 'CODE': {
      // 行内 code:内含反引号时用更长的反引号包裹
      if (!inner) return ''
      const ticks = '`'.repeat(Math.max(1, (inner.match(/`/g) || []).length + 1))
      const needsPad = /^\s|\s$/.test(inner)
      return ticks + (needsPad ? ' ' : '') + inner + (needsPad ? ' ' : '') + ticks
    }
    case 'A': {
      const href = node.getAttribute('href') || ''
      if (!href || !inner) return inner
      return `[${inner}](${href})`
    }
    default:
      // <span>、<u>、未知行内标签 — 当透明
      return inner
  }
}

/**
 * 块级序列化:把一个 DOM 子树转成多行 markdown。
 * 用于 Tiptap 输出的根级 <p>/<h1>/<ul>/<pre> 等。
 */
function serializeBlock(node) {
  if (!node) return ''
  if (node.nodeType === TEXT_NODE) {
    return (node.nodeValue || '').trim()
  }
  if (node.nodeType !== ELEMENT_NODE) return ''

  const tag = node.tagName

  // 水平线
  if (tag === 'HR') return '\n\n---\n\n'

  // 图片(单独成行)
  if (tag === 'IMG') {
    const src = node.getAttribute('src') || ''
    const alt = node.getAttribute('alt') || ''
    if (!src) return ''
    return `\n\n![${alt}](${src})\n\n`
  }

  if (/^H[1-6]$/.test(tag)) {
    const level = Number(tag[1])
    const inner = serializeInline(node).trim()
    return inner ? `${'#'.repeat(level)} ${inner}\n\n` : '\n'
  }

  if (tag === 'P') {
    const inner = serializeInline(node).trim()
    return inner ? `${inner}\n\n` : '\n'
  }

  if (tag === 'BLOCKQUOTE') {
    // 递归子节点,把它们当作块序列,再给每行加 '> ' 前缀
    // 引用内:多行紧凑排列(\n 而非 \n\n)
    let body = ''
    for (const child of node.childNodes) {
      body += serializeBlock(child)
    }
    // 把空行压成 \n,去首尾空行
    body = body.replace(/\n{2,}/g, '\n').replace(/^\n+|\n+$/g, '')
    if (!body) return '\n'
    return body
      .split('\n')
      .map((l) => `> ${l}`)
      .join('\n') + '\n\n'
  }

  if (tag === 'PRE') {
    // 优先从 <code> 子节点取 textContent(避免 inline 序列化破坏空白)
    const codeEl = node.querySelector('code')
    const code = (codeEl ? codeEl.textContent : node.textContent) || ''
    return '```\n' + code.replace(/\n+$/, '') + '\n```\n\n'
  }

  if (tag === 'UL' || tag === 'OL') {
    const ordered = tag === 'OL'
    let out = ''
    let idx = 1
    for (const child of node.children) {
      if (child.tagName !== 'LI') continue
      const liContent = serializeInline(child).trim()
      const prefix = ordered ? `${idx}. ` : '- '
      out += `${prefix}${liContent}\n`
      idx++
    }
    return out + '\n'
  }

  // <div> / <section> / <article> / 未知块容器 — 透明,递归子节点
  let out = ''
  for (const child of node.childNodes) {
    out += serializeBlock(child)
  }
  return out
}

/**
 * 把 Tiptap 输出的 HTML 转成 markdown 字符串。
 * 输入:可能是 "<p>xxx</p><p>yyy</p>" 这种多块结构。
 */
export function htmlToMarkdown(html) {
  if (!html) return ''
  // Tiptap 有时给空编辑区返回 "<p></p>",trim 后是空字符串
  if (html.trim() === '<p></p>') return ''

  const wrapped = `<div id="__rte_root__">${html}</div>`
  const doc = new DOMParser().parseFromString(wrapped, 'text/html')
  const root = doc.getElementById('__rte_root__')
  if (!root) return ''

  let out = ''
  for (const child of root.childNodes) {
    out += serializeBlock(child)
  }

  return out
    .replace(/\n{3,}/g, '\n\n')
    .replace(/^\s+|\s+$/g, '')
}
