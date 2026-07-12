<script setup>
// SkillScopePanel - 作用域(生效位置)展示面板
//
// 从 SkillFileInlinePanel 拆出来独立组件,2026-07-07 v4 重写。
// 独立组件的好处:
//   1. 即使本组件内部的 t() 出现 ProxyObject 异常,也只影响自身 sub-tree
//   2. main InlinePanel 不会因此崩掉(render function 拿不到)
//   3. 作用域逻辑独立维护(loadScopeStatus / apply / undo / fold)
//
// 文案策略:跟 InlinePanel 同 — 全部常量字符串,不调 t(),避免 i18n Proxy 报错。

import { ref, computed, onMounted, onUpdated, onUnmounted, onErrorCaptured } from 'vue'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import ToolIcon from '@/components/ToolIcon.vue'
import { useToastStore } from '@/core/store/toast'
import { useToolsStore } from '@/core/store/tools'
import { getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply, toggleGlobalAgent, getSkill } from '@/api/skillbox/skills'
import { inspectApplyResult, formatFailedDetail } from '@/api/skillbox/apply_result.js'

const props = defineProps({
  skill: { type: Object, default: () => ({}) },
})

const toast = useToastStore()
// 2026-07-07 改:作用域区图标 fallback — 后端 scope-status 返回的 tool 元数据不一定带 icon_file,
// 优先用 store 里全量工具的 ToolView(自定义图标 + mdi_icon 完整配置)。
const toolsStore = useToolsStore()
const toolsById = computed(() => {
  const m = {}
  for (const t of toolsStore.items || []) {
    if (t && t.tool_id) m[t.tool_id] = t
  }
  return m
})
function findTool(toolID) {
  // 先在 toolsStore 全量工具表里查(有完整 icon_file / mdi_icon / display_name)
  const fromStore = toolsById.value[toolID]
  if (fromStore) return fromStore
  // 兜底:scope-status 返回的轻量 tool 元数据
  const fromScope = scopeTools.value.find((x) => x.tool_id === toolID)
  return fromScope || null
}
function toolDisplay(toolID) {
  const t = findTool(toolID)
  if (!t) return toolShort(toolID)
  return t.display_name || t.display || t.tool_id || toolShort(toolID)
}

const LABEL_SCOPE = 'skill 作用域'
const LABEL_GLOBAL = '全局'
const LABEL_PROJECT_PREFIX = '项目 #'
const LABEL_EMPTY = '该技能尚未写入任何工具/位置'
const LABEL_LOADING = '加载中...'
const LABEL_ENABLE = '启用作用域'
const LABEL_DISABLE = '停用作用域'
const LABEL_TITLE_ERROR = '作用域加载出错'
const LABEL_RETRY = '重试'
// 2026-07-12 增:顶部"全局 Agent" tag 文案常量(跟 TreeNode 卡片上的 tag 一致)。
const LABEL_GLOBAL_AGENT = '全局 Agent'
const LABEL_GLOBAL_AGENT_TIP = '同步到 ~/.agents/skills/ 共享池(所有工具均可读取)'

const scopeTools = ref([])
const scopeHits = ref([])
const scopeLoading = ref(false)
const scopeError = ref('')
const scopeCollapsed = ref(null)
// 2026-07-07 增:作用域区整体可展开/收起(标题栏点击切换)。
// true = 整体收起(只看到标题);false = 展开(看到工具列表)。
// 2026-07-07 改 v2:默认 false — 用户进首页就应该能看到作用域生效位置,
// 不需要先点开"问号图标"。sectionCollapsed 控制整块可见性。
const sectionCollapsed = ref(false)
function toggleSection() {
  sectionCollapsed.value = !sectionCollapsed.value
}

const busyKey = ref('')
function busyKeyFor(toolID, scope, projectID) {
  return `${toolID}|${scope}|${projectID || 0}`
}
function isScopeTargetBusy(group, target) {
  return busyKey.value === busyKeyFor(group.tool_id, target.scope, target.project_id)
}

function toolShort(toolID) {
  if (!toolID) return '?'
  return toolID.charAt(0).toUpperCase() + toolID.slice(1)
}
function targetLabel(target) {
  if (!target) return ''
  if (target.scope === 'global') return LABEL_GLOBAL
  return `${LABEL_PROJECT_PREFIX}${target.project_id}`
}

