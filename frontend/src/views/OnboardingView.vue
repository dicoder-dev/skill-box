<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { getOnboardingStatus, runOnboardingScan, runOnboardingImport } from '@/api/skillbox/onboarding'

const { t } = useI18n()

const phase = ref('status')
const appBus = inject('appBus', null)

const loading = ref(false)
const error = ref('')
const success = ref('')

const adapters = ref([])
const lastScan = ref(null)
const totalFound = ref(0)
const hasReport = ref(false)

const scanReport = ref(null)
const selected = ref(new Set())
// 当前激活的工具 tab id;phase === 'scan' 时使用。
const activeToolId = ref('')

const importResult = ref(null)

// tool_id → mdi 图标映射。
// 后端 IconEmoji 字段已废弃(2026-06-23 清理乱码字节,项目规范禁 emoji),
// 改由前端按 tool_id 决定图标,5 个工具都给到语义化的 mdi 图标。
const toolIconMap = {
  claude: 'mdi:robot-outline',
  codex: 'mdi:cube-outline',
  cursor: 'mdi:cursor-default-click-outline',
  opencode: 'mdi:code-braces',
  trae: 'mdi:shield-outline',
}
function iconOf(toolId) {
  return toolIconMap[toolId] || 'mdi:puzzle-outline'
}

async function loadStatus() {
  loading.value = true
  error.value = ''
  try {
    const res = await getOnboardingStatus()
    adapters.value = res?.adapters || []
    lastScan.value = res?.last_scan || null
    totalFound.value = res?.total_found || 0
    hasReport.value = !!res?.has_report
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function doScan() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const res = await runOnboardingScan()
    scanReport.value = res
    // 默认勾选:仅 user 级别。system 级别(工具自带 / vendor curated /
    // plugin 内建)只读展示,不能误导入覆盖本地 store。
    selected.value = new Set(
      (res.found || [])
        .filter((f) => f.category !== 'system')
        .map((f) => keyOf(f)),
    )
    // 默认激活第一个有 user 级别发现的 tab,避免空 tab。
    const firstTid = (res.tools || []).find(
      (tid) =>
        (res.found || []).some(
          (f) => f.tool_id === tid && f.category !== 'system',
        ),
    ) || (res.tools || [])[0]
    activeToolId.value = firstTid || ''
    phase.value = 'scan'
  } catch (e) {
    error.value = t('onboarding.errScan', { msg: e?.message || e })
  } finally {
    loading.value = false
  }
}

function keyOf(found) {
  return `${found.tool_id}::${found.name}@${found.version}`
}

function toggleSelect(found) {
  const k = keyOf(found)
  const s = new Set(selected.value)
  if (s.has(k)) s.delete(k)
  else s.add(k)
  selected.value = s
}

// 按 tool_id 分组的 skill 列表 + 元数据(显示名 / 数量 / 当前 tab id)。
//
// 组内排序:user 级别在前 system 级别在后 —— 用户日常用的 skill 优先展示,
// 系统自带 / plugin 内建 / vendor curated 的列在下方,只读不可勾选。
const foundByTool = computed(() => {
  const groups = {}
  // 先按 scanReport.tools 顺序建空组,保证 tab 顺序稳定(后端已排序)
  for (const tid of scanReport.value?.tools || []) {
    groups[tid] = { name: '', items: [] }
  }
  for (const f of scanReport.value?.found || []) {
    if (!groups[f.tool_id]) groups[f.tool_id] = { name: f.tool_name, items: [] }
    if (!groups[f.tool_id].name) groups[f.tool_id].name = f.tool_name
    groups[f.tool_id].items.push(f)
  }
  // 组内按 category(user 先 system 后)稳定排序。
  for (const tid of Object.keys(groups)) {
    groups[tid].items.sort((a, b) => {
      const ax = a.category === 'system' ? 1 : 0
      const bx = b.category === 'system' ? 1 : 0
      if (ax !== bx) return ax - bx
      return a.name.localeCompare(b.name)
    })
  }
  return groups
})

