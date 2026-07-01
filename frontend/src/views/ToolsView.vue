<script setup>
// ToolsView - 工具元数据管理视图。
//
// 2026-07-01 新建。对应后端 7 个 ctool 接口(2026-06-30 上线)——
// 用户可在此浏览 / 启停 / 编辑 / 增删 / 调 reload 的工具表。
//
// 布局结构(从上到下):
//   1. 页面头:标题 + 副标题(展示总数 / 系统 / 用户)
//   2. toolbar:搜索 + 三选一过滤 + 新建 + Reload
//   3. 错误提示条(只在 store.error 非空时)
//   4. 卡片网格:每张卡 1 个工具
//      - 顶部:icon + display_name + tool_id + 系统/用户徽章
//      - 中部:maturity chip + path 数 + note
//      - 底部:enabled switch(主交互) + 编辑 / 删除操作图标(hover 显)
//   5. 新建 / 编辑 Modal(size="lg",含 paths 子表) — 由 store.formOpen 控
//   6. 删除确认 Modal(size="sm")          — 由 store.confirmOpen 控
//
// 复用模式:
//   - HTTP:    @/api/skillbox/tools.js
//   - store:   @/core/store/tools.js
//   - 弹窗:    @/components/Modal.vue(已封 size / 标题图标 / 滚动锁定)
//   - 时间:    @/core/utils/time.js#formatRelative
//   - 文件夹:  platform.fs.pickFolder()
//   - i18n:    useI18n() -> t('tools.*'),共命名空间

import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { useToolsStore } from '@/core/store/tools'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'
import { formatRelative } from '@/core/utils/time.js'
import { platform } from '@/platform'

const { t } = useI18n()
const tools = useToolsStore()
const toast = useToastStore()

// 搜索框 draft:输入实时回显,Enter / @input 触发写入 store.setKeyword
const keywordDraft = ref('')
watch(() => tools.filter.keyword, (v) => {
  if (v !== keywordDraft.value) keywordDraft.value = v
})
function applyKeyword() {
  tools.setKeyword(keywordDraft.value)
}

// 过滤按钮互斥
function selectSource(src) {
  tools.setSource(src)
}

// reload:成功后弹 toast(toast 单独抽出来,失败也提示)
async function onReload() {
  try {
    await tools.reloadRegistry()
    toast.success(t('tools.reloadedOk'))
  } catch (e) {
    toast.error(t('tools.reloadFailed', { msg: e?.message || e }))
  }
}

// 新建 / 编辑 / 删除 / 启停 由 store 内部集中处理;view 只负责 toast 反馈
async function onSubmitForm() {
  try {
    await tools.submitForm()
    toast.success(t('tools.savedOk'))
  } catch (e) {
    toast.error(t('tools.saveFailed', { msg: e?.message || e }))
  }
}

async function onConfirmDelete() {
  try {
    await tools.confirmDelete()
    toast.success(t('tools.deletedOk'))
  } catch (e) {
    toast.error(t('tools.deleteFailed', { msg: e?.message || e }))
  }
}

async function onToggleEnabled(t_item) {
  try {
    // store.toggleEnabled 内部顺序:update -> load -> reloadRegistry
    // load 后列表中 t_item 引用会被替换,这里按"调用前的状态"取反提示
    const willEnable = !t_item.enabled
    await tools.toggleEnabled(t_item)
    toast.success(willEnable ? t('tools.enabledOk') : t('tools.disabledOk'))
  } catch (err) {
    toast.error(t('tools.toggleFailed', { msg: err?.message || err }))
  }
}

// pickFolder 辅助函数:用户取消时静默(返空串不报错)
async function pickPath(p) {
  try {
    const v = await platform.fs.pickFolder()
    if (v) p.path = v
  } catch (e) {
    toast.error(t('tools.pickFolderFailed', { msg: e?.message || e }))
  }
}

// maturity 在三个不同位置用到,集中一个 helper 返回 mdi 图标
function maturityIcon(m) {
  if (m === 'stable') return 'mdi:check-decagram-outline'
  if (m === 'experimental') return 'mdi:flask-outline'
  if (m === 'deprecated') return 'mdi:archive-arrow-down-outline'
  return 'mdi:help-circle-outline'
}

// 取 store 计算结果(用 computed 是为了响应式跟随 state 变化)
const items = computed(() => tools.filteredItems)
const total = computed(() => tools.totalCount)
const systemCount = computed(() => tools.systemCount)
const userCount = computed(() => tools.userCount)
const loading = computed(() => tools.loading)
const error = computed(() => tools.error)

