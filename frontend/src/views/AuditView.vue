<script setup>
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { listAuditLogs, getAuditStats } from '@/api/skillbox/audit'

// 后端就绪检测
const backendReady = ref(false)
const loading = ref(false)
const error = ref('')

// 数据
const logs = ref([])
const total = ref(0)
const page = ref(1)
const size = 20
const stats = ref({ total: 0, by_action: {}, by_actor: {} })

// 过滤
const filterAction = ref('')
const filterActor = ref('')
const filterTargetType = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

const ACTION_OPTIONS = [
  '', 'create', 'update', 'delete',
  'apply', 'undo',
  'tag_create', 'tag_delete', 'tag_rollback',
  'test_run', 'market_install', 'onboarding_import',
  'project_create', 'project_delete',
]

async function loadStats() {
  try {
    const s = await getAuditStats()
    stats.value = s || { total: 0, by_action: {}, by_actor: {} }
    backendReady.value = true
  } catch (e) {
    backendReady.value = false
  }
}

async function loadLogs() {
  loading.value = true
  error.value = ''
  try {
    const res = await listAuditLogs({
      page: page.value,
      size,
      action: filterAction.value || undefined,
      actor: filterActor.value || undefined,
      target_type: filterTargetType.value || undefined,
    })
    logs.value = res?.items || []
    total.value = res?.total || 0
    backendReady.value = true
  } catch (e) {
    error.value = e?.message || String(e)
    logs.value = []
    total.value = 0
    backendReady.value = false
  } finally {
    loading.value = false
  }
}

function reload() { page.value = 1; loadLogs() }
function gotoPage(p) { if (p >= 1 && p <= totalPages.value) { page.value = p; loadLogs() } }

const actionColor = (a) => {
  if (!a) return ''
  if (a.startsWith('create') || a.startsWith('tag_create') || a === 'market_install' || a === 'onboarding_import' || a === 'project_create') return 'ok'
  if (a.startsWith('delete') || a === 'project_delete' || a === 'tag_delete') return 'err'
  if (a.startsWith('undo') || a === 'tag_rollback') return 'warn'
  return ''
}

onMounted(async () => {
  await loadStats()
  await loadLogs()
})
</script>

