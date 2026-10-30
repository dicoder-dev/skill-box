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
 *   - { is_group: false, name, path, skill_meta: { name, version, description, applied_agents, ... } }
 *
 * 行为:
 *   - 分组:可展开/折叠(箭头 + 点击行)
 *   - skill:点击选中(emit select-skill),右键弹菜单(emit context-menu-skill)
 *   - 分组右键:emit context-menu-group
 *   - 根区域右键:emit context-menu-root
 *   - 拖拽:@dragstart 把 source payload 塞到 dataTransfer;@drop 由父级 .tree-container
 *     统一处理(用 document.elementsFromPoint 定位鼠标下的目标 group),见 commit
 *     下一条。TreeNode 内部不再 bind drop 事件。
 *
 * 状态从外部 prop 传入(collapsedPaths 是个 Set,记录当前折叠的 path 列表),
 * 让父组件能跨节点共享展开状态(搜索时自动展开匹配路径用)。
 *
 * 2026-06-29 改:skill 叶子改"卡片"样式 — 明显的圆角边框、内边距、阴影,
 *   卡片内部分两行:头(name + @version + description)、尾(Agent调用小标题 + Agent chip 列表);
 *   折叠时缩进(分组)不影响 skill 卡片视觉。
 */
import { computed } from 'vue'
import IconPark from '@/components/IconPark.vue'
import { plainT } from '@/core/i18n/index.js'

// 2026-07-13 增:TreeNode 文案走 i18n(避免在 template 内联 t(),用常量 key)
// 跟 SkillScopePanel / SkillFileInlinePanel 同样的 plainT 兜底策略。
const t = (key, values) => plainT(key, values)

const props = defineProps({
  // 当前节点的 children 列表(从树根传入)
  nodes: { type: Array, default: () => [] },
  // 当前选中 skill 的 path(用于高亮)
  selectedPath: { type: String, default: '' },
  // 当前折叠的 path 集合(从父组件传入,跨节点共享)
  collapsedPaths: { type: Object, default: () => new Set() },
  // 当前正在被拖入的 path(用于视觉高亮)— 2026-06-30 改:还保留 prop,
  // 但 SkillsView 不再通过它驱动高亮(改用 elementsFromPoint 实时算)。
  // 保留 prop 是为了不破坏未来其它场景(如搜索时显示"匹配 drop 目标")。
  dropTargetPath: { type: String, default: '' },
  // 缩进级别(根为 0,每深一级 +1)
  depth: { type: Number, default: 0 },
  // 父路径(根为空)— 用于构建完整 path
  parentPath: { type: String, default: '' },
  // 2026-07-03 增:Agent元数据映射 { [agent_id]: { mdi_icon, icon_file, display_name, ... } }
  // 用于把首页 skill 树 chip 前的图标从硬编码 mdi 改成"icon_file 优先 + mdi 兜底",
  // 跟 AgentsView 的 AgentIcon 组件保持一致的视觉。
  // 父组件(SkillsView)传整个 useAgentsStore.items,TreeNode 内部按 agent_id 查。
  agentsById: { type: Object, default: () => ({}) },
})

// 2026-07-03 改:首页分组只支持单级,走单级拍平函数,把后端可能仍返回的
// 嵌套 children 拍平到一层(后端没动,前端兜底)。computed 缓存避免每次
// 响应式触发都重算;nodes 变化时才失效。
const displayNodes = computed(() => flattenSingleLevel(props.nodes))

// 2026-06-29 增:显式声明组件 name,允许模板里 <TreeNode /> 自引用递归。
defineOptions({ name: 'TreeNode' })