const ALLOWED_MATURITY = ['stable', 'experimental', 'deprecated']

onMounted(async () => {
  try {
    await tools.load()
  } catch (e) {
    // 错误已经在 store.error 里;view 只显示
  }
})
</script>

<template>
  <div class="tools-view">
    <!-- 1. 页面头 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-emerald">
          <Icon icon="mdi:tools" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('tools.title') }}</h1>
          <p>{{ t('tools.subtitle', { total, system: systemCount, user: userCount }) }}</p>
        </div>
      </div>
    </header>

    <!-- 2. 工具栏 -->
    <div class="toolbar">
      <div class="search-box">
        <Icon icon="mdi:magnify" width="16" height="16" class="search-icon" />
        <input
          v-model="keywordDraft"
          type="text"
          :placeholder="t('tools.searchPlaceholder')"
          class="search-input"
          @keyup.enter="applyKeyword"
        />
      </div>

      <!-- 三选一过滤 -->
      <div class="filter-group">
        <button
          :class="['filter-btn', { active: tools.filter.source === 'all' }]"
          @click="selectSource('all')"
        >
          <Icon icon="mdi:view-list" width="13" height="13" />
          {{ t('tools.filterAll') }}
          <span class="filter-count">{{ total }}</span>
        </button>
        <button
          :class="['filter-btn', { active: tools.filter.source === 'system' }]"
          @click="selectSource('system')"
        >
          <Icon icon="mdi:shield-check-outline" width="13" height="13" />
          {{ t('tools.filterSystem') }}
          <span class="filter-count">{{ systemCount }}</span>
        </button>
        <button
          :class="['filter-btn', { active: tools.filter.source === 'user' }]"
          @click="selectSource('user')"
        >
          <Icon icon="mdi:account-outline" width="13" height="13" />
          {{ t('tools.filterUser') }}
          <span class="filter-count">{{ userCount }}</span>
        </button>
      </div>

      <div class="toolbar-right">
        <button class="ghost" :disabled="tools.reloading" @click="onReload">
          <Icon icon="mdi:refresh" width="14" height="14" />
          {{ t('tools.btnReload') }}
        </button>
        <button class="primary" @click="tools.openCreate()">
          <Icon icon="mdi:plus" width="14" height="14" />
          {{ t('tools.btnNew') }}
        </button>
      </div>
    </div>

    <!-- 3. 错误提示 -->
    <p v-if="error" class="error-message">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>

    <!-- 4. 卡片网格 -->
    <div v-if="loading && !items.length" class="loading-state">
      <span class="spinner"></span>
      <p>{{ t('tools.loading') }}</p>
    </div>

    <div v-else-if="items.length" class="tools-grid">
      <article
        v-for="t_item in items"
        :key="t_item.tool_id"
        :class="['tool-card', {
          'is-system': t_item.is_system,
          'is-disabled': !t_item.enabled,
        }]"
      >
        <!-- 顶部 -->
        <header class="tool-card-top">
          <div class="tool-card-icon">
            <Icon
              :icon="t_item.mdi_icon || 'mdi:cog-outline'"
              width="22"
              height="22"
            />
          </div>
          <div class="tool-card-titles">
            <h3 class="tool-card-name" :title="t_item.display_name">
              {{ t_item.display_name }}
            </h3>
            <code class="tool-card-id">{{ t_item.tool_id }}</code>
          </div>
          <span v-if="t_item.is_system" class="badge badge-system">
            <Icon icon="mdi:shield-check-outline" width="10" height="10" />
            {{ t('tools.systemBadge') }}
          </span>
        </header>

        <!-- 中部 -->
        <div class="tool-card-meta">
          <span :class="['maturity-chip', `maturity-${t_item.maturity || 'stable'}`]">
            <Icon :icon="maturityIcon(t_item.maturity)" width="10" height="10" />
            {{ t(`tools.maturity.${t_item.maturity || 'stable'}`) }}
          </span>
          <span class="meta-item">
            <Icon icon="mdi:folder-multiple-outline" width="11" height="11" />
            {{ t('tools.pathCount', { n: (t_item.paths || []).length }) }}
          </span>
        </div>

        <p
          v-if="t_item.note"
          class="tool-card-note"
          :title="t_item.note"
        >
          {{ t_item.note }}
        </p>

        <!-- 底部 -->
        <footer class="tool-card-bottom">
          <label class="switch" @click.stop>
            <input
              type="checkbox"
              :checked="t_item.enabled"
              :disabled="tools.saving"
              @change="onToggleEnabled(t_item)"
            />
            <span class="switch-slider"></span>
          </label>
          <span class="tool-card-time">
            {{ formatRelative(t_item.updated_at) }}
          </span>
          <div class="tool-card-actions" @click.stop>
            <Icon
              icon="mdi:pencil-outline"
              class="action-icon action-icon-edit"
              :title="t('tools.btnEdit')"
              width="14"
              height="14"
              @click="tools.openEdit(t_item)"
            />
            <Icon
              v-if="!t_item.is_system"
              icon="mdi:delete-outline"
              class="action-icon action-icon-danger"
              :title="t('common.delete')"
              width="14"
              height="14"
              @click="tools.askDelete(t_item)"
            />
            <Icon
              v-else
              icon="mdi:lock-outline"
              class="action-icon action-icon-locked"
              :title="t('tools.systemLocked')"
              width="14"
              height="14"
            />
          </div>
        </footer>
      </article>
    </div>

    <div v-else class="empty-state">
      <Icon icon="mdi:tools" width="48" height="48" />
      <p class="empty-title">{{ t('tools.empty') }}</p>
      <p class="empty-hint">{{ t('tools.emptyHint') }}</p>
    </div>

    <!-- 5. 新建 / 编辑 Modal -->
    <Modal
      v-model="tools.formOpen"
      size="lg"
      :title="tools.formMode === 'create'
        ? t('tools.formNewTitle')
        : t('tools.formEditTitle', { name: tools.form.display_name || tools.editingToolId })"
      :close-on-mask="!tools.saving"
    >
      <template #title-icon>
        <Icon
          :icon="tools.formMode === 'create' ? 'mdi:plus-box-outline' : 'mdi:pencil-outline'"
          width="18"
          height="18"
        />
      </template>
      <form class="form" @submit.prevent="onSubmitForm">
        <p class="form-hint">
          <Icon icon="mdi:information-outline" width="14" height="14" />
          {{ t('tools.formHint') }}
        </p>

        <div class="form-grid">
          <!-- tool_id:新建可填,编辑锁死 -->
          <div class="form-field">
            <label>
              {{ t('tools.field.toolId') }}
              <span v-if="tools.formMode === 'create'" class="required">*</span>
            </label>
            <input
              v-model="tools.form.tool_id"
              :disabled="tools.formMode === 'edit'"
              :placeholder="t('tools.hint.toolId')"
              :readonly="tools.formMode === 'edit'"
            />
            <p v-if="tools.formMode === 'edit'" class="field-hint">
              {{ t('tools.hint.toolIdLocked') }}
            </p>
          </div>

          <div class="form-field">
            <label>
              {{ t('tools.field.displayName') }}
              <span class="required">*</span>
            </label>
            <input
              v-model="tools.form.display_name"
              :placeholder="t('tools.hint.displayName')"
              :disabled="tools.saving"
            />
          </div>

          <div class="form-field">
            <label>
              {{ t('tools.field.mdiIcon') }}
              <span class="required">*</span>
            </label>
            <input
              v-model="tools.form.mdi_icon"
              placeholder="mdi:tools"
              :disabled="tools.saving"
            />
            <p class="field-hint">{{ t('tools.hint.mdiIcon') }}</p>
          </div>

          <div class="form-field">
            <label>{{ t('tools.field.maturity') }}</label>
            <select v-model="tools.form.maturity" :disabled="tools.saving">
              <option v-for="m in ALLOWED_MATURITY" :key="m" :value="m">
                {{ t(`tools.maturity.${m}`) }}
              </option>
            </select>
          </div>

          <div class="form-field">
            <label>{{ t('tools.field.sortOrder') }}</label>
            <input
              v-model.number="tools.form.sort_order"
              type="number"
              :disabled="tools.saving"
            />
          </div>

          <div class="form-field form-field-switch">
            <label>{{ t('tools.field.enabled') }}</label>
            <label class="switch">
              <input
                type="checkbox"
                v-model="tools.form.enabled"
                :disabled="tools.saving"
              />
              <span class="switch-slider"></span>
            </label>
          </div>

          <div class="form-field form-field-full">
            <label>{{ t('tools.field.note') }}</label>
            <input
              v-model="tools.form.note"
              :placeholder="t('tools.hint.note')"
              :disabled="tools.saving"
            />
          </div>
        </div>

        <!-- paths 子表 -->
        <div class="paths-section">
          <div class="paths-section-header">
            <h4>{{ t('tools.paths.title') }}</h4>
            <button
              type="button"
              class="ghost small"
              :disabled="tools.saving"
              @click="tools.addPathRow()"
            >
              <Icon icon="mdi:plus" width="13" height="13" />
              {{ t('tools.paths.add') }}
            </button>
          </div>

          <table v-if="tools.form.paths.length" class="paths-table">
            <thead>
              <tr>
                <th style="width: 110px">{{ t('tools.paths.scope') }}</th>
                <th style="width: 110px">{{ t('tools.paths.category') }}</th>
                <th>{{ t('tools.paths.path') }}</th>
                <th style="width: 80px">{{ t('tools.paths.order') }}</th>
                <th style="width: 40px"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(p, i) in tools.form.paths" :key="i">
                <td>
                  <select v-model="p.scope" :disabled="tools.saving">
                    <option value="global">global</option>
                    <option value="project">project</option>
                  </select>
                </td>
                <td>
                  <select v-model="p.category" :disabled="tools.saving">
                    <option value="user">user</option>
                    <option value="system">system</option>
                  </select>
                </td>
                <td>
                  <div class="input-with-action">
                    <input
                      v-model="p.path"
                      :placeholder="t('tools.paths.pathHint')"
                      :disabled="tools.saving"
                    />
                    <button
                      type="button"
                      class="ghost icon-btn"
                      :disabled="tools.saving"
                      :title="t('tools.paths.pickFolder')"
                      @click="pickPath(p)"
                    >
                      <Icon icon="mdi:folder-search-outline" width="14" height="14" />
                    </button>
                  </div>
                </td>
                <td>
                  <input
                    v-model.number="p.path_order"
                    type="number"
                    :disabled="tools.saving"
                  />
                </td>
                <td class="paths-action-cell">
                  <Icon
                    icon="mdi:close"
                    class="action-icon action-icon-danger"
                    :title="t('common.delete')"
                    width="14"
                    height="14"
                    @click="tools.removePathRow(i)"
                  />
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="paths-empty">{{ t('tools.paths.empty') }}</p>

          <p class="field-hint">{{ t('tools.paths.hint') }}</p>
        </div>
      </form>

      <template #footer>
        <button
          type="button"
          class="ghost"
          :disabled="tools.saving"
          @click="tools.closeForm()"
        >
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="primary"
          :disabled="tools.saving"
          @click="onSubmitForm"
        >
          <span v-if="tools.saving" class="spinner spinner-sm"></span>
          <Icon v-else icon="mdi:check" width="14" height="14" />
          {{ tools.saving ? t('common.processing') : t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 6. 删除确认 Modal -->
    <Modal
      v-model="tools.confirmOpen"
      size="sm"
      :title="t('tools.confirmDeleteTitle')"
      :close-on-mask="!tools.removing"
    >
      <p class="confirm-message">
        {{
          t('tools.confirmDeleteMsg', {
            name: tools.confirmTarget?.display_name || tools.confirmTarget?.tool_id,
          })
        }}
        <span v-if="tools.confirmTarget?.note" class="confirm-hint">
          {{ tools.confirmTarget.note }}
        </span>
      </p>
      <template #footer>
        <button
          type="button"
          class="ghost"
          :disabled="tools.removing"
          @click="tools.cancelDelete()"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="danger"
          :disabled="tools.removing"
          @click="onConfirmDelete"
        >
          <span v-if="tools.removing" class="spinner spinner-sm"></span>
          <Icon v-else icon="mdi:delete-outline" width="14" height="14" />
          {{ tools.removing ? t('common.processing') : t('common.delete') }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.tools-view {
  /* 占满内容区宽度(与 MarketView 一致) */
  width: 100%;
  color: var(--text);
  transition: color 0.3s ease;
}

/* ===== 工具主题:Emerald Workshop(独立作用域变量) ===== */
.tools-view {
  /* 主色:teal-500(冷一点的青色,和翠绿 emerald 拉开层次) */
  --tool-primary: #14b8a6;
  --tool-primary-hover: #0d9488;
  /* 强调:emerald-500(绿,代表"工具可用/成功") */
  --tool-accent: #10b981;
  /* 派生浅底/边/字 */
  --tool-bg: #f0fdfa;          /* teal-50 */
  --tool-bg-strong: #ccfbf1;   /* teal-100 */
  --tool-border: #99f6e4;      /* teal-200 */
  --tool-text: #0f766e;        /* teal-700 */
}
:global(html.dark) .tools-view {
  --tool-primary: #2dd4bf;     /* teal-400 提亮 */
  --tool-primary-hover: #5eead4; /* teal-300 */
  --tool-accent: #34d399;      /* emerald-400 */
  --tool-bg: #042f2e;          /* teal-950 */
  --tool-bg-strong: #134e4a;  /* teal-900 */
  --tool-border: #115e59;      /* teal-800 */
  --tool-text: #99f6e4;        /* teal-200 */
}

/* ===== 页面头 ===== */
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

.view-icon-emerald {
  /* 主题色块:teal→emerald 渐变 + 发光阴影 */
  background: linear-gradient(135deg, var(--tool-primary) 0%, var(--tool-accent) 100%);
  color: #ffffff;
  box-shadow: 0 2px 8px -2px color-mix(in srgb, var(--tool-primary) 40%, transparent);
}

.view-title h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 4px;
}

.view-title p {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0;
}

/* ===== 工具栏 ===== */
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 240px;
  min-width: 200px;
  max-width: 360px;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-faint);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding-left: 38px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  /* 主题浅底 + 主题淡边(柔和不刺眼) */
  background: var(--tool-bg);
  border: 1px solid var(--tool-border);
  border-radius: var(--radius-sm);
}

