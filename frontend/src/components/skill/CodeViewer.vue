<script setup>
// CodeViewer - 技能包内单文件预览/编辑器
//
// 三种渲染分支:
//   1. Markdown(.md / .markdown)→ renderMarkdownView 渲染(只读预览)
//   2. 纯文本 / 代码(.py / .js / .json / ...)→ Monaco 只读(Commit 3) / 可编辑(Commit 4)
//   3. 二进制(.png / .jpg / .pdf / .zip / ...)→ 兜底"不支持预览"
//
// Commit 3 实现:Monaco 只读预览(所有文本文件走 Monaco,theme 跟随站点)。
// 不实现编辑(Commit 4)、"在文件夹打开"按钮(Commit 5)。
//
// 2026-07-04 增:首页技能文件浏览器(Commit 3)。

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import IconPark from '@/components/IconPark.vue'
import { renderMarkdownView } from '@/core/utils/markdown_view.js'
import { handleExternalClick } from '@/core/utils/external_link.js'
import { loadMonaco } from '@/core/composables/useMonaco.js'

const props = defineProps({
  // 文件相对路径
  path: { type: String, default: '' },
  // 文件内容
  content: { type: String, default: '' },
})

const emit = defineEmits([])

// 文件后缀
const ext = computed(() => {
  if (!props.path) return ''
  const idx = props.path.lastIndexOf('.')
  return idx >= 0 ? props.path.slice(idx + 1).toLowerCase() : ''
})

const isMarkdown = computed(() => ['md', 'markdown'].includes(ext.value))

// 二进制扩展名(图片 / pdf / 压缩包)
const BINARY_EXTS = [
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico',
  'pdf',
  'zip', 'tar', 'gz', 'tgz', '7z', 'rar',
]
const isBinary = computed(() => BINARY_EXTS.includes(ext.value))

// 大文件阈值(Monaco 加载保护)
const LARGE_FILE_BYTES = 1024 * 1024
const isLarge = computed(() => (props.content || '').length > LARGE_FILE_BYTES)

const fileName = computed(() => {
  if (!props.path) return ''
  return props.path.slice(props.path.lastIndexOf('/') + 1)
})

// markdown 渲染
const renderedMd = computed(() => isMarkdown.value ? renderMarkdownView(props.content || '') : '')

// markdown 容器点击委托
function onMdClick(e) {
  handleExternalClick(e)
}

// ====== Monaco 集成 ======
const monacoContainer = ref(null)
const monacoLoading = ref(false)
let editor = null
let model = null
let suppressEmit = false

// 文件后缀 → monaco language id
const EXT_TO_LANG = {
  py: 'python',
  js: 'javascript', jsx: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', tsx: 'typescript',
  json: 'json',
  yaml: 'yaml', yml: 'yaml',
  sh: 'shell', bash: 'shell', zsh: 'shell',
  go: 'go', rs: 'rust',
  html: 'html', htm: 'html',
  css: 'css', scss: 'css', less: 'css',
  sql: 'sql',
  toml: 'ini', ini: 'ini', cfg: 'ini',
  xml: 'xml', svg: 'xml',
  java: 'java', kt: 'kotlin', scala: 'scala',
  rb: 'ruby', php: 'php',
  c: 'c', h: 'cpp', cpp: 'cpp', cc: 'cpp', cxx: 'cpp', hpp: 'cpp',
  md: 'markdown', markdown: 'markdown',
  txt: 'plaintext', log: 'plaintext',
  vue: 'html', svelte: 'html',
  dockerfile: 'dockerfile',
}

const language = computed(() => EXT_TO_LANG[ext.value] || 'plaintext')

// 是否应使用 Monaco(非 markdown / 非二进制 / 非过大)
const useMonaco = computed(() => !isMarkdown.value && !isBinary.value && !isLarge.value)

async function ensureMonaco() {
  if (editor) return
  monacoLoading.value = true
  try {
    const { monaco } = await loadMonaco()
    editor = monaco.editor.create(monacoContainer.value, {
      value: props.content || '',
      language: language.value,
      readOnly: true, // Commit 3 只读,Commit 4 加编辑
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
      wordWrap: 'on',
      scrollBeyondLastLine: false,
      lineNumbers: 'on',
      folding: true,
      renderWhitespace: 'selection',
      smoothScrolling: true,
      contextmenu: false, // 桌面端 webview 右键菜单体验差,禁用
    })
    model = editor.getModel()
  } finally {
    monacoLoading.value = false
  }
}

function disposeEditor() {
  if (model) { try { model.dispose() } catch (_) {} model = null }
  if (editor) { try { editor.dispose() } catch (_) {} editor = null }
}

// 监听 path / content 变化,更新 Monaco 内容
watch(
  [() => props.path, () => props.content, useMonaco],
  async () => {
    if (!useMonaco.value) {
      disposeEditor()
      return
    }
    if (!editor) {
      await ensureMonaco()
    }
    if (!editor) return
    // 切文件时,先 dispose 旧 model,创建新 model(切换 language)
    if (model) { try { model.dispose() } catch (_) {} }
    const { monaco } = await loadMonaco()
    model = monaco.editor.createModel(props.content || '', language.value)
    suppressEmit = true
    editor.setModel(model)
    suppressEmit = false
  },
  { immediate: false },
)

// 当 useMonaco 由 false 变 true 时(切到代码文件),初始化
watch(useMonaco, async (v) => {
  if (v && !editor) {
    await nextTick()
    await ensureMonaco()
  }
})

import { nextTick } from 'vue'

onBeforeUnmount(() => {
  disposeEditor()
})
</script>

<template>
  <div class="code-viewer">
    <!-- 二进制兜底 -->
    <div v-if="isBinary" class="cv-binary">
      <IconPark icon="mdi:file-document-outline" width="56" height="56" />
      <p class="cv-binary-title">{{ fileName }}</p>
      <p class="cv-binary-hint">二进制文件(.{{ ext }})不支持在线预览</p>
      <p class="cv-binary-hint">Commit 5 将支持"在文件夹打开"</p>
    </div>

    <!-- Markdown 渲染 -->
    <div
      v-else-if="isMarkdown"
      class="cv-md md-body"
      v-html="renderedMd"
      @click="onMdClick"
    />

    <!-- 大文件提示 -->
    <div v-else-if="isLarge" class="cv-large">
      <IconPark icon="mdi:file-alert-outline" width="56" height="56" />
      <p class="cv-large-title">{{ fileName }}</p>
      <p class="cv-large-hint">文件过大({{ Math.round((content || '').length / 1024) }} KB),不支持在线预览</p>
      <p class="cv-large-hint">Commit 5 将支持"在文件夹打开"</p>
    </div>

    <!-- Monaco(代码 / 纯文本) -->
    <div v-else class="cv-monaco-wrap">
      <div v-if="monacoLoading" class="cv-monaco-loading">
        <span class="spinner" />
        <span>加载编辑器…</span>
      </div>
      <div ref="monacoContainer" class="cv-monaco" />
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
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
}
.cv-monaco-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
}
.cv-monaco {
  flex: 1;
  width: 100%;
  height: 100%;
}
.cv-monaco-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-faint);
  font-size: 13px;
  background: var(--bg-card);
  z-index: 2;
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
</style>