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
    <header class="head">
      <h2 class="flex items-center gap-2">
        <Icon icon="mdi:folder-multiple-outline" width="20" height="20" class="text-sb-primary" />
        {{ t('projects.title') }}
      </h2>
      <p class="muted">{{ t('projects.subtitle') }}</p>
    </header>

    <div class="toolbar">
      <div class="search">
        <input
          v-model="filter.keyword"
          :placeholder="t('projects.searchPlaceholder')"
          @keyup.enter="() => { filter.page = 1; reload() }"
        />
        <button @click="() => { filter.page = 1; reload() }">{{ t('common.search') }}</button>
      </div>
      <button class="primary" @click="showForm = !showForm">
        {{ showForm ? t('projects.btnCancel') : t('projects.btnNew') }}
      </button>
    </div>

    <p v-if="error" class="error inline-flex items-center gap-1.5">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />{{ error }}
    </p>

    <form v-if="showForm" class="card form" @submit.prevent="submit">
      <h3>{{ t('projects.formTitle') }}</h3>
      <div class="form-grid">
        <label>
          <span>{{ t('projects.name') }}</span>
          <input v-model="form.name" :placeholder="t('projects.nameHint')" />
        </label>
        <label>
          <span>{{ t('projects.alias') }}</span>
          <input v-model="form.alias" :placeholder="t('projects.aliasHint')" />
        </label>
        <label class="full">
          <span>{{ t('projects.rootPath') }}</span>
          <input v-model="form.root_path" :placeholder="t('projects.rootPathHint')" />
        </label>
        <label class="full">
          <span>{{ t('projects.description') }}</span>
          <input v-model="form.description" :placeholder="t('projects.descriptionHint')" />
        </label>
      </div>
      <div class="form-actions">
        <button type="submit" class="primary">{{ t('common.create') }}</button>
      </div>
    </form>

    <div class="card">
      <h3>{{ t('projects.listTitle') }}
        <span class="card-sub">— {{ t('common.totalCount', { count: total }) }}</span>
        <span v-if="loading" class="spinner" style="margin-left: auto"></span>
      </h3>

      <table v-if="items.length" class="grid">
        <thead>
          <tr>
            <th style="width: 60px">{{ t('projects.colId') }}</th>
            <th>{{ t('projects.colName') }}</th>
            <th>{{ t('projects.colAlias') }}</th>
            <th>{{ t('projects.colRootPath') }}</th>
            <th>{{ t('projects.colDescription') }}</th>
            <th style="width: 90px">{{ t('projects.colActions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in items" :key="p.ID">
            <td>{{ p.ID }}</td>
            <td><b>{{ p.Name }}</b></td>
            <td><code>{{ p.Alias }}</code></td>
            <td class="path">{{ p.RootPath }}</td>
            <td class="desc-cell">{{ p.Description || t('common.dash') }}</td>
            <td>
              <button class="link danger" @click="remove(p.ID)">{{ t('common.delete') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!loading" class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:folder-open-outline" width="36" height="36" />
        </span>
        {{ t('projects.empty') }}
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="filter.page <= 1" @click="gotoPage(filter.page - 1)">{{ t('common.prev') }}</button>
        <span>{{ filter.page }} / {{ totalPages }} ({{ t('common.totalCount', { count: total }) }})</span>
        <button :disabled="filter.page >= totalPages" @click="gotoPage(filter.page + 1)">{{ t('common.next') }}</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.projects-view { max-width: 1100px; margin: 0 auto; color: var(--text); }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; gap: 12px; }
.search { display: flex; gap: 6px; }
.search input { width: 220px; }

.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 14px; }
.form-grid label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-dim); }
.form-grid label.full { grid-column: 1 / -1; }
.form-grid input { width: 100%; }
.form-actions { display: flex; justify-content: flex-end; margin-top: 12px; }

.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { text-align: left; padding: 8px 10px; border-bottom: 1px solid #f3f4f6; }
.grid th { background: #f9fafb; color: var(--text-dim); font-weight: 600; }
.grid .path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text-dim); font-size: 12px; }
.desc-cell { color: var(--text-dim); }

.pager { display: flex; align-items: center; gap: 12px; margin-top: 12px; font-size: 13px; color: var(--text-dim); justify-content: flex-end; }
</style>
