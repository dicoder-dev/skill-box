<script setup>
// SkillFileInlinePanel - 首页右侧主区域:目录树 + 预览/编辑
//
// 2026-07-04 改 v2:替换原来的 detail-body(SKILL.md 单独渲染区),
// 把整个右侧详情区换成"左目录树 + 右预览/编辑"两栏布局:
//   - 左侧 260px:技能包全文件树(含 SKILL.md)
//   - 右侧:文件预览/编辑(代码走 Monaco,markdown 也走 Monaco 不再单独渲染)
//   - SKILL.md 也走 Monaco 编辑(用 markdown language,统一编辑器风格)
//   - 支持编辑保存(updateSkill)
//
// 不用原 detail-body 的 .md-body markdown 渲染(它专为"只读 SKILL.md"设计),
// 也不再用抽屉 / 弹窗。

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
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

watch(
  () => props.files,
  (files) => {
    if (files && files.length) {
      // 默认选 SKILL.md(主入口)
      const sk = files.find((f) => f.path === 'SKILL.md')
      const target = sk || files[0]
      selectedFile.value = target
      selectedKey.value = target?.path || ''
    } else {
      selectedFile.value = null
      selectedKey.value = ''
    }
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

const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  return localFiles.get(path) !== (selectedFile.value?.content || '')
})

const fileSize = computed(() => (currentContent.value || '').length)

function onContentChange(v) {
  const path = selectedFile.value?.path
  if (!path) return
  localFiles.set(path, v || '')
  const orig = selectedFile.value?.content || ''
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
    const newContent = localFiles.get(path) || ''
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
  const orig = selectedFile.value.content || ''
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
</script>

<template>
  <div class="sfip">
    <header class="sfip-header">
      <div class="sfip-title-block">
        <IconPark icon="mdi:folder-multiple-outline" width="16" height="16" />
        <span class="sfip-name">{{ skill?.name || '' }}<span v-if="skill?.version" class="sfip-version">@{{ skill.version }}</span></span>
        <span class="sfip-count">{{ (files || []).length }} files</span>
      </div>
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
          :content="currentContent"
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
</style>