const toolTabs = computed(() =>
  Object.entries(foundByTool.value).map(([tid, g]) => ({
    toolId: tid,
    name: g.name,
    count: g.items.filter((f) => f.category !== 'system').length,
    totalCount: g.items.length,
    icon: iconOf(tid),
  })),
)

// 工具内"全选/全不选":只动当前 tab 的 user 级别 skill,不影响 system(不可勾)。
function selectAllInTool(tid) {
  const s = new Set(selected.value)
  for (const f of foundByTool.value[tid]?.items || []) {
    if (f.category === 'system') continue
    s.add(keyOf(f))
  }
  selected.value = s
}
function selectNoneInTool(tid) {
  const s = new Set(selected.value)
  for (const f of foundByTool.value[tid]?.items || []) {
    if (f.category === 'system') continue
    s.delete(keyOf(f))
  }
  selected.value = s
}
function selectedInTool(tid) {
  let n = 0
  for (const f of foundByTool.value[tid]?.items || []) {
    if (f.category === 'system') continue
    if (selected.value.has(keyOf(f))) n++
  }
  return n
}

async function doImport() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const items = (selected.value.size === (scanReport.value?.found || []).length)
      ? []
      : Array.from(selected.value).map((k) => {
          const [tool_id, nameVer] = k.split('::')
          const [name, version] = nameVer.split('@')
          return { tool_id, name, version }
        })
    const res = await runOnboardingImport(items)
    importResult.value = res
    phase.value = 'import'
    success.value = t('onboarding.okImport', { ok: res.ok, failed: res.failed })
    await loadStatus()
  } catch (e) {
    error.value = t('onboarding.errImport', { msg: e?.message || e })
  } finally {
    loading.value = false
  }
}

function reset() {
  phase.value = 'status'
  scanReport.value = null
  importResult.value = null
  selected.value = new Set()
  activeToolId.value = ''
}

function goSkills() {
  if (appBus) {
    appBus.emit('switch-tab', 'skills')
  } else {
    window.dispatchEvent(new CustomEvent('skillbox:switch-tab', { detail: 'skills' }))
  }
}

onMounted(loadStatus)
</script>

