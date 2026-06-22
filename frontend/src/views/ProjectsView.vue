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
    error.value = 'name / alias / root_path 都不能为空'
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
  if (!confirm(`确定删除项目 #${id} ?`)) return
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
        Projects
      </h2>
      <p class="muted">登记项目根目录,后续 skill 可绑定到 project scope 走项目级覆盖。</p>
    </header>

    <div class="toolbar">
      <div class="search">
        <input
          v-model="filter.keyword"
          placeholder="按 name 过滤"
          @keyup.enter="() => { filter.page = 1; reload() }"
        />
        <button @click="() => { filter.page = 1; reload() }">搜索</button>
      </div>
      <button class="primary" @click="showForm = !showForm">
        {{ showForm ? '取消' : '+ 新建项目' }}
      </button>
    </div>

    <p v-if="error" class="error inline-flex items-center gap-1.5">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />{{ error }}
    </p>

    <form v-if="showForm" class="card form" @submit.prevent="submit">
      <h3>新建项目</h3>
      <div class="form-grid">
        <label>
          <span>Name</span>
          <input v-model="form.name" placeholder="显示名,如 My App" />
        </label>
        <label>
          <span>Alias</span>
          <input v-model="form.alias" placeholder="唯一别名,英文短码" />
        </label>
        <label class="full">
          <span>Root Path</span>
          <input v-model="form.root_path" placeholder="项目根绝对路径" />
        </label>
        <label class="full">
          <span>Description</span>
          <input v-model="form.description" placeholder="可选,描述项目用途" />
        </label>
      </div>
      <div class="form-actions">
        <button type="submit" class="primary">创建</button>
      </div>
    </form>

    <div class="card">
      <h3>项目列表
        <span class="card-sub">— 共 {{ total }} 条</span>
        <span v-if="loading" class="spinner" style="margin-left: auto"></span>
      </h3>

      <table v-if="items.length" class="grid">
        <thead>
          <tr>
            <th style="width: 60px">ID</th>
            <th>Name</th>
            <th>Alias</th>
            <th>Root Path</th>
            <th>Description</th>
            <th style="width: 90px">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in items" :key="p.ID">
            <td>{{ p.ID }}</td>
            <td><b>{{ p.Name }}</b></td>
            <td><code>{{ p.Alias }}</code></td>
            <td class="path">{{ p.RootPath }}</td>
            <td class="desc-cell">{{ p.Description || '—' }}</td>
            <td>
              <button class="link danger" @click="remove(p.ID)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!loading" class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:folder-open-outline" width="36" height="36" />
        </span>
        还没有登记项目。点右上角"+ 新建项目"开始
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="filter.page <= 1" @click="gotoPage(filter.page - 1)">上一页</button>
        <span>{{ filter.page }} / {{ totalPages }} (共 {{ total }} 条)</span>
        <button :disabled="filter.page >= totalPages" @click="gotoPage(filter.page + 1)">下一页</button>
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
