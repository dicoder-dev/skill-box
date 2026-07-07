<script setup>
// FileTreeNode - 技能包文件树节点(递归组件)
//
// 用于 SkillFileDrawer 内展示技能包的全部文件 / 目录。
// 与 TreeNode.vue 的差异:
//   - 不支持拖拽 / 右键菜单 / 工具 chip,纯只读浏览
//   - 支持任意深度嵌套(现有 TreeNode 2026-07-03 改单级拍平,这里不复用)
//   - 节点数据更简单:{ type: 'dir'|'file', name, path, children?, files?, size? }
//   - 文件按后缀分配 iconpark 图标(语言 / markdown / 配置 / 二进制)
//
// 2026-07-04 增:首页技能文件浏览器(Commit 1)。

import { computed } from 'vue'
import IconPark from '@/components/IconPark.vue'

defineOptions({ name: 'FileTreeNode' })

const props = defineProps({
  // 当前节点的 dirs 和 files
  dirs: { type: Array, default: () => [] },
  files: { type: Array, default: () => [] },
  // 当前选中文件的 path
  selectedPath: { type: String, default: '' },
  // 当前折叠的目录 path 集合(跨节点共享)
  collapsedPaths: { type: Object, default: () => new Set() },
  // 缩进级别(根为 0,每深一级 +1)
  depth: { type: Number, default: 0 },
  // dirty 标记的文件 path 集合(给节点加橙色圆点)
  dirtyPaths: { type: Object, default: () => new Set() },
})

const emit = defineEmits(['select-file', 'toggle-collapse'])

// 文件后缀 → iconpark 图标 mdi 字符串(映射走项目统一 iconpark,见 iconparkMap.js)
const FILE_ICON = {
  // 文档
  md: 'mdi:language-markdown-outline',
  markdown: 'mdi:language-markdown-outline',
  txt: 'mdi:file-document-outline',
  // 代码(走 Monaco 的语言)
  py: 'mdi:language-python',
  js: 'mdi:language-javascript',
  jsx: 'mdi:language-javascript',
  ts: 'mdi:language-typescript',
  tsx: 'mdi:language-typescript',
  go: 'mdi:language-go',
  rs: 'mdi:language-rust',
  java: 'mdi:language-java',
  rb: 'mdi:language-ruby',
  php: 'mdi:language-php',
  c: 'mdi:language-c',
  h: 'mdi:language-cpp',
  cpp: 'mdi:language-cpp',
  sh: 'mdi:language-bash',
  bash: 'mdi:language-bash',
  zsh: 'mdi:language-bash',
  sql: 'mdi:database-outline',
  // 配置 / 数据
  json: 'mdi:code-json',
  yaml: 'mdi:format-list-bulleted-square',
  yml: 'mdi:format-list-bulleted-square',
  toml: 'mdi:cog-outline',
  ini: 'mdi:cog-outline',
  xml: 'mdi:file-xml-box-outline',
  html: 'mdi:language-html5',
  css: 'mdi:language-css3',
  scss: 'mdi:language-css3',
  less: 'mdi:language-css3',
  // 二进制
  png: 'mdi:file-image-outline',
  jpg: 'mdi:file-image-outline',
  jpeg: 'mdi:file-image-outline',
  gif: 'mdi:file-image-outline',
  webp: 'mdi:file-image-outline',
  svg: 'mdi:file-image-outline',
  bmp: 'mdi:file-image-outline',
  pdf: 'mdi:file-pdf-box',
  zip: 'mdi:folder-zip-outline',
  tar: 'mdi:folder-zip-outline',
  gz: 'mdi:folder-zip-outline',
  tgz: 'mdi:folder-zip-outline',
  '7z': 'mdi:folder-zip-outline',
  rar: 'mdi:folder-zip-outline',
}

// 文件名 → iconpark 图标;未匹配走默认 file 图标
function fileIcon(fileName) {
  if (!fileName) return 'mdi:file-outline'
  const idx = fileName.lastIndexOf('.')
  if (idx <= 0) return 'mdi:file-outline'
  const ext = fileName.slice(idx + 1).toLowerCase()
  return FILE_ICON[ext] || 'mdi:file-outline'
}

// 大文件判定阈值(展示提示用,Monaco 加载控制由父组件负责)
const LARGE_FILE_BYTES = 1024 * 1024
function isLargeFile(size) {
  return typeof size === 'number' && size > LARGE_FILE_BYTES
}

// 目录是否折叠
function isCollapsed(node) {
  return props.collapsedPaths.has(node.path)
}