const scopeGroupByTool = computed(() => {
  const out = []
  for (const tool of scopeTools.value) {
    const toolHits = scopeHits.value.filter((h) => h.tool_id === tool.tool_id)
    const map = new Map()
    for (const h of toolHits) {
      const key = h.scope === 'global' ? 'global' : `p:${h.project_id}`
      if (!map.has(key)) {
        map.set(key, {
          key,
          scope: h.scope,
          project_id: h.project_id || 0,
          project_label: targetLabel({ scope: h.scope, project_id: h.project_id }),
          exists: !!h.exists,
          path: h.path,
        })
      } else if (h.exists) {
        map.get(key).exists = true
      }
    }
    const targets = Array.from(map.values())
    targets.sort((a, b) => {
      if (a.scope !== b.scope) return a.scope === 'global' ? -1 : 1
      return a.project_id - b.project_id
    })
    out.push({
      tool_id: tool.tool_id,
      display: toolDisplay(tool.tool_id),
      // 不在 group 上挂 icon 字段了,template 用 ToolIcon + findTool(tool_id) 直接取
      hitCount: toolHits.filter((h) => h.exists).length,
      hasHit: toolHits.some((h) => h.exists),
      // 2026-07-08 增:工具是否在全局作用域启用。template 据此在数量标签前显示"全局"chip。
      hasGlobal: toolHits.some((h) => h.scope === 'global' && h.exists),
      targets,
    })
  }
  // 2026-07-07 改:作用域区工具排序 — 按 hitCount(存在命中数)降序,越多越前面;
  // 同数按 tool_id 字母序兜底,保证排序稳定。
  out.sort((a, b) => {
    if (b.hitCount !== a.hitCount) return b.hitCount - a.hitCount
    return String(a.tool_id).localeCompare(String(b.tool_id))
  })
  return out
})

function isCollapsed(toolID) {
  if (!scopeCollapsed.value) return false
  return scopeCollapsed.value.has(toolID)
}
function toggle(toolID) {
  const cur = scopeCollapsed.value || new Set()
  const next = new Set(cur)
  if (next.has(toolID)) next.delete(toolID)
  else next.add(toolID)
  // 2026-07-07 修:全展开时(null)删除最后一个折叠项 → 重新回到全展开(null);
  // 全折叠(所有 tool 都折叠)时,保留 Set(否则所有 group 同时展开太挤)。
  const allCollapsed = next.size === scopeTools.value.length && scopeTools.value.length > 0
  scopeCollapsed.value = next.size === 0 ? null : (allCollapsed ? next : next)
}

// 2026-07-07 改 v3:默认折叠全部工具,用户主动点开才展开。
// 旧版 scopeCollapsed = null → isCollapsed 返 false → 全展开,信息密度太高。
function resetCollapsed() {
  if (scopeTools.value && scopeTools.value.length) {
    scopeCollapsed.value = new Set(scopeTools.value.map((t) => t.tool_id))
  } else {
    scopeCollapsed.value = null
  }
}

async function loadScope({ resetCollapsed: shouldReset } = {}) {
  const sk = props.skill
  if (!sk || !sk.name) return
  scopeLoading.value = true
  scopeError.value = ''
  try {
    const resp = await getSkillScopeStatus({
      name: sk.name,
      version: sk.version,
    })
    scopeTools.value = resp?.tools || []
    scopeHits.value = resp?.hits || []
    // 2026-07-07 改 v4:
    //   - resetCollapsed=true  → 切 skill(初次进入)时调用,默认折叠全部
    //   - resetCollapsed=false → apply/undo / scope-refresh 事件后调用,保留用户展开态
    if (shouldReset) resetCollapsed()
  } catch (e) {
    scopeError.value = e?.message || String(e)
  } finally {
    scopeLoading.value = false
  }
}

// 2026-07-07 改 v6:不依赖 vue 的 watch(同 SkillFileInlinePanel 修复理由),
// 改用 onUpdated + 手动引用比较。
let _lastSkillName = null
let _lastSkillVersion = null
function _syncWatch() {
  const sk = props.skill
  const curName = sk?.name
  const curVersion = sk?.version
  if (curName === _lastSkillName && curVersion === _lastSkillVersion) return
  _lastSkillName = curName
  _lastSkillVersion = curVersion
  if (!curName) return
  // 2026-07-07 改 v4:切 skill 必须 reset 折叠态,旧折叠对该 skill 的 tool_id 集合无意义。
  // loadScope 内部根据参数决定是否 reset。
  loadScope({ resetCollapsed: true })
  // 2026-07-12 增:切 skill 也要重查"全局 Agent"状态,旧 skill 的 is_global_agent
  // 不能继承到新 skill。同时清掉 toggleLoading(避免新 skill 状态被旧 toggle 的
  // loading 卡住)。
  loadGlobalAgentStatus({ reset: true })
}
onUpdated(_syncWatch)

