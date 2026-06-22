<script setup>
import { ref, reactive, computed, onMounted, inject } from 'vue'
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

const importResult = ref(null)

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
    selected.value = new Set((res.found || []).map((f) => keyOf(f)))
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

function selectAll() { selected.value = new Set((scanReport.value?.found || []).map(keyOf)) }
function selectNone() { selected.value = new Set() }

const foundByTool = computed(() => {
  const groups = {}
  for (const f of scanReport.value?.found || []) {
    if (!groups[f.tool_id]) groups[f.tool_id] = { name: f.tool_name, items: [] }
    groups[f.tool_id].items.push(f)
  }
  return groups
})

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
        <div class="view-icon view-icon-teal">
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
            <th style="width: 50px"></th>
            <th>{{ t('onboarding.phase1.colTool') }}</th>
            <th>{{ t('onboarding.phase1.colId') }}</th>
            <th>{{ t('onboarding.phase1.colGlobalPath') }}</th>
            <th>{{ t('onboarding.phase1.colStatus') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in adapters" :key="a.tool_id">
            <td class="icon-cell">{{ a.icon }}</td>
            <td><strong>{{ a.display_name }}</strong></td>
            <td><code>{{ a.tool_id }}</code></td>
            <td class="td-path">{{ a.global_path || t('common.dash') }}</td>
            <td>
              <span v-if="a.global_ok" class="badge badge-success">{{ t('onboarding.phase1.detected') }}</span>
              <span v-else class="badge badge-muted">{{ t('onboarding.phase1.missing') }}</span>
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

    <!-- 阶段 2: 扫描 + 勾选 -->
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
        <div class="bulk-actions">
          <button class="sm" @click="selectAll">{{ t('onboarding.phase2.selectAll') }}</button>
          <button class="sm ghost" @click="selectNone">{{ t('onboarding.phase2.selectNone') }}</button>
          <span class="selection-info">{{ t('onboarding.phase2.selected', { sel: selected.size, total: scanReport.found.length }) }}</span>
        </div>

        <div v-for="(g, tid) in foundByTool" :key="tid" class="tool-group">
          <header class="group-header">
            <span class="group-name">{{ g.name }}</span>
            <code class="group-id">{{ tid }}</code>
            <span class="group-count">{{ g.items.length }}</span>
          </header>
          <ul class="found-list">
            <li v-for="f in g.items" :key="keyOf(f)" :class="{ selected: selected.has(keyOf(f)) }">
              <label class="found-item">
                <input
                  type="checkbox"
                  :checked="selected.has(keyOf(f))"
                  @change="toggleSelect(f)"
                />
                <span class="f-name"><code>{{ f.name }}</code></span>
                <span class="f-ver">v{{ f.version }}</span>
                <span class="f-path">{{ f.source_path }}</span>
              </label>
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
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0d9488 0%, #f59e0b 100%);
  color: white;
  flex-shrink: 0;
}

.view-icon-teal {
  background: linear-gradient(135deg, #0d9488 0%, #14b8a6 100%);
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
  background: var(--primary-dim);
  border-color: var(--primary);
}

.step.done {
  background: var(--success-dim);
  border-color: var(--success);
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
  background: var(--bg-hover);
  color: var(--text-dim);
  flex-shrink: 0;
  transition: all 0.3s ease;
}

.step.active .step-number {
  background: var(--primary);
  color: white;
}

.step.done .step-number {
  background: var(--success);
  color: white;
}

.step-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-dim);
  transition: color 0.3s ease;
}

.step.active .step-title {
  color: var(--primary);
}

.step.done .step-title {
  color: var(--success);
}

.step-connector {
  flex: 1;
  height: 2px;
  background: var(--border);
  margin: 0 8px;
  transition: background 0.3s ease;
}

.step-connector.done {
  background: var(--success);
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
  background: var(--bg-hover);
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
  font-size: 20px;
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
  background: var(--success-dim);
  color: var(--success);
}

.badge-muted {
  background: var(--bg-hover);
  color: var(--text-dim);
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

/* 工具组 */
.tool-group {
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  transition: border-color 0.3s ease;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--bg-hover);
  border-bottom: 1px solid var(--border);
  font-size: 13px;
}

.group-name {
  font-weight: 600;
  color: var(--text);
}

.group-id {
  font-size: 11px;
  color: var(--text-faint);
}

.group-count {
  margin-left: auto;
  padding: 2px 8px;
  background: var(--bg-card);
  border-radius: var(--radius-full);
  font-size: 11px;
  color: var(--text-dim);
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
  background: var(--primary-dim);
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
  background: var(--bg-hover);
  border-radius: var(--radius);
  text-align: center;
  transition: all 0.3s ease;
}

.stat-success {
  background: var(--success-dim);
}

.stat-error {
  background: var(--danger-dim);
}

.stat-number {
  display: block;
  font-size: 32px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
  margin-bottom: 8px;
}

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
  color: var(--success);
}

.result-error {
  color: var(--danger);
}

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
  background: var(--bg-hover);
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