const emit = defineEmits([
  'select-skill',
  'context-menu-skill',
  'context-menu-group',
  'context-menu-root',
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
// 2026-06-30 改:完全移除 @drop / @dragenter / @dragleave / @dragover 绑定。
// 之前 5 次 commit 在"DOM 冒泡 + TreeNode 路由 drop"上反复打补丁,每次都出新 bug。
// 现在把 drop 路由**唯一化到 .tree-container**(在 SkillsView),用 document.elementsFromPoint
// 实时判断鼠标下到底是哪个 group 节点。TreeNode 内部不再关心 drop,只负责:
//   1. 标记每个 .tree-row 的 path(给 .tree-container 的 drop handler 用)
//   2. @dragstart 把 source payload 塞到 dataTransfer
//
// data-node-path 是关键 — elementsFromPoint 返回的 z-stack 数组里,
// 第一个带这个属性的元素就是"鼠标下最顶层的 group/skill 节点"。

function onDragStart(node, e) {
  if (!e.dataTransfer) return
  const payload = JSON.stringify({
    type: node.is_group ? 'group' : 'skill',
    path: fullPath(node),
    name: node.name,
  })
  e.dataTransfer.setData('application/x-skillbox-node', payload)
  e.dataTransfer.effectAllowed = 'move'
  // 用透明 dragImage,让默认的"卡片副本"不显示 — 默认那个半透明副本
  // 会跟目标位置视觉冲突,用户看着累。W3C 推荐做法。
  const ghost = document.createElement('div')
  ghost.style.cssText = 'position:fixed;top:-1000px;width:1px;height:1px;'
  document.body.appendChild(ghost)
  e.dataTransfer.setDragImage(ghost, 0, 0)
  setTimeout(() => ghost.remove(), 0)
}

// 2026-07-17 增:图标栈的悬浮提示 — 显示完整的可读名(优先 display_name,否则 agent_id),
// 避免用户看到重叠图标想确认"这是哪个Agent"时无处可查。
function agentFullTitle(agentID) {
  const meta = props.agentsById?.[agentID]
  if (meta?.display_name) return meta.display_name
  return agentID
}
const TOOL_ICON_MAP = {
  codex: 'mdi:console',
  claude: 'mdi:robot-outline',
  opencode: 'mdi:code-tags',
  cursor: 'mdi:cursor-default-click-outline',
  trae: 'mdi:leaf',
}
function agentIcon(tid) { return TOOL_ICON_MAP[tid] || 'mdi:puzzle-outline' }

// 2026-07-11 增:chip 按Agent分色 — key 与 TOOL_ICON_MAP 同步,
// value 是 style.css 已有的 accent 名(浅色 / 暗黑模式自动切换)。
// opencode=紫仅作用于 chip 强调色,不会蔓延到卡片主体,符合
// memory:avoid-violet-as-primary-color.md "紫色 AI 感强,主色禁用"。
const TOOL_ACCENT = {
  codex: 'accent-blue',     // 蓝 — 编程 AI
  claude: 'accent-emerald',  // 翠 — AI 助手
  cursor: 'accent-amber',    // 橙 — 编辑器
  opencode: 'accent-violet', // 紫 — 开源 CLI(chip only,克制)
  trae: 'accent-rose',       // 玫 — 字节 IDE
}

// 2026-07-03 增:agent 的 icon_file 优先级 > mdi。
// chip 上有 icon_file 就走 <img :src="/api/files/agent-icons/<basename>">,
// 否则走原来 agentIcon(tid) 拿 mdi 字符串。
function agentIconFile(tid) {
  const t = props.agentsById && props.agentsById[tid]
  return t && t.icon_file ? t.icon_file : ''
}

// 给 chip 的 <img> 拼绝对 URL(同 AgentIcon 组件的逻辑,baseURL 由 main.js 注入)
function resolveBaseURL() {
  if (typeof window === 'undefined') return ''
  const cfg = window.__APP_CONFIG__
  if (cfg && typeof cfg.baseURL === 'string') return cfg.baseURL.replace(/\/$/, '')
  if (window.location) return `${window.location.protocol}//${window.location.host}`
  return ''
}
function agentIconURL(name) {
  if (!name) return ''
  return `${resolveBaseURL()}/api/files/agent-icons/${name}`
}

// 是否 drop 目标 = 当前节点
function isDropTarget(node) {
  return props.dropTargetPath && props.dropTargetPath === fullPath(node)
}

// 2026-07-12 改:全局 Agent 判定 — 看 skill 的 source_path 磁盘绝对路径是否
// 命中 ~/.agents/skills/ 段(用户家目录下的全局 skills 池,所有Agent可自动读取)。
// source_path 是后端 ListTree 在 SkillTreeMeta.source_path 字段注入的:
// skill 从 ~/.agents/skills/<name>/ 导入到 ~/.skill-box/skills/ 时,后端
// importOneFromDir 会把原始目录 EvalSymlinks 后的真实路径写到
// <store>/<name>/.skillbox-source.json sidecar,buildTreeNode 读它注入。
// 用正则 [\\/]\.agents[\\/]skills[\\/] 跨平台匹配:
//   - macOS / Linux: `/Users/x/.agents/skills/xxx` 或 `/home/x/.agents/skills/xxx`
//   - Windows:       `C:\Users\x\.agents\skills\xxx`
// 不能用 startsWith('~/.agents/...'),因为后端给的是 EvalSymlinks 后的真实
// 绝对路径(macOS 是 /private/var/.../~/.agents/skills/...),而 ~ 这种 shell
// 缩写形式只出现在用户视角,不在 store 落盘数据里。
function isGlobalAgent(node) {
  if (!node || node.is_group) return false
  const src = node.skill_meta?.source_path || ''
  if (!src) return false
  return /[\\/]\.agents[\\/]skills[\\/]/.test(src)
}

// 2026-07-17 增:卡片前的「图标栈」— 把原来底部完整Agent名 chip 列表收敛成
// head 最左侧的重叠Agent图标,节省卡片纵向空间,让列表更简洁。
// 规则(用户需求):
//   - 全局 Agent:只显示当前默认图标(mdi:earth,emerald),不叠Agent图标;
//   - 没有任何Agent启用:显示默认图标(mdi:puzzle-outline);
//   - 有Agent启用:用各Agent自己的图标横向重叠排列,最多显示 3 个;
//   - 启用Agent > 3 个:前 2 个显示Agent图标,第 3 个位置显示「+N」表示还有更多。
// 每个栈元素结构:
//   { key, kind:'default'|'agent'|'more', icon?, img?, title?, accent? }
const MAX_ICON_STACK = 3
function iconStack(node) {
  // 全局 Agent:单独一个 earth 图标,语义已足够,不再叠Agent图标
  if (isGlobalAgent(node)) {
    return [{
      key: '__global__',
      kind: 'default',
      icon: 'mdi:earth',
      accent: 'is-global-agent',
      title: t('skills.treeNode.globalAgentTip'),
    }]
  }
  const agents = node.skill_meta?.applied_agents || []
  // 无任何Agent启用:回退默认拼图图标
  if (!agents.length) {
    return [{ key: '__default__', kind: 'default', icon: 'mdi:puzzle-outline' }]
  }
  // 有Agent启用:构造Agent图标栈
  const stack = []
  // 超过上限时,前 (MAX-1) 个显示真实Agent图标,最后一格留给「+N」
  const showCount = agents.length > MAX_ICON_STACK ? MAX_ICON_STACK - 1 : agents.length
  for (let i = 0; i < showCount; i++) {
    const tid = agents[i]
    stack.push({
      key: tid,
      kind: 'agent',
      img: agentIconFile(tid) ? agentIconURL(agentIconFile(tid)) : '',
      icon: agentIcon(tid),
      accent: TOOL_ACCENT[tid] || 'default',
      title: agentFullTitle(tid),
    })
  }
  // 溢出:最后一格显示「+N」,hover 提示剩余Agent完整名
  if (agents.length > MAX_ICON_STACK) {
    const rest = agents.slice(showCount)
    stack.push({
      key: '__more__',
      kind: 'more',
      more: rest.length,
      title: rest.map((tid) => agentFullTitle(tid)).join('、'),
    })
  }
  return stack
}
</script>

<template>
  <ul class="tree" role="tree" :style="{ '--drop-hint': t('skills.treeNode.dropHere') }">
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
      :aria-expanded="node.is_group ? !isCollapsed(node) : undefined"
      :aria-selected="!node.is_group && selectedPath === fullPath(node)"
    >
      <!-- 分组行:箭头 + 图标 + 名称 + 子项计数。
           2026-06-30 改:draggable + @dragstart 移到 .tree-row 自身,不再用 <li> 整个 draggable。
           原因:之前 <li> draggable 时,鼠标在 li 内任意子元素上 mousedown 都会触发 dragstart,
           用户从 code-review 划到 aa group 行时,aa 行的 dragstart 会再次触发并重置 dataTransfer,
           导致拖动源从 skill 变成 group,误触发 moveIntoDescendant / alreadyAtRoot。
           现在 dragstart 只在用户实际点中的那个 .tree-row 上触发,跨行 hover 不会重置。 -->
      <!-- 2026-07-10 改 v3:删除 group 行最左侧的 + / − caret 图标(用户反馈图标多余),
           folder 图标左移到 li 容器左边缘,跟子 skill 卡片左边缘对齐(子 skill 有
           margin-left: 4 抵消 li padding,group 没有 margin,所以反向吸收 li padding)。
           根 group folder 起点 = 4(li) + 8(.row padding) + 0(spacer 宽) - 12(margin) = 0
           根 skill 卡片起点 = 4(li) + 0(.row padding,被 skill margin 抵消) - 4(skill margin) = 0
           深度 N 的 group/skill 同步右移 14px,父子关系清晰。
           点击行展开/折叠的行为保留(.tree-row-group 整体 @click)。 -->
      <div
        v-if="node.is_group"
        class="tree-row tree-row-group"
        :data-node-path="fullPath(node)"
        :data-node-is-group="1"
        :data-node-parent-path="parentPath"
        draggable="true"
        @click="onClickGroup(node, $event)"
        @contextmenu="onContextMenu(node, $event)"
        @dragstart="onDragStart(node, $event)"
      >
        <span class="tree-caret-spacer" aria-hidden="true" />
        <IconPark
          :icon="isCollapsed(node) ? 'mdi:folder-outline' : 'mdi:folder-open-outline'"
          width="18"
          height="18"
          class="tree-group-icon"
        />
        <span class="tree-name tree-name-group">{{ node.name }}</span>
        <span v-if="(node.children || []).length" class="tree-count">
          {{ (node.children || []).length }}
        </span>
      </div>

      <!-- skill 行:卡片样式 — 点击选中;卡片下方显示已被哪些Agent全局启用。
           2026-06-30 改:draggable + @dragstart 放到 .tree-row-skill 自己(同 .tree-row-group)。 -->
      <div
        v-else
        class="tree-row tree-row-skill"
        :data-node-path="fullPath(node)"
        :data-node-is-group="0"
        :data-node-parent-path="parentPath"
        draggable="true"
        @click="onClickSkill(node, $event)"
        @contextmenu="onContextMenu(node, $event)"
        @dragstart="onDragStart(node, $event)"
      >
        <div class="tree-skill-main">
          <div class="tree-skill-head">
            <!-- 2026-07-17 改:卡片前的图标从「单个 skill 类型图标」升级为「Agent图标栈」。
                 - 全局 Agent:单个 earth 图标(emerald);
                 - 无Agent启用:默认拼图图标;
                 - 有Agent启用:各Agent真实图标横向重叠,最多 3 个,超出末位显「+N」。
                 重叠靠 margin-left 负值实现,后一个盖在前一个上,视觉更紧凑。
                 移除了原底部完整Agent名 chip 列表(tree-skill-agents),节省卡片空间。 -->
            <div class="tree-skill-iconstack" :title="iconStack(node).length === 1 ? iconStack(node)[0].title : ''">
              <span
                v-for="(it, idx) in iconStack(node)"
                :key="it.key"
                class="tree-icon-slot"
                :class="[
                  it.kind === 'default' ? 'is-default' : '',
                  it.kind === 'more' ? 'is-more' : '',
                  it.accent ? 'accent-' + it.accent : '',
                ]"
                :style="{ zIndex: iconStack(node).length - idx }"
                :title="it.title || ''"
              >
                <!-- 溢出「+N」 -->
                <span v-if="it.kind === 'more'" class="tree-icon-more">+{{ it.more }}</span>
                <!-- Agent真实 logo -->
                <img
                  v-else-if="it.img"
                  :src="it.img"
                  :alt="it.key"
                  width="12"
                  height="12"
                  class="tree-icon-img"
                />
                <!-- 兜底 mdi 图标(默认图标 / Agent无 icon_file) -->
                <IconPark v-else :icon="it.icon" width="13" height="13" />
              </span>
            </div>
            <span class="tree-name tree-name-skill">{{ node.skill_meta?.name || node.name }}</span>
            <!-- 2026-10-30 v2 改:版本号重新放回列表项(2026-07-16 曾移除)。
                 当时移除的原因是栏宽 320→260 太挤,名字+版本号挤在一行。
                 现在栏宽收窄到 224 + 行改成扁平行后,行内反而空出一大块,
                 用户反馈"两侧空白太多"。
                 用版本号填掉这段空白 = 顶级 source list 惯例
                 (VS Code 资源管理器在文件右侧显示大小/git 状态),
                 既消除死白又提供真实信息,不是纯装饰。 -->
            <span v-if="node.skill_meta?.version" class="tree-version">@{{ node.skill_meta.version }}</span>
          </div>
        </div>
      </div>

      <!-- 递归子节点(仅分组,展开时)。
           2026-07-08 修:必须把 :agents-by-id 也透传,否则子分组下的 skill 节点
           拿不到 agentsById,agentIconFile(tid) 永远拿不到 icon_file → 退化到
           硬编码 mdi 兜底,显示不出"数据库配置的图标"。 -->
      <TreeNode
        v-if="node.is_group && !isCollapsed(node) && (node.children || []).length"
        :nodes="node.children"
        :selected-path="selectedPath"
        :collapsed-paths="collapsedPaths"
        :drop-target-path="dropTargetPath"
        :agents-by-id="agentsById"
        :depth="depth + 1"
        :parent-path="fullPath(node)"
        @select-skill="(n) => emit('select-skill', n)"
        @context-menu-skill="(p) => emit('context-menu-skill', p)"
        @context-menu-group="(p) => emit('context-menu-group', p)"
        @context-menu-root="(p) => emit('context-menu-root', p)"
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
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 0.1s ease, color 0.1s ease, border-color 0.1s ease;
  min-height: 24px;
  border: 1px solid transparent;
}
/* 2026-06-29 改:分组行加大,容纳更大的 icon(18px) + 加粗 name。
   2026-10-30 改:紧凑化 — min-height 32→26,padding 5→3,
   分组行只是导航骨架,不该跟内容行抢纵向空间。 */