.filter-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  background: transparent;
  border: none;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-btn:hover:not(.active) {
  /* hover 时上主题色,文字也跟随 */
  color: var(--tool-text);
  background: var(--tool-bg-strong);
}

.filter-btn.active {
  /* 激活态:teal→emerald 渐变(类似 Market 的 source-tab) */
  background: linear-gradient(135deg, var(--tool-primary) 0%, var(--tool-accent) 100%);
  color: #ffffff;
  box-shadow: 0 2px 6px -2px color-mix(in srgb, var(--tool-primary) 50%, transparent);
}

.filter-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 6px;
  font-size: 10px;
  font-weight: 600;
  background: var(--border);
  color: var(--text-dim);
  border-radius: 999px;
  min-width: 18px;
}

.filter-btn.active .filter-count {
  /* 激活态时:count 数字用主色突出 */
  background: color-mix(in srgb, #ffffff 25%, transparent);
  color: #ffffff;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

/* 新建按钮(CTA):主题 teal→emerald 渐变 + hover 上浮(视觉关键转化点) */
.toolbar-right button.primary {
  background: linear-gradient(135deg, var(--tool-primary) 0%, var(--tool-accent) 100%);
  color: #ffffff;
  border-color: transparent;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--tool-primary) 30%, transparent);
}
.toolbar-right button.primary:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--tool-primary-hover) 0%, var(--tool-accent) 100%);
  border-color: transparent;
  transform: translateY(-1px);
  box-shadow: 0 3px 8px -2px color-mix(in srgb, var(--tool-primary) 45%, transparent);
}

