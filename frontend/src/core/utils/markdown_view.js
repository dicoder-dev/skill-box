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

import MarkdownIt from 'markdown-it'
import taskLists from 'markdown-it-task-lists'
import hljs from 'highlight.js/lib/core'

// 只注册本项目实际会出现的语言,大幅减小打包体积(common languages)
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import bash from 'highlight.js/lib/languages/bash'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import sql from 'highlight.js/lib/languages/sql'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import markdown from 'highlight.js/lib/languages/markdown'

hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('json', json)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('yml', yaml)
hljs.registerLanguage('python', python)
hljs.registerLanguage('py', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('md', markdown)

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

/**
 * 把 markdown 字符串渲染成 HTML,用于详情预览区。
 * @param {string} src markdown 源码
 * @returns {string} HTML 字符串
 */
export function renderMarkdownView(src) {
  if (!src) return ''
  return md.render(src)
}