.tree-row-group {
  min-height: 26px;
  padding: 3px 8px;
}
.tree-row:hover {
  background: var(--bg-hover);
}
/* 2026-07-10 改 v3:用户反馈根 group 和根 skill 应同垂直线,但当前根 group folder
   向右缩进。根因:.tree-row-skill 有 margin-left: 4 抵消 li padding,根 skill
   实际起点 = 0;group 没有 margin,folder 起点 = li padding(4) + .row padding(8)
   + 原本 spacer = 20,跟根 skill 差了 20px。

   重算:
   - 根 skill 卡片实际起点 = 0px(li paddingLeft=4 + skill margin-left=4 = 抵消)
   - 子 skill 卡片(depth=1)实际起点 = 14px(li paddingLeft=18 - margin=4)
   - 同理,根 group folder 应该 = 0px,深度 N 的 group folder = 14*N px
   - 路径:spacer width 改 0,加 margin-left: -12px 吸收 li padding(4) + .row padding(8)
   - 验证:root folder 起点 = 4(li) + 8(.row padding) + 0(spacer) - 12(margin) = 0 ✓
           depth=1 group folder 起点 = 18(li) + 8 + 0 - 12 = 14 = 子 skill 起点 ✓
*/
.tree-caret-spacer {
  display: inline-block;
  width: 0; /* 占位宽度归零,folder 起始位置由 margin-left 决定 */
  margin-left: -12px; /* 吸收 li padding(4) + .tree-row padding(8) */
  flex-shrink: 0;
}
.tree-group-icon {
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
.tree-name-group {
  /* 2026-06-29 改:分组名加大到 14px + 加粗 600,与 skill 卡片视觉对等
     2026-10-30 改:14→13,分组名跟 skill 名拉开层级靠字重(600)而不是字号,
     字号一样时上下行高更紧凑,扫读也更快。 */
  color: var(--text);
  font-weight: 600;
  font-size: 13px;
}
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
   skill 卡片样式
   2026-10-30 重做(用户反馈左栏"有点丑" + 布局要紧凑):
   旧版是"渐变卡顶 + 三层蓝色光晕阴影 + 10px 外边距"的浮夸卡片风,
   在窄侧栏里 7 个技能就撑满整列,视觉噪音大、密度低。
   改成 Linear / VS Code / Arc 那一套"扁平行"范式:
     - 去掉渐变、去掉所有 box-shadow、去掉浮起位移
     - 默认无边框无底色,靠留白分组
     - hover 才出中性底色(--bg-hover)
     - 选中 = 淡强调底 + 左侧 2px 色条 + 强调色文字
       (经典 source list 选中范式,不靠光晕)
     - 外边距 10px→3px,内边距 8px→5px,行更紧凑
   保留能力:拖拽落点高亮(dashed + 蓝底)语义不变。
   ===================================================== */
.tree-row-skill {
  --card-accent: var(--accent-blue);
  /* 2026-10-30 v2 改:行内更紧凑 —— 内边距 5/10 → 4/8,
     外边距 3/4 → 2/2,图标槽 16 → 14px。
     用户反馈"技能列表行内空白太多":名字普遍很短(code-review),
     原来图标 16px + gap 6px + padding 10px 一套下来,名字右侧能空出 100px+。
     这里把行内元素本身收紧,让"内容区"跟着名字走,右侧不再是大片死白。 */
  padding: 4px 8px;
  margin: 2px 2px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  transition: background-color 0.12s ease, border-color 0.12s ease, color 0.12s ease;
  cursor: pointer;
  position: relative;
}
.tree-row-skill:hover {
  background: var(--bg-hover);
  border-color: transparent;
}
.tree-row-skill:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: -1px;
}