/* ===== 错误提示 ===== */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--danger-dim);
  color: var(--danger);
  border: 1px solid var(--danger);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin: 0 0 16px;
}

/* ===== 加载状态 ===== */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 64px 24px;
  color: var(--text-faint);
}

.loading-state .spinner {
  width: 24px;
  height: 24px;
  border-width: 3px;
}

/* ===== 卡片网格 ===== */
.tools-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}

.tool-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 14px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-left: 4px solid transparent;
  border-radius: var(--radius);
  position: relative;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

.tool-card:hover {
  /* hover 时显示主题色淡边 + 轻微抬起 */
  border-color: color-mix(in srgb, var(--tool-primary) 35%, var(--border));
  box-shadow: var(--shadow-card);
}

/* 系统工具:左侧 emerald 渐变条 */
.tool-card.is-system {
  border-left: 4px solid;
  border-image: linear-gradient(180deg, var(--tool-primary) 0%, var(--tool-accent) 100%) 1;
}

/* 用户工具:左侧浅主题色条(轻提示,与系统工具区分但不刺眼) */
.tool-card:not(.is-system) {
  border-left-color: color-mix(in srgb, var(--tool-primary) 25%, transparent);
}

.tool-card.is-disabled {
  opacity: 0.55;
}

.tool-card-top {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tool-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  /* 主题浅底 + 主题主色图标 */
  background: var(--tool-bg);
  color: var(--tool-primary);
  flex-shrink: 0;
  border: 1px solid var(--tool-border);
}