// 2026-07-12 增:全局 Agent 状态(顶部 tag 的"选中态"=该 skill 同步到
// ~/.agents/skills/ 全局目录下)。
//
// 数据来源:后端 get_skill 接口返回的 is_global_agent / global_source_path 字段
// (后端走 skillstore.ResolveGlobalSourcePath 实时 stat ~/.agents/skills/<name>/
// SKILL.md 是否存在,跟 buildTreeNode 注入到 TreeNode 的 source_path 是同一个
// 判定函数 —— 避免出现"列表说不是全局,详情说是全局"的割裂)。
//
// 切 skill / 收到 scope-refresh 事件(用户保存了 SKILL.md)时重查;toggle 完
// 成后立刻用后端返回值覆盖本地状态,无需再发请求。
const isGlobalAgent = ref(false)
const globalAgentLoading = ref(false)
const toggleAgentBusy = ref(false)
async function loadGlobalAgentStatus({ reset } = {}) {
  const sk = props.skill
  if (!sk || !sk.name) {
    isGlobalAgent.value = false
    return
  }
  // 切 skill 时立刻重置本地状态,避免旧 skill 的全局 Agent tag 在新 skill
  // 加载完成前闪一下。
  if (reset) {
    isGlobalAgent.value = false
  }
  globalAgentLoading.value = true
  try {
    const r = await getSkill({
      name: sk.name,
      version: sk.version || '',
      path: sk.group_path ? `${sk.group_path}/${sk.name}` : sk.name,
    })
    isGlobalAgent.value = !!r?.is_global_agent
  } catch (_) {
    // 静默失败:tag 默认 false,不影响主流程
    isGlobalAgent.value = false
  } finally {
    globalAgentLoading.value = false
  }
}

async function onToggleGlobalAgentClick() {
  if (toggleAgentBusy.value) return
  const sk = props.skill
  if (!sk || !sk.name) return
  toggleAgentBusy.value = true
  const next = !isGlobalAgent.value
  try {
    await toggleGlobalAgent({
      name: sk.name,
      version: sk.version || '',
      enabled: next,
    })
    isGlobalAgent.value = next
    // 通知左侧 skill 卡片列表 reload —— store.ListTree 是实时检测
    // ~/.agents/skills/<name>/ 物理文件,新镜像写盘后下次拉取即可看到 tag。
    // 走 dispatchEvent('skillbox:scope-refresh') 让 SkillsView / ScopePanel 内部
    // 各自刷新(loadScope 自己不重查 isGlobalAgent,这里再单独补一次)。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    // 顺便 reload 自身 scope-status(用户可能希望看到工具列表里某些 tool 的
    // applied_tools 发生变化 — 但其实跟全局 Agent tag 无关,这里保留以防万一)。
    loadScope()
    // 再主动重查一次 is_global_agent 后端权威值(写盘后端 store 可能要 stat
    // 真实路径,跟前端期望严格一致)。
    await loadGlobalAgentStatus()
    toast.success(next
      ? `已同步到 ~/.agents/skills/${sk.name}/`
      : `已从 ~/.agents/skills/${sk.name}/ 移除`)
  } catch (e) {
    toast.error(`切换全局 Agent 失败: ${e?.message || e}`)
  } finally {
    toggleAgentBusy.value = false
  }
}

function onScopeRefresh() {
  // 2026-07-07 改 v4:外部 scope-refresh(用户保存 SKILL.md / 重拉数据等)
  // 不应该把用户的折叠状态清掉。apply/undo 完成后自己显式调 loadScope() 也是保留态。
  loadScope()
  // 2026-07-12 增:同步重查全局 Agent 状态,确保用户保存 SKILL.md 后 tag 仍是最新值。
  loadGlobalAgentStatus()
}

onMounted(() => {
  if (props.skill?.name) {
    loadScope({ resetCollapsed: true })
    loadGlobalAgentStatus({ reset: true })
  }
  window.addEventListener('skillbox:scope-refresh', onScopeRefresh)
})
// 2026-07-08 增:旧版漏 onUnmounted 清理 listener,导致 ScopePanel 实例重建时
// (InlinePanel 走 :key 重 mount)旧的 window listener 滞留,后续 dispatchEvent
// 会触发所有还活着的 instance 各发一次 scope-status。证据:用户 apply
// code-review 后日志里出现 canvas-design 的 scope-status 请求,实际 UI 上
// canvas-design 不在视野内,是上一个 ScopePanel 实例残留监听造成的幽灵请求。
onUnmounted(() => {
  window.removeEventListener('skillbox:scope-refresh', onScopeRefresh)
})