<template>
  <div class="onb">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-violet">
          <Icon icon="mdi:compass-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('onboarding.title') }}</h1>
          <p>{{ t('onboarding.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- 阶段指示器 -->
    <div class="steps">
      <div :class="['step', { active: phase === 'status', done: ['scan', 'import'].includes(phase) }]">
        <div class="step-number">1</div>
        <div class="step-content">
          <span class="step-title">{{ t('onboarding.steps.status') }}</span>
        </div>
      </div>
      <div class="step-connector" :class="{ done: ['scan', 'import'].includes(phase) }"></div>
      <div :class="['step', { active: phase === 'scan', done: phase === 'import' }]">
        <div class="step-number">2</div>
        <div class="step-content">
          <span class="step-title">{{ t('onboarding.steps.scan') }}</span>
        </div>
      </div>
      <div class="step-connector" :class="{ done: phase === 'import' }"></div>
      <div :class="['step', { active: phase === 'import' }]">
        <div class="step-number">3</div>
        <div class="step-content">
          <span class="step-title">{{ t('onboarding.steps.done') }}</span>
        </div>
      </div>
    </div>

    <p v-if="error" class="message message-error">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>
    <p v-if="success" class="message message-success">
      <Icon icon="mdi:check-circle-outline" width="14" height="14" />
      {{ success }}
    </p>

    <!-- 阶段 1: 状态 -->
    <section v-if="phase === 'status'" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
          {{ t('onboarding.phase1.title') }}
          <span class="card-sub">— {{ t('onboarding.phase1.total', { n: adapters.length }) }}</span>
        </h3>
      </header>

      <div v-if="!adapters.length" class="empty-state">
        <Icon icon="mdi:inbox-outline" width="48" height="48" />
        <p class="empty-title">{{ t('onboarding.phase1.empty') }}</p>
      </div>

      <table v-else class="grid">
        <thead>
          <tr>
            <th style="width: 44px"></th>
            <th>{{ t('onboarding.phase1.colTool') }}</th>
            <th>{{ t('onboarding.phase1.colId') }}</th>
            <th>{{ t('onboarding.phase1.colGlobalPath') }}</th>
            <th>{{ t('onboarding.phase1.colStatus') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in adapters" :key="a.tool_id">
            <td class="icon-cell">
              <Icon :icon="iconOf(a.tool_id)" width="20" height="20" class="tool-icon" />
            </td>
            <td><strong>{{ a.display_name }}</strong></td>
            <td><code>{{ a.tool_id }}</code></td>
            <td class="td-path">{{ a.global_path || t('common.dash') }}</td>
            <td>
              <span v-if="a.global_ok" class="badge badge-success">{{ t('onboarding.phase1.detected') }}</span>
              <span v-else class="badge badge-warning">{{ t('onboarding.phase1.missing') }}</span>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="card-footer">
        <span class="footer-info">
          {{ t('onboarding.phase1.lastScan') }}{{ lastScan ? new Date(lastScan).toLocaleString() : t('onboarding.phase1.neverScanned') }}
          <span v-if="hasReport">{{ t('onboarding.phase1.foundSuffix', { n: totalFound }) }}</span>
        </span>
        <button class="primary" :disabled="loading" @click="doScan">
          <span v-if="loading" class="spinner"></span>
          <Icon v-else icon="mdi:magnify" width="14" height="14" />
          {{ loading ? t('onboarding.phase1.scanning') : t('onboarding.phase1.btnScan') }}
        </button>
      </div>
    </section>

    <!-- 阶段 2: 扫描 + 勾选(tab 面板,按工具拆分) -->
    <section v-else-if="phase === 'scan'" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:folder-search" width="16" height="16" />
          {{ t('onboarding.phase2.title') }}
          <span class="card-sub">— {{ t('onboarding.phase2.foundSuffix', { n: scanReport?.found?.length || 0 }) }}</span>
        </h3>
      </header>

      <div v-if="!scanReport?.found?.length" class="empty-state">
        <Icon icon="mdi:magnify" width="48" height="48" />
        <p class="empty-title">{{ t('onboarding.phase2.empty') }}</p>
      </div>

      <div v-else>
        <!-- 工具 tab 栏:瑞士风,上划线 + 数字徽章 -->
        <div class="tool-tabs" role="tablist">
          <button
            v-for="tab in toolTabs"
            :key="tab.toolId"
            role="tab"
            :aria-selected="activeToolId === tab.toolId"
            :class="['tool-tab', { active: activeToolId === tab.toolId }]"
            @click="activeToolId = tab.toolId"
          >
            <Icon :icon="tab.icon" width="16" height="16" class="tab-icon" />
            <span class="tab-name">{{ tab.name }}</span>
            <span class="tab-count">
              {{ tab.count }}<span v-if="tab.totalCount > tab.count" class="tab-count-sys">+{{ tab.totalCount - tab.count }}</span>
            </span>
          </button>
        </div>

        <!-- 当前 tab 内容 -->
        <div v-if="activeToolId && foundByTool[activeToolId]" class="tool-panel">
          <div class="bulk-actions">
            <button class="sm" @click="selectAllInTool(activeToolId)">
              {{ t('onboarding.phase2.selectAll') }}
            </button>
            <button class="sm ghost" @click="selectNoneInTool(activeToolId)">
              {{ t('onboarding.phase2.selectNone') }}
            </button>
            <span class="selection-info">
              {{ t('onboarding.phase2.selected', {
                  sel: selectedInTool(activeToolId),
                  total: foundByTool[activeToolId].items.filter((f) => f.category !== 'system').length,
              }) }}
            </span>
          </div>

          <!-- 分档小标题:用户 skill 在前,系统 skill 在后 -->
          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'user' || !f.category)"
            class="cat-label cat-user"
          >
            <Icon icon="mdi:account-circle-outline" width="14" height="14" />
            {{ t('onboarding.phase2.catUser') }}
          </div>
          <ul v-if="foundByTool[activeToolId].items.some((f) => f.category !== 'system')" class="found-list">
            <li v-for="f in foundByTool[activeToolId].items.filter((x) => x.category !== 'system')"
                :key="keyOf(f)"
                :class="{ selected: selected.has(keyOf(f)) }">
              <label class="found-item">
                <input
                  type="checkbox"
                  :checked="selected.has(keyOf(f))"
                  @change="toggleSelect(f)"
                />
                <span class="f-name"><code>{{ f.name }}</code></span>
                <span class="f-ver">v{{ f.version }}</span>
                <span class="f-path" :title="f.source_path">{{ f.source_path }}</span>
              </label>
            </li>
          </ul>

          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')"
            class="cat-divider"
          >
            <span class="cat-divider-text">{{ t('onboarding.phase2.catSectionDivider') }}</span>
          </div>
          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')"
            class="cat-label cat-system"
          >
            <Icon icon="mdi:lock-outline" width="14" height="14" />
            {{ t('onboarding.phase2.catSystem') }}
            <span class="cat-hint">— {{ t('onboarding.phase2.catSystemHint') }}</span>
          </div>
          <ul v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')" class="found-list found-list-system">
            <li v-for="f in foundByTool[activeToolId].items.filter((x) => x.category === 'system')"
                :key="keyOf(f)"
                class="system-item">
              <span class="found-item found-item-system">
                <input
                  type="checkbox"
                  disabled
                  aria-disabled="true"
                  :title="t('onboarding.phase2.catSystemHint')"
                />
                <span class="f-name"><code>{{ f.name }}</code></span>
                <span class="f-ver">v{{ f.version }}</span>
                <span class="f-path" :title="f.source_path">{{ f.source_path }}</span>
                <Icon icon="mdi:lock-outline" width="12" height="12" class="lock-icon" />
              </span>
            </li>
          </ul>
        </div>

        <div class="card-footer">
          <button class="ghost" @click="reset">
            <Icon icon="mdi:arrow-left" width="14" height="14" />
            {{ t('onboarding.phase2.btnBack') }}
          </button>
          <button class="primary" :disabled="loading || selected.size === 0" @click="doImport">
            <span v-if="loading" class="spinner"></span>
            <Icon v-else icon="mdi:download" width="14" height="14" />
            {{ loading ? t('onboarding.phase2.importing') : t('onboarding.phase2.btnImport', { n: selected.size }) }}
          </button>
        </div>
      </div>
    </section>

    <!-- 阶段 3: 完成 -->
    <section v-else-if="phase === 'import'" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:check-circle" width="16" height="16" />
          {{ t('onboarding.phase3.title') }}
        </h3>
      </header>

      <div v-if="importResult" class="result-stats">
        <div class="stat-card stat-success">
          <span class="stat-number">{{ importResult.ok }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statOk') }}</span>
        </div>
        <div class="stat-card stat-error">
          <span class="stat-number">{{ importResult.failed }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statErr') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-number">{{ importResult.total }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statTotal') }}</span>
        </div>
      </div>

      <ul v-if="importResult?.results?.length" class="result-list">
        <li v-for="(r, i) in importResult.results" :key="i" :class="r.ok ? 'result-ok' : 'result-error'">
          <span class="r-tool">{{ r.tool_id || r.tool || t('common.dash') }}</span>
          <span class="r-name"><code>{{ r.name || r.canonical?.manifest?.name }}</code></span>
          <span class="r-msg">{{ r.error || r.message || (r.ok ? 'OK' : 'failed') }}</span>
        </li>
      </ul>

      <div class="card-footer">
        <button class="ghost" @click="reset">
          <Icon icon="mdi:refresh" width="14" height="14" />
          {{ t('onboarding.phase3.btnAgain') }}
        </button>
        <button class="primary" @click="goSkills">
          <Icon icon="mdi:arrow-right" width="14" height="14" />
          {{ t('onboarding.phase3.btnGoSkills') }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.onb {
  max-width: 980px;
  margin: 0 auto;
  color: var(--text);
  transition: color 0.3s ease;
}

/* 页面头部 */
.view-header {
  margin-bottom: 24px;
}

.view-title {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.view-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-violet);
  color: #ffffff;
  flex-shrink: 0;
}

.view-icon-violet {
  background: var(--accent-violet);
  color: #ffffff;
}

.view-title h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 4px;
  transition: color 0.3s ease;
}

.view-title p {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0;
  transition: color 0.3s ease;
}

/* 步骤指示器 */
.steps {
  display: flex;
  align-items: center;
  gap: 0;
  margin-bottom: 24px;
}

.step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: all 0.3s ease;
}

.step.active {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue);
}

