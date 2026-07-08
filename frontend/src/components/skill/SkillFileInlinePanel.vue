<script setup>
// SkillFileInlinePanel - 首页右侧主区域:目录树 + 预览/编辑
//
// 2026-07-07 v4 重写:完全脱离 vue-i18n,template 内不调 t(),所有展示文本
// 直接用常量字符串写死。原因:vue-i18n 9 + legacy:false + useI18n 在 v-if 懒挂载
// 子组件里,t 会被 Proxy 包装,render function 抛 "t is not a function" 把整段
// component update 弄崩。其他故障(scope 区一直转圈、[i] 信息按钮首次点击
// 无效)均是同一根因的副作用。重写后组件内 0 个 t() 调用。
//
// 二次保险:
//   - onErrorCaptured 兜底 render 错,降级到安全 UI
//   - safeReload 允许用户点"重试"清错误状态
//
// 拆分:
//   - scope 区迁移到 <SkillScopePanel> 子组件,本组件只传 props
//   - Frontmatter / CodeViewer / FileTreeView 已是独立子组件,不重写

import { computed, onMounted, onUnmounted, onUpdated, reactive, ref, onErrorCaptured } from 'vue'
import { plainT, messages } from '@/core/i18n/index.js'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import SkillScopePanel from './SkillScopePanel.vue'
import { updateSkill, getStoreInfo } from '@/api/skillbox/skills'
import { useToastStore } from '@/core/store/toast'

// 2026-07-07 临时调试:桌面端 webview 缓存导致浏览器拉到旧 chunk,
// 用 console 时间戳确认这次是否拿到新版本。
// 用户在桌面端启用 devtools (wails3 dev 默认开 Cmd+Opt+I) 看 console 输出。
console.log('[SkillFileInlinePanel v6] loaded at', new Date().toISOString(), 'no-watch import, no-console')

// 2026-07-07 v4:不再尝试从 vue-i18n 拿 t,直接读 messages 对象兜底。
// 但为了避免"再抛"再次发生,这里完全不再调 plainT()。template 内所有
// 用户可见文案一律用常量字符串(下方 LABEL_* 常量)。
// 仅 messages 在 <script> 中以 import 形式留存,供未来读 key 用,
// 这里就当 unused import 处理。
void messages
void plainT

// ===== 常量文案(原 i18n key,直接写中文) =====
const LABEL_NO_FILE = '未选择文件'
const LABEL_EDIT = '编辑'
const LABEL_PREVIEW = '返回预览'
const LABEL_PICK = '请选择一个文件开始浏览'
const LABEL_DIRTY = '● 未保存'
const LABEL_DISCARD = '放弃修改'
const LABEL_SAVE = '保存'
const LABEL_SAVING = '保存中...'
const LABEL_FILES = 'files'
const LABEL_FRONTMATTER_TITLE = '查看 frontmatter'
const LABEL_RENDER_ERROR_TITLE = '技能详情加载出错'
const LABEL_RETRY = '重试'

const toast = useToastStore()