// ===== Apply / Undo =====
async function doApplyOne(target, group) {
  busyKey.value = busyKeyFor(group.tool_id, target.scope, target.project_id)
  const sk = props.skill
  try {
    const res = await applySkill({
      name: sk.name,
      scope: target.scope,
      project_id: target.project_id || 0,
      tools: [group.tool_id],
    })
    await loadScope()
    // 2026-07-08 修:apply 完成后必须 dispatch 事件,让 SkillsView 重新拉
    // skill 树 — 树节点的 applied_tools 字段不重算的话,左侧 chip 永远显示
    // 旧状态(用户禁用某 tool 后,左侧 chip 仍显示该 tool)。
    // 注释里早就写了"apply 完成后 dispatch skillbox:scope-refresh",
    // 但实际代码漏掉了,这次补上。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    const ins = inspectApplyResult(res)
    if (ins.allOk) {
      toast.success(`已启用 ${group.display} · ${targetLabel(target)}`)
    } else {
      const detail = formatFailedDetail(ins.failedItems)
      toast.error(`部分失败: ${detail}`, 6000)
      scopeError.value = detail
    }
  } catch (e) {
    toast.error(`启用失败: ${e?.message || e}`)
  } finally {
    busyKey.value = ''
  }
}

async function doUnapplyOne(target, group) {
  busyKey.value = busyKeyFor(group.tool_id, target.scope, target.project_id)
  const sk = props.skill
  try {
    const list = await listApplies({
      scope: target.scope,
      name: sk.name,
      tool: group.tool_id,
      status: 'applied',
      page: 1,
      size: 1,
    })
    const last = list?.items?.[0]
    if (!last) {
      await forceUndoApply({
        scope: target.scope,
        project_id: target.project_id || 0,
        name: sk.name,
        tool: group.tool_id,
      })
    } else {
      await undoApply({ apply_id: last.id })
    }
    await loadScope()
    // 2026-07-08 修:同 doApplyOne,unapply 后也必须 dispatch,左侧 chip 才会同步消失。
    window.dispatchEvent(new CustomEvent('skillbox:scope-refresh'))
    toast.success(`已停用 ${group.display} · ${targetLabel(target)}`)
  } catch (e) {
    toast.error(`停用失败: ${e?.message || e}`)
  } finally {
    busyKey.value = ''
  }
}

async function handleClick(group, target) {
  if (busyKey.value) return
  if (target.exists) {
    // 2026-07-07 改:用自管 Modal 替代 window.confirm。
    // 旧版 window.confirm 在 wails desktop webview 内被默认禁用/拦截,
    // 用户点完"生效位置"按钮后 confirm 默默返回 false → 流程中断,
    // 用户感受就是"点了没反应"。
    confirmAction.value = {
      kind: 'unapply',
      title: '停用作用域',
      message: `确定要从 ${group.display} · ${targetLabel(target)} 删除 skill "${props.skill.name}"?`,
      group, target,
    }
  } else {
    confirmAction.value = {
      kind: 'apply',
      title: '启用作用域',
      message: `确定要把 skill "${props.skill.name}" 复制到 ${group.display} · ${targetLabel(target)}?`,
      group, target,
    }
  }
}

// 2026-07-07 增:自管确认弹窗状态(替代 window.confirm,适配 wails webview)
const confirmAction = ref(null)
const confirmOpen = computed({
  get: () => !!confirmAction.value,
  set: (v) => { if (!v) confirmAction.value = null },
})
async function onConfirmYes() {
  const a = confirmAction.value
  confirmAction.value = null
  if (!a) return
  if (a.kind === 'apply') await doApplyOne(a.target, a.group)
  else await doUnapplyOne(a.target, a.group)
}
function onConfirmNo() {
  confirmAction.value = null
}

// ====== ErrorBoundary 兜底(本组件独立 render,出错只影响自己) ======
const localError = ref(null)
function safeReload() { localError.value = null; loadScope() }
// 2026-07-07 改 v2:必须 return false 阻止错误继续冒泡到父组件
// (父 SkillFileInlinePanel 也有 onErrorCaptured,如果不 return false,会被父级再次捕获,
// 显示成父组件的"加载出错"覆盖页,而不是 scope 区自己的降级 UI)。
onErrorCaptured((err) => {
  console.error('[SkillScopePanel captured]', err)
  localError.value = err?.message || String(err)
  return false
})
</script>

