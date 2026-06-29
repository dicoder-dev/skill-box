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
 * 节点数据:
 *   - { is_group: true,  name, path, children: [...] }
 *   - { is_group: false, name, path, skill_meta: { name, version, description, applied_tools, ... } }
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
 *
 * 2026-06-29 改:skill 叶子改"卡片"样式 — 明显的圆角边框、内边距、阴影,
 *   卡片内部分两行:头(name + @version + description)、尾(工具调用小标题 + 工具 chip 列表);
 *   折叠时缩进(分组)不影响 skill 卡片视觉。
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

// 2026-06-29 改:onRootContextMenu 删除(根区域右键事件已上移到 SkillsView 的
// .tree-container 元素上 — 那里覆盖整个左侧,无论是否有节点 / 折叠)。
// @context-menu-root emit 仍保留在 defineEmits 里(供父组件透传 / 调试用),
// 但不再有 emit 触发者。

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
    <!-- 2026-06-29 改:删除原 .tree-root-blank 占位 li(根区域右键事件已上移到
         SkillsView 的 .tree-container 元素上 — 那里覆盖整个左侧,无论是否有节点 / 折叠) -->

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

      <!-- skill 行:卡片样式 — 点击选中;卡片下方显示已被哪些工具全局启用 -->
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

/* =====================================================
   skill 卡片样式(2026-06-29 改:从"行"强化为"卡片")
   保留改动前的内容(icon + name + @version + 工具 chip 列表),
   仅加强容器视觉(圆角 / 边框 / 阴影 / 内边距),不改字段。
   ===================================================== */
.tree-row-skill {
  /* 比 .tree-row 更厚的内边距 + 圆角 + 边框,体现"卡片"感 */
  padding: 8px 10px;
  margin: 2px 4px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  transition: border-color 0.12s ease, transform 0.12s ease, box-shadow 0.12s ease;
  cursor: pointer;
}
.tree-row-skill:hover {
  border-color: var(--text-faint);
  transform: translateY(-1px);
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.08);
}
.tree-row-skill:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 1px;
}

.tree-skill-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tree-skill-head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.tree-skill-head .tree-skill-icon {
  color: var(--text-dim);
  flex-shrink: 0;
}
.tree-skill-head .tree-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
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

/* 选中态(skill 卡片):蓝色边框 */
.tree-node-selected > .tree-row-skill {
  border-color: var(--accent-blue);
  background: var(--bg-card);
  box-shadow: 0 0 0 1px var(--accent-blue);
}
.tree-node-selected > .tree-row-skill:hover {
  border-color: var(--accent-blue);
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
