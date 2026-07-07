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

import { ref, computed, onMounted, onErrorCaptured } from 'vue'
import IconPark from '@/components/IconPark.vue'
import { useToastStore } from '@/core/store/toast'
import { getSkillScopeStatus, applySkill, listApplies, undoApply, forceUndoApply } from '@/api/skillbox/skills'
import { inspectApplyResult, formatFailedDetail } from '@/api/skillbox/apply_result.js'

const props = defineProps({
  skill: { type: Object, default: () => ({}) },
})

const toast = useToastStore()

const LABEL_SCOPE = '作用域'
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
function toolIcon(toolID) {
  const t = scopeTools.value.find((x) => x.tool_id === toolID)
  return t?.icon || 'mdi:puzzle-outline'
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
      display: tool.display_name || tool.tool_id,
      icon: toolIcon(tool.tool_id),
      hitCount: toolHits.filter((h) => h.exists).length,
      hasHit: toolHits.some((h) => h.exists),
      targets,
    })
  }
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
  scopeCollapsed.value = next.size === scopeTools.value.length ? null : next
}

async function loadScope() {
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
  } catch (e) {
    scopeError.value = e?.message || String(e)
  } finally {
    scopeLoading.value = false
  }
}

watch(
  () => [props.skill?.name, props.skill?.version],
  () => {
    if (!props.skill?.name) return
    scopeCollapsed.value = null
    loadScope()
  },
)

function onScopeRefresh() { loadScope() }

onMounted(() => {
  if (props.skill?.name) loadScope()
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
    const ok = window.confirm(`确定要从 ${group.display} · ${targetLabel(target)} 删除 skill "${props.skill.name}"?`)
    if (!ok) return
    await doUnapplyOne(target, group)
  } else {
    const ok = window.confirm(`确定要把 skill "${props.skill.name}" 复制到 ${group.display} · ${targetLabel(target)}?`)
    if (!ok) return
    await doApplyOne(target, group)
  }
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
    <header class="ssp-scope-header">
      <IconPark icon="mdi:earth" width="13" height="13" />
      <span>{{ LABEL_SCOPE }}</span>
    </header>
    <ul class="ssp-scope-list">
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
            :icon="isCollapsed(group.tool_id) ? 'mdi:chevron-right' : 'mdi:chevron-down'"
            width="12"
            height="12"
            class="ssp-scope-chevron"
          />
          <IconPark :icon="group.icon" width="12" height="12" />
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
</template>

<style scoped>
.ssp-scope {
  border-bottom: 1px solid var(--border);
  background: var(--bg-subtle);
  max-height: 50%;
  overflow: auto;
  flex-shrink: 0;
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
.ssp-scope-list {
  list-style: none;
  margin: 0;
  padding: 0 0 6px;
}
.ssp-scope-group { padding: 0; }
.ssp-scope-row {
  width: 100%;
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
}
.ssp-scope-row:hover { background: var(--bg-hover); }
.ssp-scope-chevron { color: var(--text-faint); flex-shrink: 0; }
.ssp-scope-row-name {
  flex: 1;
  font-weight: 500;
}
.ssp-scope-row-count {
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
  border-radius: 999px;
}
.ssp-scope-targets {
  list-style: none;
  margin: 0;
  padding: 0 0 4px 26px;
}
.ssp-scope-target {
  width: 100%;
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
.ssp-scope-target-name { flex: 1; }
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
</style>
