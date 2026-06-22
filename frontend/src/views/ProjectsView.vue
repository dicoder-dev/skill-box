<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listProjects, createProject, deleteProject } from '@/api/skillbox/projects'

const { t } = useI18n()

const items = ref([])
const total = ref(0)
const loading = ref(false)
const error = ref('')
const showForm = ref(false)

const form = reactive({ name: '', alias: '', root_path: '', description: '' })
const filter = reactive({ keyword: '', page: 1, size: 10 })

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filter.size)))

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const resp = await listProjects({
      page: filter.page,
      size: filter.size,
      keyword: filter.keyword || undefined,
    })
    items.value = resp?.items || []
    total.value = resp?.total || 0
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function submit() {
  error.value = ''
  if (!form.name.trim() || !form.alias.trim() || !form.root_path.trim()) {
    error.value = t('projects.errRequired')
    return
  }
  try {
    await createProject({ ...form })
    showForm.value = false
    Object.assign(form, { name: '', alias: '', root_path: '', description: '' })
    filter.page = 1
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

async function remove(id) {
  if (!confirm(t('projects.confirmDelete', { id }))) return
  try {
    await deleteProject(id)
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

function gotoPage(p) {
  if (p < 1 || p > totalPages.value) return
  filter.page = p
  reload()
}

onMounted(reload)
</script>

<template>
  <div class="projects-view">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-purple">
          <Icon icon="mdi:folder-multiple-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('projects.title') }}</h1>
          <p>{{ t('projects.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="search-box">
        <Icon icon="mdi:magnify" width="16" height="16" class="search-icon" />
        <input
          v-model="filter.keyword"
          :placeholder="t('projects.searchPlaceholder')"
          class="search-input"
          @keyup.enter="() => { filter.page = 1; reload() }"
        />
      </div>
      <button class="primary" @click="showForm = !showForm">
        <Icon :icon="showForm ? 'mdi:close' : 'mdi:plus'" width="16" height="16" />
        {{ showForm ? t('projects.btnCancel') : t('projects.btnNew') }}
      </button>
    </div>

    <p v-if="error" class="error-message">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>

    <!-- 创建表单 -->
    <form v-if="showForm" class="card form-card" @submit.prevent="submit">
      <header class="form-header">
        <h3>
          <Icon icon="mdi:folder-plus" width="18" height="18" />
          {{ t('projects.formTitle') }}
        </h3>
      </header>

      <div class="form-grid">
        <div class="form-field">
          <label>{{ t('projects.name') }}</label>
          <input v-model="form.name" :placeholder="t('projects.nameHint')" />
        </div>
        <div class="form-field">
          <label>{{ t('projects.alias') }}</label>
          <input v-model="form.alias" :placeholder="t('projects.aliasHint')" />
        </div>
        <div class="form-field form-field-full">
          <label>{{ t('projects.rootPath') }}</label>
          <input v-model="form.root_path" :placeholder="t('projects.rootPathHint')" />
        </div>
        <div class="form-field form-field-full">
          <label>{{ t('projects.description') }}</label>
          <input v-model="form.description" :placeholder="t('projects.descriptionHint')" />
        </div>
      </div>

      <div class="form-actions">
        <button type="button" class="ghost" @click="showForm = false">
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button type="submit" class="primary">
          <Icon icon="mdi:check" width="14" height="14" />
          {{ t('common.create') }}
        </button>
      </div>
    </form>

    <!-- 列表卡片 -->
    <div class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
          {{ t('projects.listTitle') }}
          <span class="card-sub">— {{ t('common.totalCount', { count: total }) }}</span>
        </h3>
        <span v-if="loading" class="spinner"></span>
      </header>

      <div class="table-container">
        <table v-if="items.length" class="grid">
          <thead>
            <tr>
              <th style="width: 60px">{{ t('projects.colId') }}</th>
              <th>{{ t('projects.colName') }}</th>
              <th>{{ t('projects.colAlias') }}</th>
              <th>{{ t('projects.colRootPath') }}</th>
              <th>{{ t('projects.colDescription') }}</th>
              <th style="width: 100px">{{ t('projects.colActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in items" :key="p.ID">
              <td class="td-id">{{ p.ID }}</td>
              <td><strong class="project-name">{{ p.Name }}</strong></td>
              <td><code class="project-alias">{{ p.Alias }}</code></td>
              <td class="td-path">{{ p.RootPath }}</td>
              <td class="td-desc">{{ p.Description || t('common.dash') }}</td>
              <td>
                <button class="action-btn action-btn-danger" @click="remove(p.ID)">
                  <Icon icon="mdi:delete" width="12" height="12" />
                  {{ t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-else-if="!loading" class="empty-state">
          <Icon icon="mdi:folder-open-outline" width="48" height="48" />
          <p class="empty-title">{{ t('projects.empty') }}</p>
        </div>
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="filter.page <= 1" @click="gotoPage(filter.page - 1)">
          <Icon icon="mdi:chevron-left" width="14" height="14" />
          {{ t('common.prev') }}
        </button>
        <span class="pager-info">{{ filter.page }} / {{ totalPages }} ({{ t('common.totalCount', { count: total }) }})</span>
        <button :disabled="filter.page >= totalPages" @click="gotoPage(filter.page + 1)">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="14" height="14" />
        </button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.projects-view {
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

.view-icon-purple {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
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

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 400px;
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
  height: 40px;
}

/* 错误消息 */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--danger-dim);
  color: var(--danger);
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 16px;
}

/* 卡片样式 */
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
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.card-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

/* 表单卡片 */
.form-card {
  animation: slideDown 0.2s ease;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.form-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.form-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
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
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
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
  text-align: left;
  padding: 12px 14px;
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

.td-id {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-faint);
}

.project-name {
  font-weight: 600;
  color: var(--text);
}

.project-alias {
  background: var(--primary-dim);
  color: var(--primary);
}

.td-path {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
  font-size: 12px;
}

.td-desc {
  color: var(--text-dim);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 操作按钮 */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--border);
  background: var(--bg-card);
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

.action-btn-danger:hover:not(:disabled) {
  background: var(--danger-dim);
  border-color: var(--danger);
  color: var(--danger);
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

/* 响应式 */
@media (max-width: 768px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    max-width: none;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .table-container {
    margin: 0 -16px;
    padding: 0 16px;
  }

  .grid th, .grid td {
    padding: 10px 8px;
  }
}
</style>
