<script setup>
// SkillFileDrawer - 首页技能文件浏览器抽屉。
//
// 2026-07-04 增:从 SkillsView 详情区的 [📁] 按钮打开,
// 展示技能包全文件树 + Monaco 预览/编辑(代码) + Markdown 渲染 + 二进制兜底。
//
// 设计要点:
//   - 不复用 Modal.vue(居中弹窗与"右侧抽屉"形态冲突)
//   - <Teleport to="body"> + position:fixed;right:0 + Transition 滑入
//   - 两栏:左 280px 文件树,右 flex:1 文件预览/编辑器
//   - SKILL.md 在抽屉里默认选中,只读(避免与主页 Tiptap 编辑态冲突)
//   - 编辑保存复用后端 updateSkill(同 saveInlineEdit 路径)

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import { updateSkill, getStoreInfo } from '@/api/skillbox/skills'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // 当前选中的 skill:{ name, version, scope, project_id, source, group_path, canonical }
  skill: { type: Object, default: () => ({}) },
  // 技能包文件列表 [{path, content}] - 来自后端 canonical.files
  files: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'closed', 'saved'])

// 编辑态独立副本(覆盖原 files 内容,保存时构造完整 files 数组提交)
const localFiles = reactive(new Map())
// 当前 dirty 状态(任一文件)
const dirtyPaths = ref(new Set())

// 同步 props.files → localFiles(打开抽屉 / 外部更新时)
watch(
  () => [props.modelValue, props.files],
  ([open, files]) => {
    if (!open) return
    localFiles.clear()
    for (const f of files || []) {
      localFiles.set(f.path, f.content || '')
    }
    dirtyPaths.value = new Set()
  },
  { immediate: true, deep: true },
)

// 当前选中的文件
const selectedFile = ref(null)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      const sk = (props.files || []).find((f) => f.path === 'SKILL.md')
      selectedFile.value = sk || props.files[0] || null
    } else {
      selectedFile.value = null
    }
  },
  { immediate: true },
)

function onSelectFile(file) {
  if (dirtyPaths.value.size > 0 && !dirtyPaths.value.has(file.path)) {
    // 切到其他文件不影响当前 dirty(每个文件独立 dirty)
  }
  selectedFile.value = file
}

// SKILL.md 在抽屉里强制只读(避免与主页 Tiptap 编辑态冲突)
const isReadOnly = computed(() => selectedFile.value?.path === 'SKILL.md')

// 当前显示的内容
const currentContent = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return ''
  return localFiles.has(path) ? localFiles.get(path) : (selectedFile.value?.content || '')
})

// dirty 检测
const isDirty = computed(() => {
  const path = selectedFile.value?.path
  if (!path) return false
  if (!localFiles.has(path)) return false
  return localFiles.get(path) !== (selectedFile.value?.content || '')
})

// 文件大小
const fileSize = computed(() => (currentContent.value || '').length)

// CodeViewer emit: content 变化
function onContentChange(v) {
  const path = selectedFile.value?.path
  if (!path) return
  localFiles.set(path, v || '')
  // 更新 dirty 集合
  const orig = selectedFile.value?.content || ''
  const s = new Set(dirtyPaths.value)
  if ((v || '') !== orig) s.add(path)
  else s.delete(path)
  dirtyPaths.value = s
}

// CodeViewer emit: dirty 状态(冗余,但用于切文件时清 dirty)
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
  if (isReadOnly.value) return
  saving.value = true
  try {
    const path = selectedFile.value.path
    const newContent = localFiles.get(path) || ''
    // 构造完整 files 数组(只改选中的那个,其它保持原样)
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
    // 同步到 selectedFile.content(去掉 dirty)
    selectedFile.value = { ...selectedFile.value, content: newContent }
    const s = new Set(dirtyPaths.value)
    s.delete(path)
    dirtyPaths.value = s
    emit('saved', { path, content: newContent })
    toast.success(`已保存 ${path}`)
  } catch (e) {
    toast.error(`保存失败: ${e?.message || e}`)
  } finally {
    saving.value = false
  }
}

// 放弃当前修改
function resetCurrent() {
  if (!selectedFile.value) return
  const path = selectedFile.value.path
  localFiles.set(path, selectedFile.value.content || '')
  // 触发 CodeWatcher 重新渲染(Monaco 实例内部 model 也要同步)
  onContentChange(selectedFile.value.content || '')
  const s = new Set(dirtyPaths.value)
  s.delete(path)
  dirtyPaths.value = s
}

// 关闭抽屉
async function close() {
  if (dirtyPaths.value.size > 0) {
    const ok = window.confirm(t('skills.fileBrowser.unsaved'))
    if (!ok) return
  }
  emit('update:modelValue', false)
  emit('closed')
}

function onMaskClick() {
  close()
}

function onKeydown(e) {
  if (e.key === 'Escape' && props.modelValue) {
    e.stopPropagation()
    close()
  }
}

// 2026-07-04 增(Commit 5):拉 store_root + 拼 skill 相对路径,用于"在文件夹打开"按钮
const storeRoot = ref('')
async function fetchStoreRoot() {
  if (storeRoot.value) return
  try {
    const info = await getStoreInfo()
    storeRoot.value = info?.store_root || ''
  } catch (_) {
    storeRoot.value = ''
  }
}
onMounted(fetchStoreRoot)
watch(() => props.modelValue, (open) => { if (open) fetchStoreRoot() })

