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

import { computed, onMounted, onUnmounted, reactive, ref, watch, onErrorCaptured } from 'vue'
import { plainT, messages } from '@/core/i18n/index.js'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import SkillScopePanel from './SkillScopePanel.vue'
import { updateSkill, getStoreInfo } from '@/api/skillbox/skills'
import { useToastStore } from '@/core/store/toast'

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

function splitSkillMd(text) {
  if (!text) return { frontmatter: '', body: text }
  const m = text.match(/^---\s*\n([\s\S]*?)\n---\s*\n?/)
  if (!m) return { frontmatter: '', body: text }
  return { frontmatter: m[0], body: text.slice(m[0].length) }
}

// 选中文件 → localFiles 填充,响应 props.files 变化
watch(
  () => [props.files, props.skill?.name],
  () => {
    const files = props.files
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
  },
  { immediate: true },
)

watch(
  () => props.files,
  () => {
    localFiles.clear()
    for (const f of props.files || []) {
      const c = f.content || ''
      const stored = f.path === 'SKILL.md' ? splitSkillMd(c).body : c
      localFiles.set(f.path, stored)
    }
    dirtyPaths.value = new Set()
  },
  { immediate: true, deep: true },
)

function onSelectFile(file) {
  selectedFile.value = file
  selectedKey.value = file.path
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
    const newMd = rebuildSkillMd()
    const files = []
    if (path === 'SKILL.md') {
      files.push({ path: 'SKILL.md', content: newMd })
    } else {
      files.push({ path, content: localFiles.get(path) || '' })
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
      files,
    })
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    emit('saved', { path, content: path === 'SKILL.md' ? newMd : localFiles.get(path) })
  } catch (e) {
    saveError.value = e?.message || String(e)
    toast.error(`保存失败: ${saveError.value}`)
  } finally {
    saving.value = false
  }
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
  localFiles.set(path, orig)
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
}

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

onMounted(() => {
  fetchStoreRoot()
})
onUnmounted(() => {})
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
        <IconPark icon="mdi:folder-multiple-outline" width="16" height="16" />
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
        :title="LABEL_FRONTMATTER_TITLE"
        :aria-label="LABEL_FRONTMATTER_TITLE"
        @click="openFrontmatter"
      >
        <IconPark icon="mdi:information-outline" width="15" height="15" />
      </button>
    </header>

    <div class="sfip-body">
      <nav class="sfip-left">
        <!-- 作用域区已迁到 <SkillScopePanel>,本组件不直接渲染 t() -->
        <SkillScopePanel :skill="skill" />

        <div class="sfip-tree-wrap">
          <FileTreeView
            v-if="(files || []).length"
            :files="files"
            :initial-selected-path="selectedKey"
            :dirty-paths="dirtyPaths"
            @select-file="onSelectFile"
          />
        </div>
      </nav>

      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <span class="sfip-viewer-path">{{ selectedFile?.path || LABEL_NO_FILE }}</span>
          <span v-if="selectedFile?.path" class="sfip-viewer-size">{{ fileSize }} B</span>
          <button
            v-if="selectedFile?.path && currentMode === 'view'"
            class="sfip-mode-btn"
            :title="LABEL_EDIT"
            :aria-label="LABEL_EDIT"
            @click="setMode(props.skill?.name, selectedFile.path, 'edit')"
          >
            <IconPark icon="mdi:pencil-outline" width="14" height="14" />
          </button>
          <button
            v-else-if="selectedFile?.path && currentMode === 'edit'"
            class="sfip-mode-btn sfip-mode-btn-active"
            :title="LABEL_PREVIEW"
            :aria-label="LABEL_PREVIEW"
            @click="setMode(props.skill?.name, selectedFile.path, 'view')"
          >
            <IconPark icon="mdi:eye-outline" width="14" height="14" />
          </button>
          <span v-if="isDirty" class="sfip-viewer-dirty">{{ LABEL_DIRTY }}</span>
          <button
            v-if="isDirty"
            class="sfip-btn"
            :disabled="saving"
            @click="resetCurrent"
          >{{ LABEL_DISCARD }}</button>
          <button
            v-if="isDirty"
            class="sfip-btn sfip-btn-primary"
            :disabled="saving"
            @click="saveCurrent"
          >
            <span v-if="saving" class="sfip-spinner"></span>
            <IconPark v-else icon="mdi:content-save" width="13" height="13" />
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

    <!-- Frontmatter modal(纯展示,不调 t()) -->
    <Modal v-if="fmOpen" size="md" :title="(skill?.name || '') + ' · frontmatter'" @close="closeFrontmatter">
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
</style>
