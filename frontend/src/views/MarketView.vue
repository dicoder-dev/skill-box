<script setup>
import { ref, reactive, computed, onMounted, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { useMarketStore } from '@/core/store/market'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'
import MarketPullConfirm from '@/components/MarketPullConfirm.vue'
import MarketSourceSettings from '@/components/MarketSourceSettings.vue'

const { t } = useI18n()
const market = useMarketStore()
const toast = useToastStore()

// 注入 appBus 用于跳到 skills tab
const appBus = inject('appBus', null)

// 状态
const error = computed(() => market.lastError)
const loading = computed(() => market.loading)
// 2026-07-01 改:全走 API 后只剩 loading 单 flag。
// 每次进入/切 tab/输入搜索,都会走远端,loading 是正反馈。
const refreshing = computed(() => market.loading)

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
  // 2026-07-01 改:全走 API,Enter 走 setKeyword + loadSkills(每次都打远端)。
  // skillhub 走 ?keyword= 搜索语义;skills.sh 走 50 页 + substring。
  market.setKeyword(keyword.value)
  market.loadSkills()
}

async function onSelectSource(id) {
  // 2026-07-01 改:全走 API 后每次切源都重新打远端(纯 API,无缓存判断)。
  market.setSourceActive(id)
  await market.loadSkills()
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
// 2026-07-01 改:MarketInstallConfirm → MarketPullConfirm。
const installOpen = ref(false)
const installItem = ref(null)
function openInstall(item) {
  installItem.value = item
  installOpen.value = true
}

async function onInstallConfirm(payload) {
  try {
    const res = await market.pull(payload)
    installOpen.value = false
    // 根据 apply 结果给 toast
    if (res?.skipped_tools?.length && res.skipped_tools.length > 0) {
      toast.push({
        type: 'info',
        message: t('market.pullDialog.applyPartial', {
          n: res.skipped_tools.length,
          tools: res.skipped_tools.join(', '),
        }),
      })
    } else if (res?.apply_result?.all_ok) {
      toast.push({
        type: 'success',
        message: t('market.pullDialog.applyAllOk', { n: res.apply_result?.applies?.length || 0 }),
      })
    } else {
      toast.push({ type: 'success', message: t('market.okPulled', { name: res?.name, version: res?.version }) })
    }
    // 刷新列表(更新 installed 标记)
    await market.loadSkills()
  } catch (e) {
    toast.push({ type: 'error', message: t('market.errPull', { msg: e?.message || e }) })
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
    if (market.activeSourceId) {
      // 2026-07-01 改:全走 API,直接 loadSkills 即可。
      // 每次都打远端,loading 是正反馈,失败有 banner + toast。
      await market.loadSkills()
    }
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
          <!-- 2026-07-01 改:全走 API 后,Enter 已经每次都打远端,搜索按钮被删除。
               工具栏右侧只留「源设置」一个 action;搜索 = 直接 Enter 输入框即可触发。 -->
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

      <!-- 中部可滚动主体:网格/空态/加载态;
           内部 overflow-y:auto 让卡片多时只滚这一段,分页栏固定在下方 -->
      <div class="market-body">
        <!-- 错误提示 -->
        <div v-if="error" class="message message-error">
          <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ error }}
        </div>

        <!-- 2026-07-01 改:列表用卡片网格(原表格) -->
        <div v-if="items.length > 0" class="market-grid">
          <article
            v-for="it in items"
            :key="it.remote_id"
            class="market-card"
            :class="{ 'is-installed': installed[it.name] }"
          >
            <header class="market-card-top">
              <div class="market-card-icon">
                <Icon icon="mdi:puzzle-outline" width="22" height="22" />
              </div>
              <div class="market-card-titles">
                <h3 class="market-card-name" :title="it.name">{{ it.name }}</h3>
                <code class="market-card-id">{{ it.remote_id }}</code>
              </div>
              <span v-if="installed[it.name]" class="badge badge-installed">
                <Icon icon="mdi:check-circle" width="10" height="10" />
                {{ t('market.installedChip') }}
              </span>
              <span v-else class="badge badge-not-installed">
                <Icon icon="mdi:circle-outline" width="10" height="10" />
                {{ t('market.notInstalledChip') }}
              </span>
            </header>

            <div class="market-card-meta">
              <span class="meta-item">
                <Icon icon="mdi:tag-outline" width="12" height="12" />
                {{ it.version || t('common.dash') }}
              </span>
              <span class="meta-item">
                <Icon icon="mdi:account-outline" width="12" height="12" />
                {{ it.author || t('common.dash') }}
              </span>
            </div>

            <p class="market-card-desc" :title="it.description">{{ it.description || t('common.dash') }}</p>

            <div v-if="it.tags" class="market-card-tags">
              <span v-for="tg in String(it.tags).split(',').filter(Boolean)" :key="tg" class="tag">{{ tg }}</span>
            </div>

            <footer class="market-card-bottom">
              <span class="market-card-bottom-spacer"></span>
              <div class="market-card-actions">
                <Icon
                  icon="mdi:eye-outline"
                  :title="t('market.btnViewSkill')"
                  class="action-icon action-icon-view"
                  @click="openDetail(it)"
                />
                <Icon
                  v-if="installed[it.name]"
                  icon="mdi:open-in-new"
                  :title="t('market.btnViewSkill')"
                  class="action-icon action-icon-jump"
                  @click="viewSkill(it.name)"
                />
                <button class="market-card-pull" :disabled="market.pulling" @click="openInstall(it)">
                  <Icon icon="mdi:download" width="13" height="13" />
                  {{ installed[it.name] ? t('market.btnRepull') : t('market.btnPull') }}
                </button>
              </div>
            </footer>
          </article>
        </div>

        <div v-else-if="refreshing || loading" class="loading-state">
          <span class="spinner"></span>
          <p>{{ t('market.btnRemoteLoading') }}</p>
        </div>

        <div v-else class="empty-state">
          <Icon icon="mdi:radio-tower" width="48" height="48" />
          <p class="empty-title">{{ t('market.emptyAfter') }}</p>
          <p class="empty-hint">{{ t('market.emptyAfterHint') }}</p>
        </div>
      </div>

      <!-- 分页 - 固定在卡片容器底部,不随内容滚动 -->
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
          :disabled="market.pulling"
          @click="detailOpen = false; openInstall(detailItem)"
        >
          <Icon icon="mdi:download" width="14" height="14" />
          {{ t('market.btnPull') }}
        </button>
      </template>
    </Modal>

    <!-- 拉取弹窗 -->
    <MarketPullConfirm
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
  display: flex;
  flex-direction: column;
  height: 100%;                  /* 占满 content-area(已被 app-container 锁为视口高度) */
  max-width: 1100px;
  margin: 0 auto;
  color: var(--text);
  transition: color 0.3s ease;
}

