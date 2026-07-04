<script setup>
// SkillFileDrawer - 首页技能文件浏览器抽屉。
//
// 2026-07-04 增:从 SkillsView 详情区的 [📁] 按钮打开,
// 展示技能包全文件树 + 单文件纯文本预览(后续 commit 会加 Monaco / markdown 渲染)。
//
// 设计要点:
//   - 不复用 Modal.vue(居中弹窗与"右侧抽屉"形态冲突)
//   - <Teleport to="body"> + position:fixed;right:0 + Transition 滑入
//   - 两栏:左 280px 文件树,右 flex:1 文件预览
//   - SKILL.md 在抽屉里默认选中,只展示纯文本(避免与主页 Tiptap 编辑态冲突)
//
// Commit 1 只做"目录树 + 纯文本预览",Monaco / 编辑 / 保存 / markdown 渲染
// 在后续 commit 加。

import { computed, ref, watch } from 'vue'
import IconPark from '@/components/IconPark.vue'
import FileTreeView from './FileTreeView.vue'

const props = defineProps({
  // v-model 控制开合
  modelValue: { type: Boolean, default: false },
  // 当前选中的 skill:{ name, version, scope, project_id, source, group_path }
  skill: { type: Object, default: () => ({}) },
  // 技能包文件列表 [{path, content}] - 来自后端 canonical.files
  files: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'closed'])

// 关闭抽屉
function close() {
  emit('update:modelValue', false)
  emit('closed')
}

// 点遮罩关闭(Commit 1 简化:不做 dirty 检测,后续 commit 加)
function onMaskClick() {
  close()
}

// ESC 关闭
function onKeydown(e) {
  if (e.key === 'Escape' && props.modelValue) {
    e.stopPropagation()
    close()
  }
}

// 当前选中的文件
const selectedFile = ref(null)

// 默认选中 SKILL.md
watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      const sk = (props.files || []).find((f) => f.path === 'SKILL.md')
      selectedFile.value = sk || props.files[0] || null
    }
  },
  { immediate: true },
)

function onSelectFile(file) {
  selectedFile.value = file
}

// 预览区显示的内容(Commit 1 简化:直接纯文本展示)
const previewContent = computed(() => selectedFile.value?.content || '')
const previewPath = computed(() => selectedFile.value?.path || '')

// 文件大小
const fileSize = computed(() => (previewContent.value || '').length)
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
                <span class="file-drawer-count">{{ (files || []).length }} files</span>
              </div>
              <button class="file-drawer-close" :aria-label="'关闭'" @click="close">
                <IconPark icon="mdi:close" width="18" height="18" />
              </button>
            </header>

            <!-- body:左树 + 右预览 -->
            <div class="file-drawer-body">
              <nav class="file-drawer-tree">
                <FileTreeView
                  :files="files"
                  :initial-selected-path="selectedFile?.path || 'SKILL.md'"
                  @select-file="onSelectFile"
                />
              </nav>
              <main class="file-drawer-viewer">
                <header class="file-drawer-viewer-header">
                  <IconPark :icon="previewPath ? 'mdi:file-document-outline' : 'mdi:file-outline'" width="14" height="14" />
                  <span class="file-drawer-viewer-path">{{ previewPath || '未选中文件' }}</span>
                  <span v-if="previewPath" class="file-drawer-viewer-size">{{ fileSize }} B</span>
                </header>
                <pre v-if="previewPath" class="file-drawer-preview"><code>{{ previewContent }}</code></pre>
                <div v-else class="file-drawer-empty">
                  <IconPark icon="mdi:file-outline" width="48" height="48" />
                  <p>从左侧选择一个文件查看</p>
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
.file-drawer-preview {
  flex: 1;
  margin: 0;
  padding: 16px 20px;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-all;
}
.file-drawer-preview code {
  font-family: inherit;
  background: transparent;
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