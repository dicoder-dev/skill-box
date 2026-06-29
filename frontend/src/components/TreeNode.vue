<script setup>
/**
 * TreeNode 树节点递归组件
 *
 * 2026-06-29 增:用于首页 skill 列表的分组树形展示。
 *
 * 节点数据:
 *   - { is_group: true,  name, path, children: [...] }
 *   - { is_group: false, name, path, skill_meta: { name, version, description, ... } }
 *
 * 行为:
 *   - 分组:可展开/折叠(箭头 + 点击行)
 *   - skill:点击选中(emit select-skill),右键弹菜单(emit context-menu-skill)
 *   - 分组右键:emit context-menu-group
 *   - 根区域右键:emit context-menu-root
 *   - 拖拽:skill 可拖到分组 / 分组可拖到另一分组(emit drop),含视觉反馈
 *
 * 状态从外部 prop 传入(collapsedPaths 是个 Set,记录当前折叠的 path 列表),
 * 让父组件能跨节点共享展开状态(搜索时自动展开匹配路径用)。
 */
import { ref, computed } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  // 当前节点的 children 列表(从树根传入)
  nodes: { type: Array, default: () => [] },
  // 当前选中 skill 的 path(用于高亮)
  selectedPath: { type: String, default: '' },
  // 当前折叠的 path 集合(从父组件传入,跨节点共享)
  collapsedPaths: { type: Object, default: () => new Set() },
  // 当前正在被拖入的 path(用于视觉高亮)
  dropTargetPath: { type: String, default: '' },
  // 缩进级别(根为 0,每深一级 +1)
  depth: { type: Number, default: 0 },
  // 父路径(根为空)— 用于构建完整 path
  parentPath: { type: String, default: '' },
})

// 2026-06-29 增:显式声明组件 name,允许模板里 <TreeNode /> 自引用递归。
defineOptions({ name: 'TreeNode' })

const emit = defineEmits([
  'select-skill',
  'context-menu-skill',
  'context-menu-group',
  'context-menu-root',
  'drop',
  'toggle-collapse',
])

// 节点自身的完整 path helper
function fullPath(node) {
  if (!node) return ''
  return node.path || node.name
}

// 判断节点是否折叠
function isCollapsed(node) {
  return props.collapsedPaths.has(fullPath(node))
}

function toggleCollapse(node) {
  emit('toggle-collapse', fullPath(node))
}

// 选中 skill
function onClickSkill(node, e) {
  if (e) e.stopPropagation()
  emit('select-skill', node)
}

function onClickGroup(node, e) {
  if (e) e.stopPropagation()
  toggleCollapse(node)
}

// ====== 右键菜单 ======
function onContextMenu(node, e) {
  e.preventDefault()
  e.stopPropagation()
  if (node.is_group) {
    emit('context-menu-group', { node, event: e })
  } else {
    emit('context-menu-skill', { node, event: e })
  }
}

function onRootContextMenu(e) {
  e.preventDefault()
  emit('context-menu-root', { event: e })
}

// ====== 拖拽 ======
const dragCounter = ref(0) // 防止子元素 dragenter/leave 抖动

function onDragStart(node, e) {
  if (!e.dataTransfer) return
  const payload = JSON.stringify({
    type: node.is_group ? 'group' : 'skill',
    path: fullPath(node),
    name: node.name,
  })
  e.dataTransfer.setData('application/x-skillbox-node', payload)
  e.dataTransfer.effectAllowed = 'move'
}

function onDragEnterGroup(node, e) {
  e.preventDefault()
  e.stopPropagation()
  if (e.dataTransfer?.types.includes('application/x-skillbox-node')) {
    dragCounter.value++
    emit('drop', { target: node, event: e, hovering: true })
  }
}

function onDragLeaveGroup(e) {
  e.preventDefault()
  e.stopPropagation()
  dragCounter.value = Math.max(0, dragCounter.value - 1)
  if (dragCounter.value === 0) {
    emit('drop', { target: null, event: e, hovering: false })
  }
}

function onDragOverGroup(e) {
  e.preventDefault()
  e.stopPropagation()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
}

function onDropGroup(node, e) {
  e.preventDefault()
  e.stopPropagation()
  dragCounter.value = 0
  const raw = e.dataTransfer?.getData('application/x-skillbox-node')
  if (!raw) return
  try {
    const payload = JSON.parse(raw)
    emit('drop', { target: node, event: e, hovering: false, source: payload })
  } catch (_) { /* 解析失败就当作无效拖放 */ }
}

// 应用工具 chip 列表(给 skill 叶子用)
function toolShort(toolID) {
  if (!toolID) return '?'
  return toolID.charAt(0).toUpperCase() + toolID.slice(1)
}
const TOOL_ICON_MAP = {
  codex: 'mdi:console',
  claude: 'mdi:robot-outline',
  opencode: 'mdi:code-tags',
  cursor: 'mdi:cursor-default-click-outline',
  trae: 'mdi:leaf',
}
function toolIcon(tid) { return TOOL_ICON_MAP[tid] || 'mdi:puzzle-outline' }

// 是否 drop 目标 = 当前节点
function isDropTarget(node) {
  return props.dropTargetPath && props.dropTargetPath === fullPath(node)
}
</script>

