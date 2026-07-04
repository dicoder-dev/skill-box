<script setup>
// CodeViewer - 技能包内单文件预览/编辑器
//
// 三种渲染分支:
//   1. Markdown(.md / .markdown)→ renderMarkdownView 渲染(只读预览)
//   2. 纯文本 / 代码(.py / .js / .json / ...)→ Monaco 只读 / 可编辑
//   3. 二进制(.png / .jpg / .pdf / .zip / ...)→ 兜底"不支持预览" + "在文件夹打开"
//
// 2026-07-04 增:首页技能文件浏览器。

import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
// 2026-07-04 增:SKILL.md 等 .md 文件在可编辑时用 Tiptap 所见即所得,
// 与 SkillsView 旧版"正文编辑"体验一致(粗体/斜体/标题/列表/链接工具栏)。
import RichTextEditor from '@/components/RichTextEditor.vue'
import { renderMarkdownView } from '@/core/utils/markdown_view.js'
import { handleExternalClick } from '@/core/utils/external_link.js'
import { loadMonaco, isDark as monacoIsDark } from '@/core/composables/useMonaco.js'
import { platform } from '@/platform'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const props = defineProps({
  // 文件相对路径
  path: { type: String, default: '' },
  // 文件内容
  content: { type: String, default: '' },
  // 2026-07-04 改:editable boolean 改成 mode 二态,语义更清晰
  //   'view'  - 只读渲染(markdown v-html / Monaco readOnly)
  //   'edit'  - 可编辑(markdown Tiptap / Monaco readOnly=false)
  // 默认 'view'(用户点编辑按钮后才进 'edit')
  mode: { type: String, default: 'view' },
  // 技能在 store_root 下的相对路径(用于拼绝对路径,显示"在文件夹打开")
  // 格式: <group_path>/<name>(group_path 可能为空)
  skillRelPath: { type: String, default: '' },
  // store_root 绝对路径(后端 /api/skillbox/skills/store-info 拿到)
  storeRoot: { type: String, default: '' },
})

const emit = defineEmits(['update:content', 'dirty-change', 'update:mode'])

// editable = mode === 'edit'(后续内部用这个,不改字段名影响太多)
const editable = computed(() => props.mode === 'edit')

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

// 2026-07-04 增(Commit 5):"在文件夹打开"按钮
// 拼绝对路径: storeRoot + skillRelPath + / + path
// storeRoot / skillRelPath 由父级 SkillFileDrawer 注入(后端 /api/skillbox/skills/store-info)
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
      // 2026-07-04 改:显式传 theme 字段,否则 Monaco 用默认 'vs',
      // useMonaco 里 setTheme 设的全局主题不会自动应用到这个 editor 实例。
      theme: monacoIsDark() ? 'skillbox-dark' : 'skillbox-light',
      readOnly: !editable.value,
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
      contextmenu: false,
    })
    model = editor.getModel()
    // 2026-07-04 增(Commit 4):内容变化 → emit update:content + dirty-change
    if (props.editable) {
      model.onDidChangeContent(() => {
        if (suppressEmit) return
        const v = model.getValue()
        emit('update:content', v)
        emit('dirty-change', v !== (props.content || ''))
      })
    }
  } finally {
    monacoLoading.value = false
  }
}

function disposeEditor() {
  if (model) { try { model.dispose() } catch (_) {} model = null }
  if (editor) { try { editor.dispose() } catch (_) {} editor = null }
}

// 监听 path / content 变化,更新 Monaco 内容
// 2026-07-04 修(Commit 6 - 修复空白 bug):
//   - 旧版 useMonaco 直接作为数组元素传,Vue 把 ref 当成 reactive 跟踪的源,可能不会立即触发
//   - 旧版 immediate: false → 首次挂载不调 ensureMonaco,容器空
//   - 修复:统一用 getter 形式,immediate: true + nextTick 等容器就绪
watch(
  [() => props.path, () => props.content, () => useMonaco.value],
  async () => {
    if (!useMonaco.value) {
      disposeEditor()
      return
    }
    // 等容器 ref 挂到 DOM 上
    if (!monacoContainer.value) {
      await nextTick()
    }
    if (!monacoContainer.value) return
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
    // 切文件后清除 dirty 状态
    emit('dirty-change', false)
  },
  { immediate: true },
)

// 监听 mode 变化(Monaco 实例化后切换 readOnly)
watch(() => props.mode, (m) => {
  if (editor) editor.updateOptions({ readOnly: m !== 'edit' })
})

onBeforeUnmount(() => {
  disposeEditor()
})
</script>

<template>
  <div class="code-viewer">
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

    <!-- Markdown 渲染:可编辑时用 Tiptap 所见即所得,只读时用 v-html -->
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
  font-size: 12.5px;
  line-height: 1.65;
  color: var(--text);
}
/* 2026-07-04 增:markdown 编辑态容器,让 RichTextEditor 自适应填满 */
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

/* 2026-07-04 改:细化预览区滚动条(默认 14px 太粗太黑,改 6px 细款 + 浅灰)。
   范围:.code-viewer 内一切可滚动节点(包含 Monaco 内部 .monaco-scrollable-element)。
   桌面端 webview 走 webkit 内核,Web 端 Chrome/Safari 走 webkit,
   Firefox 走 scrollbar-width 兜底。 */
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