const props = defineProps({
  files: { type: Array, default: () => [] },
  skill: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['saved'])

// ====== File selection state ======
const selectedFile = ref(null)
const selectedKey = ref('')
const localFiles = reactive(new Map())
const dirtyPaths = ref(new Set())

// per-file edit mode, keyed by "<skillName>/<path>"
const editModeMap = reactive({})
function modeKey(skillName, path) {
  if (!path) return ''
  return skillName ? `${skillName}/${path}` : path
}
function getMode(skillName, path) {
  const k = modeKey(skillName, path)
  if (!k) return 'view'
  return editModeMap[k] || 'view'
}
function setMode(skillName, path, m) {
  const k = modeKey(skillName, path)
  if (!k) return
  editModeMap[k] = m
}
// 2026-07-07 增:清指定文件的编辑态,回到 view 模式。
// 用在切换 skill/file 前,保证新选中的文件默认是 view 模式,
// 而不是继承前一个 skill 的 edit 状态。
function clearMode(skillName, path) {
  const k = modeKey(skillName, path)
  if (!k) return
  delete editModeMap[k]
}

function splitSkillMd(text) {
  if (!text) return { frontmatter: '', body: text }
  const m = text.match(/^---\s*\n([\s\S]*?)\n---\s*\n?/)
  if (!m) return { frontmatter: '', body: text }
  return { frontmatter: m[0], body: text.slice(m[0].length) }
}

// 选中文件 → localFiles 填充,响应 props.files 变化
// 2026-07-07 改 v6:不依赖 vue 的 watch(esm cache 缺 watch 函数,
// webview 拿到的 chunk 里 ReferenceError: Can't find variable: watch),
// 改用 onUpdated + 手动依赖追踪 — 每次父组件 patch 后重新检查 props。
let _lastFilesRef = null
let _lastSkillName = null
function _syncSelectedFile() {
  const sk = props.skill
  const files = props.files
  const curFilesRef = files
  const curName = sk?.name
  if (curFilesRef === _lastFilesRef && curName === _lastSkillName) return
  _lastFilesRef = curFilesRef
  _lastSkillName = curName
  if (!files || !files.length) {
    selectedFile.value = null
    selectedKey.value = ''
    localFiles.clear()
    dirtyPaths.value = new Set()
    return
  }
  const prev = selectedKey.value
  const target = (prev && files.find((f) => f.path === prev))
    || files.find((f) => f.path === 'SKILL.md')
    || files[0]
  selectedFile.value = target
  selectedKey.value = target?.path || ''
}
function _syncLocalFiles() {
  const sk = props.skill
  const curFilesRef = props.files
  const curName = sk?.name
  if (curFilesRef === _lastFilesRef && curName === _lastSkillName) return
  // 跟 _syncSelectedFile 共享判断,省一次比较
  _lastFilesRef = curFilesRef
  _lastSkillName = curName
  localFiles.clear()
  for (const f of props.files || []) {
    const c = f.content || ''
    const stored = f.path === 'SKILL.md' ? splitSkillMd(c).body : c
    localFiles.set(f.path, stored)
  }
  dirtyPaths.value = new Set()
}
onUpdated(() => {
  _syncSelectedFile()
  _syncLocalFiles()
})
// 首次同步在 onMounted 里跑一次
onMounted(() => {
  _syncSelectedFile()
  _syncLocalFiles()
  fetchStoreRoot()
})

// 2026-07-07 改:切换文件前也走 dirty 检查。
// 用户改完 SKILL.md 后点其他文件(目录树里)→ 弹三选项。
// dirty 检查按"任一文件 dirty"算,不只是当前选中文件。
//
// 2026-07-07 修:file 参数从 FileTreeView emit 出来,FileTreeView.buildTree
// 输出的 file 对象只有 {name, path, size},**没有 content**。如果直接
// selectedFile.value = file → displayContent 算 selectedFile.value.content
// 是 undefined → 空白。修法:从 props.files 里 find 出含 content 的原始对象。
async function onSelectFile(file) {
  if (!file || !file.path) return
  if (file.path === selectedKey.value) return
  const verdict = await ensureCleanBeforeSwitch()
  if (verdict === 'cancel') return
  // 从 props.files 里拿完整的 {path, content} 对象(包含 content)
  const full = props.files.find((f) => f.path === file.path) || file
  selectedFile.value = full
  selectedKey.value = full.path
}

// 2026-07-07 增:切换前 dirty 检查 + 询问逻辑。
// 父组件在切换 skill / 切换文件前调 ensureCleanBeforeSwitch(),有 dirty 弹
// 三选项(保存 / 放弃 / 取消),等用户决策后再决定是否继续切换。
// 设计目标:无论保存/放弃,dirty 状态都要被清掉(用户的核心诉求),编辑态也
// 回到 view(避免切换后新文件继承 edit 模式)。
//
// 返回值约定(给父调用方):
//   'proceed'  → 允许切换(用户已保存或放弃)
//   'cancel'   → 不切换(用户取消)
//   'busy'     → 当前正在另一个保存流程中,父应该也阻断切换
const _discardResolve = ref(null)
const discardOpen = ref(false)
const discardFilePath = ref('')
const discardFileName = ref('')

async function ensureCleanBeforeSwitch() {
  // 2026-07-07 改:用最新的 dirtyPaths 计算 — computed isDirty 只反映当前选中文件,
  // 这里要"任一文件 dirty 都算脏"。
  if (!dirtyPaths.value || dirtyPaths.value.size === 0) {
    // 即使没 dirty,如果当前文件处于 edit 模式,也要清掉(切走后不该继承 edit)。
    // 这是用户反馈"打开其他 skill 默认是处于编辑状态"那个 bug 的修法。
    clearModeOnLeave()
    return 'proceed'
  }
  // 拿第一个 dirty 文件(多文件 dirty 时也只问一次,统一处理)
  const firstDirty = Array.from(dirtyPaths.value)[0]
  discardFilePath.value = firstDirty
  discardFileName.value = (firstDirty || '').split('/').pop() || firstDirty
  discardOpen.value = true
  return new Promise((resolve) => { _discardResolve.value = resolve })
}

// 切换前清当前选中文件的 edit 模式(进入 view)
function clearModeOnLeave() {
  const sk = props.skill
  const path = selectedFile.value?.path
  if (!path) return
  clearMode(sk?.name, path)
}

async function onDiscardSave() {
  discardOpen.value = false
  // 保存当前 dirty 文件(只保存第一个 dirty;实际 InlinePanel 里只支持单文件保存,
  // 这里就按现有 saveCurrent 走)
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  // 把 selectedFile 切到 dirty 文件再保存
  const target = props.files.find((f) => f.path === discardFilePath.value)
  if (target) {
    selectedFile.value = target
    selectedKey.value = target.path
  }
  try {
    await saveCurrent()
  } catch (_) { /* saveCurrent 内部已 toast */ }
  // saveCurrent 成功后 dirtyPaths 已被清;编辑态也清掉。
  clearModeOnLeave()
  r('proceed')
}

function onDiscardDrop() {
  discardOpen.value = false
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  // 放弃:直接清掉所有 dirty(同步 localFiles 到原内容),并清编辑态。
  resetAllDirty()
  clearModeOnLeave()
  r('proceed')
}

function onDiscardCancel() {
  discardOpen.value = false
  const r = _discardResolve.value
  _discardResolve.value = null
  if (!r) return
  r('cancel')
}

function resetAllDirty() {
  // 把 dirty 文件的 localFiles 同步回原 content
  for (const dirtyPath of dirtyPaths.value) {
    const f = props.files.find((x) => x.path === dirtyPath)
    if (!f) continue
    const orig = dirtyPath === 'SKILL.md' ? splitSkillMd(f.content || '').body : (f.content || '')
    localFiles.set(dirtyPath, orig)
  }
  dirtyPaths.value = new Set()
}

const currentContent = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return ''
  return localFiles.has(path) ? localFiles.get(path) : (selectedFile.value?.content || '')
})