<template>
  <!-- 错误降级 UI -->
  <div v-if="localError" class="ssp-error">
    <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
    <span>{{ LABEL_TITLE_ERROR }}: {{ localError }}</span>
    <button class="link" @click="safeReload">{{ LABEL_RETRY }}</button>
  </div>
  <!-- 2026-07-08 改:作用域区高度锁死策略
       - sectionCollapsed=true(整体收起):只显示 header 一行,高度由内容决定,约 36px
       - sectionCollapsed=false(整体展开):面板高度锁成固定值(SECTION_EXPANDED_HEIGHT),
        内部 list 用 overflow:auto 滚动;无论用户点击几个工具展开其下 targets,
        面板自身高度永远不变,只内部滚动,不再把 .sfip-tree-wrap 挤压缩小
       - 实现:.ssp-scope 切换 .is-expanded class,展开态下设 height + flex-basis 固定,
        .ssp-scope-list 设 max-height + overflow-y auto 内部滚动 -->
  <section
    v-else-if="!scopeLoading && (scopeGroupByTool?.length || 0)"
    :class="['ssp-scope', { 'is-expanded': !sectionCollapsed }]"
  >
    <!-- 2026-07-07 改:作用域标题栏改成可点击 button,点击切换整体展开/收起。
         右侧加 chevron 图标提示状态。sectionCollapsed = true 时只显示标题,
         隐藏 .ssp-scope-list;收起态下 max-height 收紧避免占太多空间。 -->
    <button
      type="button"
      class="ssp-scope-header ssp-scope-header-toggle"
      :aria-expanded="!sectionCollapsed"
      @click="toggleSection"
    >
      <!-- 2026-07-08 改:用 Local(图标位置 marker)替代原 mdi:help-circle-outline(Help)。
           "作用域" 语义核心是"这个 skill 在哪些工具/位置上生效",位置/坐标定位图标比问号更贴切。
           用 PascalCase 直传 iconpark 组件名,避免 mdi 映射兜底导致 fallback 到问号。 -->
      <IconPark icon="Local" width="13" height="13" />
      <span>{{ LABEL_SCOPE }}</span>
      <span class="ssp-scope-header-count">{{ scopeGroupByTool.length }} 个工具</span>
      <!-- 2026-07-07 改:展开收起箭头 chevron-up/down → plus/minus。
           plus = 当前是合上点开后展开,minus = 当前是展开点合上。
           跟 TreeNode/FileTreeNode 同步。 -->
      <IconPark
        :icon="sectionCollapsed ? 'mdi:plus' : 'mdi:minus'"
        width="13"
        height="13"
        class="ssp-scope-header-chevron"
      />
    </button>
    <ul v-if="!sectionCollapsed" class="ssp-scope-list">
      <!-- 2026-07-12 增:作用域区首位"全局 Agent" tag。
           跟左侧 skill 卡片"全局 Agent" tag 同源(后端 getSkill 返回的 is_global_agent
           + 走 skillstore.ResolveGlobalSourcePath 实时 stat ~/.agents/skills/<name>/
           SKILL.md),用户点 tag 切换后会立刻刷新工具列表(scope-refresh event)。
           设计要点:
             - 永远排在第一个(放在 v-for 上方,跟其他工具互不影响)
             - 没有展开/折叠(没有 targets 列表)
             - 纯 tag 胶囊样式:选中=emerald 浅底+边框+主色字,未选中=透明+灰文字+无边框
             - 整个 tag 可点击切换,spinner 在 toggle 中显示避免重复点击 -->
      <li class="ssp-scope-group ssp-scope-group-global">
        <button
          type="button"
          :class="['ssp-global-agent-tag', { 'ssp-global-agent-tag-active': isGlobalAgent }]"
          :data-tip="LABEL_GLOBAL_AGENT_TIP"
          :disabled="toggleAgentBusy || globalAgentLoading"
          @click="onToggleGlobalAgentClick"
        >
          <span
            v-if="toggleAgentBusy"
            class="ssp-spinner ssp-spinner-xs ssp-global-agent-spinner"
          ></span>
          <IconPark
            v-else
            icon="mdi:earth"
            width="11"
            height="11"
            class="ssp-global-agent-icon"
          />
          <span class="ssp-global-agent-label">{{ LABEL_GLOBAL_AGENT }}</span>
        </button>
      </li>
      <li
        v-for="group in (scopeGroupByTool || [])"
        :key="group.tool_id"
        class="ssp-scope-group"
      >
        <button
          type="button"
          class="ssp-scope-row"
          :data-tip="group.display"
          @click="toggle(group.tool_id)"
        >
          <IconPark
            :icon="isCollapsed(group.tool_id) ? 'mdi:plus' : 'mdi:minus'"
            width="12"
            height="12"
            class="ssp-scope-chevron"
          />
          <!-- 2026-07-07 改:用 ToolIcon 渲染真图标(icon_file 优先 + mdi 兜底),
               findTool 优先查 toolsStore 全量工具表;store 没数据时降级到
               scopeTools(后端 scope-status 返回的轻量工具元数据)。 -->
          <ToolIcon
            :tool="findTool(group.tool_id) || { mdi_icon: 'mdi:puzzle-outline' }"
            :size="13"
            class="ssp-scope-tool-icon"
          />
          <span class="ssp-scope-row-name">{{ group.display }}</span>
          <!-- 2026-07-08 增:工具启用了全局时,在数量标签前显示"全局"chip,
               让用户一眼看出"这个 skill 是生效在全局还是项目级"。 -->
          <span v-if="group.hasGlobal" class="ssp-scope-row-global" :title="'已启用全局作用域'">全局</span>
          <span v-if="group.hitCount > 0" class="ssp-scope-row-count">{{ group.hitCount }}</span>
        </button>
        <ul v-if="!isCollapsed(group.tool_id)" class="ssp-scope-targets">
          <li v-for="target in group.targets" :key="target.key">
            <button
              type="button"
              :class="['ssp-scope-target', target.exists ? 'ssp-scope-target-active' : '']"
              :title="target.exists ? LABEL_DISABLE : LABEL_ENABLE"
              :disabled="!!busyKey"
              @click="handleClick(group, target)"
            >
              <span
                v-if="isScopeTargetBusy(group, target)"
                class="ssp-spinner ssp-spinner-xs"
              ></span>
              <IconPark
                v-else
                :icon="target.scope === 'global' ? 'mdi:earth' : 'mdi:folder-outline'"
                width="11"
                height="11"
              />
              <span class="ssp-scope-target-name">{{ targetLabel(target) }}</span>
            </button>
          </li>
          <li v-if="!group.targets.length" class="ssp-scope-empty">
            {{ LABEL_EMPTY }}
          </li>
        </ul>
      </li>
    </ul>
    <p v-if="scopeError" class="ssp-scope-error">
      <IconPark icon="mdi:alert-circle-outline" width="11" height="11" />
      {{ scopeError }}
    </p>
  </section>
  <p v-else-if="scopeLoading" class="ssp-scope-loading">
    <span class="ssp-spinner ssp-spinner-xs"></span>
  </p>
  <p v-else class="ssp-scope-empty-tip">{{ LABEL_EMPTY }}</p>

  <!-- 2026-07-07 增:自管确认弹窗,替代 window.confirm。
       wails desktop webview 默认禁用 window.confirm,直接调确认会被静默拒绝。 -->
  <Modal
    v-model="confirmOpen"
    size="sm"
    :title="confirmAction?.title || ''"
    :close-on-mask="false"
    @close="onConfirmNo"
  >
    <p class="ssp-confirm-msg">{{ confirmAction?.message || '' }}</p>
    <template #footer>
      <button type="button" class="ghost" @click="onConfirmNo">取消</button>
      <button type="button" class="primary" @click="onConfirmYes">确定</button>
    </template>
  </Modal>