.step.done {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald);
}

.step-number {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  background: var(--bg-subtle);
  color: var(--text-dim);
  flex-shrink: 0;
  transition: all 0.3s ease;
}

.step.active .step-number {
  background: var(--accent-blue);
  color: #ffffff;
}

.step.done .step-number {
  background: var(--accent-emerald);
  color: #ffffff;
}

.step-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-dim);
  transition: color 0.3s ease;
}

.step.active .step-title {
  color: var(--accent-blue);
}

.step.done .step-title {
  color: var(--accent-emerald);
}

.step-connector {
  flex: 1;
  height: 2px;
  background: var(--border);
  margin: 0 8px;
  transition: background 0.3s ease;
}

.step-connector.done {
  background: var(--accent-emerald);
}

/* 消息提示 */
.message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 16px;
}

.message-success {
  background: var(--success-dim);
  color: var(--success);
}

.message-error {
  background: var(--danger-dim);
  color: var(--danger);
}

/* 卡片 */
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  padding: 20px;
  margin-bottom: 16px;
  transition: all 0.3s ease;
}

.card-header {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.card-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.footer-info {
  font-size: 13px;
  color: var(--text-dim);
}

/* 表格 */
.grid {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.grid th, .grid td {
  padding: 12px 14px;
  text-align: left;
  border-bottom: 1px solid var(--border);
  transition: background-color 0.3s ease;
}

.grid th {
  background: var(--bg-subtle);
  color: var(--text-dim);
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.grid tbody tr {
  transition: background-color 0.15s ease;
}

.grid tbody tr:hover {
  background: var(--bg-hover);
}

.icon-cell {
  color: var(--accent-blue);
}
.tool-icon {
  vertical-align: middle;
}

.td-path {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
  font-size: 12px;
}

/* 徽章 */
.badge {
  display: inline-flex;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
}

.badge-success {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border: 1px solid var(--accent-emerald-border);
}

.badge-warning {
  background: var(--accent-amber-bg);
  color: var(--accent-amber);
  border: 1px solid var(--accent-amber-border);
}

/* 批量操作 */
.bulk-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.selection-info {
  font-size: 13px;
  color: var(--text-dim);
  margin-left: auto;
}

/* 工具 tab 栏:瑞士风,上划线 + 数字徽章 */
.tool-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
  scrollbar-width: thin;
}

.tool-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.tool-tab:hover:not(.active) {
  color: var(--text);
}

.tool-tab.active {
  color: var(--accent-blue);
  border-bottom-color: var(--accent-blue);
}

.tool-tab .tab-icon {
  flex-shrink: 0;
}

.tool-tab .tab-name {
  font-weight: 500;
}

.tool-tab .tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 18px;
  padding: 0 6px;
  border-radius: 9px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  font-size: 11px;
  font-weight: 600;
  font-feature-settings: 'tnum';
  transition: background 0.15s ease, color 0.15s ease;
}

.tool-tab.active .tab-count {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
}

.tool-tab .tab-count-sys {
  margin-left: 4px;
  color: var(--text-faint);
  font-weight: 500;
}

.tool-tab.active .tab-count-sys {
  color: var(--text-dim);
}

/* 分档小标题(user / system) */
.cat-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  margin-bottom: 10px;
}

