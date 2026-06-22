<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { getOnboardingStatus, runOnboardingScan, runOnboardingImport } from '@/api/skillbox/onboarding'

// 阶段: status(初始状态) → scan(扫描结果) → import(导入结果)
const phase = ref('status')

const loading = ref(false)
const error = ref('')
const success = ref('')

// 状态
const adapters = ref([])
const lastScan = ref(null)
const totalFound = ref(0)
const hasReport = ref(false)

// 扫描报告
const scanReport = ref(null) // { tools, summary, found, scanned_at }
const selected = ref(new Set()) // 选中的 key: `${tool_id}::${name}@${version}`

// 导入结果
const importResult = ref(null) // { total, ok, failed, results }

async function loadStatus() {
  loading.value = true
  error.value = ''
  try {
    const res = await getOnboardingStatus()
    adapters.value = res?.adapters || []
    lastScan.value = res?.last_scan || null
    totalFound.value = res?.total_found || 0
    hasReport.value = !!res?.has_report
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function doScan() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const res = await runOnboardingScan()
    scanReport.value = res
    // 默认全选
    selected.value = new Set((res.found || []).map((f) => keyOf(f)))
    phase.value = 'scan'
  } catch (e) {
    error.value = `扫描失败: ${e?.message || e}`
  } finally {
    loading.value = false
  }
}

function keyOf(found) {
  return `${found.tool_id}::${found.name}@${found.version}`
}

function toggleSelect(found) {
  const k = keyOf(found)
  const s = new Set(selected.value)
  if (s.has(k)) s.delete(k)
  else s.add(k)
  selected.value = s
}

function selectAll() { selected.value = new Set((scanReport.value?.found || []).map(keyOf)) }
function selectNone() { selected.value = new Set() }

const foundByTool = computed(() => {
  const groups = {}
  for (const f of scanReport.value?.found || []) {
    if (!groups[f.tool_id]) groups[f.tool_id] = { name: f.tool_name, items: [] }
    groups[f.tool_id].items.push(f)
  }
  return groups
})

async function doImport() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    // 后端:items=空 = 全部。我们这里把"全选"映射成空,否则按 selected 构造 items
    const items = (selected.value.size === (scanReport.value?.found || []).length)
      ? []
      : Array.from(selected.value).map((k) => {
          const [tool_id, nameVer] = k.split('::')
          const [name, version] = nameVer.split('@')
          return { tool_id, name, version }
        })
    const res = await runOnboardingImport(items)
    importResult.value = res
    phase.value = 'import'
    success.value = `导入完成: ${res.ok} 成功 / ${res.failed} 失败`
    await loadStatus()
  } catch (e) {
    error.value = `导入失败: ${e?.message || e}`
  } finally {
    loading.value = false
  }
}

function reset() {
  phase.value = 'status'
  scanReport.value = null
  importResult.value = null
  selected.value = new Set()
}

onMounted(loadStatus)
</script>