const displayContent = computed(() => {
  if (!selectedFile.value) return ''
  if (selectedFile.value.path === 'SKILL.md') {
    return splitSkillMd(currentContent.value).body
  }
  return currentContent.value
})

const currentMode = computed(() => getMode(props.skill?.name, selectedFile.value?.path || ''))

const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  const current = localFiles.get(path) || ''
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  return current !== orig
})

const fileSize = computed(() => (currentContent.value || '').length)

function onContentChange(v) {
  // 2026-07-08 增:重置锁窗口内的 emit 丢弃,避免 Tiptap 飞行中的异步 update 把
  // 已经重置的 localFiles 重新写回用户的最新内容(详见 resetCurrent 注释)。
  if (Date.now() < resetLockUntil) return
  const path = selectedFile.value?.path
  if (!path) return
  localFiles.set(path, v || '')
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  const s = new Set(dirtyPaths.value)
  if ((v || '') !== orig) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

function onDirtyChange(d) { /* no-op:用 localFiles 即时算 */ }

// ===== Frontmatter =====
const frontmatter = computed(() => {
  const md = (props.files || []).find((f) => f.path === 'SKILL.md')
  return parseFrontmatter(md?.content || '')
})

function parseFrontmatter(text) {
  if (!text) return {}
  const m = text.match(/^---\s*\n([\s\S]*?)\n---/)
  if (!m) return {}
  const block = m[1]
  const out = {}
  for (const line of block.split('\n')) {
    const kv = line.match(/^([a-zA-Z_][\w]*)\s*:\s*(.*)$/)
    if (!kv) continue
    const key = kv[1]
    let v = kv[2].trim()
    if (v.startsWith('[') && v.endsWith(']')) {
      v = v.slice(1, -1).split(',').map((s) => {
        let x = s.trim()
        if ((x.startsWith('"') && x.endsWith('"')) || (x.startsWith("\'") && x.endsWith("\'"))) {
          x = x.slice(1, -1)
        }
        return x
      }).filter(Boolean)
    } else if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("\'") && v.endsWith("\'"))) {
      v = v.slice(1, -1)
    }
    out[key] = v
  }
  return out
}