.cat-user {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border: 1px solid var(--accent-emerald-border);
}

.cat-system {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.cat-hint {
  text-transform: none;
  letter-spacing: 0;
  font-weight: normal;
  color: var(--text-faint);
  margin-left: 6px;
}

.cat-divider {
  margin: 18px 0 12px;
  text-align: center;
  position: relative;
}

.cat-divider::before {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  height: 1px;
  background: var(--border);
  z-index: 0;
}

.cat-divider-text {
  position: relative;
  z-index: 1;
  background: var(--bg-card);
  padding: 0 12px;
  font-size: 11px;
  color: var(--text-faint);
  letter-spacing: 0.3px;
}

/* 系统级 skill 列表:灰色背景,checkbox 禁用 + 锁图标 */
.found-list-system {
  opacity: 0.78;
}

.found-list-system .found-item-system {
  cursor: not-allowed;
}

.found-list-system .found-item-system input[type="checkbox"]:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.found-list-system .f-name code,
.found-list-system .f-ver,
.found-list-system .f-path {
  color: var(--text-dim);
}

.lock-icon {
  color: var(--text-faint);
  flex-shrink: 0;
  margin-left: 4px;
}

/* 当前 tab 内容区(单工具的 skill 列表) */
.tool-panel {
  padding-top: 4px;
}

/* 发现列表 */
.found-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.found-list li {
  border-bottom: 1px solid var(--border);
  transition: background 0.15s ease;
}

.found-list li:last-child {
  border-bottom: none;
}

.found-list li.selected {
  background: var(--accent-blue-bg);
}

.found-item input[type="checkbox"]:checked {
  accent-color: var(--accent-blue);
}

.found-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  cursor: pointer;
}

