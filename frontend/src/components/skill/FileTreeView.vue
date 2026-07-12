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

const emit = defineEmits([
  'select-file',
  // 2026-07-11 增:转发 3 种位置的右键事件
  // - context-menu-file  文件节点右键
  // - context-menu-folder 目录节点右键
  // - context-menu-root 树空白处右键(本组件自己绑)
  'context-menu-file',
  'context-menu-folder',
  'context-menu-root',
])

// 把扁平 files[] 构造成嵌套树。
// 数据结构:
//   root.dirs: [{ name, path, dirs: [...], files: [{name, path, size}] }]
//   root.files: [{ name, path, size }]
//
// 2026-07-04 增(Commit 7+):过滤掉 macOS 系统元数据文件(.DS_Store / ._*),
// 这些是 Finder 留下的,跟 skill 内容无关,展示出来干扰用户。
// 走"以 . 开头"为统一规则,顺手过滤 .git / .vscode 等其它隐藏文件。
//
// 2026-07-11 改:业务占位 .skillbox-placeholder 允许 buildDir 但不挂成 file —
// 用它建一个空目录时,父目录必须能 ensureDir 出来(否则空目录用户看不到)。
// 实现:用 业务占位白名单 BUSINESS_PLACEHOLDERS = {'.skillbox-placeholder'} 在两处分别处理:
//   - ensureDir 时:这种名字**不**按 . 开头跳过(让父目录能建)
//   - 文件循环时:这种名字整体 skip(用户视觉上不出现)
const BUSINESS_PLACEHOLDERS = new Set(['.skillbox-placeholder'])
function isBusinessPlaceholder(seg) {
  return BUSINESS_PLACEHOLDERS.has(seg)
}
function buildTree(files) {
  const root = { dirs: [], files: [] }
  // 用 path 前缀找 / 建中间目录
  const dirIndex = new Map() // path -> dirNode
  function ensureDir(fullPath) {
    if (!fullPath) return root
    if (dirIndex.has(fullPath)) return dirIndex.get(fullPath)
    const parentPath = fullPath.includes('/') ? fullPath.slice(0, fullPath.lastIndexOf('/')) : ''
    const name = fullPath.slice(fullPath.lastIndexOf('/') + 1)
    // 中间目录名过滤:仅过滤"非业务占位的 . 开头"名字(.git / .vscode 等空目录)
    // 业务占位 .skillbox-placeholder 仍建出父目录(否则新建的空目录不显示)。
    if (name.startsWith('.') && !isBusinessPlaceholder(name)) return root
    const parent = ensureDir(parentPath)
    const dirNode = { name, path: fullPath, dirs: [], files: [], children: [] }
    parent.dirs.push(dirNode)
    dirIndex.set(fullPath, dirNode)
    return dirNode
  }
  for (const f of files || []) {
    if (!f || !f.path) continue
    const parts = f.path.split('/')
    const fileName = parts[parts.length - 1]
    const dirPath = parts.length > 1 ? parts.slice(0, -1).join('/') : ''
    // 业务占位文件(比如 .skillbox-placeholder):仅用于让父目录在 buildTree 里
    // 被建出来(空目录要能显示),自身不挂成 file。先 ensureDir 父目录,再 continue。
    //
    // 2026-07-12 改:父目录 ensureDir 后,打 isEmpty=true 标记 —
    // FileTreeNode.visibleDirs 默认过滤 children+files==0 的空目录,
    // 没有这个标记,后端 listEmptyDirs 注入的占位条目会被自身循环 continue 掉,
    // 父目录 dirs[] 永远是 0 长度 → visibleDirs 把它过滤掉 → 视觉上还是看不到。
    // 标记后 visibleDirs 保留该 dirNode,渲染时因 files 为空自然没文件行,
    // 树里就是一个空的 folder 节点 — 跟磁盘一致。
    if (parts.some((seg) => isBusinessPlaceholder(seg))) {
      if (dirPath) {
        const dirNode = ensureDir(dirPath)
        dirNode.isEmpty = true
      }
      continue
    }
    // 过滤以 . 开头的隐藏文件(.DS_Store / ._* / .git 等)
    if (f.path.startsWith('.') || parts.some((seg) => seg.startsWith('.'))) continue
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
    // 2026-07-11 改:同步 children 别名 — FileTreeNode 模板递归用 :dirs="dir.children"
    // (历史遗留的字段名,跟 buildTree 输出的 dirs 不一致),不补会传 undefined,
    // 递归不渲染 → 子目录看不到。
    for (const d of n.dirs) {
      d.children = d.dirs
    }
  }
  sortNode(root)
  // 顶层 root 也要补 children
  root.children = root.dirs
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

// 2026-07-11 增:根区域(树空白处)右键 — 转发给父级,由 InlinePanel 弹
// "新建文件 / 新建目录"。注意:FileTreeNode 内部的文件/目录行已经 stopPropagation,
// 所以这个 handler 不会被子节点冒泡触发,只在用户点到真正的空白区域时触发。
function onRootContextMenu(e) {
  e.preventDefault()
  // 2026-07-11 增:诊断日志(确认根区域右键事件是否真的触达 handler)
  console.log('[FileTreeView] onRootContextMenu fired at', e.clientX, e.clientY, 'target=', e.target?.tagName, e.target?.className)
  emit('context-menu-root', { event: e })
}
</script>

<template>
  <!-- 2026-07-11 改:在 .file-tree-view 容器上加 @contextmenu,作为"根区域右键"
       入口(用户点到树空白处时触发);子节点文件/目录行已在 FileTreeNode
       内 stopPropagation 不会冒泡到这里。 -->
  <div class="file-tree-view" @contextmenu="onRootContextMenu">
    <FileTreeNode
      :dirs="tree.dirs"
      :files="tree.files"
      :selected-path="selectedPath"
      :collapsed-paths="collapsedPaths"
      :dirty-paths="dirtyPaths"
      :depth="0"
      @select-file="onSelectFile"
      @toggle-collapse="onToggleCollapse"
      @context-menu-file="(p) => emit('context-menu-file', p)"
      @context-menu-folder="(p) => emit('context-menu-folder', p)"
    />
  </div>
</template>

<style scoped>
/* 2026-07-11 改:.file-tree-view 撑满父容器高度,让"根区域右键"事件在用户点到
   文件树底部大片空白时也能触达容器的 @contextmenu。
   之前 padding: 4px 0 + 高度 auto 时,容器高度 = ul 子内容高度,父级
   .sfip-tree-wrap flex:1 撑满的剩余空间都在 .file-tree-view 之外 — 用户
   点那些空白时事件不冒泡到 .file-tree-view,右键菜单不弹。 */
.file-tree-view {
  padding: 4px 0;
  height: 100%;
  min-height: 100%;
  display: flex;
  flex-direction: column;
}
.file-tree-view > ul.file-tree {
  flex: 1;
}
</style>