/* 页面头部 - flex 子项,不收缩不滚动 */
.view-header {
  margin-bottom: 24px;
  flex-shrink: 0;
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

/* 卡片 - flex 列占满 .market 剩余高度,内部三段各自负责;
   顶部工具栏/源标签 flex-shrink:0 不滚,.market-body flex:1 接管滚动,
   .pager flex-shrink:0 始终在 .card 底部 = 视口内可见 */
.card {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;                /* 关键:允许子项收缩到内容以下,触发内部 overflow */
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
  margin-bottom: 16px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.toolbar-left, .toolbar-center, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 工具栏内 ghost 按钮:常驻可见背景+对齐图标文字(覆盖全局 ghost 透明) */
.toolbar .ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-card);
  border-color: var(--border);
}
.toolbar .ghost:hover:not(:disabled) {
  background: var(--bg-hover);
  border-color: var(--text-faint);
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
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
  flex-shrink: 0;
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

/* 卡片网格 - 中部可滚动容器,卡片多时只滚这一段;
   flex:1 占 .card 剩余高度,min-height:0 允许收缩触发内部 overflow。
   配合 .market height:100% + .card flex:1 链路,自适应任何屏幕,
   保证顶部工具栏/分页栏始终在 .card 上下两端 = 视口上下两端。 */
.market-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding-right: 4px;       /* 给滚动条留位置,避免遮住卡片 */
  margin-right: -4px;       /* 抵消 padding-right,保持外边距不变 */
}

/* 卡片网格 */
.market-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}

.market-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 14px 12px 18px; /* 左侧 18px 给 box-shadow 留位,避开 border-radius 切割 */
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: inset 4px 0 0 transparent; /* 默认透明占位,避免布局抖动 */
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.market-card:hover {
  border-color: var(--text-faint);
  box-shadow: inset 4px 0 0 transparent, var(--shadow-card);
}

/* 2026-07-01 修:已安装/未安装用 inset box-shadow 显示左侧条带,
   替代 border-left(避免被 border-radius 圆角切割导致不可见)。
   inset shadow 不会被 border-radius 裁剪,边角清晰。 */
.market-card.is-installed {
  box-shadow: inset 4px 0 0 var(--success);
}

.market-card.is-installed:hover {
  box-shadow: inset 4px 0 0 var(--success), var(--shadow-card);
}

.market-card-top {
  display: flex;
  align-items: center;
  gap: 10px;
}

.market-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--bg-subtle);
  color: var(--text);
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.market-card-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.market-card-name {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.market-card-id {
  font-size: 11px;
  font-family: 'JetBrains Mono', monospace;
  background: var(--primary-dim);
  color: var(--primary);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  align-self: flex-start;
  max-width: fit-content;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 10px;
  font-weight: 600;
  border-radius: 999px;
  flex-shrink: 0;
}

.badge-installed {
  background: var(--success-dim);
  color: var(--success);
}

.badge-not-installed {
  background: var(--bg-subtle);
  color: var(--text-faint);
}

.market-card-meta {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-faint);
}

.market-card-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.market-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.market-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.market-card-bottom-spacer {
  flex: 1;
}

.market-card-actions {
  display: flex;
  gap: 4px;
  align-items: center;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.market-card:hover .market-card-actions {
  opacity: 1;
}

.action-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  color: var(--text-dim);
  transition: background 0.15s ease, color 0.15s ease;
}

.action-icon:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.action-icon-view:hover {
  background: var(--primary-dim);
  color: var(--primary);
}

.action-icon-jump:hover {
  background: var(--success-dim);
  color: var(--success);
}

.market-card-pull {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  font-size: 12px;
  font-weight: 500;
  background: var(--text);
  border: 1px solid var(--text);
  color: var(--bg-card);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.market-card-pull:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

/* 标签 chips(卡片里也用) */
.tag {
  display: inline-block;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 11px;
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

/* 分页器 - flex-shrink:0 在 .card flex 列里保持自然高度,不被压掉 */
.pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex-shrink: 0;
  margin: 16px auto 0;
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

.empty-hint {
  font-size: 13px;
  color: var(--text-faint);
  margin: 6px 0 0;
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

  .market-grid {
    grid-template-columns: 1fr;
  }

  .market-card-actions {
    opacity: 1;
  }
}
</style>
