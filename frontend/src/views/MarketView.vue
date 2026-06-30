<script setup>
import { ref, reactive, computed, onMounted, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { useMarketStore } from '@/core/store/market'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'
import MarketInstallConfirm from '@/components/MarketInstallConfirm.vue'
import MarketSourceSettings from '@/components/MarketSourceSettings.vue'

const { t } = useI18n()
const market = useMarketStore()
const toast = useToastStore()

// 注入 appBus 用于跳到 skills tab
const appBus = inject('appBus', null)

// 状态
const error = computed(() => market.lastError)
const loading = computed(() => market.loading)
const refreshing = computed(() => market.refreshing)

// 列表
const items = computed(() => market.skills)
const total = computed(() => market.total)
const page = computed(() => market.page)
const size = computed(() => market.size)
const totalPages = computed(() => market.totalPages)
const installed = computed(() => market.installed)

// 源
const sources = computed(() => market.sources)
const activeSourceId = computed(() => market.activeSourceId)

// 工具栏
const keyword = ref('')

function onSearch() {
  market.setKeyword(keyword.value)
  market.loadSkills()
}

function onSelectSource(id) {
  market.setSourceActive(id)
  market.loadSkills()
}

async function onRefresh() {
  try {
    await market.refreshActive()
    toast.push({ type: 'success', message: t('market.lastRefresh', market.lastRefresh || {}) })
  } catch (e) {
    toast.push({ type: 'error', message: t('market.errRefresh', { msg: e?.message || e }) })
  }
}

// 详情弹窗
const detailOpen = ref(false)
const detailItem = ref(null)
function openDetail(item) {
  detailItem.value = item
  detailOpen.value = true
}

// 源设置弹窗
const settingsOpen = ref(false)
function openSettings() {
  settingsOpen.value = true
}

// 安装弹窗
const installOpen = ref(false)
const installItem = ref(null)
function openInstall(item) {
  installItem.value = item
  installOpen.value = true
}

async function onInstallConfirm(payload) {
  try {
    const res = await market.install(payload)
    installOpen.value = false
    // 根据 apply 结果给 toast
    if (res?.skipped_tools?.length && res.skipped_tools.length > 0) {
      toast.push({
        type: 'info',
        message: t('market.installDialog.applyPartial', {
          n: res.skipped_tools.length,
          tools: res.skipped_tools.join(', '),
        }),
      })
    } else if (res?.apply_result?.all_ok) {
      toast.push({
        type: 'success',
        message: t('market.installDialog.applyAllOk', { n: res.apply_result?.applies?.length || 0 }),
      })
    } else {
      toast.push({ type: 'success', message: t('market.okInstalled', { name: res?.name, version: res?.version }) })
    }
    // 刷新列表(更新 installed 标记)
    await market.loadSkills()
  } catch (e) {
    toast.push({ type: 'error', message: t('market.errInstall', { msg: e?.message || e }) })
  }
}

// 跳到 skills tab(已安装时查看)
function viewSkill(name) {
  if (appBus && typeof appBus.emit === 'function') {
    appBus.emit('switch-tab', 'skills')
  } else {
    window.dispatchEvent(new CustomEvent('skillbox:switch-tab', { detail: 'skills' }))
  }
}

onMounted(async () => {
  try {
    await market.loadSources()
    await market.loadProjects()
    await market.loadSkills()
  } catch (e) {
    // error 已在 store 里
  }
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
          <button class="ghost" :disabled="!sources.length" @click="openSettings">
            <Icon icon="mdi:cog-outline" width="14" height="14" />
            {{ t('market.btnSourceSettings') }}
          </button>
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

      <!-- 错误提示 -->
      <div v-if="error" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ error }}
      </div>
      <div v-if="market.lastRefresh" class="message message-success">
        <Icon icon="mdi:check-circle-outline" width="14" height="14" />
        {{ t('market.lastRefresh', { pulled: market.lastRefresh.pulled_count, inserted: market.lastRefresh.inserted, updated: market.lastRefresh.updated }) }}
        <span class="muted">({{ market.lastRefresh.finished_at }})</span>
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
              <th>{{ t('market.colStatus') }}</th>
              <th style="width: 200px"></th>
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
                <span v-if="installed[it.name]" class="installed-chip">
                  <Icon icon="mdi:check-circle" width="12" height="12" />
                  {{ t('market.installedChip') }}
                </span>
                <span v-else class="not-installed-chip">
                  <Icon icon="mdi:circle-outline" width="12" height="12" />
                  {{ t('market.notInstalledChip') }}
                </span>
              </td>
              <td>
                <div class="row-actions">
                  <button class="action-btn" :title="t('common.edit')" @click="openDetail(it)">
                    <Icon icon="mdi:eye-outline" width="12" height="12" />
                  </button>
                  <button v-if="installed[it.name]" class="action-btn" :title="t('market.btnViewSkill')" @click="viewSkill(it.name)">
                    <Icon icon="mdi:open-in-new" width="12" height="12" />
                  </button>
                  <button class="install-btn" :disabled="market.installing" @click="openInstall(it)">
                    <Icon icon="mdi:download" width="12" height="12" />
                    {{ installed[it.name] ? t('market.btnReinstall') : t('market.btnInstall') }}
                  </button>
                </div>
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
        <button :disabled="page <= 1" @click="market.page--; market.loadSkills()">
          <Icon icon="mdi:chevron-left" width="14" height="14" />
          {{ t('common.prev') }}
        </button>
        <span class="pager-info">{{ t('common.pageOf', { page, total: totalPages, count: total }) }}</span>
        <button :disabled="page >= totalPages" @click="market.page++; market.loadSkills()">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="14" height="14" />
        </button>
      </footer>
    </div>

    <!-- 详情弹窗 -->
    <Modal
      v-model="detailOpen"
      size="lg"
      :title="detailItem?.name || ''"
    >
      <template #title-icon>
        <Icon icon="mdi:information-outline" width="18" height="18" />
      </template>
      <div v-if="detailItem" class="detail-grid">
        <div class="detail-row">
          <span class="detail-label">{{ t('market.colVersion') }}</span>
          <code>{{ detailItem.version || t('common.dash') }}</code>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('market.colAuthor') }}</span>
          <span>{{ detailItem.author || t('common.dash') }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">ID</span>
          <code class="detail-id">{{ detailItem.remote_id }}</code>
        </div>
        <div class="detail-row detail-row-full">
          <span class="detail-label">{{ t('market.colDescription') }}</span>
          <p class="detail-desc">{{ detailItem.description || t('common.dash') }}</p>
        </div>
        <div v-if="detailItem.tags" class="detail-row detail-row-full">
          <span class="detail-label">{{ t('market.colTags') }}</span>
          <div class="detail-tags">
            <span v-for="tg in String(detailItem.tags).split(',').filter(Boolean)" :key="tg" class="tag">
              {{ tg }}
            </span>
          </div>
        </div>
        <div class="detail-row detail-row-full">
          <span class="detail-label">{{ t('market.colStatus') }}</span>
          <span v-if="installed[detailItem.name]" class="installed-chip">
            <Icon icon="mdi:check-circle" width="12" height="12" />
            {{ t('market.installedChip') }}
          </span>
          <span v-else class="not-installed-chip">
            <Icon icon="mdi:circle-outline" width="12" height="12" />
            {{ t('market.notInstalledChip') }}
          </span>
        </div>
      </div>
      <template #footer>
        <button type="button" class="ghost" @click="detailOpen = false">
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.close') }}
        </button>
        <button
          type="button"
          class="primary"
          :disabled="market.installing"
          @click="detailOpen = false; openInstall(detailItem)"
        >
          <Icon icon="mdi:download" width="14" height="14" />
          {{ t('market.btnInstall') }}
        </button>
      </template>
    </Modal>

    <!-- 安装弹窗 -->
    <MarketInstallConfirm
      v-model="installOpen"
      :item="installItem"
      :installed="installed"
      :projects="market.projects"
      @confirm="onInstallConfirm"
      @cancel="installOpen = false"
    />

    <!-- 源设置弹窗 -->
    <MarketSourceSettings v-model="settingsOpen" />
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

.view-icon-orange {
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
  color: var(--bg-card);
}

.source-type {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.2);
  text-transform: uppercase;
}

.source-tab:not(.active) .source-type {
  background: var(--bg-subtle);
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

.muted {
  color: var(--text-faint);
  font-size: 11px;
  margin-left: 4px;
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
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 11px;
  margin: 2px;
}

/* 已安装/未安装 chip */
.installed-chip,
.not-installed-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}

.installed-chip {
  background: var(--success-dim);
  color: var(--success);
  border: 1px solid var(--success);
}

.not-installed-chip {
  background: var(--bg-subtle);
  color: var(--text-faint);
  border: 1px solid var(--border);
}

/* 安装按钮 */
.install-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  background: var(--text);
  border: 1px solid var(--text);
  color: var(--bg-card);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.install-btn:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

/* 行内操作按钮组 */
.row-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  border-color: var(--text-faint);
  color: var(--text);
}

/* 详情弹窗 */
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.detail-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.detail-row-full {
  grid-column: 1 / -1;
}

.detail-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.detail-id {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-faint);
  word-break: break-all;
}

.detail-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
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

.loading-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--text-faint);
}

.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--text-faint);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
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