const hasFrontmatter = computed(() => Object.keys(frontmatter.value).length > 0)
const fmOpen = ref(false)
function openFrontmatter() { fmOpen.value = true }
function closeFrontmatter() { fmOpen.value = false }

const FM_KEY_ORDER = [
  'name', 'version', 'description', 'triggers',
  'author', 'license', 'depends_on', 'target_tools',
  'group_path', 'source', 'source_ref',
]
const frontmatterEntries = computed(() => {
  const fm = frontmatter.value
  const ordered = []
  for (const k of FM_KEY_ORDER) {
    if (k in fm) ordered.push([k, fm[k]])
  }
  for (const k of Object.keys(fm)) {
    if (!FM_KEY_ORDER.includes(k)) ordered.push([k, fm[k]])
  }
  return ordered
})

// ===== store root =====
const storeRoot = ref('')
async function fetchStoreRoot() {
  try {
    const info = await getStoreInfo()
    storeRoot.value = info?.store_root || ''
  } catch (_) { storeRoot.value = '' }
}

const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})

// ===== Save / Discard =====
const saving = ref(false)
const saveError = ref('')
async function saveCurrent() {
  const path = selectedFile.value?.path
  if (!path) return
  const sk = props.skill
  if (!sk || !sk.name) {
    saveError.value = '当前未选中 skill'
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    // 2026-07-08 改:后端 store.Save 是"原子全量覆盖"语义(SKILL.md 用 manifest 重渲,
    // 其它 files 走 c.Files 全量写到 tmp 目录再 rename)— 必须由 caller 拼出完整的
    // files 数组,否则就会丢文件。旧实现只 send 当前 dirty 文件 → 其他文件
    // 走 tmp 写不到 → os.RemoveAll 把原目录删了 → 那些文件就消失了。
    // 这里拿 props.files 当骨架,dirty 行用 localFiles 最新值,未 dirty 行用
    // 原 content 透传,确保发出去跟磁盘原本一致。
    const incomingFiles = (props.files || []).map((f) => {
      if (!f || !f.path) return null
      // SKILL.md 比较特殊:后端会用 RenderSkillMD(c.Manifest) 强制重写,
      // 这里送不送 content 都会被覆盖;但为了语义清晰,dirty 时仍按 localFiles 的 body 重拼
      if (dirtyPaths.value.has(f.path)) {
        if (f.path === 'SKILL.md') {
          const localBody = localFiles.get('SKILL.md') || ''
          return { path: 'SKILL.md', content: rebuildSkillMdFromBody(localBody) }
        }
        return { path: f.path, content: localFiles.get(f.path) || '' }
      }
      return { path: f.path, content: f.content || '' }
    }).filter(Boolean)
    // 兜底:极端情况 props.files 为空(还没回填)→ 退化为老行为 + 弹警告
    if (incomingFiles.length === 0) {
      const fallback = path === 'SKILL.md'
        ? { path: 'SKILL.md', content: rebuildSkillMd() }
        : { path, content: localFiles.get(path) || '' }
      incomingFiles.push(fallback)
      saveError.value = '提示:文件列表为空,只提交了当前文件,保存后其他文件会丢失 — 请等待目录加载完成后再保存。'
    }
    await updateSkill({
      scope: sk.scope || 'global',
      project_id: sk.project_id || 0,
      name: sk.name,
      version: sk.version,
      source: sk.source || 'local',
      manifest: sk.canonical?.manifest || {
        name: sk.name, version: sk.version,
      },
      files: incomingFiles,
    })
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    // 同步 clean 所有"刚被发出去"的文件(包括 SKILL.md 重建后的新内容)
    for (const f of incomingFiles) {
      const stored = f.path === 'SKILL.md' ? splitSkillMd(f.content || '').body : (f.content || '')
      localFiles.set(f.path, stored)
      s.delete(f.path)
    }
    dirtyPaths.value = s
    const savedContent = path === 'SKILL.md' ? rebuildSkillMd() : (localFiles.get(path) || '')
    emit('saved', { path, content: savedContent })
  } catch (e) {
    saveError.value = e?.message || String(e)
    toast.error(`保存失败: ${saveError.value}`)
  } finally {
    saving.value = false
  }
}

