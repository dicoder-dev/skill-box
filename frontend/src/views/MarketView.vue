<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
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
  <div class="market">
    <header class="head">
      <h2 class="flex items-center gap-2">
        <Icon icon="mdi:cart-outline" width="20" height="20" class="text-sb-primary" />
        三方市场
      </h2>
      <p class="muted">从 skillhub.cn / skills.sh 等三方源拉取 skill,直接装到 Skill Box 本地 store。</p>
    </header>

    <div class="card">
      <div class="toolbar">
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
        </div>
        <button class="primary flex items-center gap-1.5" :disabled="refreshing || !activeSourceId" @click="onRefresh">
          <span v-if="refreshing" class="spinner"></span>
          <Icon v-else icon="mdi:refresh" width="14" height="14" />
          {{ refreshing ? '刷新中…' : '刷新源' }}
        </button>
      </div>

      <nav class="srcbar">
        <button
          v-for="s in sources"
          :key="s.id"
          :class="{ active: s.id === activeSourceId }"
          @click="onSelectSource(s.id)"
        >
          <span class="src-icon">
            <Icon icon="mdi:radio-tower" width="14" height="14" />
          </span>
          {{ s.name }}
          <span class="src-type">{{ s.type }}</span>
        </button>
        <span v-if="!sources.length && !loading" class="src-empty">没有可用的源</span>
      </nav>

      <div v-if="error" class="err">⚠️ {{ error }}</div>
      <div v-if="lastRefresh" class="ok">
        ✅ 上次刷新:pulled {{ lastRefresh.pulled_count }} · 新增 {{ lastRefresh.inserted }} · 更新 {{ lastRefresh.updated }}
        <span class="muted">({{ lastRefresh.finished_at }})</span>
      </div>
      <div v-if="installOk" class="ok">✅ {{ installOk }}</div>
      <div v-if="installError" class="err">⚠️ {{ installError }}</div>

      <table v-if="items.length > 0" class="grid">
        <thead>
          <tr>
            <th>name</th>
            <th>version</th>
            <th>author</th>
            <th>description</th>
            <th>tags</th>
            <th style="width: 90px"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="it in items" :key="it.remote_id">
            <td>
              <span class="name">{{ it.name }}</span>
              <span class="rid">{{ it.remote_id }}</span>
            </td>
            <td><code>{{ it.version || '—' }}</code></td>
            <td>{{ it.author || '—' }}</td>
            <td class="desc">{{ it.description || '—' }}</td>
            <td>
              <span v-for="t in (it.tags || '').split(',').filter(Boolean)" :key="t" class="tag">
                {{ t }}
              </span>
            </td>
            <td>
              <button class="link primary-link" :disabled="installing" @click="onInstall(it)">
                {{ installing ? '装中…' : '安装' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!loading" class="empty-state">
        <span class="empty-icon">📡</span>
        当前源还没拉过。点 "↻ 刷新源" 把三方目录拉到本地。
      </div>
      <div v-else class="empty-state">
        <span class="spinner"></span>
        <p style="margin: 8px 0 0">加载中…</p>
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="page <= 1" @click="page--; fetchSkills()">上一页</button>
        <span>第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 条</span>
        <button :disabled="page >= totalPages" @click="page++; fetchSkills()">下一页</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.market { max-width: 1100px; margin: 0 auto; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }

.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.row { display: flex; gap: 8px; align-items: center; }
.label { color: var(--text-dim); font-size: 13px; }
.row input[type="text"] { width: 240px; }

.srcbar { display: flex; gap: 4px; margin-bottom: 12px; border-bottom: 1px solid var(--border); }
.srcbar button {
  border: none; background: transparent; padding: 8px 14px;
  border-bottom: 2px solid transparent; cursor: pointer; color: var(--text-dim);
  display: inline-flex; align-items: center; gap: 6px; font-size: 14px;
}
.srcbar button:hover:not(.active) { color: var(--text); }
.srcbar button.active { color: var(--primary); border-bottom-color: var(--primary); font-weight: 600; }
.src-icon { font-size: 13px; }
.src-type { font-size: 11px; color: var(--text-faint); }
.srcbar button.active .src-type { color: var(--primary); }
.src-empty { padding: 8px 12px; color: var(--text-faint); font-size: 13px; }

.err { background: var(--danger-dim); color: var(--danger); border: 1px solid #fecaca; padding: 8px 12px; border-radius: var(--radius-sm); margin-bottom: 10px; font-size: 13px; }
.ok { background: var(--success-dim); color: var(--success); border: 1px solid #bbf7d0; padding: 8px 12px; border-radius: var(--radius-sm); margin-bottom: 10px; font-size: 13px; }

.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { padding: 8px 10px; text-align: left; vertical-align: top; border-bottom: 1px solid #f3f4f6; }
.grid th { background: #f9fafb; color: var(--text-dim); font-weight: 600; }
.name { font-weight: 500; color: var(--text); display: block; }
.rid { font-size: 11px; color: var(--text-faint); font-family: ui-monospace, monospace; }
.desc { color: var(--text-dim); max-width: 360px; }
.tag { display: inline-block; background: #eef2ff; color: #4338ca; border-radius: 3px; padding: 1px 6px; font-size: 11px; margin-right: 4px; }
.link.primary-link { color: var(--success); font-weight: 500; }
.link.primary-link:hover:not(:disabled) { background: var(--success-dim); }

.pager { margin-top: 12px; display: flex; gap: 12px; align-items: center; justify-content: flex-end; color: var(--text-dim); font-size: 13px; }
</style>