<template>
  <div class="onb">
    <header class="head">
      <h2 class="flex items-center gap-2">
        <Icon icon="mdi:compass-outline" width="20" height="20" class="text-sb-primary" />
        首次 Onboarding
      </h2>
      <p class="muted">扫描本机 5 个 AI 编程工具的 skill 目录,把发现的 skill 勾选导入到 Skill Box 自己的 store(global scope)。</p>
    </header>

    <!-- 阶段指示器 -->
    <ol class="steps">
      <li :class="{ active: phase === 'status', done: ['scan', 'import'].includes(phase) }">
        <span class="step-no">1</span>
        <span class="step-text">查看状态</span>
      </li>
      <li :class="{ active: phase === 'scan', done: phase === 'import' }">
        <span class="step-no">2</span>
        <span class="step-text">扫描 + 勾选</span>
      </li>
      <li :class="{ active: phase === 'import' }">
        <span class="step-no">3</span>
        <span class="step-text">完成</span>
      </li>
    </ol>

    <p v-if="error" class="error inline-flex items-center gap-1.5">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />{{ error }}
    </p>
    <p v-if="success" class="success inline-flex items-center gap-1.5">
      <Icon icon="mdi:check-circle-outline" width="14" height="14" />{{ success }}
    </p>

    <!-- 阶段 1:状态 -->
    <section v-if="phase === 'status'" class="card">
      <h3>工具 adapter 状态
        <span class="card-sub">— 共 {{ adapters.length }} 个</span>
      </h3>
      <div v-if="!adapters.length" class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:inbox-outline" width="36" height="36" />
        </span>
        还没注册 adapter
      </div>
      <table v-else class="grid">
        <thead>
          <tr>
            <th style="width: 50px"></th>
            <th>Tool</th>
            <th>ID</th>
            <th>Global Path</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in adapters" :key="a.tool_id">
            <td class="icon-cell">{{ a.icon }}</td>
            <td><b>{{ a.display_name }}</b></td>
            <td><code>{{ a.tool_id }}</code></td>
            <td class="path">{{ a.global_path || '—' }}</td>
            <td>
              <span v-if="a.global_ok" class="tag ok">已检测到</span>
              <span v-else class="tag missing">未找到</span>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="actions">
        <span class="muted">
          上次扫描:{{ lastScan ? new Date(lastScan).toLocaleString() : '从未' }}
          <span v-if="hasReport">· 共发现 {{ totalFound }} 个 skill</span>
        </span>
        <button class="primary" :disabled="loading" @click="doScan">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? '扫描中…' : '开始扫描' }}
        </button>
      </div>
    </section>

    <!-- 阶段 2:扫描 + 勾选 -->
    <section v-else-if="phase === 'scan'" class="card">
      <h3>扫描结果
        <span class="card-sub">— 发现 {{ scanReport?.found?.length || 0 }} 个 skill</span>
      </h3>

      <div v-if="!scanReport?.found?.length" class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:magnify" width="36" height="36" />
        </span>
        这次扫描没找到任何 skill。可以重扫或先装一些。
      </div>

      <div v-else>
        <div class="bulk-actions">
          <button class="sm" @click="selectAll">全选</button>
          <button class="sm ghost" @click="selectNone">全不选</button>
          <span class="muted">已选 {{ selected.size }} / {{ scanReport.found.length }}</span>
        </div>

        <div v-for="(g, tid) in foundByTool" :key="tid" class="group">
          <header class="group-head">
            <span class="g-name">{{ g.name }}</span>
            <code class="g-id">{{ tid }}</code>
            <span class="muted">{{ g.items.length }} 个</span>
          </header>
          <ul class="found-list">
            <li v-for="f in g.items" :key="keyOf(f)" :class="{ sel: selected.has(keyOf(f)) }">
              <label>
                <input
                  type="checkbox"
                  :checked="selected.has(keyOf(f))"
                  @change="toggleSelect(f)"
                />
                <span class="f-name"><code>{{ f.name }}</code></span>
                <span class="f-ver muted">v{{ f.version }}</span>
                <span class="f-path muted">{{ f.source_path }}</span>
              </label>
            </li>
          </ul>
        </div>

        <div class="actions">
          <button class="ghost" @click="reset">返回上一步</button>
          <button class="primary" :disabled="loading || selected.size === 0" @click="doImport">
            <span v-if="loading" class="spinner"></span>
            {{ loading ? '导入中…' : `导入 ${selected.size} 个到 store` }}
          </button>
        </div>
      </div>
    </section>

    <!-- 阶段 3:完成 -->
    <section v-else-if="phase === 'import'" class="card">
      <h3>导入完成</h3>
      <div v-if="importResult" class="result-stats">
        <div class="stat ok">
          <span class="stat-num">{{ importResult.ok }}</span>
          <span class="stat-lbl">成功</span>
        </div>
        <div class="stat err">
          <span class="stat-num">{{ importResult.failed }}</span>
          <span class="stat-lbl">失败</span>
        </div>
        <div class="stat">
          <span class="stat-num">{{ importResult.total }}</span>
          <span class="stat-lbl">总计</span>
        </div>
      </div>
      <ul v-if="importResult?.results?.length" class="result-list">
        <li v-for="(r, i) in importResult.results" :key="i" :class="r.ok ? 'ok' : 'err'">
          <span class="r-tool">{{ r.tool_id || r.tool || '—' }}</span>
          <span class="r-name"><code>{{ r.name || r.canonical?.manifest?.name }}</code></span>
          <span class="r-msg">{{ r.error || r.message || (r.ok ? 'OK' : 'failed') }}</span>
        </li>
      </ul>
      <div class="actions">
        <button class="ghost" @click="reset">再扫一次</button>
        <button class="primary" @click="$emit && $emit('jump', 'skills')">去 Skills 页查看</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.onb { max-width: 980px; margin: 0 auto; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }

