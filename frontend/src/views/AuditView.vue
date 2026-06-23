<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listAuditLogs, getAuditStats } from '@/api/skillbox/audit'

const { t } = useI18n()

const backendReady = ref(false)
const loading = ref(false)
const error = ref('')

const logs = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const stats = ref({ total: 0, by_action: {}, by_actor: {} })

const filterAction = ref('')
const filterActor = ref('')
const filterTargetType = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

const ACTION_OPTIONS = [
  '', 'create', 'update', 'delete',
  'apply', 'undo',
  'tag_create', 'tag_delete', 'tag_rollback',
  'test_run', 'market_install', 'onboarding_import',
  'project_create', 'project_delete',
]

async function loadStats() {
  try {
    const s = await getAuditStats()
    stats.value = s || { total: 0, by_action: {}, by_actor: {} }
    backendReady.value = true
  } catch (e) {
    backendReady.value = false
  }
}

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const res = await listAuditLogs({
      page: page.value,
      size,
      action: filterAction.value || undefined,
      actor: filterActor.value || undefined,
      target_type: filterTargetType.value || undefined,
    })
    logs.value = res?.items || []
    total.value = res?.total || 0
    backendReady.value = true
  } catch (e) {
    error.value = e?.message || String(e)
    logs.value = []
    total.value = 0
    backendReady.value = false
  } finally {
    loading.value = false
  }
}

function reload() { page.value = 1; loadLogs() }
function gotoPage(p) { if (p >= 1 && p <= totalPages.value) { page.value = p; loadLogs() } }

const actionColor = (a) => {
  if (!a) return ''
  if (a.startsWith('create') || a.startsWith('tag_create') || a === 'market_install' || a === 'onboarding_import' || a === 'project_create') return 'ok'
  if (a.startsWith('delete') || a === 'project_delete' || a === 'tag_delete') return 'err'
  if (a.startsWith('undo') || a === 'tag_rollback') return 'warn'
  return ''
}

onMounted(async () => {
  await loadStats()
  await loadLogs()
})
</script>

