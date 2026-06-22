<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import {
  listSources,
  listMarketSkills,
  refreshSource,
  installMarketSkill,
} from '@/api/skillbox/market.js'

const { t } = useI18n()

// 状态
const loading = ref(false)
const error = ref('')

// 源
const sources = ref([])
const activeSourceId = ref(0)
const refreshing = ref(false)
const lastRefresh = ref(null)

// 列表
const keyword = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

// 安装
const installScope = ref('global')
const installing = ref(false)
const installError = ref('')
const installOk = ref('')

async function fetchSources() {
  try {
    const res = await listSources()
    sources.value = res.items || []
    if (sources.value.length > 0 && !activeSourceId.value) {
      activeSourceId.value = sources.value[0].id
    }
  } catch (e) {
    error.value = t('market.errLoadSources', { msg: e?.message || e })
  }
}

async function fetchSkills() {
  if (!activeSourceId.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await listMarketSkills({
      source_id: activeSourceId.value,
      keyword: keyword.value,
      page: page.value,
      size,
    })
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    error.value = t('market.errLoadList', { msg: e?.message || e })
  } finally {
    loading.value = false
  }
}

async function onRefresh() {
  if (!activeSourceId.value || refreshing.value) return
  refreshing.value = true
  error.value = ''
  try {
    const res = await refreshSource(activeSourceId.value)
    lastRefresh.value = res
    page.value = 1
    await fetchSkills()
  } catch (e) {
    error.value = t('market.errRefresh', { msg: e?.message || e })
  } finally {
    refreshing.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchSkills()
}

function onSelectSource(id) {
  activeSourceId.value = id
  page.value = 1
  lastRefresh.value = null
  fetchSkills()
}

async function onInstall(item) {
  if (!confirm(t('market.installConfirm', { name: item.name, scope: installScope.value }))) return
  installing.value = true
  installError.value = ''
  installOk.value = ''
  try {
    const res = await installMarketSkill({
      source_id: activeSourceId.value,
      remote_id: item.remote_id,
      scope: installScope.value,
      project_id: 0,
    })
    installOk.value = t('market.okInstalled', { name: res?.skill?.name || item.name, version: res?.skill?.version || '?' })
  } catch (e) {
    installError.value = t('market.errInstall', { msg: e?.message || e })
  } finally {
    installing.value = false
  }
}

onMounted(async () => {
  await fetchSources()
  await fetchSkills()
})
</script>

<template>
  <div class="market">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-orange">
          <Icon icon="mdi:cart-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('market.title') }}</h1>
          <p>{{ t('market.subtitle') }}</p>
        </div>
      </div>
    </header>

    <div class="card">
      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <span class="toolbar-label">{{ t('market.scopeLabel') }}</span>
          <select v-model="installScope" class="scope-select">
            <option value="global">{{ t('market.scopeGlobal') }}</option>
            <option value="project" disabled>{{ t('market.scopeProject') }}</option>
          </select>
        </div>
        <div class="toolbar-center">
          <div class="search-box">
            <Icon icon="mdi:magnify" width="16" height="16" class="search-icon" />
            <input
              v-model="keyword"
              type="text"
              :placeholder="t('market.searchPlaceholder')"
              class="search-input"
              @keyup.enter="onSearch"
            />
          </div>
          <button class="ghost" @click="onSearch">
            <Icon icon="mdi:magnify" width="14" height="14" />
            {{ t('common.search') }}
          </button>
        </div>
        <div class="toolbar-right">
          <button class="primary" :disabled="refreshing || !activeSourceId" @click="onRefresh">
            <span v-if="refreshing" class="spinner"></span>
            <Icon v-else icon="mdi:refresh" width="14" height="14" />
            {{ refreshing ? t('market.refreshing') : t('market.btnRefresh') }}
          </button>
        </div>
      </div>

      <!-- 源选择标签 -->
      <nav class="source-tabs">
        <button
          v-for="s in sources"
          :key="s.id"
          :class="['source-tab', { active: s.id === activeSourceId }]"
          @click="onSelectSource(s.id)"
        >
          <Icon icon="mdi:radio-tower" width="14" height="14" />
          {{ s.name }}
          <span class="source-type">{{ s.type }}</span>
        </button>
        <span v-if="!sources.length && !loading" class="source-empty">{{ t('market.noSources') }}</span>
      </nav>

      <!-- 消息提示 -->
      <div v-if="error" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ error }}
      </div>
      <div v-if="lastRefresh" class="message message-success">
        <Icon icon="mdi:check-circle-outline" width="14" height="14" />
        {{ t('market.lastRefresh', { pulled: lastRefresh.pulled_count, inserted: lastRefresh.inserted, updated: lastRefresh.updated }) }}
        <span class="muted">({{ lastRefresh.finished_at }})</span>
      </div>
      <div v-if="installOk" class="message message-success">
        <Icon icon="mdi:check-circle-outline" width="14" height="14" />
        {{ installOk }}
      </div>
      <div v-if="installError" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ installError }}
      </div>

      <!-- 列表 -->
      <div class="table-container">
        <table v-if="items.length > 0" class="grid">
          <thead>
            <tr>
              <th>{{ t('market.colName') }}</th>
              <th>{{ t('market.colVersion') }}</th>
              <th>{{ t('market.colAuthor') }}</th>
              <th>{{ t('market.colDescription') }}</th>
              <th>{{ t('market.colTags') }}</th>
              <th style="width: 100px"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in items" :key="it.remote_id">
              <td>
                <span class="item-name">{{ it.name }}</span>
                <span class="item-id">{{ it.remote_id }}</span>
              </td>
              <td><code>{{ it.version || t('common.dash') }}</code></td>
              <td>{{ it.author || t('common.dash') }}</td>
              <td class="item-desc">{{ it.description || t('common.dash') }}</td>
              <td>
                <span v-for="tg in (it.tags || '').split(',').filter(Boolean)" :key="tg" class="tag">
                  {{ tg }}
                </span>
              </td>
              <td>
                <button class="install-btn" :disabled="installing" @click="onInstall(it)">
                  <Icon icon="mdi:download" width="12" height="12" />
                  {{ installing ? t('market.installing') : t('market.btnInstall') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else-if="!loading" class="empty-state">
          <Icon icon="mdi:radio-tower" width="48" height="48" />
          <p class="empty-title">{{ t('market.emptyFirstTime') }}</p>
        </div>
        <div v-else class="loading-state">
          <span class="spinner"></span>
          <p>{{ t('market.loading') }}</p>
        </div>
      </div>

      <!-- 分页 -->
      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="page <= 1" @click="page--; fetchSkills()">
          <Icon icon="mdi:chevron-left" width="14" height="14" />
          {{ t('common.prev') }}
        </button>
        <span class="pager-info">{{ t('common.pageOf', { page, total: totalPages, count: total }) }}</span>
        <button :disabled="page >= totalPages" @click="page++; fetchSkills()">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="14" height="14" />
        </button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.market {
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

.view-icon-orange {
  background: linear-gradient(135deg, #f59e0b 0%, #ea580c 100%);
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

/* 卡片 */
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  padding: 20px;
  transition: all 0.3s ease;
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.toolbar-left, .toolbar-center, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.toolbar-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
}

.scope-select {
  padding: 8px 12px;
  min-width: 120px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-faint);
  pointer-events: none;
}

.search-input {
  padding-left: 36px;
  width: 280px;
}

/* 源标签 */
.source-tabs {
  display: flex;
  gap: 6px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}

.source-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s ease;
}

.source-tab:hover:not(.active) {
  background: var(--bg-hover);
  border-color: var(--text-faint);
  color: var(--text);
}

.source-tab.active {
  background: var(--primary);
  border-color: var(--primary);
  color: white;
}

.source-type {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  text-transform: uppercase;
}

.source-tab:not(.active) .source-type {
  background: var(--bg-hover);
  color: var(--text-faint);
}

.source-empty {
  padding: 8px 12px;
  color: var(--text-faint);
  font-size: 13px;
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
  vertical-align: top;
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

.item-name {
  font-weight: 600;
  color: var(--text);
  display: block;
}

.item-id {
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
}

.item-desc {
  color: var(--text-dim);
  max-width: 360px;
}

.tag {
  display: inline-block;
  background: linear-gradient(135deg, rgba(124, 58, 237, 0.1) 0%, rgba(124, 58, 237, 0.05) 100%);
  color: #7c3aed;
  border: 1px solid rgba(124, 58, 237, 0.2);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 11px;
  margin: 2px;
}

/* 安装按钮 */
.install-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(16, 185, 129, 0.05) 100%);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #059669;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.install-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, rgba(16, 185, 129, 0.1) 100%);
  border-color: rgba(16, 185, 129, 0.4);
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

.loading-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--text-faint);
}

/* 响应式 */
@media (max-width: 768px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left, .toolbar-center, .toolbar-right {
    justify-content: center;
    flex-wrap: wrap;
  }

  .search-input {
    width: 100%;
  }

  .table-container {
    margin: 0 -16px;
    padding: 0 16px;
  }
}
</style>