</template>

<style scoped>
.ssp-scope {
  border-bottom: 1px solid var(--border);
  /* 2026-07-10 改 v7:用户反馈"作用于区域面板"主体要白底,只保留 header 灰底做层级区分。
     之前 v6 改回 var(--bg-subtle) 是为了跟 file tree 视觉对齐,现在按用户最新需求反转:
     外层 panel(包含 body 列表)改用 var(--bg-card) 纯白,header 仍用 var(--bg-subtle) 灰,
     header 自身带 border-bottom 自然分隔 header 与 body 区域。 */
  background: var(--bg-card);
  /* 2026-07-07 改 v4:作用域区移到文件树底部,不要再 max-height:50%(占满左栏下半),
     让它作为底部一块自然收缩,文件树占主空间。 */
  flex-shrink: 0;
  /* 2026-07-07 修:必须显式 width:100% + max-width:100% + box-sizing,
     否则 .ssp-scope-list/.ssp-scope-row 在 flex 子项里按内容撑开,
     把左栏宽度顶出去超出界面。 */
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  min-width: 0;
  /* 2026-07-08 改:折叠态(sectionCollapsed=true)下只显示 header,
     高度由内容自然撑开,不设固定高度,避免空占空间 */
}
/* 2026-07-08 增:展开态下高度锁死 —— 用户反馈"点击工具展开 targets 时
   此面板高度不能变"。实现方式:
   - panel 自身 height + flex-basis 锁成 SECTION_EXPANDED_HEIGHT(300px),
     不论里面 tool 列表展开几个 targets,panel 占用的 flex 空间恒定
   - .ssp-scope-list(工具组列表)内部 max-height + overflow-y auto,
     内容溢出走内部滚动,不会再撑开 panel
   这样保证展开状态下 .sfip-tree-wrap(文件树)的高度不被压缩 */