<template>
  <div class="audit">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-amber">
          <Icon icon="mdi:script-text-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('audit.title') }}</h1>
          <p>{{ t('audit.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- 概览卡片 -->
    <div class="stats-row">
      <div class="stat-card stat-main">
        <div class="stat-label">{{ t('audit.statTotal') }}</div>
        <div class="stat-value">{{ stats.total || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ t('audit.statByAction') }}</div>
        <div class="chips-container">
          <span v-for="(c, a) in (stats.by_action || {})" :key="a" class="chip">
            <code>{{ a }}</code> <strong>×{{ c }}</strong>
          </span>
          <span v-if="!Object.keys(stats.by_action || {}).length" class="muted">{{ t('common.dash') }}</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ t('audit.statByActor') }}</div>
        <div class="chips-container">
          <span v-for="(c, a) in (stats.by_actor || {})" :key="a" class="chip">
            <code>{{ a }}</code> <strong>×{{ c }}</strong>
          </span>
          <span v-if="!Object.keys(stats.by_actor || {}).length" class="muted">{{ t('common.dash') }}</span>
        </div>
      </div>
    </div>

    <!-- 后端未就绪占位 -->
    <div v-if="!backendReady" class="card placeholder">
      <div class="empty-state">
        <Icon icon="mdi:construction" width="48" height="48" />
        <h3 class="empty-title">{{ t('audit.placeholderTitle') }}</h3>
        <p class="empty-desc">{{ t('audit.placeholderHint1') }}</p>
        <p class="empty-desc">{{ t('audit.placeholderHint2') }}</p>
      </div>
    </div>

    <!-- 列表 -->
    <div v-else class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
          {{ t('audit.listTitle') }}
          <span class="card-sub">— {{ t('common.totalCount', { count: total }) }}</span>
        </h3>
      </header>

      <!-- 过滤器 -->
      <div class="filters">
        <div class="filter-group">
          <label class="filter-label">{{ t('audit.filterAction') }}</label>
          <select v-model="filterAction" @change="reload">
            <option v-for="a in ACTION_OPTIONS" :key="a" :value="a">{{ a || t('common.all') }}</option>
          </select>
        </div>
        <div class="filter-group">
          <label class="filter-label">{{ t('audit.filterActor') }}</label>
          <input v-model="filterActor" :placeholder="t('audit.actorPlaceholder')" @keyup.enter="reload" />
        </div>
        <div class="filter-group">
          <label class="filter-label">{{ t('audit.filterTargetType') }}</label>
          <input v-model="filterTargetType" :placeholder="t('audit.targetTypePlaceholder')" @keyup.enter="reload" />
        </div>
        <button class="primary filter-btn" @click="reload">
          <Icon icon="mdi:magnify" width="14" height="14" />
          {{ t('common.applyFilter') }}
        </button>
      </div>

      <div class="table-container">
        <table v-if="logs.length" class="grid">
          <thead>
            <tr>
              <th style="width: 70px">{{ t('audit.colId') }}</th>
              <th style="width: 160px">{{ t('audit.colTime') }}</th>
              <th style="width: 120px">{{ t('audit.colActor') }}</th>
              <th style="width: 140px">{{ t('audit.colAction') }}</th>
              <th style="width: 180px">{{ t('audit.colTarget') }}</th>
              <th>{{ t('audit.colPayload') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.ID || log.id">
              <td class="td-id">{{ log.ID || log.id }}</td>
              <td class="td-time">{{ (log.CreatedAt || log.created_at || '').slice(0, 19) }}</td>
              <td><code>{{ log.Actor || log.actor }}</code></td>
              <td>
                <span :class="['action-badge', `action-${actionColor(log.Action || log.action)}`]">
                  {{ log.Action || log.action }}
                </span>
              </td>
              <td>
                <code class="target-code">{{ (log.TargetType || log.target_type) }}#{{ log.TargetID || log.target_id }}</code>
              </td>
              <td class="td-payload">
                <details class="payload-details">
                  <summary>{{ t('audit.seeMore') }}</summary>
                  <pre class="payload-content">{{ log.Payload || log.payload || t('common.dash') }}</pre>
                </details>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else-if="!loading" class="empty-state">
          <Icon icon="mdi:inbox-outline" width="48" height="48" />
          <p class="empty-title">{{ t('audit.empty') }}</p>
        </div>
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">
          <Icon icon="mdi:chevron-left" width="14" height="14" />
          {{ t('common.prev') }}
        </button>
        <span class="pager-info">{{ t('common.pageOf', { page, total: totalPages, count: total }) }}</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="14" height="14" />
        </button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.audit {
  max-width: 1100px;
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
  background: var(--text);
  color: var(--bg-card);
  flex-shrink: 0;
}

.view-icon-amber {
  background: var(--text-dim);
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

/* 统计卡片 */
.stats-row {
  display: grid;
  grid-template-columns: 1fr 2fr 2fr;
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
  transition: all 0.3s ease;
}

.stat-main {
  background: var(--text);
  border: none;
  color: var(--bg-card);
}

.stat-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-dim);
  margin-bottom: 8px;
}

.stat-main .stat-label {
  color: rgba(255, 255, 255, 0.8);
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
}

.stat-main .stat-value {
  color: var(--bg-card);
}

.chips-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 4px;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  background: var(--bg-subtle);
  border-radius: var(--radius-full);
  font-size: 11px;
  color: var(--text-dim);
  transition: all 0.3s ease;
}

.chip code {
  background: transparent;
  color: var(--primary);
  padding: 0;
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

/* 过滤器 */
.filters {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 150px;
}

.filter-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.filter-group select,
.filter-group input {
  padding: 8px 12px;
  min-width: 120px;
}

.filter-btn {
  height: 38px;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* 表格 */
.table-container {
  overflow-x: auto;
  margin: 0 -20px;
  padding: 0 20px;
}

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

.td-id {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-faint);
}

.td-time {
  color: var(--text-dim);
  font-size: 12px;
  white-space: nowrap;
}

.target-code {
  font-size: 12px;
  background: var(--primary-dim);
  color: var(--primary);
  padding: 2px 8px;
  border-radius: 4px;
}

/* 操作徽章 */
.action-badge {
  display: inline-flex;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  background: var(--bg-subtle);
  color: var(--text-dim);
  white-space: nowrap;
}

.action-ok {
  background: var(--success-dim);
  color: var(--success);
}

.action-err {
  background: var(--danger-dim);
  color: var(--danger);
}

.action-warn {
  background: var(--warning-dim);
  color: var(--warning);
}

/* Payload 详情 */
.td-payload {
  max-width: 200px;
}

.payload-details summary {
  cursor: pointer;
  color: var(--primary);
  font-size: 12px;
  user-select: none;
}

.payload-content {
  background: var(--bg-subtle);
  padding: 12px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  max-height: 200px;
  overflow: auto;
  margin-top: 8px;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 分页器 */
.pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.pager button {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
}

.pager-info {
  font-size: 13px;
  color: var(--text-dim);
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

.empty-desc {
  font-size: 13px;
  color: var(--text-dim);
  margin: 4px 0 0;
}

/* 占位符 */
.placeholder .empty-state {
  padding: 48px 24px;
}

/* 响应式 */
@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .filters {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-group {
    min-width: auto;
  }

  .filter-group select,
  .filter-group input {
    width: 100%;
  }

  .table-container {
    margin: 0 -16px;
    padding: 0 16px;
  }
}
</style>
