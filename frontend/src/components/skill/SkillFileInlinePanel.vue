<script setup>
// SkillFileInlinePanel - 首页右侧主区域:目录树 + 预览/编辑
//
// 2026-07-04 改 v2:替换原来的 detail-body(SKILL.md 单独渲染区),
// 把整个右侧详情区换成"左目录树 + 右预览/编辑"两栏布局:
//   - 左侧 200px:技能包全文件树(含 SKILL.md)
//   - 右侧:文件预览/编辑(代码走 Monaco,markdown 也走 Monaco 不再单独渲染)
//   - SKILL.md 也走 Monaco 编辑(用 markdown language,统一编辑器风格)
//   - 支持编辑保存(updateSkill)
//
// 2026-07-04 改 v3:SKILL.md 的 frontmatter 不直接显示在编辑器里,
// 在面板顶部右侧加一个 [info] 图标,点击后弹窗显示完整的 frontmatter
// (name / version / description / triggers / author / license / depends_on / target_tools)。

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import { updateSkill, getStoreInfo } from '@/api/skillbox/skills'
import { useToastStore } from '@/core/store/toast'
import { useAppStore } from '@/core/store/app'

const { t } = useI18n()
const toast = useToastStore()
const appStore = useAppStore()

const props = defineProps({
  // 技能包文件列表 [{path, content}] - 来自后端 canonical.files
  files: { type: Array, default: () => [] },
  // 当前选中的 skill:{ name, version, scope, project_id, source, group_path, canonical }
  skill: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['saved'])

// 当前选中的文件
const selectedFile = ref(null)
const selectedKey = ref('')  // 用于 FileTreeView 的 selectedPath

// 监听 props.files 变化,更新 selectedFile
// 2026-07-04 修(Commit 8+):保存代码文件后,父组件 onDrawerSaved 会 reload 整个 skill,
// props.files 重新赋值,这个 watch 会触发。旧版总是 fallback 到 SKILL.md,
// 导致用户编辑了 examples/foo.py 点保存 → 跳回 SKILL.md,体验很糟。
// 修复:files 变化时优先保留 selectedKey(用户正在编辑的文件),找不到再 fallback SKILL.md。
watch(
  () => props.files,
  (files) => {
    if (!files || !files.length) {
      selectedFile.value = null
      selectedKey.value = ''
      return
    }
    // 优先用用户当前选中的 path 在新 files 里找
    const prev = selectedKey.value
    if (prev) {
      const found = files.find((f) => f.path === prev)
      if (found) {
        // 保留 selectedKey,只更新 selectedFile 的 content(防 stale)
        // 但 selectedFile 不能直接用 found 替换,因为 selectedFile 是 ref,会触发 watch
        // 用 nextTick 等一帧再设(实际上 selectedFile 在外面已经被 saveCurrent 同步过)
        if (!selectedFile.value || selectedFile.value.path !== prev) {
          selectedFile.value = found
        }
        return
      }
    }
    // 首次打开/没选中:默认选 SKILL.md
    const sk = files.find((f) => f.path === 'SKILL.md')
    const target = sk || files[0]
    selectedFile.value = target
    selectedKey.value = target?.path || ''
  },
  { immediate: true, deep: true },
)

function onSelectFile(file) {
  selectedFile.value = file
  selectedKey.value = file.path
}

// 编辑态独立副本
const localFiles = reactive(new Map())
const dirtyPaths = ref(new Set())

watch(
  () => [props.files],
  () => {
    localFiles.clear()
    for (const f of props.files || []) {
      localFiles.set(f.path, f.content || '')
    }
    dirtyPaths.value = new Set()
  },
  { immediate: true, deep: true },
)

const isReadOnly = computed(() => false)  // v2 改:所有文件都可编辑
const currentContent = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return ''
  return localFiles.has(path) ? localFiles.get(path) : (selectedFile.value?.content || '')
})