.ssp-scope.is-expanded {
  height: 300px;
  flex: 0 0 300px;
  display: flex;
  flex-direction: column;
}
.ssp-scope-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  /* 2026-07-07 改 v6:作用域标题色跟 sfip-tree-header 完全一致 var(--text-dim),
     图标 svg 强制 currentColor 继承,避免 IconPark svg 默认 fill 跟文字对比过大
     看上去像"白色"。 */
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.ssp-scope-header :deep(svg) { color: inherit; fill: currentColor; }
/* 2026-07-07 改 v6:标题内 IconPark svg 显式 currentColor,跟文字色统一 */
.ssp-scope-header > :deep(svg),
.ssp-scope-header svg { color: inherit; fill: currentColor; }
/* 2026-07-07 改:作用域标题栏变成可点击 button 后的样式 reset。
   跟 .sfip-tree-header 同款风格(uppercase + sticky + bg-subtle + border-bottom)。
   收起态(sectionCollapsed=true)下整块只占标题一行,不再 max-height:45%,自然收缩。 */
.ssp-scope-header-toggle {
  width: 100%;
  /* 2026-07-07 改 v6:作用域标题栏跟面板同底 var(--bg-subtle),hover 时 var(--bg-hover)
     加深。 */
  background: var(--bg-subtle);
  border: none;
  border-bottom: 1px solid var(--border);
  border-top: 1px solid var(--border);
  cursor: pointer;
  text-align: left;
  font: inherit;
  text-transform: uppercase;
  transition: background 0.12s ease;
}
.ssp-scope-header-toggle:hover { background: var(--bg-hover); }
.ssp-scope-header-count {
  margin-left: auto;
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0;
  text-transform: none;
  color: var(--text-faint);
  padding: 1px 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
}
.ssp-scope-header-chevron {
  margin-left: 4px;
  color: var(--text-faint);
}
.ssp-scope-list {
  list-style: none;
  margin: 0;
  padding: 0 0 6px;
  /* 2026-07-07 修:列表容器限制宽度,长工具名/路径不撑出父级 */
  width: 100%;
  max-width: 100%;
  min-width: 0;
  box-sizing: border-box;
  /* 2026-07-08 改:展开态下,list 占 panel 减 header 的剩余高度,
     内部工具组展开后溢出走 overflow-y auto,不再撑开外层 panel */
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  /* 2026-07-08 增:作用域面板内滚动条美化 — 整体 4px 细,圆角 thumb,
     默认半透明跟 var(--bg-subtle) 灰底融合,hover 时变明显。
     只作用域 panel 内部生效,不污染全局 ::-webkit-scrollbar */
  scrollbar-width: thin;
  scrollbar-color: var(--border) transparent;
}
.ssp-scope-list::-webkit-scrollbar {
  width: 4px;
  height: 4px;
}
.ssp-scope-list::-webkit-scrollbar-track {
  background: transparent;
  margin: 2px 0;
}
.ssp-scope-list::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 999px;
  /* 默认半透明,跟灰底融合不抢眼 */
  opacity: 0.6;
  transition: background 0.15s ease, opacity 0.15s ease;
}
.ssp-scope-list:hover::-webkit-scrollbar-thumb {
  background: var(--text-faint);
  opacity: 1;
}
.ssp-scope-list::-webkit-scrollbar-thumb:hover {
  background: var(--accent-blue);
  opacity: 1;
}
.ssp-scope-group {
  padding: 0;
  min-width: 0;
}
.ssp-scope-row {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: transparent;
  border: none;
  color: var(--text);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
}
.ssp-scope-row:hover { background: var(--bg-hover); }
.ssp-scope-chevron { color: var(--text-faint); flex-shrink: 0; }
.ssp-scope-tool-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}
.ssp-scope-row-name {
  flex: 1;
  font-weight: 500;
  /* 2026-07-07 修:长工具名截断,否则"启用"按钮会被挤出 */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-scope-row-count {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-radius: 999px;
  flex-shrink: 0;
}
/* 2026-07-08 增:工具启用全局时的"全局"chip。
   2026-07-08 改 v2:跟 .ssp-scope-row-count 视觉一致(同色同字号同圆角),
   之前用实心蓝底+白字跟数量徽章视觉打架,看着像两个不同元素;
   用户反馈希望保持同款 — 区分靠位置(数量标签在前)+ 文案("全局" vs 数字)。 */
.ssp-scope-row-global {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-radius: 999px;
  flex-shrink: 0;
  font-weight: 500;
}
.ssp-scope-targets {
  list-style: none;
  margin: 0;
  padding: 0 0 4px 26px;
}
.ssp-scope-target {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  margin: 1px 0;
  background: transparent;
  border: 1px dashed var(--border);
  color: var(--text-dim);
  font: inherit;
  font-size: 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
}
.ssp-scope-target-active {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-style: solid;
  border-color: var(--accent-blue-border);
}
.ssp-scope-target:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
}
.ssp-scope-target:disabled { opacity: 0.5; cursor: not-allowed; }
.ssp-scope-target-name {
  flex: 1;
  /* 2026-07-07 修:长路径/项目名截断 */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-scope-empty,
.ssp-scope-empty-tip {
  font-size: 11px;
  color: var(--text-faint);
  font-style: italic;
  padding: 4px 14px;
  margin: 0;
}
.ssp-scope-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
}
.ssp-scope-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 14px;
  border-bottom: 1px solid var(--border);
  color: var(--text-faint);
}
.ssp-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: ssp-spin 0.8s linear infinite;
}
.ssp-spinner-xs { width: 12px; height: 12px; }
@keyframes ssp-spin { to { transform: rotate(360deg); } }
.ssp-error {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}
.ssp-error .link {
  background: transparent;
  border: none;
  color: var(--accent-blue);
  text-decoration: underline;
  cursor: pointer;
  font-size: 12px;
  margin-left: auto;
}

