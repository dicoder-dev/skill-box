<script setup>
// CodeViewer - 技能包内单文件预览/编辑器
//
// 三种渲染分支:
//   1. Markdown(.md / .markdown)→ renderMarkdownView 渲染(只读预览)
//   2. 纯文本 / 代码(.py / .js / .json / ...)→ 原生 textarea(可编辑)+ <pre>(只读),
//      简单语法高亮走 highlight.js(已有依赖)
//   3. 二进制(.png / .jpg / .pdf / .zip / ...)→ 兜底"不支持预览" + "在文件夹打开"
//
// 2026-07-07 大改:彻底去掉 Monaco。
// Monaco 在 wails3 dev + macOS webview 环境下持续出问题:
//   1. ?worker URL 被 Vite dev server 当 SPA fallback 返回 HTML,worker 解析失败
//   2. editor.main chunk 也偶发不稳定
//   3. SyntaxError: Unexpected token '<' + 文件区空白
// 改用 textarea + highlight.js <pre>,零外部 chunk 依赖,稳。

import { computed, nextTick, onMounted, onUpdated, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import { renderMarkdownView } from '@/core/utils/markdown_view.js'
import { handleExternalClick } from '@/core/utils/external_link.js'
import { platform } from '@/platform'
import { useToastStore } from '@/core/store/toast'
// highlight.js 用于代码高亮(只读视图),跟 markdown_view.js 共享 CSS。
import hljs from 'highlight.js/lib/common'

const { t } = useI18n()
const toast = useToastStore()

const props = defineProps({
  path: { type: String, default: '' },
  content: { type: String, default: '' },
  mode: { type: String, default: 'view' },
  skillRelPath: { type: String, default: '' },
  storeRoot: { type: String, default: '' },
})

const emit = defineEmits(['update:content', 'dirty-change', 'update:mode'])

const editable = computed(() => props.mode === 'edit')

const ext = computed(() => {
  if (!props.path) return ''
  const idx = props.path.lastIndexOf('.')
  return idx >= 0 ? props.path.slice(idx + 1).toLowerCase() : ''
})

const isMarkdown = computed(() => ['md', 'markdown'].includes(ext.value))

const BINARY_EXTS = [
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico',
  'pdf',
  'zip', 'tar', 'gz', 'tgz', '7z', 'rar',
]
const isBinary = computed(() => BINARY_EXTS.includes(ext.value))

const LARGE_FILE_BYTES = 1024 * 1024
const isLarge = computed(() => (props.content || '').length > LARGE_FILE_BYTES)

const fileName = computed(() => {
  if (!props.path) return ''
  return props.path.slice(props.path.lastIndexOf('/') + 1)
})

const renderedMd = computed(() => isMarkdown.value ? renderMarkdownView(props.content || '') : '')

function onMdClick(e) {
  handleExternalClick(e)
}

// 文件后缀 → highlight.js language id(常见覆盖,找不到就 plaintext)
const EXT_TO_HLJS = {
  py: 'python',
  js: 'javascript', jsx: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', tsx: 'typescript',
  json: 'json',
  yaml: 'yaml', yml: 'yaml',
  sh: 'bash', bash: 'bash', zsh: 'bash',
  go: 'go', rs: 'rust',
  html: 'xml', htm: 'xml',
  css: 'css', scss: 'scss', less: 'less',
  sql: 'sql',
  toml: 'ini', ini: 'ini', cfg: 'ini',
  xml: 'xml',
  java: 'java', kt: 'kotlin',
  rb: 'ruby', php: 'php',
  c: 'c', h: 'cpp', cpp: 'cpp', cc: 'cpp', cxx: 'cpp', hpp: 'cpp',
  md: 'markdown', markdown: 'markdown',
  txt: 'plaintext', log: 'plaintext',
  vue: 'xml', svelte: 'xml',
}
const language = computed(() => EXT_TO_HLJS[ext.value] || 'plaintext')

// 高亮后的 HTML:每次 props.path/content 变化重新计算。
// highlight.js 是同步库,直接调用即可。
const highlightedHtml = computed(() => {
  const v = props.content || ''
  try {
    if (language.value && hljs.getLanguage(language.value)) {
      const result = hljs.highlight(v, { language: language.value, ignoreIllegals: true })
      return result.value
    }
    return escapeHtml(v)
  } catch (_) {
    return escapeHtml(v)
  }
})

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// 编辑器本地缓冲(避免 typing 时把 input value 跟 props.content 不同步造成光标跳动)。
// 注意:父级 InlinePanel 也用 localFiles 缓存,这里再缓存一次纯粹为了 input 用,
// 跟父级双向同步用 watch + emit 实现。
const localText = ref(props.content || '')
// 用 watch 在 props.content 变化时同步本地(防止父级 loadCurrent 后内容没进 input)
watch(() => props.content, (v) => {
  if (v !== localText.value) localText.value = v || ''
})

function onTextareaInput(e) {
  const v = e.target.value
  localText.value = v
  emit('update:content', v)
  emit('dirty-change', v !== (props.content || ''))
}

// 在文件夹打开
async function openInFolder() {
  if (!props.storeRoot || !props.skillRelPath) {
    toast.error(t('common.openFailed', { msg: 'storeRoot 未知' }))
    return
  }
  const relPath = `${props.skillRelPath}/${props.path}`.replace(/^\/+/, '')
  const abs = `${props.storeRoot.replace(/\/+$/, '')}/${relPath}`
  try {
    const r = await platform.fs.reveal(abs)
    if (r && r.ok === false && r.fallbackUrl) {
      await platform.platform.openExternal(r.fallbackUrl)
    }
  } catch (e) {
    toast.error(t('common.openFailed', { msg: e?.message || e }))
  }
}

// 行号生成:把内容按 \n split,渲染成左侧数字列(只读视图用)。
// 性能:1MB 文件按 \n 分 ~ 数万行,Vue 渲染 <li> 会卡;所以行号限制在 1万 以内。
const lineCount = computed(() => {
  const v = props.content || ''
  if (!v) return 1
  return Math.min(v.split('\n').length, 10000)
})
const lineNumbers = computed(() => {
  const n = lineCount.value
  const arr = new Array(n)
  for (let i = 0; i < n; i++) arr[i] = i + 1
  return arr
})

// tab 缩进支持:Tab 键插入 2 空格而不是跳焦
function onTextareaKeydown(e) {
  if (e.key !== 'Tab') return
  e.preventDefault()
  const ta = e.target
  const start = ta.selectionStart
  const end = ta.selectionEnd
  const v = localText.value
  const insert = '  '
  const next = v.slice(0, start) + insert + v.slice(end)
  localText.value = next
  emit('update:content', next)
  emit('dirty-change', next !== (props.content || ''))
  nextTick(() => {
    ta.selectionStart = ta.selectionEnd = start + insert.length
  })
}
</script>

<template>
  <div class="code-viewer">
    <!-- 临时调试:看 hljs 输出 -->
    <div style="padding: 4px 8px; background: #dbeafe; color: #1e40af; font-size: 11px; font-family: monospace;">
      HLJS: lang={{ language }}, hasLang={{ !!language && !!hljs.getLanguage(language) }}, preview={{ (highlightedHtml || '').slice(0, 80) }}
    </div>
    <!-- 二进制兜底 -->
    <div v-if="isBinary" class="cv-binary">
      <IconPark icon="mdi:file-image-outline" width="56" height="56" />
      <p class="cv-binary-title">{{ fileName }}</p>
      <p class="cv-binary-hint">{{ t('skills.fileBrowser.binaryTitle') }}</p>
      <p class="cv-binary-hint">{{ t('skills.fileBrowser.binaryHint', { ext }) }}</p>
      <button class="cv-open-folder-btn" @click="openInFolder">
        <IconPark icon="mdi:folder-open-outline" width="14" height="14" />
        {{ t('skills.fileBrowser.openInFolder') }}
      </button>
    </div>

    <!-- Markdown:可编辑用 Tiptap,只读用 v-html -->
    <div v-else-if="isMarkdown" class="cv-md-wrap">
      <RichTextEditor
        v-if="editable"
        :model-value="content || ''"
        :placeholder="t('skills.list.bodyEmpty', '开始输入正文…')"
        :disabled="false"
        min-height="100%"
        class="cv-md-rte"
        @update:model-value="(v) => $emit('update:content', v)"
      />
      <div
        v-else
        class="cv-md md-body"
        v-html="renderedMd"
        @click="onMdClick"
      />
    </div>

    <!-- 大文件提示 -->
    <div v-else-if="isLarge" class="cv-large">
      <IconPark icon="mdi:file-alert-outline" width="56" height="56" />
      <p class="cv-large-title">{{ fileName }}</p>
      <p class="cv-large-hint">{{ t('skills.fileBrowser.largeTitle') }}</p>
      <p class="cv-large-hint">{{ t('skills.fileBrowser.largeHint', { kb: Math.round((content || '').length / 1024) }) }}</p>
      <button class="cv-open-folder-btn" @click="openInFolder">
        <IconPark icon="mdi:folder-open-outline" width="14" height="14" />
        {{ t('skills.fileBrowser.openInFolder') }}
      </button>
    </div>

    <!-- 代码/纯文本:可编辑模式用 textarea + 行号,只读模式用 <pre> + highlight.js -->
    <div v-else class="cv-text-wrap">
      <div class="cv-text-toolbar">
        <span class="cv-text-lang">{{ language }}</span>
      </div>
      <div class="cv-text-body">
        <div v-if="editable" class="cv-text-edit">
          <div class="cv-text-gutter">
            <span v-for="n in lineNumbers" :key="n" class="cv-text-line-no">{{ n }}</span>
          </div>
          <textarea
            class="cv-text-input"
            spellcheck="false"
            :value="localText"
            @input="onTextareaInput"
            @keydown="onTextareaKeydown"
          />
        </div>
        <div v-else class="cv-text-view">
          <div class="cv-text-gutter">
            <span v-for="n in lineNumbers" :key="n" class="cv-text-line-no">{{ n }}</span>
          </div>
          <pre class="cv-text-pre hljs"><code v-html="highlightedHtml" /></pre>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.code-viewer {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cv-md {
  flex: 1;
  overflow: auto;
  padding: 20px 28px;
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--text);
}
.cv-md-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cv-md-rte {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.cv-binary,
.cv-large {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-dim);
  padding: 40px 20px;
}
.cv-binary-title,
.cv-large-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text);
  margin: 4px 0 0 0;
}
.cv-binary-hint,
.cv-large-hint {
  font-size: 13px;
  color: var(--text-faint);
  margin: 0;
}
.cv-open-folder-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 6px 14px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: background 120ms ease;
}
.cv-open-folder-btn:hover {
  background: var(--bg-hover);
  border-color: var(--accent-blue);
  color: var(--accent-blue);
}

