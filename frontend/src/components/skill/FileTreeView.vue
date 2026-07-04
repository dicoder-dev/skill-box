<script setup>
// FileTreeView - 技能包文件树容器(单层)。
//
// 把后端返回的 files[] 扁平数组([{path, content}])构造成嵌套树,然后交给
// FileTreeNode 递归渲染。自身只负责:
//   - buildTree 构造树数据
//   - collapsedPaths / selectedPath 状态管理
//   - 向父组件 emit select-file 事件
//
// 2026-07-04 增:首页技能文件浏览器(Commit 1)。

import { computed, ref } from 'vue'
import FileTreeNode from './FileTreeNode.vue'

const props = defineProps({
  // [{path, content}] - 来自后端 getSkill({full:true}).canonical.files
  files: { type: Array, default: () => [] },
  // 初始选中的文件 path(默认选 SKILL.md)
  initialSelectedPath: { type: String, default: '' },
  // dirty 文件 path 集合(可选)
  dirtyPaths: { type: Object, default: () => new Set() },
})

const emit = defineEmits(['select-file'])

// 把扁平 files[] 构造成嵌套树。
// 数据结构:
//   root.dirs: [{ name, path, dirs: [...], files: [{name, path, size}] }]
//   root.files: [{ name, path, size }]
function buildTree(files) {
  const root = { dirs: [], files: [] }
  // 用 path 前缀找 / 建中间目录
  const dirIndex = new Map() // path -> dirNode
  function ensureDir(fullPath) {
    if (!fullPath) return root
    if (dirIndex.has(fullPath)) return dirIndex.get(fullPath)
    const parentPath = fullPath.includes('/') ? fullPath.slice(0, fullPath.lastIndexOf('/')) : ''
    const name = fullPath.slice(fullPath.lastIndexOf('/') + 1)
    const parent = ensureDir(parentPath)
    const dirNode = { name, path: fullPath, dirs: [], files: [] }
    parent.dirs.push(dirNode)
    dirIndex.set(fullPath, dirNode)
    return dirNode
  }
  for (const f of files || []) {
    if (!f || !f.path) continue
    const parts = f.path.split('/')
    const fileName = parts[parts.length - 1]
    const dirPath = parts.length > 1 ? parts.slice(0, -1).join('/') : ''
    const parent = ensureDir(dirPath)
    parent.files.push({
      name: fileName,
      path: f.path,
      size: (f.content || '').length,
    })
  }
  // 排序:目录在前,文件在后,均按字母序
  function sortNode(n) {
    n.dirs.sort((a, b) => a.name.localeCompare(b.name))
    n.files.sort((a, b) => a.name.localeCompare(b.name))
    n.dirs.forEach(sortNode)
  }
  sortNode(root)
  return root
}

const tree = computed(() => buildTree(props.files))

// 选中态(默认选 SKILL.md)
const selectedPath = ref(props.initialSelectedPath || 'SKILL.md')

// 折叠态:路径 Set,跨 FileTreeNode 递归共享
const collapsedPaths = ref(new Set())

function onSelectFile(file) {
  selectedPath.value = file.path
  emit('select-file', file)
}

function onToggleCollapse(p) {
  const s = new Set(collapsedPaths.value)
  if (s.has(p)) s.delete(p)
  else s.add(p)
  collapsedPaths.value = s
}
</script>

<template>
  <div class="file-tree-view">
    <FileTreeNode
      :dirs="tree.dirs"
      :files="tree.files"
      :selected-path="selectedPath"
      :collapsed-paths="collapsedPaths"
      :dirty-paths="dirtyPaths"
      :depth="0"
      @select-file="onSelectFile"
      @toggle-collapse="onToggleCollapse"
    />
  </div>
</template>

<style scoped>
.file-tree-view {
  padding: 4px 0;
}
</style>