/* 2026-07-07 增:自管确认弹窗文案 */
/* 2026-07-12 改 v2:全局 Agent 改成纯 tag 胶囊样式(2026-07-12 用户反馈)。
   设计目标:
     - 选中 = emerald 浅底 + 边框 + 主色字(跟 TreeNode .tree-skill-badge-global-agent
       同款色相,视觉一致)
     - 未选中 = 完全透明背景 + 无边框 + 灰文字(跟工具列表的"未启用"状态
       视觉接近,只是一个轻量的标签)
   实现:
     - 胶囊 padding 1px 8px(比原 row padding 小很多,贴合 tag 形态)
     - 整行可点击 — cursor:pointer,hover 时未选中态显示淡灰底提示可点
     - 选中态 hover 加深一档(emerald-bg → emerald-border),强化反馈
     - 字体 11px(比 row 文字小一档,贴合 tag 视觉权重)
     - 切换中 toggleAgentBusy 时显示 spinner 替代图标 */
.ssp-global-agent-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 4px 12px 6px;
  padding: 1px 8px;
  border-radius: 999px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-faint);
  font-size: 11px;
  font-weight: 500;
  line-height: 1.5;
  white-space: nowrap;
  cursor: pointer;
  user-select: none;
  transition: background 0.12s ease, color 0.12s ease, border-color 0.12s ease;
}
.ssp-global-agent-tag:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text-dim);
}
/* 选中态:emerald 三件套 + 字色加深 */
.ssp-global-agent-tag-active {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
  color: var(--accent-emerald);
  font-weight: 600;
}
.ssp-global-agent-tag-active:hover:not(:disabled) {
  /* 选中态 hover 时把边框加深一档,告诉用户"还能再点切换",而不是淡化。
     背景保持 emerald-bg(不能加深太多,亮色底反复叠会发灰)。 */
  border-color: var(--accent-emerald);
  background: var(--accent-emerald-bg);
}
.ssp-global-agent-tag:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.ssp-global-agent-icon { color: inherit; flex-shrink: 0; }
.ssp-global-agent-spinner { flex-shrink: 0; }
/* 2026-07-12 改:tag 自身有 margin-bottom 跟下方工具行分隔,
   旧的 border-bottom 分隔线去掉(避免视觉割裂 — tag 是 inline-flex 自带间距)。 */
.ssp-scope-group-global {
  margin-bottom: 2px;
}

.ssp-confirm-msg {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
</style>