// 2026-07-04 增:SKILL.md 在 Monaco 里**不显示 frontmatter 区域**(用户反馈太干扰)。
// 策略:
//   - Monaco 看到的内容 = body(去掉开头 --- 块)
//   - localFiles / selectedFile.content 始终存完整 SKILL.md 原文
//   - 保存时用 rebuildSkillMd 把 frontmatter + 编辑后 body 重新拼回
//   - 顶部加 [i] frontmatter 弹窗,告诉用户这些元数据存在但不在编辑器里
function splitSkillMd(text) {
  if (!text) return { frontmatter: '', body: '' }
  // 匹配开头 --- 到下一个 --- 的 frontmatter 块(允许末尾有空行)
  const m = text.match(/^---\s*\n([\s\S]*?)\n---\s*\n?/)
  if (!m) return { frontmatter: '', body: text }
  return {
    frontmatter: m[0],         // 含 --- 包裹的完整块
    body: text.slice(m[0].length),  // frontmatter 之后的内容
  }
}

// 给 Monaco 显示用的内容(SKILL.md 去掉 frontmatter,其它文件原样)
const displayContent = computed(() => {
  if (!selectedFile.value) return ''
  if (selectedFile.value.path === 'SKILL.md') {
    return splitSkillMd(currentContent.value).body
  }
  return currentContent.value
})

const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  // 2026-07-04 改:SKILL.md 时比较 body(同 displayContent 的逻辑)
  const current = localFiles.get(path) || ''
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  return current !== orig
})

const fileSize = computed(() => (currentContent.value || '').length)

function onContentChange(v) {
  const path = selectedFile.value?.path
  if (!path) return
  // 2026-07-04 改:SKILL.md 时,Monaco 拿到的是 body,原文件含 frontmatter,
  // 不能直接比 localFiles 跟 selectedFile.content(永远不等 → 永远 dirty)。
  // 统一存 localFiles = "Monaco 看到的内容"(SKILL.md 是 body,其它文件是原文)。
  // orig(用于 dirty 判定)同步剥 frontmatter。
  localFiles.set(path, v || '')
  const origFull = selectedFile.value?.content || ''
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  const s = new Set(dirtyPaths.value)
  if ((v || '') !== orig) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

function onDirtyChange(d) {
  const path = selectedFile.value?.path
  if (!path) return
  const s = new Set(dirtyPaths.value)
  if (d) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

// 保存当前文件
const saving = ref(false)
async function saveCurrent() {
  if (!selectedFile.value) return
  saving.value = true
  try {
    const path = selectedFile.value.path
    let newContent = localFiles.get(path) || ''

    // 2026-07-04 改:SKILL.md 保存时,把 Monaco 编辑的 body + 原 frontmatter 拼回。
    // 否则保存的就是"剥离 frontmatter 的 body",磁盘文件就丢了元数据。
    if (path === 'SKILL.md') {
      const orig = selectedFile.value?.content || ''
      const { frontmatter } = splitSkillMd(orig)
      // 如果原文件有 frontmatter,拼回去;如果用户把 frontmatter 全删了,新文件也不加(尊重用户)
      if (frontmatter) {
        newContent = frontmatter + (newContent.startsWith('\n') ? '' : '\n') + newContent
      }
    }

    const updatedFiles = (props.files || []).map((f) =>
      f.path === path ? { ...f, content: newContent } : f,
    )
    await updateSkill({
      scope: props.skill.scope,
      project_id: props.skill.project_id,
      name: props.skill.name,
      version: props.skill.version,
      source: props.skill.source || 'local',
      manifest: props.skill.canonical?.manifest || {
        name: props.skill.name,
        version: props.skill.version,
      },
      files: updatedFiles,
    })
    selectedFile.value = { ...selectedFile.value, content: newContent }
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    emit('saved', { path, content: newContent })
    toast.success(t('skills.fileBrowser.saved', { path }))
  } catch (e) {
    toast.error(t('skills.fileBrowser.saveFailed', { msg: e?.message || e }))
  } finally {
    saving.value = false
  }
}

function resetCurrent() {
  if (!selectedFile.value) return
  const path = selectedFile.value.path
  const origFull = selectedFile.value.content || ''
  // 2026-07-04 改:SKILL.md 时 Monaco 看到的是 body,reset 也要把 body 写回 localFiles
  // 否则 onContentChange 比对会判 dirty(完整 vs body 永远不等)
  const orig = path === 'SKILL.md' ? splitSkillMd(origFull).body : origFull
  localFiles.set(path, orig)
  onContentChange(orig)
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
}

// store_root(用于"在文件夹打开")
const storeRoot = ref('')
async function fetchStoreRoot() {
  if (storeRoot.value) return
  try {
    const info = await getStoreInfo()
    storeRoot.value = info?.store_root || ''
  } catch (_) { storeRoot.value = '' }
}
onMounted(fetchStoreRoot)

const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})