.tree-skill-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.tree-skill-head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
/* 2026-07-17 改:tree-skill-icon 系列样式退役,替换为「Agent图标栈」样式。
   图标栈:head 最左侧一排重叠的圆形图标(Agent logo / 默认图标 / +N)。 */
.tree-skill-iconstack {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}
/* 2026-07-17 改:去掉圆圈轮廓 — 用户反馈图标外的圆圈描边/底色多余。
   现在是纯图标重叠,无 border / background / border-radius。 */
.tree-icon-slot {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  /* 2026-10-30 v2 改:16 → 14px,配合模板里的 svg/img 尺寸一起收小 */
  width: 14px;
  height: 14px;
  color: var(--text-dim);
  box-sizing: border-box;
}
/* 第二个及之后的槽向左重叠,盖在前一个上(zIndex 在模板里按序递减) */
.tree-icon-slot + .tree-icon-slot {
  margin-left: -4px;
}
/* 2026-10-30 v2 改:图标槽 16 → 14px,行内元素更紧凑 */
.tree-icon-slot.is-default {
  color: var(--text-dim);
}
.tree-icon-slot.accent-is-global-agent {
  color: var(--accent-emerald);
}
/* 2026-07-17 改:去掉Agent分色的边框/淡底 — 用户反馈图标外圈那层颜色多余。
   现在所有槽统一中性描边 + 卡片底色,只保留重叠的视觉层次,不再按Agent染色。 */