// 2026-07-08 增:跟 rebuildSkillMd 配套的"从 body 反推完整 SKILL.md"的工具。
// 复用已有 frontmatter(不从 localFiles 拿 frontmatter,因为 editor 只编辑 body),
// 拼上 body 得到完整字符串。原 rebuildSkillMd() 默认从 props.files 取 SKILL.md 的
// frontmatter;这里签名保持一致,通过参数显式传入 body。
function rebuildSkillMdFromBody(body) {
  const fmLines = []
  for (const k of FM_KEY_ORDER) {
    if (!(k in frontmatter.value)) continue
    const v = frontmatter.value[k]
    if (Array.isArray(v)) fmLines.push(`${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`)
    else fmLines.push(`${k}: ${JSON.stringify(v)}`)
  }
  for (const k of Object.keys(frontmatter.value)) {
    if (FM_KEY_ORDER.includes(k)) continue
    const v = frontmatter.value[k]
    if (Array.isArray(v)) fmLines.push(`${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`)
    else fmLines.push(`${k}: ${JSON.stringify(v)}`)
  }
  return `---\n${fmLines.join('\n')}\n---\n\n${body || ''}\n`
}

function rebuildSkillMd() {
  const path = selectedFile.value?.path
  const body = path === 'SKILL.md' ? (localFiles.get('SKILL.md') || '') : ''
  const fm = frontmatter.value
  const fmLines = []
  for (const k of FM_KEY_ORDER) {
    if (!(k in fm)) continue
    const v = fm[k]
    if (Array.isArray(v)) fmLines.push(`${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`)
    else fmLines.push(`${k}: ${JSON.stringify(v)}`)
  }
  for (const k of Object.keys(fm)) {
    if (FM_KEY_ORDER.includes(k)) continue
    const v = fm[k]
    if (Array.isArray(v)) fmLines.push(`${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`)
    else fmLines.push(`${k}: ${JSON.stringify(v)}`)
  }
  return `---\n${fmLines.join('\n')}\n---\n\n${body}\n`
}

function resetCurrent() {
  const path = selectedFile.value?.path
  if (!path) return
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  // 2026-07-08 修:放弃修改要点 2 次才生效的根因 — Tiptap 编辑器的 onUpdate 是
  // 防抖异步(且 Markdown→HTML→Markdown 转换后字符串跟编辑器实时内容不完全等价),
  // 用户点"放弃修改"瞬间,可能还有一帧 onUpdate 在飞行中:resetCurrent 把 localFiles
  // 重置回 orig 后,Tiptap 那一帧 emit 触发 onContentChange 再次写回用户的最新内容,
  // dirtyPaths 重新被加上,看起来像"放弃失败"。py/json 等 Monaco 走同步
  // onDidChangeContent,没有这个异步窗口,所以一次就生效。
  //
  // 修法:resetCurrent 时打 resetLock 标记,接下来 80ms 内 CodeViewer 传回来的
  // update:content 全部丢弃(避免被 Tiptap 飞行中那一帧覆盖)。Monaco 走同样逻辑,
  // 保证两个编辑器的重置行为统一。
  resetLockUntil = Date.now() + 80
  localFiles.set(path, orig)
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
}

// 2026-07-08 增:resetLock 时间窗,绝对时间戳;onContentChange 检查 now < resetLockUntil。
let resetLockUntil = 0

// ====== ErrorBoundary (render error 兜底) ======
const renderError = ref(null)
function safeReload() {
  renderError.value = null
  fetchStoreRoot()
}
onErrorCaptured((err) => {
  console.error('[SkillFileInlinePanel captured]', err)
  renderError.value = err?.message || String(err)
  return false
})