<template>
  <ul class="tree" role="tree">
    <!-- 根区域右键(在 ul 空白处右键) -->
    <li
      v-if="depth === 0"
      class="tree-root-blank"
      @contextmenu="onRootContextMenu"
    >
      <span v-if="!nodes.length" class="tree-empty-hint">右键新建分组,或拖拽 skill 到此处</span>
    </li>

    <li
      v-for="node in nodes"
      :key="fullPath(node)"
      role="treeitem"
      :class="[
        'tree-node',
        node.is_group ? 'tree-node-group' : 'tree-node-skill',
        isCollapsed(node) ? 'tree-node-collapsed' : '',
        selectedPath === fullPath(node) ? 'tree-node-selected' : '',
        isDropTarget(node) ? 'tree-node-drop-target' : '',
      ]"
      :style="{ paddingLeft: `${depth * 14 + 4}px` }"
      :draggable="true"
      :aria-expanded="node.is_group ? !isCollapsed(node) : undefined"
      :aria-selected="!node.is_group && selectedPath === fullPath(node)"
      @dragstart="onDragStart(node, $event)"
    >
      <!-- 分组行:箭头 + 图标 + 名称 + 子项计数 -->
      <div
        v-if="node.is_group"
        class="tree-row tree-row-group"
        @click="onClickGroup(node, $event)"
        @contextmenu="onContextMenu(node, $event)"
        @dragenter="onDragEnterGroup(node, $event)"
        @dragleave="onDragLeaveGroup($event)"
        @dragover="onDragOverGroup($event)"
        @drop="onDropGroup(node, $event)"
      >
        <Icon
          :icon="isCollapsed(node) ? 'mdi:chevron-right' : 'mdi:chevron-down'"
          width="14"
          height="14"
          class="tree-caret"
        />
        <Icon
          :icon="isCollapsed(node) ? 'mdi:folder-outline' : 'mdi:folder-open-outline'"
          width="14"
          height="14"
          class="tree-group-icon"
        />
        <span class="tree-name tree-name-group">{{ node.name }}</span>
        <span v-if="(node.children || []).length" class="tree-count">
          {{ (node.children || []).length }}
        </span>
      </div>

      <!-- skill 行:无箭头,点击选中 -->
      <div
        v-else
        class="tree-row tree-row-skill"
        @click="onClickSkill(node, $event)"
        @contextmenu="onContextMenu(node, $event)"
        @dragenter="onDragEnterGroup(node, $event)"
        @dragleave="onDragLeaveGroup($event)"
        @dragover="onDragOverGroup($event)"
        @drop="onDropGroup(node, $event)"
      >
        <span class="tree-caret-spacer"></span>
        <Icon icon="mdi:bookmark-multiple-outline" width="13" height="13" class="tree-skill-icon" />
        <div class="tree-skill-main">
          <div class="tree-skill-head">
            <span class="tree-name tree-name-skill">{{ node.skill_meta?.name || node.name }}</span>
            <span v-if="node.skill_meta?.version" class="tree-version">@{{ node.skill_meta.version }}</span>
          </div>
          <div v-if="(node.skill_meta?.applied_tools || []).length" class="tree-skill-tools">
            <span
              v-for="tid in (node.skill_meta.applied_tools || [])"
              :key="tid"
              class="tree-tool-chip"
              :title="tid"
            >
              <Icon :icon="toolIcon(tid)" width="10" height="10" />
              <span>{{ toolShort(tid) }}</span>
            </span>
          </div>
        </div>
      </div>

      <!-- 递归子节点(仅分组,展开时) -->
      <TreeNode
        v-if="node.is_group && !isCollapsed(node) && (node.children || []).length"
        :nodes="node.children"
        :selected-path="selectedPath"
        :collapsed-paths="collapsedPaths"
        :drop-target-path="dropTargetPath"
        :depth="depth + 1"
        :parent-path="fullPath(node)"
        @select-skill="(n) => emit('select-skill', n)"
        @context-menu-skill="(p) => emit('context-menu-skill', p)"
        @context-menu-group="(p) => emit('context-menu-group', p)"
        @context-menu-root="(p) => emit('context-menu-root', p)"
        @drop="(p) => emit('drop', p)"
        @toggle-collapse="(p) => emit('toggle-collapse', p)"
      />
    </li>
  </ul>
</template>

<style scoped>
.tree {
  list-style: none;
  margin: 0;
  padding: 0;
}

.tree-root-blank {
  padding: 8px 12px 4px;
  color: var(--text-faint);
  font-size: 11px;
}
.tree-empty-hint { font-style: italic; }

.tree-node {
  position: relative;
  user-select: none;
  list-style: none;
}
.tree-node-collapsed > .tree-row { /* 折叠态视觉无变化,只是隐藏子树 */ }

.tree-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 0.1s ease, color 0.1s ease, border-color 0.1s ease;
  min-height: 26px;
  border: 1px solid transparent;
}
.tree-row:hover {
  background: var(--bg-hover);
}
.tree-caret {
  color: var(--text-faint);
  flex-shrink: 0;
}
.tree-caret-spacer {
  display: inline-block;
  width: 14px;
  flex-shrink: 0;
}
.tree-group-icon, .tree-skill-icon {
  color: var(--text-dim);
  flex-shrink: 0;
}

.tree-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.tree-name-group { color: var(--text); font-weight: 500; }
.tree-name-skill { color: var(--text); font-weight: 500; }

.tree-count {
  font-size: 11px;
  color: var(--text-faint);
  background: var(--bg-subtle);
  padding: 1px 6px;
  border-radius: 999px;
  flex-shrink: 0;
}

.tree-skill-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.tree-skill-head {
  display: flex;
  align-items: baseline;
  gap: 4px;
  min-width: 0;
}
.tree-version {
  font-size: 10px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  flex-shrink: 0;
}
.tree-skill-tools {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.tree-tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 5px;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
  font-size: 10px;
  line-height: 1;
}

.tree-node-selected > .tree-row {
  background: var(--bg-card);
  border-color: var(--accent-blue);
}
.tree-node-selected > .tree-row .tree-name { color: var(--accent-blue); }

/* 拖入目标高亮 */
.tree-node-drop-target > .tree-row {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue);
  border-style: dashed;
  border-width: 1px;
}
</style>