/* ===== 代码/纯文本分支 ===== */
.cv-text-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
}
.cv-text-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 12px;
  font-size: 11px;
  color: var(--text-faint);
  background: var(--bg-subtle);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
  text-transform: lowercase;
  letter-spacing: 0.04em;
}
.cv-text-lang {
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-weight: 500;
}
.cv-text-body {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

/* 行号列(共用) */
.cv-text-gutter {
  flex-shrink: 0;
  width: 48px;
  padding: 12px 8px;
  background: var(--bg-subtle);
  border-right: 1px solid var(--border);
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-faint);
  text-align: right;
  user-select: none;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.cv-text-line-no {
  display: block;
}

/* 编辑模式:textarea */
.cv-text-edit {
  flex: 1;
  display: flex;
  min-height: 0;
}
.cv-text-input {
  flex: 1;
  min-width: 0;
  padding: 12px 16px;
  border: none;
  outline: none;
  resize: none;
  background: var(--bg-card);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre;
  tab-size: 2;
}
.cv-text-input:focus {
  background: var(--bg-card);
}

/* 只读模式:<pre> + highlight.js */
.cv-text-view {
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: auto;
}
.cv-text-pre {
  flex: 1;
  margin: 0;
  padding: 12px 16px;
  background: var(--bg-card);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre;
  overflow: visible;
}

/* 2026-07-07 增:highlight.js token class 配色。
   hljs.highlight() 返回的 HTML 里有 <span class="hljs-keyword"> 等,
   这些 class 默认在 github.css(已在 SkillsView 顶部 import)里定义颜色。
   但 CodeViewer 的 .cv-text-pre { color: var(--text) } + scoped CSS 顺序
   可能导致 hljs token 颜色被覆盖。这里在 scoped 内重新定义 token 配色,
   优先级 100% 高于 hljs 全局 CSS,确保颜色生效。
   配色对齐项目 UI 调色板(主色禁用紫,token 紫仅用于变量,跟 monaco 主题一致)。 */
.cv-text-pre :deep(.hljs-keyword),
.cv-text-pre :deep(.hljs-selector-tag),
.cv-text-pre :deep(.hljs-built_in),
.cv-text-pre :deep(.hljs-name) { color: #2563eb; font-weight: 600; }
.cv-text-pre :deep(.hljs-string),
.cv-text-pre :deep(.hljs-attr),
.cv-text-pre :deep(.hljs-symbol),
.cv-text-pre :deep(.hljs-bullet),
.cv-text-pre :deep(.hljs-link) { color: #16a34a; }
.cv-text-pre :deep(.hljs-number),
.cv-text-pre :deep(.hljs-literal),
.cv-text-pre :deep(.hljs-meta-number) { color: #ea580c; }
.cv-text-pre :deep(.hljs-comment),
.cv-text-pre :deep(.hljs-quote) { color: #94a3b8; font-style: italic; }
.cv-text-pre :deep(.hljs-function),
.cv-text-pre :deep(.hljs-title),
.cv-text-pre :deep(.hljs-attribute),
.cv-text-pre :deep(.hljs-class),
.cv-text-pre :deep(.hljs-type) { color: #0891b2; }
.cv-text-pre :deep(.hljs-tag),
.cv-text-pre :deep(.hljs-meta) { color: #dc2626; }
.cv-text-pre :deep(.hljs-variable),
.cv-text-pre :deep(.hljs-template-variable),
.cv-text-pre :deep(.hljs-params) { color: #7c3aed; }
.cv-text-pre :deep(.hljs-deletion) { color: #dc2626; background: #fee2e2; }
.cv-text-pre :deep(.hljs-addition) { color: #16a34a; background: #dcfce7; }

.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

.code-viewer * {
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}
.code-viewer ::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.code-viewer ::-webkit-scrollbar-track {
  background: transparent;
}
.code-viewer ::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 999px;
  transition: background 160ms ease;
}
.code-viewer ::-webkit-scrollbar-thumb:hover {
  background: var(--text-faint);
}
.code-viewer ::-webkit-scrollbar-corner {
  background: transparent;
}
</style>