// ====== Frontmatter 弹窗 ======
// 2026-07-04 增:从 SKILL.md 文件内容解析 YAML frontmatter,弹窗展示。
// 不在 Monaco 里直接显示 frontmatter(让用户专心编辑正文)。
//
// 简易解析:不引 js-yaml 依赖(打包又 +30KB),自己写一个最小解析器,
// 只支持扁平的 key: value 和 key: [array] 语法(skillbox manifest 实际只用这些)。
const fmOpen = ref(false)

// 从 SKILL.md 文件内容里抽 frontmatter 块
function parseFrontmatter(text) {
  if (!text) return {}
  const m = text.match(/^---\s*\n([\s\S]*?)\n---/)
  if (!m) return {}
  const block = m[1]
  const out = {}
  // 每行格式:key: value  或  key: [a, b]
  for (const line of block.split('\n')) {
    const kv = line.match(/^([a-zA-Z_][\w]*)\s*:\s*(.*)$/)
    if (!kv) continue
    const key = kv[1]
    let v = kv[2].trim()
    // 数组:[a, b] → 拆
    if (v.startsWith('[') && v.endsWith(']')) {
      v = v.slice(1, -1).split(',').map((s) => {
        let x = s.trim()
        // 去掉外层引号
        if ((x.startsWith('"') && x.endsWith('"')) || (x.startsWith("'") && x.endsWith("'"))) {
          x = x.slice(1, -1)
        }
        return x
      }).filter(Boolean)
    } else if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
      v = v.slice(1, -1)
    }
    out[key] = v
  }
  return out
}

const frontmatter = computed(() => {
  const md = (props.files || []).find((f) => f.path === 'SKILL.md')
  return parseFrontmatter(md?.content || '')
})

const hasFrontmatter = computed(() => Object.keys(frontmatter.value).length > 0)

// 展示用的 key 顺序(常用在前)
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
  // 其它 key 追加
  for (const k of Object.keys(fm)) {
    if (!FM_KEY_ORDER.includes(k)) ordered.push([k, fm[k]])
  }
  return ordered
})

function openFrontmatter() { fmOpen.value = true }
function closeFrontmatter() { fmOpen.value = false }
</script>

