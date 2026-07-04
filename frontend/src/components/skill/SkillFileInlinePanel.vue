<script setup>
// SkillFileInlinePanel - 首页正文右侧内联的文件浏览器面板
//
// 2026-07-04 改:从 SkillFileDrawer 抽取出来,直接嵌入到 SkillsView 的正文区右侧,
// 不再走 Teleport + 抽屉动画,布局更紧凑(避免每次点右上角打开)。
//
// 复用 FileTreeView(目录树) + CodeViewer(渲染) + 父组件传 skill 用于保存。

import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import FileTreeView from './FileTreeView.vue'
import CodeViewer from './CodeViewer.vue'
import { updateSkill, getStoreInfo } from '@/api/skillbox/skills'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const props = defineProps({
  // 技能包文件列表 [{path, content}] - 来自后端 canonical.files
  files: { type: Array, default: () => [] },
  // 当前选中的 skill:{ name, version, scope, project_id, source, group_path, canonical }
  skill: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['saved'])

// 当前选中的文件
const selectedFile = ref(null)
watch(
  () => [props.files],
  () => {
    if (props.files && props.files.length) {
      const sk = props.files.find((f) => f.path === 'SKILL.md')
      // 默认不选 SKILL.md(它已在主区渲染),选第一个非 SKILL.md 文件
      const firstNonMd = props.files.find((f) => f.path !== 'SKILL.md')
      selectedFile.value = firstNonMd || sk || props.files[0] || null
    } else {
      selectedFile.value = null
    }
  },
  { immediate: true, deep: true },
)

function onSelectFile(file) {
  selectedFile.value = file
}

// SKILL.md 强制只读(避免与主区 Tiptap 编辑态冲突)
const isReadOnly = computed(() => selectedFile.value?.path === 'SKILL.md')

// 当前内容(暂不支持内联编辑,只展示;后续可加)
const currentContent = computed(() => selectedFile.value?.content || '')
const fileSize = computed(() => (currentContent.value || '').length)

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
watch(() => props.skill?.name, () => { if (!storeRoot.value) fetchStoreRoot() })

const skillRelPath = computed(() => {
  const gp = props.skill.group_path || ''
  return gp ? `${gp}/${props.skill.name || ''}` : (props.skill.name || '')
})

// 文件数量(去掉 SKILL.md,因为它已在主区)
const otherFilesCount = computed(() => (props.files || []).filter((f) => f.path !== 'SKILL.md').length)
</script>

<template>
  <div class="sfip">
    <header class="sfip-header">
      <IconPark icon="mdi:folder-multiple-outline" width="14" height="14" />
      <span class="sfip-title">{{ t('skills.fileBrowser.open') }}</span>
      <span class="sfip-count">{{ otherFilesCount }}</span>
    </header>

    <div class="sfip-body">
      <!-- 左:文件树(去掉 SKILL.md,因为它已在主区展示) -->
      <nav class="sfip-tree">
        <FileTreeView
          v-if="otherFilesCount"
          :files="(files || []).filter((f) => f.path !== 'SKILL.md')"
          :initial-selected-path="selectedFile?.path || ''"
          @select-file="onSelectFile"
        />
        <p v-else class="sfip-tree-empty">该技能包没有其它文件</p>
      </nav>

      <!-- 右:文件预览/编辑(暂只读,后续可加内联编辑) -->
      <main class="sfip-viewer">
        <header class="sfip-viewer-header">
          <span class="sfip-viewer-path">{{ selectedFile?.path || t('skills.fileBrowser.noFile') }}</span>
          <span v-if="selectedFile?.path" class="sfip-viewer-size">{{ fileSize }} B</span>
        </header>
        <CodeViewer
          v-if="selectedFile?.path"
          :key="selectedFile.path"
          :path="selectedFile.path"
          :content="currentContent"
          :editable="false"
          :store-root="storeRoot"
          :skill-rel-path="skillRelPath"
        />
        <div v-else class="sfip-empty">
          <IconPark icon="mdi:file-outline" width="32" height="32" />
          <p>暂无文件</p>
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
  border-left: 1px solid var(--border);
  background: var(--bg-subtle);
}
.sfip-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--border);
  font-size: 12px;
  color: var(--text-dim);
  background: var(--bg-card);
  flex-shrink: 0;
}
.sfip-title {
  flex: 1;
  font-weight: 500;
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
  width: 180px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  overflow: auto;
  padding: 6px 8px;
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
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 11px;
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
}
.sfip-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-faint);
  font-size: 12px;
}
</style>