onUnmounted(() => {})

// 2026-07-07 增:暴露给父组件的方法。
// 父在切换 skill / 切换文件前调 ensureCleanBeforeSwitch(),有 dirty 时弹
// 三选项弹窗,等用户决策返回 'proceed' / 'cancel' 后再决定是否继续。
// resetDirtyNow 是父级强制清 dirty 的兜底(例如删除 skill 流程不需要询问)。
defineExpose({
  ensureCleanBeforeSwitch,
  resetAllDirty,
  isAnyDirty: () => dirtyPaths.value.size > 0,
})
</script>

<template>
  <!-- 渲染异常时的降级 UI -->
  <div v-if="renderError" class="sfip-error">
    <IconPark icon="mdi:alert-circle-outline" width="22" height="22" />
    <h4>{{ LABEL_RENDER_ERROR_TITLE }}</h4>
    <p class="sfip-error-msg">{{ renderError }}</p>
    <button class="primary sm" @click="safeReload">
      <IconPark icon="mdi:refresh" width="14" height="14" />
      {{ LABEL_RETRY }}
    </button>
  </div>
  <div v-else class="sfip">
    <header class="sfip-header">
      <div class="sfip-title-block">
        <IconPark icon="FileCabinet" width="16" height="16" />
        <span class="sfip-name">{{ skill?.name || '' }}<span v-if="skill?.version" class="sfip-version">@{{ skill.version }}</span></span>
        <span v-if="skill?.source" :class="['badge', skill.source === 'market' ? 'blue' : 'gray']">{{ skill.source }}</span>
        <span class="sfip-count">{{ (files || []).length }} {{ LABEL_FILES }}</span>
        <span class="sfip-name-actions">
          <slot name="name-actions" />
        </span>
      </div>
      <div class="sfip-actions">
        <slot name="actions" />
      </div>
      <button
        v-if="hasFrontmatter"
        class="sfip-fm-btn"
        :data-tip="LABEL_FRONTMATTER_TITLE"
        :aria-label="LABEL_FRONTMATTER_TITLE"
        @click="openFrontmatter"
      >
        <IconPark icon="Info" width="15" height="15" />
      </button>
    </header>

    <div class="sfip-body">
      <nav class="sfip-left">
        <!-- 2026-07-07 改 v3:作用域区移到文件树底部。
             旧版:作用域在顶部 → 用户第一眼看到的是 scope,文件树被挤。
             新版:文件树在上(占主要空间),作用域在底部(辅助信息,默认折叠,
             用户主动展开才看得到生效位置)。 -->
        <div class="sfip-tree-wrap">
          <!-- 2026-07-07 增:文件树加标题栏,跟 .ssp-scope-header 风格一致 -->
          <header class="sfip-tree-header">
            <!-- 2026-07-08 改:PascalCase 直传 FileCabinet(避免 mdi 映射兜底导致的"看不见"
                 现象)。多文件柜图标跟"skill 目录树"语义贴合(文件夹集合)。 -->
            <IconPark icon="FileCabinet" width="13" height="13" />
            <span>skill 目录</span>
            <span class="sfip-tree-header-count">{{ (files || []).length }} 个</span>
          </header>
          <FileTreeView
            v-if="(files || []).length"
            :files="files"
            :initial-selected-path="selectedKey"
            :dirty-paths="dirtyPaths"
            @select-file="onSelectFile"
          />
        </div>
        <SkillScopePanel :skill="skill" />
      </nav>

      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <!-- 2026-07-08 改:不再显示 selectedFile.path + fileSize,
               CodeViewer 内部 cv-text-toolbar 已经显示 language 标签,
               顶部 path + size 是冗余信息(用户从左边文件树知道当前选的是哪个)。
               只保留右侧的操作按钮(铅笔/眼睛 + dirty + 放弃/保存)。 -->
          <span class="sfip-viewer-spacer" />
          <button
            v-if="selectedFile?.path && currentMode === 'view'"
            class="sfip-mode-btn"
            :data-tip="LABEL_EDIT"
            :aria-label="LABEL_EDIT"
            @click="setMode(props.skill?.name, selectedFile.path, 'edit')"
          >
            <IconPark icon="Edit" width="14" height="14" />
          </button>
          <button
            v-else-if="selectedFile?.path && currentMode === 'edit'"
            class="sfip-mode-btn sfip-mode-btn-active"
            :data-tip="LABEL_PREVIEW"
            :aria-label="LABEL_PREVIEW"
            @click="setMode(props.skill?.name, selectedFile.path, 'view')"
          >
            <IconPark icon="View" width="14" height="14" />
          </button>
          <span v-if="isDirty" class="sfip-viewer-dirty">{{ LABEL_DIRTY }}</span>
          <button
            v-if="isDirty"
            class="sfip-btn"
            :disabled="saving"
            :data-tip="LABEL_DISCARD"
            :aria-label="LABEL_DISCARD"
            @click="resetCurrent"
          >{{ LABEL_DISCARD }}</button>
          <button
            v-if="isDirty"
            class="sfip-btn sfip-btn-primary"
            :disabled="saving"
            :data-tip="saving ? LABEL_SAVING : LABEL_SAVE"
            :aria-label="LABEL_SAVE"
            @click="saveCurrent"
          >
            <span v-if="saving" class="sfip-spinner"></span>
            <IconPark v-else icon="Save" width="13" height="13" />
            {{ saving ? LABEL_SAVING : LABEL_SAVE }}
          </button>
        </header>
        <CodeViewer
          v-if="selectedFile?.path"
          :key="selectedFile.path"
          :path="selectedFile.path"
          :content="displayContent"
          :mode="currentMode"
          :store-root="storeRoot"
          :skill-rel-path="skillRelPath"
          @update:content="onContentChange"
          @dirty-change="onDirtyChange"
        />
        <div v-else class="sfip-empty">
          <IconPark icon="mdi:file-outline" width="48" height="48" />
          <p>{{ LABEL_PICK }}</p>
        </div>
      </main>
    </div>

    <!-- 2026-07-07 修:Modal 必须用 v-model 绑 modelValue(组件内部 watch modelValue 控制渲染),
         旧版用 v-if="fmOpen" + @close="closeFrontmatter" 看似能调,但组件内部 <div v-if="modelValue">
         modelValue 始终是 undefined,所以 mask 永远不渲染 → 弹窗不出现。 -->
    <Modal
      v-model="fmOpen"
      size="md"
      :title="(skill?.name || '') + ' · frontmatter'"
      @close="closeFrontmatter"
    >
      <div class="sfip-fm-body">
        <table v-if="frontmatterEntries.length" class="sfip-fm-table">
          <tbody>
            <tr v-for="[k, v] in frontmatterEntries" :key="k">
              <th>{{ k }}</th>
              <td>
                <template v-if="Array.isArray(v)">
                  <span v-for="(x, i) in v" :key="i" class="sfip-fm-chip">{{ x }}</span>
                </template>
                <template v-else>{{ v }}</template>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else class="sfip-fm-empty">无 frontmatter</p>
      </div>
      <template #footer>
        <button class="primary" @click="closeFrontmatter">关闭</button>
      </template>
    </Modal>

    <!-- 2026-07-07 增:切换前的 dirty 询问弹窗(三选项:保存/放弃/取消)。
         ensureCleanBeforeSwitch() 返回的 Promise 由 _discardResolve 接住,
         等用户点击对应按钮再 resolve。 -->
    <Modal
      v-model="discardOpen"
      size="sm"
      title="文件已修改"
      :close-on-mask="false"
    >
      <p class="sfip-discard-msg">
        文件 <code>{{ discardFileName }}</code> 已被修改,切换前请选择如何处理:
      </p>
      <ul class="sfip-discard-tips">
        <li><strong>保存修改</strong>:写盘后再切换</li>
        <li><strong>放弃修改</strong>:丢弃本地编辑,加载目标 skill / 文件</li>
        <li><strong>取消</strong>:留在当前页面继续编辑</li>
      </ul>
      <template #footer>
        <button type="button" class="ghost" @click="onDiscardCancel">取消</button>
        <button type="button" class="danger" @click="onDiscardDrop">放弃修改</button>
        <button type="button" class="primary" @click="onDiscardSave">保存修改</button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.sfip {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--bg-card);
}
.sfip-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 100%;
  padding: 40px 24px;
  background: var(--bg-card);
  color: var(--text-dim);
  text-align: center;
}
.sfip-error h4 { margin: 0; font-size: 15px; color: var(--text); }
.sfip-error-msg {
  margin: 0;
  max-width: 480px;
  font-size: 12px;
  color: var(--danger);
  font-family: ui-monospace, SFMono-Regular, monospace;
  word-break: break-all;
}
.sfip-header {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
  gap: 6px;
}
.sfip-title-block {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-dim);
}
.sfip-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.sfip-version {
  color: var(--text-faint);
  font-weight: 400;
  margin-left: 2px;
}
.sfip-count {
  color: var(--text-faint);
  font-size: 11px;
  padding: 1px 6px;
  background: var(--bg-subtle);
  border-radius: 999px;
}
.sfip-name-actions { display: inline-flex; gap: 6px; }
.sfip-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}
.sfip-actions :deep(.icon-btn) {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  padding: 0;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}
.sfip-actions :deep(.icon-btn:hover:not(:disabled)) {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue);
}
.sfip-fm-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-faint);
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}
.sfip-fm-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue);
}
.sfip-body {
  display: flex;
  flex: 1;
  min-height: 0;
}
.sfip-left {
  width: 240px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
}
.sfip-tree-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* 2026-07-07 增:文件树标题,跟作用域 .ssp-scope-header 风格一致 */
.sfip-tree-header {
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
  background: var(--bg-subtle);
  position: sticky;
  top: 0;
  z-index: 1;
}
.sfip-tree-header-count {
  margin-left: auto;
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
}

