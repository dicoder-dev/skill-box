<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { listProjects, createProject, deleteProject } from '@/api/skillbox/projects'

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
  <section class="projects-view">
    <header class="bar">
      <h2>Projects</h2>
      <div class="search">
        <input
          v-model="filter.keyword"
          placeholder="按 name 过滤"
          @keyup.enter="() => { filter.page = 1; reload() }"
        />
        <button @click="() => { filter.page = 1; reload() }">搜索</button>
        <button class="primary" @click="showForm = !showForm">
          {{ showForm ? '取消' : '新建项目' }}
        </button>
      </div>
    </header>

    <form v-if="showForm" class="form" @submit.prevent="submit">
      <label>
        <span>Name</span>
        <input v-model="form.name" placeholder="显示名,如 My App" />
      </label>
      <label>
        <span>Alias</span>
        <input v-model="form.alias" placeholder="唯一别名,英文短码" />
      </label>
      <label>
        <span>Root Path</span>
        <input v-model="form.root_path" placeholder="项目根绝对路径" />
      </label>
      <label class="full">
        <span>Description</span>
        <input v-model="form.description" placeholder="可选,描述项目用途" />
      </label>
      <div class="form-actions">
        <button type="submit" class="primary">创建</button>
      </div>
    </form>

    <p v-if="error" class="error">{{ error }}</p>

    <table class="grid" v-if="items.length || !loading">
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
        <tr v-if="!items.length">
          <td colspan="6" class="empty">暂无项目,点右上角"新建项目"开始</td>
        </tr>
        <tr v-for="p in items" :key="p.ID">
          <td>{{ p.ID }}</td>
          <td>{{ p.Name }}</td>
          <td><code>{{ p.Alias }}</code></td>
          <td class="path">{{ p.RootPath }}</td>
          <td>{{ p.Description }}</td>
          <td>
            <button class="link danger" @click="remove(p.ID)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <footer class="pager" v-if="totalPages > 1">
      <button :disabled="filter.page <= 1" @click="gotoPage(filter.page - 1)">上一页</button>
      <span>{{ filter.page }} / {{ totalPages }} (共 {{ total }} 条)</span>
      <button :disabled="filter.page >= totalPages" @click="gotoPage(filter.page + 1)">下一页</button>
    </footer>
  </section>
</template>

<style scoped>
.projects-view {
  padding: 16px 20px;
  max-width: 1100px;
  margin: 0 auto;
  color: #1a1a1a;
}
.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
}
.bar h2 { margin: 0; font-size: 18px; }
.search { display: flex; gap: 6px; }
.search input { width: 200px; }
input, button {
  font-size: 14px;
  padding: 5px 9px;
  border: 1px solid #d0d0d0;
  border-radius: 4px;
  background: #fff;
  color: #1a1a1a;
}
button { cursor: pointer; }
button.primary { background: #2563eb; color: #fff; border-color: #2563eb; }
button.link { border: none; background: none; padding: 2px 4px; color: #2563eb; }
button.link.danger { color: #b91c1c; }
button:disabled { opacity: 0.45; cursor: not-allowed; }
.form {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 10px 14px;
  margin-bottom: 12px;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fafafa;
}
.form label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: #4b5563; }
.form label.full { grid-column: 1 / -1; }
.form input { width: 100%; }
.form-actions { grid-column: 1 / -1; display: flex; justify-content: flex-end; }
.error { color: #b91c1c; margin: 6px 0; }
.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { text-align: left; padding: 7px 9px; border-bottom: 1px solid #eef0f3; }
.grid th { background: #f7f8fa; color: #4b5563; font-weight: 600; }
.grid .path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #374151; }
.grid .empty { text-align: center; color: #9ca3af; padding: 18px; }
.pager { display: flex; align-items: center; gap: 12px; margin-top: 12px; font-size: 13px; color: #4b5563; }
</style>
