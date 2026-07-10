<script setup>
// CodeViewer - 技能包内单文件预览/编辑器
//
// 渲染分支:
//   1. Markdown(.md / .markdown)→ Tiptap 编辑 / v-html 只读
//   2. 纯文本 / 代码(.py / .js / .json / ...):
//      - 只读模式: highlight.js <pre> 高亮
//      - 编辑模式: Monaco Editor(完整语法高亮 + 自动补全 + 括号匹配 + 缩进参考线)
//   3. Office 文档(.docx / .pdf / .xlsx / .xls / .pptx)→ @vue-office 在线预览
//   4. 二进制(.png / .jpg / .zip / ...)→ 兜底"不支持预览" + "在文件夹打开"
//   5. 大文件(> 1MB)→ 兜底 + "在文件夹打开"
//
// 2026-07-08 改:编辑模式从 textarea 重新换回 Monaco。useMonaco.js 已经修了
// wails3 webview 下 worker 被 SPA fallback 截胡的问题(动态 import + MonacoEnvironment
// 内联 Blob worker 指向 jsdelivr workerMain.js),这次直接复用。

import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import OfficeViewer from '@/components/skill/OfficeViewer.vue'
import CsvViewer from '@/components/skill/CsvViewer.vue'
import { renderMarkdownView, extractHeadings } from '@/core/utils/markdown_view.js'
import { handleExternalClick } from '@/core/utils/external_link.js'
import { platform } from '@/platform'
import { useToastStore } from '@/core/store/toast'
import { loadMonaco, isDark } from '@/core/composables/useMonaco'
// highlight.js 用于只读视图高亮(全量包, 384 语言)。
import hljs from 'highlight.js'

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

// 2026-07-08 增:CSV 文件走 CsvViewer 表格化预览(只读),编辑模式仍走 Monaco。
const isCsv = computed(() => ext.value === 'csv')

// 2026-07-08 增:office 文档类型(.docx / .pdf / .xlsx / .xls / .pptx)走 vue-office 在线预览,
// 不再归到二进制兜底。OFFICE_EXTS 是可预览类型,BINARY_EXTS 是真不能预览(图片/压缩包)。
const OFFICE_EXTS = ['docx', 'pdf', 'xlsx', 'xls', 'pptx']
// ext → OfficeViewer 子组件 kind(因为 docx/excel/pdf/pptx 是 vue-office 4 个不同组件入口)
const OFFICE_KIND_BY_EXT = {
  docx: 'docx',
  pdf: 'pdf',
  xlsx: 'excel', xls: 'excel',
  pptx: 'pptx',
}
const officeKind = computed(() => OFFICE_KIND_BY_EXT[ext.value] || '')
const isOffice = computed(() => OFFICE_EXTS.includes(ext.value))