/* 溢出「+N」槽:主色数字,提示还有更多Agent */
.tree-icon-slot.is-more {
  color: var(--text-dim);
}
.tree-icon-more {
  font-size: 9px;
  font-weight: 700;
  line-height: 1;
  font-family: 'JetBrains Mono', monospace;
}
/* Agent真实 logo:圆内居中,不拉伸 */
.tree-icon-img {
  display: block;
  /* 2026-10-30 v2 改:14 → 12px,跟 14px 图标槽对齐 */
  width: 12px;
  height: 12px;
  object-fit: contain;
}
.tree-skill-head .tree-name {
  /* 2026-10-30 v2 改:字号 13 → 12.5px,配合更小的图标,整行更紧凑 */
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text);
}
/* 2026-07-16 删:.tree-skill-badge-global-agent — 卡片不再渲染 badge,
   图标(mdi:earth + emerald)已能代表全局 Agent 语义。 */

.tree-version {
  /* 2026-10-30 v2 改:重新启用(见模板注释)。
     靠右贴边 + 极淡色,填掉名字右侧的空白带,不抢名字的视觉焦点。 */
  font-size: 10px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  flex-shrink: 0;
  margin-left: auto;
  padding-left: 6px;
}

/* 选中态(skill 行):淡强调底 + 左侧 2px 强调色条 + 强调色文字。
   2026-10-30 重做:旧版是"蓝色边框 + 三层光晕阴影",在窄栏里过于抢眼、
   且阴影让整列显得脏。改成 source list 经典范式 ——
     淡蓝底(--accent-blue-bg)+ 左侧色条 + 蓝色文字,
   焦点清晰但安静,跟 hover 的中性底色能明确区分。 */