<template>
  <div class="sfip">
    <header class="sfip-header">
      <div class="sfip-title-block">
        <IconPark icon="mdi:folder-multiple-outline" width="16" height="16" />
        <span class="sfip-name">{{ skill?.name || '' }}<span v-if="skill?.version" class="sfip-version">@{{ skill.version }}</span></span>
        <span class="sfip-count">{{ (files || []).length }} files</span>
      </div>
      <!-- 2026-07-04 增:SKILL.md frontmatter 弹窗按钮(只读展示,不影响编辑)
           frontmatter 里有 name / version / triggers / description 等元数据,
           单独看比混在 markdown 正文里更清晰。 -->
      <button
        v-if="hasFrontmatter"
        class="sfip-fm-btn"
        :title="'查看 frontmatter'"
        :aria-label="'查看 frontmatter'"
        @click="openFrontmatter"
      >
        <IconPark icon="mdi:information-outline" width="15" height="15" />
      </button>
    </header>

    <div class="sfip-body">
      <!-- 左:文件树 -->
      <nav class="sfip-tree">
        <FileTreeView
          v-if="(files || []).length"
          :files="files"
          :initial-selected-path="selectedKey"
          :dirty-paths="dirtyPaths"
          @select-file="onSelectFile"
        />
        <p v-else class="sfip-tree-empty">该技能包为空</p>
      </nav>

      <!-- 右:文件预览/编辑 -->
      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <span class="sfip-viewer-path">{{ selectedFile?.path || t('skills.fileBrowser.noFile') }}</span>
          <span v-if="selectedFile?.path" class="sfip-viewer-size">{{ fileSize }} B</span>
          <span v-if="isDirty" class="sfip-viewer-dirty">● {{ t('skills.fileBrowser.unsavedShort') }}</span>
          <button
            v-if="isDirty"
            class="sfip-btn"
            :disabled="saving"
            @click="resetCurrent"
          >{{ t('skills.fileBrowser.discard') }}</button>
          <button
            v-if="isDirty"
            class="sfip-btn sfip-btn-primary"
            :disabled="saving"
            @click="saveCurrent"
          >
            <span v-if="saving" class="sfip-spinner"></span>
            <IconPark v-else icon="mdi:content-save" width="13" height="13" />
            {{ saving ? t('skills.fileBrowser.saving') : t('skills.fileBrowser.save') }}
          </button>
        </header>
        <CodeViewer
          v-if="selectedFile?.path"
          :key="selectedFile.path"
          :path="selectedFile.path"
          :content="displayContent"
          :editable="!isReadOnly"
          :store-root="storeRoot"
          :skill-rel-path="skillRelPath"
          @update:content="onContentChange"
          @dirty-change="onDirtyChange"
        />
        <div v-else class="sfip-empty">
          <IconPark icon="mdi:file-outline" width="48" height="48" />
          <p>{{ t('skills.fileBrowser.pickOne') }}</p>
        </div>
      </main>
    </div>

    <!-- 2026-07-04 增:Frontmatter 弹窗(SKILL.md 元数据,只读展示) -->
    <Modal v-model="fmOpen" size="md" :title="`${skill?.name || ''} · frontmatter`">
      <div class="sfip-fm">
        <p class="sfip-fm-hint">SKILL.md 文件头部的元数据,主入口信息从这里来。</p>
        <table class="sfip-fm-table">
          <tbody>
            <tr v-for="[k, v] in frontmatterEntries" :key="k">
              <th>{{ k }}</th>
              <td>
                <template v-if="Array.isArray(v)">
                  <span v-for="(item, i) in v" :key="i" class="sfip-fm-chip">{{ item }}</span>
                  <span v-if="!v.length" class="sfip-fm-empty">[]</span>
                </template>
                <template v-else>
                  <span class="sfip-fm-value">{{ v || '""' }}</span>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
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
.sfip-header {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
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
  margin-left: auto;
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
.sfip-tree {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  overflow: auto;
  padding: 8px 10px;
  background: var(--bg-subtle);
}
.sfip-tree-empty {
  color: var(--text-faint);
  font-size: 12px;
  padding: 12px 8px;
  margin: 0;
}
.sfip-viewer {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
}
.sfip-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 12px;
  background: var(--bg-card);
  flex-shrink: 0;
}
.sfip-viewer-path {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sfip-viewer-size {
  color: var(--text-faint);
  font-size: 11px;
}
.sfip-viewer-dirty {
  color: var(--accent-amber, #d97706);
  font-weight: 500;
}
.sfip-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text-dim);
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease;
}
.sfip-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
}
.sfip-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.sfip-btn-primary {
  background: var(--accent-blue);
  color: white;
  border-color: var(--accent-blue);
}
.sfip-btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
  color: white;
}
.sfip-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: sfip-spin 0.8s linear infinite;
}
@keyframes sfip-spin { to { transform: rotate(360deg); } }
.sfip-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-faint);
}

/* 2026-07-04 增:Frontmatter 弹窗内容样式 */
.sfip-fm {
  font-size: 13px;
}
.sfip-fm-hint {
  color: var(--text-faint);
  font-size: 12px;
  margin: 0 0 14px;
}
.sfip-fm-table {
  width: 100%;
  border-collapse: collapse;
}
.sfip-fm-table th,
.sfip-fm-table td {
  text-align: left;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}
.sfip-fm-table th {
  width: 130px;
  color: var(--text-dim);
  font-weight: 500;
  font-size: 12px;
}
.sfip-fm-table td {
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 12.5px;
  word-break: break-all;
}
.sfip-fm-table tr:last-child th,
.sfip-fm-table tr:last-child td {
  border-bottom: none;
}
.sfip-fm-value {
  white-space: pre-wrap;
}
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
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}
.sfip-fm-empty {
  color: var(--text-faint);
  font-style: italic;
}
</style>