function toggleCollapse(node) {
  emit('toggle-collapse', node.path)
}

// 文件点击
function onClickFile(file, e) {
  if (e) e.stopPropagation()
  emit('select-file', file)
}

// 文件 dirty 检测
function isDirty(filePath) {
  return props.dirtyPaths.has(filePath)
}

// 过滤后的子节点:空目录不渲染(避免空 folder 节点干扰视觉)
const visibleDirs = computed(() => (props.dirs || []).filter((d) => (d.children || []).length + (d.files || []).length > 0))
</script>

<template>
  <ul class="file-tree" role="tree">
    <!-- 目录(递归展示) -->
    <li
      v-for="dir in visibleDirs"
      :key="dir.path"
      role="treeitem"
      :class="['file-tree-dir', isCollapsed(dir) ? 'file-tree-collapsed' : '']"
      :style="{ paddingLeft: `${depth * 12 + 4}px` }"
      :aria-expanded="!isCollapsed(dir)"
    >
      <div class="file-row file-row-dir" @click="toggleCollapse(dir)">
        <IconPark
          :icon="isCollapsed(dir) ? 'mdi:plus' : 'mdi:minus'"
          width="14"
          height="14"
          class="file-caret"
        />
        <IconPark
          :icon="isCollapsed(dir) ? 'mdi:folder-outline' : 'mdi:folder-open-outline'"
          width="16"
          height="16"
          class="file-dir-icon"
        />
        <span class="file-name">{{ dir.name }}</span>
        <span v-if="(dir.children || []).length + (dir.files || []).length" class="file-count">
          {{ (dir.children || []).length + (dir.files || []).length }}
        </span>
      </div>
      <!-- 递归:目录展开时展示其内部子目录与文件 -->
      <FileTreeNode
        v-if="!isCollapsed(dir)"
        :dirs="dir.children"
        :files="dir.files"
        :selected-path="selectedPath"
        :collapsed-paths="collapsedPaths"
        :dirty-paths="dirtyPaths"
        :depth="depth + 1"
        @select-file="onClickFile"
        @toggle-collapse="(p) => emit('toggle-collapse', p)"
      />
    </li>

    <!-- 文件 -->
    <li
      v-for="file in files"
      :key="file.path"
      role="treeitem"
      :class="[
        'file-tree-file',
        selectedPath === file.path ? 'file-tree-selected' : '',
      ]"
      :style="{ paddingLeft: `${depth * 12 + 4}px` }"
      :aria-selected="selectedPath === file.path"
    >
      <div class="file-row file-row-file" @click="onClickFile(file, $event)">
        <!-- 占位缩进:文件没有 caret,占 14px 让文件名对齐目录里的文件名 -->
        <span class="file-caret-placeholder" />
        <IconPark :icon="fileIcon(file.name)" width="16" height="16" class="file-file-icon" />
        <span class="file-name">{{ file.name }}</span>
        <span v-if="isLargeFile(file.size)" class="file-large-tip" :title="`${file.size} bytes`">大</span>
        <span v-if="isDirty(file.path)" class="file-dirty-dot" :title="'有未保存的修改'" />
      </div>
    </li>
  </ul>
</template>

<style scoped>
.file-tree {
  list-style: none;
  margin: 0;
  padding: 0;
}
.file-tree-dir,
.file-tree-file {
  list-style: none;
}
.file-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 4px 0;
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
  color: var(--text-dim);
  transition: background 120ms ease, color 120ms ease;
}
.file-row:hover {
  background: var(--bg-hover);
  color: var(--text);
}
.file-row-dir {
  font-weight: 500;
}
.file-caret {
  flex-shrink: 0;
  color: var(--text-faint);
}
.file-caret-placeholder {
  display: inline-block;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}
.file-dir-icon,
.file-file-icon {
  flex-shrink: 0;
  color: var(--text-faint);
}
.file-row-file:hover .file-file-icon {
  color: var(--accent-blue);
}
.file-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.file-count {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg-subtle);
  border-radius: 999px;
}
.file-tree-selected > .file-row {
  background: var(--accent-blue);
  color: white;
}
.file-tree-selected > .file-row:hover {
  background: var(--accent-blue);
}
.file-tree-selected > .file-row .file-file-icon {
  color: white;
}
.file-large-tip {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--accent-amber, #d97706);
  background: rgba(217, 119, 6, 0.12);
  padding: 1px 4px;
  border-radius: 4px;
}
.file-dirty-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-amber, #d97706);
}
</style>