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

import { ref, computed, onMounted, onUpdated, onErrorCaptured } from 'vue'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import ToolIcon from '@/components/ToolIcon.vue'
import { useToastStore } from '@/core/store/toast'
import { useToolsStore } from '@/core/store/tools'
import { getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply } from '@/api/skillbox/skills'
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
}
onUpdated(_syncWatch)

function onScopeRefresh() {
  // 2026-07-07 改 v4:外部 scope-refresh(用户保存 SKILL.md / 重拉数据等)
  // 不应该把用户的折叠状态清掉。apply/undo 完成后自己显式调 loadScope() 也是保留态。
  loadScope()
}

onMounted(() => {
  if (props.skill?.name) loadScope({ resetCollapsed: true })
  window.addEventListener('skillbox:scope-refresh', onScopeRefresh)
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
  <!-- 2026-07-07 改 v2:? + 兜底。computed 偶发返回 undefined 时?. 退化到 undefined.length 也返 undefined,
       v-else-if 自动判断为 false → 走 v-else fallback(v-else 显示 EMPTY 文案 + 不转圈)。 -->
  <section v-else-if="!scopeLoading && (scopeGroupByTool?.length || 0)" class="ssp-scope">
    <!-- 2026-07-07 改:作用域标题栏改成可点击 button,点击切换整体展开/收起。
         右侧加 chevron 图标提示状态。sectionCollapsed = true 时只显示标题,
         隐藏 .ssp-scope-list;收起态下 max-height 收紧避免占太多空间。 -->
    <button
      type="button"
      class="ssp-scope-header ssp-scope-header-toggle"
      :aria-expanded="!sectionCollapsed"
      @click="toggleSection"
    >
      <!-- 2026-07-07 改:问号图标(mdi:help-circle-outline),跟"作用域"语义最贴。
           "作用域"本质上就是"这个 skill 在哪些工具/位置上生效",本质是个问号问题。
           mdi:help-circle-outline 是 mdi 标准图标,确定存在。 -->
      <IconPark icon="mdi:help-circle-outline" width="13" height="13" />
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
      <li
        v-for="group in (scopeGroupByTool || [])"
        :key="group.tool_id"
        class="ssp-scope-group"
      >
        <button
          type="button"
          class="ssp-scope-row"
          :title="group.display"
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
  /* 2026-07-07 改 v5:作用域区不再用 bg-subtle 灰色面板,改成 transparent,
     跟文件树视觉一致。bg-subtle 在浅色主题下是浅灰,信息密度没意义还显脏。 */
  background: transparent;
  /* 2026-07-07 改 v4:作用域区移到文件树底部,不要再 max-height:50%(占满左栏下半),
     让它作为底部一块自然收缩,文件树占主空间。 */
  max-height: 45%;
  overflow: auto;
  flex-shrink: 0;
  /* 2026-07-07 修:必须显式 width:100% + max-width:100% + box-sizing,
     否则 .ssp-scope-list/.ssp-scope-row 在 flex 子项里按内容撑开,
     把左栏宽度顶出去超出界面。 */
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  min-width: 0;
}
.ssp-scope-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
/* 2026-07-07 改:作用域标题栏变成可点击 button 后的样式 reset。
   跟 .sfip-tree-header 同款风格(uppercase + sticky + bg-subtle + border-bottom)。
   收起态(sectionCollapsed=true)下整块只占标题一行,不再 max-height:45%,自然收缩。 */
.ssp-scope-header-toggle {
  width: 100%;
  /* 2026-07-07 改 v5:作用域标题栏不再 bg-subtle 灰底,transparent 跟整块一致 */
  background: transparent;
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
.ssp-confirm-msg {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
}
</style>