// skill 相对路径(在 store_root 下的相对路径)
const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer-fade">
      <div
        v-if="modelValue"
        class="file-drawer-mask"
        @click="onMaskClick"
        @keydown="onKeydown"
        tabindex="-1"
      >
        <Transition name="drawer-slide" appear>
          <aside
            v-if="modelValue"
            class="file-drawer-panel"
            role="dialog"
            aria-modal="true"
            :aria-label="`${skill?.name || ''} 文件浏览器`"
            @click.stop
          >
            <!-- header -->
            <header class="file-drawer-header">
              <div class="file-drawer-title">
                <IconPark icon="mdi:folder-multiple-outline" width="18" height="18" />
                <span class="file-drawer-name">{{ skill?.name || '' }}<span v-if="skill?.version" class="file-drawer-version">@{{ skill.version }}</span></span>
                <span class="file-drawer-count">{{ t('skills.fileBrowser.files', { n: (files || []).length }) }}</span>
              </div>
              <button class="file-drawer-close" :aria-label="t('common.close')" @click="close">
                <IconPark icon="mdi:close" width="18" height="18" />
              </button>
            </header>

            <!-- body:左树 + 右预览 -->
            <div class="file-drawer-body">
              <nav class="file-drawer-tree">
                <FileTreeView
                  :files="files"
                  :initial-selected-path="selectedFile?.path || 'SKILL.md'"
                  :dirty-paths="dirtyPaths"
                  @select-file="onSelectFile"
                />
              </nav>
              <main class="file-drawer-viewer">
                <header class="file-drawer-viewer-header">
                  <IconPark :icon="selectedFile?.path ? 'mdi:file-document-outline' : 'mdi:file-outline'" width="14" height="14" />
                  <span class="file-drawer-viewer-path">{{ selectedFile?.path || t('skills.fileBrowser.noFile') }}</span>
                  <span v-if="selectedFile?.path" class="file-drawer-viewer-size">{{ fileSize }} B</span>
                  <span v-if="isReadOnly && selectedFile?.path" class="file-drawer-viewer-readonly" :title="t('skills.fileBrowser.readOnlyHint')">{{ t('skills.fileBrowser.readOnly') }}</span>
                  <span v-if="isDirty" class="file-drawer-viewer-dirty">● {{ t('skills.fileBrowser.unsavedShort', '未保存') }}</span>
                  <button
                    v-if="isDirty && !isReadOnly"
                    class="file-drawer-viewer-btn"
                    :disabled="saving"
                    @click="resetCurrent"
                  >{{ t('skills.fileBrowser.discard') }}</button>
                  <button
                    v-if="isDirty && !isReadOnly"
                    class="file-drawer-viewer-btn file-drawer-viewer-btn-primary"
                    :disabled="saving"
                    @click="saveCurrent"
                  >
                    <span v-if="saving" class="spinner spinner-sm"></span>
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
                <div v-else class="file-drawer-empty">
                  <IconPark icon="mdi:file-outline" width="48" height="48" />
                  <p>{{ t('skills.fileBrowser.pickOne', '从左侧选择一个文件查看') }}</p>
                </div>
              </main>
            </div>
          </aside>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.file-drawer-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}
.file-drawer-panel {
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: min(1080px, 80vw);
  background: var(--bg-card);
  border-left: 1px solid var(--border);
  box-shadow: -16px 0 32px rgba(0, 0, 0, 0.18);
  display: flex;
  flex-direction: column;
  outline: none;
}
.file-drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
}
.file-drawer-title {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text);
}
.file-drawer-name {
  font-size: 15px;
  font-weight: 600;
}
.file-drawer-version {
  color: var(--text-faint);
  font-weight: 400;
  margin-left: 4px;
}
.file-drawer-count {
  color: var(--text-faint);
  font-size: 12px;
  padding: 2px 8px;
  background: var(--bg-subtle);
  border-radius: 999px;
}
.file-drawer-close {
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: background 120ms ease, color 120ms ease;
}
.file-drawer-close:hover {
  background: var(--bg-hover);
  color: var(--text);
}
.file-drawer-body {
  display: flex;
  flex: 1;
  min-height: 0;
}
.file-drawer-tree {
  width: 280px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  overflow: auto;
  padding: 8px 12px;
  background: var(--bg-subtle);
}
.file-drawer-viewer {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
}
.file-drawer-viewer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-bottom: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 12px;
  background: var(--bg-card);
  flex-shrink: 0;
}
.file-drawer-viewer-path {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}
.file-drawer-viewer-size {
  color: var(--text-faint);
  font-size: 11px;
}
.file-drawer-viewer-readonly {
  font-size: 11px;
  color: var(--text-faint);
  background: var(--bg-subtle);
  padding: 2px 8px;
  border-radius: 4px;
}
.file-drawer-viewer-dirty {
  font-size: 12px;
  color: var(--accent-amber, #d97706);
  font-weight: 500;
}
.file-drawer-viewer-btn {
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
.file-drawer-viewer-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
}
.file-drawer-viewer-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.file-drawer-viewer-btn-primary {
  background: var(--accent-blue);
  color: white;
  border-color: var(--accent-blue);
}
.file-drawer-viewer-btn-primary:hover:not(:disabled) {
  background: var(--accent-blue);
  filter: brightness(1.1);
  color: white;
}
.file-drawer-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-faint);
}
.spinner-sm {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 滑入动画 */
.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 200ms ease;
}
.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}
.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 240ms cubic-bezier(0.16, 1, 0.3, 1);
}
.drawer-slide-enter-from,
.drawer-slide-leave-to {
  transform: translateX(100%);
}
</style>