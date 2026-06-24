// core/utils/markdown.js - 极简 Markdown 渲染器(纯前端,不引第三方库)
//
// 设计目标:只渲染 SKILL.md 的 body,够用即可:
//   - 标题:# / ## / ### / ####
//   - 段落:连续非空行视为一个段落
//   - 强调:**bold** / *italic* / `code`
//   - 链接:[text](url)
//   - 列表:- / 1. (单层缩进)
//   - 代码块:```lang ... ```
//   - 引用:> quote
//   - 分割线:---
//   - 表格:| col | col | (含表头分隔行)
//
// 安全:对所有输出做 HTML escape,仅在白名单标签(<br/><hr/><b>...)上开禁;
// 内联 HTML 全部 escape,这是 SKILL.md 的合理做法(我们控制内容来源但仍按安全写)。
//
// 使用:
//   import { renderMarkdown } from '@/core/utils/markdown'
//   const html = renderMarkdown(source)

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function escapeAttr(s) {
  return escapeHtml(s)
}

// 行内解析(在 escape 后的字符串上工作):
//   - 链接: [text](url)    允许 http/https/mailto/file
//   - 代码: `code`
//   - 强调: **bold** / *italic*
//   - 跳过转义反斜杠
function renderInline(text) {
  // 1) 行内代码(优先,避免内部被进一步解析)
  const codeStash = []
  text = text.replace(/`([^`]+)`/g, (_, code) => {
    const i = codeStash.length
    codeStash.push(`<code>${escapeHtml(code)}</code>`)
    return `\u0000CODE${i}\u0000`
  })

  // 2) 链接 [text](url) - 只对 []() 这层 escape 后的字符处理
  text = text.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, url) => {
    const safeUrl = /^(https?:|mailto:|file:|\/|#)/i.test(url) ? url : '#'
    return `<a href="${escapeAttr(safeUrl)}" target="_blank" rel="noopener noreferrer">${label}</a>`
  })

  // 3) 粗体 / 斜体
  text = text.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  text = text.replace(/(^|[^*])\*([^*]+)\*/g, '$1<em>$2</em>')

  // 4) 还原 code stash
  text = text.replace(/\u0000CODE(\d+)\u0000/g, (_, i) => codeStash[Number(i)])

  return text
}

function renderTableRow(cells, isHeader) {
  const tag = isHeader ? 'th' : 'td'
  const html = cells.map((c) => `<${tag}>${renderInline(c.trim())}</${tag}>`).join('')
  return `<tr>${html}</tr>`
}

export function renderMarkdown(src) {
  if (!src) return ''
  const lines = String(src).replace(/\r\n?/g, '\n').split('\n')
  const out = []
  let i = 0
  let inList = null // 'ul' | 'ol' | null
  let inCode = false
  let codeBuf = []
  let codeLang = ''

  function closeList() {
    if (inList) { out.push(`</${inList}>`); inList = null }
  }

  while (i < lines.length) {
    const rawLine = lines[i]

    // 代码块 ```...```
    const fence = rawLine.match(/^```(\s*[\w+-]*)?\s*$/)
    if (fence) {
      if (!inCode) {
        closeList()
        inCode = true
        codeBuf = []
        codeLang = (fence[1] || '').trim()
      } else {
        const lang = codeLang ? ` data-lang="${escapeAttr(codeLang)}"` : ''
        out.push(`<pre${lang}><code>${escapeHtml(codeBuf.join('\n'))}</code></pre>`)
        inCode = false
        codeBuf = []
        codeLang = ''
      }
      i++
      continue
    }
    if (inCode) {
      codeBuf.push(rawLine)
      i++
      continue
    }

    const line = rawLine

    // 标题
    const h = line.match(/^(#{1,6})\s+(.*)$/)
    if (h) {
      closeList()
      const level = h[1].length
      out.push(`<h${level}>${renderInline(escapeHtml(h[2].trim()))}</h${level}>`)
      i++
      continue
    }

    // 分割线
    if (/^---+\s*$/.test(line) || /^\*\*\*+\s*$/.test(line)) {
      closeList()
      out.push('<hr/>')
      i++
      continue
    }

    // 引用
    if (/^>\s?/.test(line)) {
      closeList()
      const buf = []
      while (i < lines.length && /^>\s?/.test(lines[i])) {
        buf.push(lines[i].replace(/^>\s?/, ''))
        i++
      }
      out.push(`<blockquote>${renderInline(escapeHtml(buf.join(' ')))}</blockquote>`)
      continue
    }

    // 列表
    const ul = line.match(/^[-*+]\s+(.*)$/)
    const ol = line.match(/^\d+\.\s+(.*)$/)
    if (ul || ol) {
      const kind = ul ? 'ul' : 'ol'
      if (inList && inList !== kind) closeList()
      if (!inList) { out.push(`<${kind}>`); inList = kind }
      out.push(`<li>${renderInline(escapeHtml((ul ? ul[1] : ol[1]).trim()))}</li>`)
      i++
      continue
    }
    if (inList && line.trim() === '') {
      // 空行可能延续列表(下一行还是 - / 1.),先看一行
      if (i + 1 < lines.length && (/^[-*+]\s+/.test(lines[i + 1]) || /^\d+\.\s+/.test(lines[i + 1]))) {
        i++
        continue
      }
      closeList()
      i++
      continue
    }

    // 表格:以 | 开始且行尾也有 |
    if (/^\s*\|.*\|\s*$/.test(line)) {
      closeList()
      const headerCells = line.trim().replace(/^\||\|$/g, '').split('|')
      // 第二行必须是 ---|---|---
      if (i + 1 < lines.length && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
        i += 2
        const rows = []
        while (i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i])) {
          const cells = lines[i].trim().replace(/^\||\|$/g, '').split('|')
          rows.push(cells)
          i++
        }
        out.push('<table>')
        out.push('<thead>')
        out.push(renderTableRow(headerCells, true))
        out.push('</thead>')
        if (rows.length) {
          out.push('<tbody>')
          for (const r of rows) out.push(renderTableRow(r, false))
          out.push('</tbody>')
        }
        out.push('</table>')
        continue
      }
    }

    // 空行:段落结束
    if (line.trim() === '') {
      closeList()
      i++
      continue
    }

    // 段落:连续非空非列表非标题行
    closeList()
    const buf = [line]
    i++
    while (i < lines.length) {
      const nx = lines[i]
      if (nx.trim() === '') break
      if (/^#{1,6}\s+/.test(nx)) break
      if (/^[-*+]\s+/.test(nx)) break
      if (/^\d+\.\s+/.test(nx)) break
      if (/^```/.test(nx)) break
      if (/^>\s?/.test(nx)) break
      if (/^\s*\|.*\|\s*$/.test(nx)) break
      buf.push(nx)
      i++
    }
    out.push(`<p>${renderInline(escapeHtml(buf.join(' ')))}</p>`)
  }

  if (inCode) {
    // 未关闭的代码块也渲染出来
    out.push(`<pre><code>${escapeHtml(codeBuf.join('\n'))}</code></pre>`)
  }
  closeList()
  return out.join('\n')
}
