<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import {
  listSources,
  listMarketSkills,
  refreshSource,
  installMarketSkill,
} from '@/api/skillbox/market.js'

// 状态
const loading = ref(false)
const error = ref('')

// 源
const sources = ref([])
const activeSourceId = ref(0)
const refreshing = ref(false)
const lastRefresh = ref(null) // RefreshResult

// 列表
const keyword = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

// 装到 store
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
    error.value = `源加载失败: ${e?.message || e}`
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
    error.value = `列表加载失败: ${e?.message || e}`
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
    // 刷新后回到第 1 页
    page.value = 1
    await fetchSkills()
  } catch (e) {
    error.value = `刷新失败: ${e?.message || e}`
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
  if (!confirm(`确定把 "${item.name}" 装到 ${installScope.value} 吗?`)) return
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
    installOk.value = `已装:${res?.skill?.name || item.name} (v${res?.skill?.version || '?'})`
  } catch (e) {
    installError.value = `装失败: ${e?.message || e}`
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
  <section class="market">
    <header class="head">
      <div class="title">三方市场</div>
      <div class="row">
        <span class="label">作用域:</span>
        <select v-model="installScope">
          <option value="global">全局 (global)</option>
          <option value="project" disabled>项目 (暂未启用)</option>
        </select>
        <input
          v-model="keyword"
          type="text"
          placeholder="按 name 搜索…"
          @keyup.enter="onSearch"
        />
        <button @click="onSearch">搜索</button>
        <button :disabled="refreshing || !activeSourceId" @click="onRefresh">
          {{ refreshing ? '刷新中…' : '刷新源' }}
        </button>
      </div>
    </header>

    <nav class="srcbar">
      <button
        v-for="s in sources"
        :key="s.id"
        :class="{ active: s.id === activeSourceId }"
        @click="onSelectSource(s.id)"
      >
        {{ s.name }}
        <span class="src-type">{{ s.type }}</span>
      </button>
    </nav>

    <div v-if="error" class="err">{{ error }}</div>
    <div v-if="lastRefresh" class="ok">
      上次刷新:pulled {{ lastRefresh.pulled_count }} / inserted {{ lastRefresh.inserted }} / updated {{ lastRefresh.updated }} ({{ lastRefresh.finished_at }})
    </div>
    <div v-if="installOk" class="ok">{{ installOk }}</div>
    <div v-if="installError" class="err">{{ installError }}</div>

    <table v-if="items.length > 0" class="grid">
      <thead>
        <tr>
          <th>name</th>
          <th>version</th>
          <th>author</th>
          <th>description</th>
          <th>tags</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="it in items" :key="it.remote_id">
          <td>
            <span class="name">{{ it.name }}</span>
            <span class="rid">{{ it.remote_id }}</span>
          </td>
          <td>{{ it.version || '—' }}</td>
          <td>{{ it.author || '—' }}</td>
          <td class="desc">{{ it.description || '—' }}</td>
          <td>
            <span v-for="t in (it.tags || '').split(',').filter(Boolean)" :key="t" class="tag">
              {{ t }}
            </span>
          </td>
          <td>
            <button :disabled="installing" @click="onInstall(it)">
              {{ installing ? '装中…' : '安装' }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else-if="!loading" class="empty">
      当前源还没拉过。点 "刷新源" 把三方目录拉到本地。
    </div>

    <footer v-if="totalPages > 1" class="pager">
      <button :disabled="page <= 1" @click="page--; fetchSkills()">上一页</button>
      <span>第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 条</span>
      <button :disabled="page >= totalPages" @click="page++; fetchSkills()">下一页</button>
    </footer>
  </section>
</template>

<style scoped>
.market { padding: 20px; max-width: 1100px; margin: 0 auto; }
.head { display: flex; flex-direction: column; gap: 10px; margin-bottom: 14px; }
.title { font-size: 18px; font-weight: 600; }
.row { display: flex; gap: 8px; align-items: center; }
.label { color: #6b7280; }
.row input[type="text"] { flex: 1; max-width: 320px; padding: 6px 8px; border: 1px solid #d1d5db; border-radius: 4px; }
.row select { padding: 6px 8px; border: 1px solid #d1d5db; border-radius: 4px; }
.row button, .pager button, table button {
  padding: 6px 12px; border: 1px solid #d1d5db; background: #ffffff;
  border-radius: 4px; cursor: pointer; font-size: 13px;
}
.row button:disabled, .pager button:disabled, table button:disabled { opacity: 0.5; cursor: not-allowed; }
.srcbar { display: flex; gap: 6px; margin-bottom: 12px; border-bottom: 1px solid #e5e7eb; }
.srcbar button {
  border: none; background: transparent; padding: 8px 12px;
  border-bottom: 2px solid transparent; cursor: pointer; color: #6b7280;
}
.srcbar button.active { color: #2563eb; border-bottom-color: #2563eb; font-weight: 600; }
.src-type { font-size: 11px; color: #9ca3af; margin-left: 4px; }
.err { background: #fef2f2; color: #b91c1c; border: 1px solid #fecaca; padding: 8px 12px; border-radius: 4px; margin-bottom: 10px; }
.ok { background: #f0fdf4; color: #166534; border: 1px solid #bbf7d0; padding: 8px 12px; border-radius: 4px; margin-bottom: 10px; }
.grid { width: 100%; border-collapse: collapse; margin-top: 10px; }
.grid th, .grid td { padding: 8px 10px; border-bottom: 1px solid #f3f4f6; text-align: left; vertical-align: top; font-size: 13px; }
.grid th { background: #f9fafb; font-weight: 600; color: #374151; }
.name { font-weight: 500; color: #111827; display: block; }
.rid { font-size: 11px; color: #9ca3af; font-family: ui-monospace, monospace; }
.desc { color: #4b5563; max-width: 360px; }
.tag { display: inline-block; background: #eef2ff; color: #4338ca; border-radius: 3px; padding: 1px 6px; font-size: 11px; margin-right: 4px; }
.empty { padding: 30px; text-align: center; color: #9ca3af; }
.pager { margin-top: 12px; display: flex; gap: 12px; align-items: center; justify-content: flex-end; color: #6b7280; font-size: 13px; }
</style>