<template>
  <div class="audit">
    <header class="head">
      <h2>📜 审计日志</h2>
      <p class="muted">记录所有关键操作的 actor / action / target / payload。第 10 步后端就绪后,这里会自动出现真实数据。</p>
    </header>

    <!-- 概览卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-label">总记录数</div>
        <div class="stat-value">{{ stats.total || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">按 action 分类</div>
        <div class="action-chips">
          <span v-for="(c, a) in (stats.by_action || {})" :key="a" class="chip">
            <code>{{ a }}</code> × <b>{{ c }}</b>
          </span>
          <span v-if="!Object.keys(stats.by_action || {}).length" class="muted">—</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">按 actor 分类</div>
        <div class="action-chips">
          <span v-for="(c, a) in (stats.by_actor || {})" :key="a" class="chip">
            <code>{{ a }}</code> × <b>{{ c }}</b>
          </span>
          <span v-if="!Object.keys(stats.by_actor || {}).length" class="muted">—</span>
        </div>
      </div>
    </div>

    <!-- 后端未就绪占位 -->
    <div v-if="!backendReady" class="card placeholder">
      <div class="empty-state">
        <span class="empty-icon">🚧</span>
        <h3 style="margin: 8px 0 4px">第 10 步后端尚未就绪</h3>
        <p class="muted">该页面会在 <code>internal/skillpkg/</code> 导出导入包 + <code>caudit</code> 审计日志控制器完成后自动启用。</p>
        <p class="muted">预计接口:<code>GET /api/skillbox/audit/logs</code> · <code>GET /api/skillbox/audit/stats</code></p>
      </div>
    </div>

    <!-- 列表 -->
    <div v-else class="card">
      <h3>日志列表
        <span class="card-sub">— 共 {{ total }} 条</span>
      </h3>

      <div class="filters">
        <label>
          <span>Action</span>
          <select v-model="filterAction" @change="reload">
            <option v-for="a in ACTION_OPTIONS" :key="a" :value="a">{{ a || '全部' }}</option>
          </select>
        </label>
        <label>
          <span>Actor</span>
          <input v-model="filterActor" placeholder="用户名" @keyup.enter="reload" />
        </label>
        <label>
          <span>Target Type</span>
          <input v-model="filterTargetType" placeholder="skill / project / ..." @keyup.enter="reload" />
        </label>
        <button class="primary" @click="reload">应用过滤</button>
      </div>

      <table v-if="logs.length" class="grid">
        <thead>
          <tr>
            <th>ID</th>
            <th>Time</th>
            <th>Actor</th>
            <th>Action</th>
            <th>Target</th>
            <th>Payload</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.ID || log.id">
            <td>{{ log.ID || log.id }}</td>
            <td class="time">{{ (log.CreatedAt || log.created_at || '').slice(0, 19) }}</td>
            <td><code>{{ log.Actor || log.actor }}</code></td>
            <td>
              <span :class="['tag', actionColor(log.Action || log.action)]">
                {{ log.Action || log.action }}
              </span>
            </td>
            <td>
              <code class="target">{{ (log.TargetType || log.target_type) }}#{{ log.TargetID || log.target_id }}</code>
            </td>
            <td class="payload">
              <details>
                <summary>查看</summary>
                <pre>{{ log.Payload || log.payload || '—' }}</pre>
              </details>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!loading" class="empty-state">
        <span class="empty-icon">📭</span>
        没有匹配的日志记录
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">上一页</button>
        <span>第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 条</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">下一页</button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.audit { max-width: 1100px; margin: 0 auto; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }

.stats-row { display: grid; grid-template-columns: 1fr 2fr 2fr; gap: 10px; margin-bottom: 14px; }
.stat-card { background: #fff; border: 1px solid var(--border); border-radius: var(--radius); padding: 12px 14px; }
.stat-label { font-size: 11px; color: var(--text-dim); text-transform: uppercase; letter-spacing: 0.5px; }
.stat-value { font-size: 22px; font-weight: 600; color: var(--text); margin-top: 4px; }
.action-chips { display: flex; flex-wrap: wrap; gap: 4px 6px; margin-top: 6px; }
.chip { font-size: 11px; padding: 2px 7px; background: #f3f4f6; border-radius: 10px; }

.placeholder .empty-state { padding: 36px 20px; }

.filters { display: flex; gap: 8px; align-items: end; margin-bottom: 12px; flex-wrap: wrap; }
.filters label { display: flex; flex-direction: column; gap: 3px; font-size: 12px; color: var(--text-dim); }

.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { padding: 8px 10px; text-align: left; border-bottom: 1px solid #f3f4f6; }
.grid th { background: #f9fafb; color: var(--text-dim); font-weight: 600; }
.time { color: var(--text-dim); font-size: 12px; }
.target { background: #f3f4f6; padding: 1px 6px; border-radius: 3px; }
.payload pre { background: #f9fafb; padding: 8px 10px; border-radius: var(--radius-sm); font-size: 11px; max-height: 200px; overflow: auto; margin: 4px 0 0; }
.payload summary { cursor: pointer; color: var(--primary); font-size: 12px; }

.tag { display: inline-block; padding: 1px 8px; border-radius: 10px; font-size: 11px; font-weight: 500; background: #f3f4f6; color: var(--text-dim); }
.tag.ok { background: var(--success-dim); color: var(--success); }
.tag.err { background: var(--danger-dim); color: var(--danger); }
.tag.warn { background: var(--warning-dim); color: var(--warning); }

.pager { display: flex; align-items: center; gap: 12px; margin-top: 12px; font-size: 13px; color: var(--text-dim); justify-content: flex-end; }
</style>