.tree-node-selected > .tree-row-skill {
  background: var(--accent-blue-bg);
  border-color: transparent;
  box-shadow: none;
  position: relative;
}
/* 左侧 2px 色条(用 inset box-shadow 实现,不占布局宽度) */
.tree-node-selected > .tree-row-skill::before {
  content: '';
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 2px;
  border-radius: 0 2px 2px 0;
  background: var(--accent-blue);
}
.tree-node-selected > .tree-row-skill:hover {
  background: var(--accent-blue-bg);
  border-color: transparent;
}
/* 2026-07-11 增:选中行内 name 也变蓝,强化焦点 */
.tree-node-selected > .tree-row-skill .tree-name {
  color: var(--accent-blue);
}

/* 2026-10-30 改:这条通用选中规则原本也会命中 skill 行
   (.tree-row-skill 同时带 .tree-row 类),同特异度下后写的这条会盖掉上面
   skill 行精心设计的"淡底 + 左色条"方案(把底色刷回白、边框刷回蓝边)。
   用 :not(.tree-row-skill) 把 skill 行排除掉,让它走专属样式。 */
.tree-node-selected > .tree-row:not(.tree-row-skill) {
  background: var(--bg-card);
  border-color: var(--accent-blue);
}
.tree-node-selected > .tree-row .tree-name { color: var(--accent-blue); }