.tool-card-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tool-card-name {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-card-id {
  font-size: 11px;
  font-family: 'JetBrains Mono', monospace;
  /* 主题色 chip */
  background: var(--tool-bg);
  color: var(--tool-text);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  align-self: flex-start;
  max-width: fit-content;
  border: 1px solid var(--tool-border);
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 6px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 999px;
  flex-shrink: 0;
}

.badge-system {
  /* 系统徽章:teal→emerald 渐变,和卡片左侧条带呼应 */
  background: linear-gradient(135deg, var(--tool-bg) 0%, var(--tool-bg-strong) 100%);
  color: var(--tool-text);
  border: 1px solid var(--tool-border);
}

.tool-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.maturity-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 999px;
}

.maturity-stable {
  background: var(--success-dim);
  color: var(--success);
}

.maturity-experimental {
  background: var(--warning-dim);
  color: var(--warning);
}

.maturity-deprecated {
  background: var(--danger-dim);
  color: var(--danger);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--text-faint);
}

.tool-card-note {
  margin: 0;
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.tool-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.tool-card-time {
  font-size: 10px;
  color: var(--text-faint);
  flex: 1;
  text-align: left;
}

.tool-card-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.tool-card:hover .tool-card-actions {
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
  transition: background 0.15s ease, color 0.15s ease;
  color: var(--text-dim);
}

.action-icon:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.action-icon-edit:hover {
  /* 编辑:hover 时用主题色(区别于危险) */
  background: var(--tool-bg);
  color: var(--tool-primary);
}

.action-icon-danger:hover {
  background: var(--danger-dim);
  color: var(--danger);
}

.action-icon-locked {
  cursor: not-allowed;
  color: var(--text-faint);
}

/* ===== 开关 ===== */
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.switch-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--border);
  transition: 0.2s;
  border-radius: 22px;
}