.found-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.f-name {
  font-weight: 500;
  min-width: 140px;
}

.f-ver {
  font-size: 12px;
  color: var(--text-dim);
  min-width: 60px;
}

.f-path {
  flex: 1;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 结果统计 */
.result-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  text-align: center;
  transition: all 0.3s ease;
}

.stat-success {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
}

.stat-error {
  background: var(--accent-rose-bg);
  border-color: var(--accent-rose-border);
}

.stat-number {
  display: block;
  font-size: 32px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
  margin-bottom: 8px;
}

.stat-success .stat-number { color: var(--accent-emerald); }
.stat-error .stat-number { color: var(--accent-rose); }

.stat-label {
  font-size: 12px;
  color: var(--text-dim);
}

/* 结果列表 */
.result-list {
  list-style: none;
  padding: 0;
  margin: 0;
  max-height: 300px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.result-list li {
  display: grid;
  grid-template-columns: 100px 180px 1fr;
  gap: 12px;
  padding: 10px 14px;
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}

.result-list li:last-child {
  border-bottom: none;
}

.result-ok {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
}

.result-error {
  background: var(--accent-rose-bg);
  color: var(--accent-rose);
}

.result-ok .r-name,
.result-error .r-name { color: var(--text); }

.result-ok .r-tool,
.result-error .r-tool { color: var(--text-dim); }

.r-tool {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
}

.r-name {
  color: var(--text);
}

.r-msg {
  color: inherit;
}

/* 空状态 */
.empty-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--text-faint);
  background: var(--bg-subtle);
  border: 1px dashed var(--border);
  border-radius: var(--radius);
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--text);
  margin: 12px 0 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .steps {
    flex-direction: column;
    align-items: stretch;
  }

  .step-connector {
    width: 2px;
    height: 12px;
    margin: 0;
    align-self: flex-start;
    margin-left: 30px;
  }

  .result-stats {
    grid-template-columns: 1fr;
  }

  .found-item {
    flex-wrap: wrap;
  }
}
</style>