const BINARY_EXTS = [
  // 2026-07-08 改:去掉 pdf(走 office 预览),保留图片 + 压缩包。
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico',
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

// 2026-07-10 增:md 文件大纲(只读视图用)。从 props.content 抽取 {level, text, id} 列表。
// 只在 view 模式 + md 文件时使用,edit 模式仍由 Tiptap 自己管 outline。
// 大纲树做"按最小 level 提一档":比如文件只有 h3 / h4,展示时按 h1 / h2 缩进,
// 避免出现"全是缩进很深的小标题"。minLevel 减 1 当作顶层。
const mdHeadings = computed(() => {
  if (!isMarkdown.value || editable.value) return []
  return extractHeadings(props.content || '')
})
const minHeadingLevel = computed(() => {
  if (!mdHeadings.value.length) return 1
  return Math.min(...mdHeadings.value.map((h) => h.level))
})

function onMdClick(e) {
  handleExternalClick(e)
}

// 大纲点击 → 滚动到对应标题。markdown-it 渲染时已经给每个 h1-h6 加了
// id="md-h-{slug}",这里直接 document.getElementById 找节点再 scrollIntoView。
// 标题节点在 .cv-md 滚动容器内,scrollIntoView 默认会找最近的滚动祖先
// (行为: smooth) 跟用户预期一致。
function scrollToHeading(id) {
  if (!id) return
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  // 视觉强调:短暂给目标标题加 active 类(用 CSS transition)
  el.classList.add('cv-md-heading-active')
  setTimeout(() => el.classList.remove('cv-md-heading-active'), 1200)
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
  // 2026-07-08 增:CSV 在 monaco 编辑模式用内置 csv language id(highlight.js 无 csv
  // language,会让 escapeHtml 走 fallback,monaco 自带 csv 至少给分隔符高亮)。
  csv: 'plaintext',
}
const language = computed(() => EXT_TO_HLJS[ext.value] || 'plaintext')

// 2026-07-08 增:Monaco 编辑器用的 language id 映射。
// 大部分跟 hljs id 一致(Monaco 也用 javascript / typescript / python / ...)。
// Monaco 没内置的(vue / svelte)fallback 到 html 拿到基础 HTML/CSS/JS 高亮。
const EXT_TO_MONACO = {
  ...EXT_TO_HLJS,
  vue: 'html', svelte: 'html',
}
const monacoLang = computed(() => EXT_TO_MONACO[ext.value] || 'plaintext')

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

// ====== Monaco 编辑器实例 ======
//
// 2026-07-08 增:编辑模式挂载 Monaco,提供语法高亮 + 自动补全 + 括号匹配。
// 实例 / model 用 let 存(不响应式),防止 Vue 包装成本;只有 editorContainer 是 ref
// 让模板挂载。suppressNextChange 是防回环的关键:
//   props.content 变 → setValue → onDidChangeContent 触发 → 用户输入回环
//   用一个 flag 让 setValue 之后第一次 onDidChangeContent 直接吞掉。

// 2026-07-08 改:localText ref 从 Monaco 块下方移到这里(在 mountMonaco 之前声明),
// 因为 mountMonaco 内 createModel 要读 localText.value。
// 同时 props.content watch 同步扩展到 Monaco model:setValue 触发 onDidChangeContent
// 时用 suppressNextChange 吞掉,避免 emit 回环。
const localText = ref(props.content || '')
watch(() => props.content, (v) => {
  const next = v || ''
  if (next !== localText.value) localText.value = next
  if (monacoModel && monacoModel.getValue() !== next) {
    suppressNextChange = true
    monacoModel.setValue(next)
  }
})

const editorContainer = ref(null)
let monacoEditor = null
let monacoModel = null
let monacoRef = null   // { monaco } from loadMonaco(),供 watch 里 setModelLanguage 用
let suppressNextChange = false

async function mountMonaco() {
  if (!editorContainer.value || monacoEditor) return
  const loaded = await loadMonaco()
  monacoRef = loaded
  const monaco = loaded.monaco
  monacoModel = monaco.editor.createModel(localText.value || '', monacoLang.value)
  monacoEditor = monaco.editor.create(editorContainer.value, {
    model: monacoModel,
    // 2026-07-08 改:强制用 skillbox-dark 主题,跟 .cv-text-wrap 黑底统一;
    // 否则站点是浅色主题时 Monaco 走 skillbox-light(背景 #fafafa 白色),
    // 在黑底容器内"白底编辑器"显得很突兀。token 配色两边都用亮色版
    // (蓝/绿/橙/青/红/紫)都好看,无视觉冲突。
    theme: 'skillbox-dark',
    automaticLayout: true,
    // 字体 / 行高跟只读视图 hljs <pre> 完全一致(13px / 1.6 → 21px),
    // 切模式视觉不跳。
    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
    fontSize: 13,
    lineHeight: 21,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    tabSize: 2,
    insertSpaces: true,
    wordWrap: 'off',
    renderLineHighlight: 'line',
    lineNumbers: 'on',                 // Monaco 自带行号,删除 CodeViewer 自定义 gutter
    glyphMargin: false,
    folding: true,
    quickSuggestions: { other: true, comments: false, strings: true },
    suggestOnTriggerCharacters: true,
    bracketPairColorization: { enabled: true },
  })
  monacoModel.onDidChangeContent(() => {
    if (suppressNextChange) {
      suppressNextChange = false
      return
    }
    const v = monacoModel.getValue()
    localText.value = v
    emit('update:content', v)
    emit('dirty-change', v !== (props.content || ''))
  })
}

function unmountMonaco() {
  if (monacoEditor) {
    monacoEditor.dispose()
    monacoEditor = null
  }
  if (monacoModel) {
    monacoModel.dispose()
    monacoModel = null
  }
}
onBeforeUnmount(unmountMonaco)

// 2026-07-08 增:editable 切换时挂载/卸载 Monaco。
//   - false → true:nextTick 等模板把 editorContainer 挂上再创建 editor
//   - true → false:立刻 dispose,避免来回切累积实例
watch(editable, (now) => {
  if (now) {
    nextTick(mountMonaco)
  } else {
    unmountMonaco()
  }
})

// 2026-07-08 增:file 类型变化时(用户在 InlinePanel 里切到其他文件),
// CodeViewer 被 :key 重建,本组件整体 unmount → onBeforeUnmount 触发 dispose。
// 这里是单文件内的扩展名变化(理论上不会发生,InlinePanel 切文件用 :key 重建),
// 防御性保留:同组件内 ext 变化时切换 model language。
watch(monacoLang, (lang) => {
  if (monacoRef && monacoModel) {
    monacoRef.monaco.editor.setModelLanguage(monacoModel, lang)
  }
})

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
//
// 2026-07-08 删:Monaco 自带 tabSize:2 / insertSpaces:true,不再需要手写 keydown。
// function onTextareaKeydown(e) { ... }
</script>

<template>
  <div class="code-viewer">
    <!-- 2026-07-08 增:office 文档(.docx / .pdf / .xlsx / .xls / .pptx)走 vue-office 在线预览 -->
    <OfficeViewer
      v-if="isOffice"
      :kind="officeKind"
      :content="content"
      class="cv-office"
    />

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

    <!-- Markdown:可编辑用 Tiptap,只读用 v-html。
         2026-07-10 改:只读视图额外渲染右侧大纲导航(mdHeadings 提取的 h1-h6 列表),
         点击大纲项 scrollIntoView 跳转到对应标题。布局走两列:左侧 md 内容(占满),
         右侧 220px 固定宽大纲(只在有标题时才显示),避免短 md 文件出现空 panel。
         编辑模式不显示大纲(由 Tiptap 自己管 outline,避免冲突)。 -->
    <div v-else-if="isMarkdown" class="cv-md-wrap">
      <div class="cv-md-content">
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
          class="cv-md md-body markdown-body"
          v-html="renderedMd"
          @click="onMdClick"
        />
      </div>
      <aside v-if="!editable && mdHeadings.length" class="cv-md-outline">
        <header class="cv-md-outline-header">
          <IconPark icon="mdi:format-list-bulleted" width="13" height="13" />
          <span>大纲</span>
          <span class="cv-md-outline-count">{{ mdHeadings.length }}</span>
        </header>
        <ul class="cv-md-outline-list">
          <li
            v-for="h in mdHeadings"
            :key="h.id"
            :class="['cv-md-outline-item', `cv-md-outline-l${h.level - minHeadingLevel + 1}`]"
            :data-tip="h.text"
          >
            <button
              type="button"
              class="cv-md-outline-btn"
              :title="h.text"
              @click="scrollToHeading(h.id)"
            >
              <span class="cv-md-outline-dot" />
              <span class="cv-md-outline-text">{{ h.text }}</span>
            </button>
          </li>
        </ul>
      </aside>
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

    <!-- 2026-07-08 改 v8:CSV 文件 - view 模式表格化预览,edit 模式 Monaco 编辑器。
         改 v-else-if 把 CSV 拉进 v-else 链(binary/md/large 同条链),跟前几个 v-if 互斥。
         历史 v5/v6 写法用独立 v-if="isCsv && !editable" 配 cv-text-wrap 的 !isCsv 排除,
         csv view 模式没问题,但 csv edit 模式两端都被卡 (CsvViewer 卡 view,cv-text-wrap 卡 csv),
         结果编辑区空白 —— 用户这次反馈的就是这个。改成 v-else-if 之后 csv 跟 md/binary/
         large 自然互斥;CSV 内部再用 v-if="!editable" 二分(view → CsvViewer 表格 / edit → Monaco)。
         OfficeViewer 维持独立 v-if(子组件多 kind,统一 v-else 复杂度高,先不动)。 -->
    <div v-else-if="isCsv" class="cv-csv-wrap">
      <CsvViewer
        v-if="!editable"
        :key="path + ':view'"
        :content="content"
        class="cv-csv"
      />
      <div v-else class="cv-text-edit">
        <div ref="editorContainer" class="cv-monaco-host" />
      </div>
    </div>

    <!-- 代码/纯文本:可编辑模式用 Monaco(自带高亮+补全),只读模式用 <pre> + highlight.js。
         2026-07-08 改 v6:加 !isMarkdown && !isOffice 排除条件(CSV 已挪到独立 v-else-if 分支),
         确保 md/office 文件绝不进 hljs plaintext 渲染链。用户之前的双视图 bug 根因跟 CSV
         一样,所以 9631e4c 才补上 isMarkdown。现在 CSV 走上面 v-else-if 分支,这里只需排除
         md/office。 -->
    <div v-if="!isMarkdown && !isOffice" class="cv-text-wrap">
      <div class="cv-text-toolbar">
        <span class="cv-text-lang">{{ language }}</span>
      </div>
      <div class="cv-text-body">
        <!-- 2026-07-08 改:编辑分支换成 Monaco 容器,删掉手工 gutter(行号 Monaco 自带)。
             只读分支保留手工 gutter,因为 hljs <pre> 没有自带行号。 -->
        <div v-if="editable" class="cv-text-edit">
          <div ref="editorContainer" class="cv-monaco-host" />
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
/* 2026-07-08 增:office 预览区占满 CodeViewer */
.cv-office {
  flex: 1;
  min-height: 0;
  display: flex;
}
/* 2026-07-08 增:CSV 表格化预览占满 CodeViewer */
.cv-csv {
  flex: 1;
  min-height: 0;
  display: flex;
}
.cv-md {
  flex: 1;
  overflow: auto;
  padding: 20px 28px;
  font-size: 14.5px;
  line-height: 1.7;
  color: var(--text);
}
/* 2026-07-10 改:.cv-md-wrap 改为横向 flex,内容 + 大纲两列。
   旧版 cv-md 自带 flex:1,现在 cv-md-wrap 内层套了 .cv-md-content,
   让内容继续 flex:1 占满,大纲侧固定 220px 宽度且只在只读 + 有标题时出现。 */
.cv-md-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: row;
  overflow: hidden;
}
.cv-md-content {
  flex: 1;
  min-width: 0;
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

/* 2026-07-10 增:md 大纲导航(右侧固定列)。 */
.cv-md-outline {
  flex-shrink: 0;
  width: 220px;
  border-left: 1px solid var(--border);
  background: var(--bg-subtle);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cv-md-outline-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  position: sticky;
  top: 0;
  z-index: 1;
  flex-shrink: 0;
}
.cv-md-outline-count {
  margin-left: auto;
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 999px;
}
.cv-md-outline-list {
  list-style: none;
  margin: 0;
  padding: 6px 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}
.cv-md-outline-item {
  list-style: none;
}
.cv-md-outline-btn {
  width: 100%;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 4px 12px;
  background: transparent;
  border: none;
  cursor: pointer;
  text-align: left;
  color: var(--text-dim);
  font-size: 12.5px;
  line-height: 1.5;
  border-radius: 0;
  transition: background 100ms ease, color 100ms ease;
}
.cv-md-outline-btn:hover {
  background: var(--bg-hover);
  color: var(--text);
}
.cv-md-outline-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-faint);
  margin-top: 7px;
}
/* 缩进按 level 差:minHeadingLevel → 0,每升一级 +14px */
.cv-md-outline-l1 .cv-md-outline-btn { padding-left: 12px; }
.cv-md-outline-l2 .cv-md-outline-btn { padding-left: 26px; }
.cv-md-outline-l3 .cv-md-outline-btn { padding-left: 40px; }
.cv-md-outline-l4 .cv-md-outline-btn { padding-left: 54px; }
.cv-md-outline-l5 .cv-md-outline-btn { padding-left: 68px; }
.cv-md-outline-l6 .cv-md-outline-btn { padding-left: 82px; }
.cv-md-outline-l2 .cv-md-outline-dot { width: 5px; height: 5px; margin-top: 8px; }
.cv-md-outline-l3 .cv-md-outline-dot,
.cv-md-outline-l4 .cv-md-outline-dot,
.cv-md-outline-l5 .cv-md-outline-dot,
.cv-md-outline-l6 .cv-md-outline-dot {
  width: 4px;
  height: 4px;
  margin-top: 8px;
  background: var(--border);
}
.cv-md-outline-text {
  flex: 1;
  min-width: 0;
  /* 2026-07-10 改:大纲标题单行截断 + tooltip 浮全名(用 :data-tip 模式靠 title 属性兜底,
     CSS 部分用 white-space + text-overflow 控制视觉) */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

/* 2026-07-10 增:大纲点击后,目标标题短暂高亮提示(蓝色背景渐隐) */
.cv-md :deep(.cv-md-heading-active) {
  background: linear-gradient(90deg, var(--accent-blue-bg, #eff6ff) 0%, transparent 100%);
  transition: background 1200ms ease;
  border-radius: 4px;
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
  /* 2026-07-07 改 v2:代码区背景改纯黑,跟 IDE 一致;之前的 var(--bg) 跟外面卡片同色,
     没有"代码区"的视觉区隔。黑色背景上 token 配色对比度更高。 */
  background: #0a0a0a;
}
.cv-text-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 12px;
  font-size: 11px;
  /* 2026-07-07 改 v2:工具栏背景深灰,跟代码区黑底过渡自然 */
  background: #171717;
  color: #94a3b8;
  border-bottom: 1px solid #262626;
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
  /* 2026-07-07 改 v2:行号列背景深灰,跟代码黑底形成层级 */
  background: #171717;
  border-right: 1px solid #262626;
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #64748b;
  text-align: right;
  user-select: none;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.cv-text-line-no {
  display: block;
}

/* 编辑模式:Monaco */
.cv-text-edit {
  flex: 1;
  display: flex;
  min-height: 0;
  min-width: 0;
  position: relative;
}
.cv-monaco-host {
  flex: 1;
  min-width: 0;
  width: 100%;
  height: 100%;
  position: relative;
}
/* 2026-07-08 增:Monaco 内部容器也需要 height 100%,
   否则父容器 height 没被吃掉,editor 显示在错误位置。 */
.cv-monaco-host .monaco-editor,
.cv-monaco-host .monaco-scrollable-element {
  position: absolute !important;
  inset: 0;
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
  /* 2026-07-07 改 v2:纯黑背景 + 浅灰底色 */
  background: #0a0a0a;
  color: #e2e8f0;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre;
  overflow: visible;
}

/* 2026-07-07 修:github.css(全局)在 .hljs / pre / code 上加了 background:#f0f0f0,
   跟 .cv-text-pre { background:#0a0a0a } 同优先级,但 github.css 是后加载的 → 赢,
   结果每一行代码都出现白底(其实是 pre 的 box-decoration-break: slice + 每行视觉块)。
   显式 :deep() 覆盖,清掉所有 hljs 默认背景,保留 token 颜色。
   同时 github.css 给 code 加 color:#24292e 深色,黑底上看不清,
   这里 :deep(code) 显式设 color:#e2e8f0 浅灰。 */
.cv-text-pre :deep(.hljs) { background: transparent; }
.cv-text-pre :deep(code) { background: transparent; color: #e2e8f0; display: block; }
.cv-text-pre :deep(.hljs-subst),
.cv-text-pre :deep(.hljs-section),
.cv-text-pre :deep(.hljs-emphasis) { color: inherit; background: transparent; }

/* 2026-07-07 改 v2:代码区黑底,hljs token 配色用浅色高对比版(浅色 token 在黑底上看不清)。
   .cv-text-pre :deep() scoped 优先级 > 全局 hljs github.css。
   颜色对应项目 UI 调色板:蓝/绿/橙/青/红;紫仅用于变量。 */
.cv-text-pre :deep(.hljs-keyword),
.cv-text-pre :deep(.hljs-selector-tag),
.cv-text-pre :deep(.hljs-built_in),
.cv-text-pre :deep(.hljs-name) { color: #60a5fa; font-weight: 600; }
.cv-text-pre :deep(.hljs-string),
.cv-text-pre :deep(.hljs-attr),
.cv-text-pre :deep(.hljs-symbol),
.cv-text-pre :deep(.hljs-bullet),
.cv-text-pre :deep(.hljs-link) { color: #4ade80; }
.cv-text-pre :deep(.hljs-number),
.cv-text-pre :deep(.hljs-literal),
.cv-text-pre :deep(.hljs-meta-number) { color: #fb923c; }
.cv-text-pre :deep(.hljs-comment),
.cv-text-pre :deep(.hljs-quote) { color: #64748b; font-style: italic; }
.cv-text-pre :deep(.hljs-function),
.cv-text-pre :deep(.hljs-title),
.cv-text-pre :deep(.hljs-attribute),
.cv-text-pre :deep(.hljs-class),
.cv-text-pre :deep(.hljs-type) { color: #22d3ee; }
.cv-text-pre :deep(.hljs-tag),
.cv-text-pre :deep(.hljs-meta) { color: #f87171; }
.cv-text-pre :deep(.hljs-variable),
.cv-text-pre :deep(.hljs-template-variable),
.cv-text-pre :deep(.hljs-params) { color: #c4b5fd; }
.cv-text-pre :deep(.hljs-deletion) { color: #fca5a5; background: #450a0a; }
.cv-text-pre :deep(.hljs-addition) { color: #86efac; background: #052e16; }

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