.steps {
  list-style: none; padding: 0; margin: 0 0 16px;
  display: flex; gap: 6px;
}
.steps li {
  flex: 1; display: flex; align-items: center; gap: 8px;
  padding: 10px 14px; background: #fff; border: 1px solid var(--border);
  border-radius: var(--radius-sm); font-size: 13px; color: var(--text-dim);
}
.steps li.active { background: var(--primary-dim); border-color: var(--primary); color: var(--primary); font-weight: 500; }
.steps li.done { color: var(--success); }
.steps li.done .step-no { background: var(--success); color: #fff; }
.step-no {
  width: 22px; height: 22px; border-radius: 50%;
  background: #e5e7eb; color: var(--text-dim);
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600;
}
.steps li.active .step-no { background: var(--primary); color: #fff; }

.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { padding: 8px 10px; text-align: left; border-bottom: 1px solid #f3f4f6; }
.grid th { background: #f9fafb; color: var(--text-dim); font-weight: 600; }
.icon-cell { font-size: 18px; }
.path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text-dim); font-size: 12px; }
.tag { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; font-weight: 500; }
.tag.ok { background: var(--success-dim); color: var(--success); }
.tag.missing { background: #f3f4f6; color: var(--text-dim); }

.actions {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 14px; gap: 12px;
}

.bulk-actions { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; font-size: 13px; }

.group { margin: 10px 0; border: 1px solid var(--border); border-radius: var(--radius-sm); background: #fafbfc; }
.group-head {
  padding: 8px 12px; background: #f3f4f6;
  display: flex; align-items: center; gap: 8px;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
}
.g-name { font-weight: 600; }
.g-id { font-size: 11px; }

.found-list { list-style: none; padding: 0; margin: 0; }
.found-list li { padding: 0; border-bottom: 1px solid #f3f4f6; }
.found-list li:last-child { border-bottom: none; }
.found-list label {
  display: flex; align-items: center; gap: 10px; padding: 8px 12px; cursor: pointer;
}
.found-list li.sel { background: var(--primary-dim); }
.f-name { font-weight: 500; min-width: 140px; }
.f-ver { font-size: 12px; min-width: 60px; }
.f-path {
  flex: 1; font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.result-stats { display: flex; gap: 12px; margin: 12px 0; }
.stat {
  flex: 1; padding: 14px; background: #f9fafb; border-radius: var(--radius-sm);
  text-align: center;
}
.stat.ok { background: var(--success-dim); }
.stat.err { background: var(--danger-dim); }
.stat-num { display: block; font-size: 24px; font-weight: 600; }
.stat-lbl { font-size: 12px; color: var(--text-dim); }

.result-list { list-style: none; padding: 0; margin: 10px 0; max-height: 280px; overflow: auto; }
.result-list li {
  display: grid; grid-template-columns: 80px 160px 1fr; gap: 10px;
  padding: 6px 10px; font-size: 12px; border-bottom: 1px solid #f3f4f6;
}
.result-list li.ok { color: var(--success); }
.result-list li.err { color: var(--danger); }
.r-tool { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>