.switch-slider::before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: var(--bg-card);
  transition: 0.2s;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.switch input:checked + .switch-slider {
  /* 开启时:teal→emerald 渐变(与卡片左侧条带呼应) */
  background: linear-gradient(135deg, var(--tool-primary) 0%, var(--tool-accent) 100%);
}

.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}

.switch input:disabled + .switch-slider {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ===== 表单(Modal 内) ===== */
.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--text-dim);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field-full {
  grid-column: 1 / -1;
}

.form-field label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
}

/* 锁 select 行高与 input 一致:macOS Chrome native select 会比 input 多 2-3px,
   同时 min-height + height 双锁避免被全局 input 规则的 padding 撑高 */
.form-field select,
.form-field input {
  height: 36px;
  min-height: 36px;
  line-height: 1.4;
}

.form-field textarea {
  min-height: 60px;
}

.form-field-switch {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.required {
  color: var(--danger);
  font-weight: 700;
}

.field-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

.input-with-action {
  display: flex;
  align-items: stretch;
  gap: 6px;
}

.input-with-action input {
  flex: 1;
  min-width: 0;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  flex-shrink: 0;
}

/* ===== paths 子表 ===== */
.paths-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.paths-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 4px;
}

.paths-section-header h4 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

button.small {
  padding: 5px 10px;
  font-size: 12px;
}

.paths-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.paths-table th {
  text-align: left;
  padding: 8px 10px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  background: var(--bg-subtle);
  border-bottom: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.paths-table td {
  padding: 6px 8px;
  border-top: 1px solid var(--border);
  vertical-align: middle;
}

.paths-table tr:first-child td {
  border-top: none;
}

.paths-table input,
.paths-table select {
  padding: 5px 8px;
  font-size: 12px;
  width: 100%;
}

.paths-action-cell {
  text-align: center;
}

.paths-empty {
  margin: 0;
  padding: 16px;
  text-align: center;
  font-size: 12px;
  color: var(--text-faint);
  background: var(--bg-subtle);
  border: 1px dashed var(--border);
  border-radius: var(--radius-sm);
}

/* ===== 删除确认 ===== */
.confirm-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
}

.confirm-hint {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-faint);
}

/* ===== 空状态 ===== */
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
  margin: 6px 0 0;
  color: var(--text-faint);
}

/* ===== 响应式 ===== */
@media (max-width: 768px) {
  .tools-grid {
    grid-template-columns: 1fr;
  }

  .tool-card-actions {
    opacity: 1;
  }

  .toolbar-right {
    width: 100%;
  }
}
</style>