/* 2026-07-07 改 v3:作用域区移到文件树底部,文件树占满主要高度,
   作用域用 flex-shrink:0 固定在底部一块,ScopePanel 内部自己 max-height:50% 兜底。 */
.sfip-viewer {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.sfip-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-subtle);
  font-size: 12px;
  color: var(--text-dim);
  flex-shrink: 0;
}
.sfip-viewer-path {
  font-family: ui-monospace, SFMono-Regular, monospace;
  color: var(--text);
  flex: 1;
}
.sfip-viewer-size { color: var(--text-faint); }
/* 2026-07-08 增:替代 .sfip-viewer-path 的弹性占位,让右侧操作按钮右对齐 */
.sfip-viewer-spacer { flex: 1; }
.sfip-viewer-dirty {
  color: var(--accent-amber, #b8860b);
  font-size: 11px;
  margin-right: 4px;
}
.sfip-mode-btn,
.sfip-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
  padding: 3px 10px;
  font-size: 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.sfip-mode-btn:hover,
.sfip-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--text-faint);
}
.sfip-mode-btn-active,
.sfip-btn-primary {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-color: var(--accent-blue-border);
}
.sfip-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-faint);
  font-size: 13px;
}
.sfip-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: sfip-spin 0.8s linear infinite;
}
@keyframes sfip-spin {
  to { transform: rotate(360deg); }
}
.sfip-fm-body {
  font-size: 13px;
  color: var(--text);
}
.sfip-fm-table {
  width: 100%;
  border-collapse: collapse;
}
.sfip-fm-table th,
.sfip-fm-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
}
.sfip-fm-table th {
  width: 120px;
  font-weight: 600;
  color: var(--text-dim);
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 12px;
}
.sfip-fm-table td { color: var(--text); }
.sfip-fm-table tr:last-child th,
.sfip-fm-table tr:last-child td { border-bottom: none; }
.sfip-fm-value { white-space: pre-wrap; }
.sfip-fm-chip {
  display: inline-block;
  margin-right: 4px;
  margin-bottom: 2px;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--accent-blue);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-family: ui-monospace, SFMono-Regular, monospace;
}
.sfip-fm-empty {
  color: var(--text-faint);
  font-style: italic;
}

/* 2026-07-07 增:dirty 询问弹窗样式 */
.sfip-discard-msg {
  margin: 0 0 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
.sfip-discard-msg code {
  font-family: ui-monospace, SFMono-Regular, monospace;
  padding: 1px 6px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
}
.sfip-discard-tips {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  line-height: 1.8;
  color: var(--text-dim);
}
.sfip-discard-tips strong {
  color: var(--text);
  font-weight: 600;
}
</style>