/* 拖入目标高亮(2026-06-29 改:从 1px 虚线升级到 2px + 强底色 + 文字 + 外发光,
   让用户拖动时一眼能看出"会落在这里") */
.tree-node-drop-target > .tree-row {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue);
  border-style: dashed;
  border-width: 2px;
  /* 外发光(蓝色光晕),让目标在视觉上"浮"出来 */
  box-shadow: 0 0 0 3px var(--accent-blue-bg);
  position: relative;
}

/* 拖入目标右侧追加"→ 放到此处"文字提示(走 i18n key skills.treeNode.dropHere)。
   父级是相对定位,after 绝对定位钉在行尾。 */
.tree-node-drop-target > .tree-row::after {
  content: var(--drop-hint, 'Drop here');
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  color: var(--accent-blue);
  background: var(--bg-card);
  border: 1px solid var(--accent-blue);
  border-radius: 999px;
  pointer-events: none;
  white-space: nowrap;
  /* 不挤压原有内容:让原本的子项计数自然隐藏 */
  z-index: 1;
}

/* 分组行特别强调:再加深一档(外发光更亮 + 微放大) */
.tree-node-group.tree-node-drop-target > .tree-row-group {
  background: var(--accent-blue-bg);
  box-shadow: 0 0 0 4px var(--accent-blue-bg), 0 0 12px var(--accent-blue-bg);
}

/* 拖入目标内的 count 徽章在提示文字背后时,默认让位给 after(隐藏避免重叠)。
   skill 叶子行没有"放到此处"文字的问题,但统一处理更稳妥 */
.tree-node-drop-target > .tree-row .tree-count {
  opacity: 0;
}
</style>
