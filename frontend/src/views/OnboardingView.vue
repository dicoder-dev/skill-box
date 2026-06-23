<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { getOnboardingStatus, runOnboardingScan, runOnboardingImport } from '@/api/skillbox/onboarding'
import { listSkills } from '@/api/skillbox/skills'
import { platform } from '@/platform'
import SkillTitle from '@/components/SkillTitle.vue'

const { t } = useI18n()

// 直接进入"扫描结果"阶段,跳过 adapter 状态展示步骤。
// 状态信息仍可通过 App.vue 顶栏的工具徽章(stats.toolsReady / toolsTotal)查看。
const phase = ref('scan')
const appBus = inject('appBus', null)

const loading = ref(false)
const error = ref('')
const success = ref('')

const adapters = ref([])
const lastScan = ref(null)
const totalFound = ref(0)
const hasReport = ref(false)

const scanReport = ref(null)
const selected = ref(new Set())
// 当前激活的工具 tab id;phase === 'scan' 时使用。
const activeToolId = ref('')

const importResult = ref(null)

// 客户端 store 中已存在的 skill 名集合(用于 phase2 重复检测)。
// name 不区分大小写,同 store 的全局作用域只保留一个 skill(name 是 unique key)。
const existingNames = ref(new Set())

// skill md 文件首行 # 标题缓存:source_path → title 或空。
// 不在 scan 时一次性读(可能 N 个文件、慢),改为 phase2 渲染时按需拉。
const skillTitles = ref({})

// tool_id → mdi 图标映射。
// 后端 IconEmoji 字段已废弃(2026-06-23 清理乱码字节,项目规范禁 emoji),
// 改由前端按 tool_id 决定图标,5 个工具都给到语义化的 mdi 图标。
const toolIconMap = {
  claude: 'mdi:robot-outline',
  codex: 'mdi:cube-outline',
  cursor: 'mdi:cursor-default-click-outline',
  opencode: 'mdi:code-braces',
  trae: 'mdi:shield-outline',
}
function iconOf(toolId) {
  return toolIconMap[toolId] || 'mdi:puzzle-outline'
}

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

async function loadExistingNames() {
  // 读客户端 store 里的所有 skill(只拿 name,page=1, size 拉满)。
  // 用于 phase2 标"已存在" + 跨工具同名互斥。
  try {
    const res = await listSkills({ page: 1, size: 1000 })
    const set = new Set()
    for (const it of res?.items || []) {
      if (it?.name) set.add(String(it.name).toLowerCase())
    }
    existingNames.value = set
  } catch (_) {
    // 拉不到也不阻塞 onboarding,只是少一个"已存在"标记
    existingNames.value = new Set()
  }
}

function keyOf(found) {
  return `${found.tool_id}::${found.name}@${found.version}`
}

async function doScan() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const [res] = await Promise.all([runOnboardingScan(), loadExistingNames()])
    scanReport.value = res
    // 默认勾选:仅 user 级别。system 级别(工具自带 / vendor curated /
    // plugin 内建)只读展示,不能误导入覆盖本地 store。
    // 同时跳过客户端已存在的同名 skill —— 那是重复,不该默认勾选。
    selected.value = new Set(
      (res.found || [])
        .filter((f) => f.category !== 'system')
        .filter((f) => !existingNames.value.has(String(f.name).toLowerCase()))
        .map((f) => keyOf(f)),
    )
    // 默认激活第一个有 user 级别发现的 tab,避免空 tab。
    const firstTid = (res.tools || []).find(
      (tid) =>
        (res.found || []).some(
          (f) => f.tool_id === tid && f.category !== 'system',
        ),
    ) || (res.tools || [])[0]
    activeToolId.value = firstTid || ''
  } catch (e) {
    error.value = t('onboarding.errScan', { msg: e?.message || e })
  } finally {
    loading.value = false
  }
}

// 选中态切换(带重复检测)。
//   - 客户端已存在 → 直接不允许切换,无效操作
//   - 跨工具同名互斥:勾选 f 时,自动把"同名但不同 tool_id"的勾选清掉
function toggleSelect(found) {
  if (isDisabled(found)) return
  const k = keyOf(found)
  let s = new Set(selected.value)
  if (s.has(k)) {
    s.delete(k)
  } else {
    // 跨工具同名互斥:同名(name, version)只能选一个 tool_id
    s = selectExclusiveByName(s, found)
    s.add(k)
  }
  selected.value = s
}

// 把 selected 中所有"与 f 同名但 tool_id 不同"的项移除。
// 在勾选方块触发时调用,实现"选一个后,另一个工具的同名项自动取消"。
function selectExclusiveByName(s, found) {
  const next = new Set(s)
  const nameKey = String(found.name).toLowerCase()
  for (const k of Array.from(next)) {
    const [tid, nameVer] = k.split('::')
    const [n, v] = nameVer.split('@')
    if (
      String(n).toLowerCase() === nameKey &&
      v === found.version &&
      tid !== found.tool_id
    ) {
      next.delete(k)
    }
  }
  return next
}

// 是否应该被禁用:
//   1) system 级别 → 不可勾选(后端也允许,但 UI 锁死)
//   2) 客户端 store 已存在同名 skill → 置灰 + 提示"客户端已存在"
//   3) 跨工具同名:另一个 tool_id 已被勾选 → 当前项禁用
function isDisabled(found) {
  if (found.category === 'system') return true
  if (existingNames.value.has(String(found.name).toLowerCase())) return true
  const k = keyOf(found)
  for (const sel of selected.value) {
    if (sel === k) continue
    const [tid, nameVer] = sel.split('::')
    const [n, v] = nameVer.split('@')
    if (
      String(n).toLowerCase() === String(found.name).toLowerCase() &&
      v === found.version &&
      tid !== found.tool_id
    ) {
      return true
    }
  }
  return false
}

function disabledReason(found) {
  if (found.category === 'system') {
    return t('onboarding.phase2.disabledSystem')
  }
  if (existingNames.value.has(String(found.name).toLowerCase())) {
    return t('onboarding.phase2.disabledExists')
  }
  return t('onboarding.phase2.disabledExclusive')
}

// 按 tool_id 分组的 skill 列表 + 元数据(显示名 / 数量 / 当前 tab id)。
//
// 组内排序:user 级别在前 system 级别在后 —— 用户日常用的 skill 优先展示,
// 系统自带 / plugin 内建 / vendor curated 的列在下方,只读不可勾选。
const foundByTool = computed(() => {
  const groups = {}
  // 先按 scanReport.tools 顺序建空组,保证 tab 顺序稳定(后端已排序:只保留有 found 的)
  for (const tid of scanReport.value?.tools || []) {
    groups[tid] = { name: '', items: [] }
  }
  for (const f of scanReport.value?.found || []) {
    if (!groups[f.tool_id]) {
      // found 里有但 tools 里没有的(理论上后端已过滤,这里是兜底)
      groups[f.tool_id] = { name: f.tool_name || f.tool_id, items: [] }
    }
    if (!groups[f.tool_id].name) groups[f.tool_id].name = f.tool_name || f.tool_id
    groups[f.tool_id].items.push(f)
  }
  // 组内按 category(user 先 system 后)稳定排序。
  for (const tid of Object.keys(groups)) {
    groups[tid].items.sort((a, b) => {
      const ax = a.category === 'system' ? 1 : 0
      const bx = b.category === 'system' ? 1 : 0
      if (ax !== bx) return ax - bx
      return a.name.localeCompare(b.name)
    })
  }
  return groups
})

const toolTabs = computed(() =>
  Object.entries(foundByTool.value).map(([tid, g]) => ({
    toolId: tid,
    // 兜底:极端情况下 name 仍为空,用 toolId 顶上
    name: g.name || tid,
    count: g.items.filter((f) => f.category !== 'system').length,
    totalCount: g.items.length,
    icon: iconOf(tid),
  })),
)

// 工具内"全选/全不选":只动当前 tab 的 user 级别 skill,跳过 system + 客户端已存在 + 跨工具已占。
function selectAllInTool(tid) {
  let s = new Set(selected.value)
  for (const f of foundByTool.value[tid]?.items || []) {
    if (isDisabled(f)) continue
    s = selectExclusiveByName(s, f)
    s.add(keyOf(f))
  }
  selected.value = s
}
function selectNoneInTool(tid) {
  const s = new Set(selected.value)
  for (const f of foundByTool.value[tid]?.items || []) {
    if (f.category === 'system') continue
    s.delete(keyOf(f))
  }
  selected.value = s
}
function selectedInTool(tid) {
  let n = 0
  for (const f of foundByTool.value[tid]?.items || []) {
    if (f.category === 'system') continue
    if (selected.value.has(keyOf(f))) n++
  }
  return n
}

// 可选数量(去掉 system + 客户端已存在 + 跨工具已占)。
function selectableInTool(tid) {
  let n = 0
  for (const f of foundByTool.value[tid]?.items || []) {
    if (!isDisabled(f)) n++
  }
  return n
}

// 拉取单个 skill 的标题(从 source_path/SKILL.md 的第一行 # ...)。
// 失败 / 解析失败 / 不存在 → 空字符串。
async function fetchTitle(sourcePath) {
  if (!sourcePath) return ''
  if (sourcePath in skillTitles.value) return skillTitles.value[sourcePath]
  try {
    const r = await platform.fs.readText(sourcePath).catch(() => null)
    const title = r ? parseMarkdownTitle(r) : ''
    skillTitles.value = { ...skillTitles.value, [sourcePath]: title }
    return title
  } catch (_) {
    skillTitles.value = { ...skillTitles.value, [sourcePath]: '' }
    return ''
  }
}

// 解析 SKILL.md 第一行 # 标题(跳 frontmatter 段)。
function parseMarkdownTitle(md) {
  if (!md) return ''
  let body = md
  if (body.startsWith('---')) {
    const end = body.indexOf('\n---', 3)
    if (end > 0) body = body.slice(end + 4)
  }
  for (const line of body.split('\n')) {
    const t = line.trim()
    if (t.startsWith('# ')) {
      return t.slice(2).trim()
    }
  }
  return ''
}

// 在系统文件管理器中显示 skill 所在目录(桌面端有实现,Web 端 no-op)。
async function revealInFileManager(sourcePath) {
  if (!sourcePath) return
  try {
    await platform.fs.reveal(sourcePath)
  } catch (_) {
    // 兜底:Web 端或桌面端没实现 reveal 时,试 openExternal 走 file:// 协议
    try {
      await platform.platform.openExternal('file://' + sourcePath)
    } catch (__) { /* ignore */ }
  }
}

async function doImport() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
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
    success.value = t('onboarding.okImport', { ok: res.ok, failed: res.failed })
    await loadStatus()
  } catch (e) {
    error.value = t('onboarding.errImport', { msg: e?.message || e })
  } finally {
    loading.value = false
  }
}

function reset() {
  phase.value = 'scan'
  scanReport.value = null
  importResult.value = null
  selected.value = new Set()
  activeToolId.value = ''
}

function goSkills() {
  if (appBus) {
    appBus.emit('switch-tab', 'skills')
  } else {
    window.dispatchEvent(new CustomEvent('skillbox:switch-tab', { detail: 'skills' }))
  }
}

onMounted(async () => {
  await loadStatus()
  await doScan()
})
</script>

<template>
  <div class="onb">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-violet">
          <Icon icon="mdi:compass-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('onboarding.title') }}</h1>
          <p>{{ t('onboarding.subtitle') }}</p>
        </div>
      </div>
      <div v-if="phase === 'scan'" class="header-actions">
        <button class="ghost" :disabled="loading" @click="doScan" :title="t('onboarding.btnRescanTitle')">
          <span v-if="loading" class="spinner"></span>
          <Icon v-else icon="mdi:refresh" width="14" height="14" />
          {{ loading ? t('onboarding.btnRescanning') : t('onboarding.btnRescan') }}
        </button>
      </div>
    </header>

    <p v-if="error" class="message message-error">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>
    <p v-if="success" class="message message-success">
      <Icon icon="mdi:check-circle-outline" width="14" height="14" />
      {{ success }}
    </p>

    <!-- 阶段 2: 扫描 + 勾选(tab 面板,按工具拆分) -->
    <section v-else-if="phase === 'scan'" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:folder-search" width="16" height="16" />
          {{ t('onboarding.phase2.title') }}
          <span class="card-sub">— {{ t('onboarding.phase2.foundSuffix', { n: scanReport?.found?.length || 0 }) }}</span>
        </h3>
      </header>

      <div v-if="!scanReport?.found?.length" class="empty-state">
        <Icon icon="mdi:magnify" width="48" height="48" />
        <p class="empty-title">{{ t('onboarding.phase2.empty') }}</p>
        <p class="empty-hint">{{ t('onboarding.phase2.emptyHint') }}</p>
      </div>

      <div v-else>
        <!-- 工具 tab 栏:瑞士风,上划线 + 数字徽章 -->
        <div class="tool-tabs" role="tablist">
          <button
            v-for="tab in toolTabs"
            :key="tab.toolId"
            role="tab"
            :aria-selected="activeToolId === tab.toolId"
            :class="['tool-tab', { active: activeToolId === tab.toolId }]"
            @click="activeToolId = tab.toolId"
          >
            <Icon :icon="tab.icon" width="16" height="16" class="tab-icon" />
            <span class="tab-name">{{ tab.name }}</span>
            <span class="tab-count">
              {{ tab.count }}<span v-if="tab.totalCount > tab.count" class="tab-count-sys">+{{ tab.totalCount - tab.count }}</span>
            </span>
          </button>
        </div>

        <!-- 当前 tab 内容 -->
        <div v-if="activeToolId && foundByTool[activeToolId]" class="tool-panel">
          <div class="bulk-actions">
            <button class="sm" @click="selectAllInTool(activeToolId)">
              {{ t('onboarding.phase2.selectAll') }}
            </button>
            <button class="sm ghost" @click="selectNoneInTool(activeToolId)">
              {{ t('onboarding.phase2.selectNone') }}
            </button>
            <span class="selection-info">
              {{ t('onboarding.phase2.selected', {
                  sel: selectedInTool(activeToolId),
                  total: selectableInTool(activeToolId),
              }) }}
            </span>
          </div>

          <!-- 分档小标题:用户 skill 在前,系统 skill 在后 -->
          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'user' || !f.category)"
            class="cat-label cat-user"
          >
            <Icon icon="mdi:account-circle-outline" width="14" height="14" />
            {{ t('onboarding.phase2.catUser') }}
          </div>
          <ul v-if="foundByTool[activeToolId].items.some((f) => f.category !== 'system')" class="found-list">
            <li v-for="f in foundByTool[activeToolId].items.filter((x) => x.category !== 'system')"
                :key="keyOf(f)"
                :class="{ selected: selected.has(keyOf(f)), disabled: isDisabled(f) }">
              <label
                class="found-item"
                :class="{ 'item-disabled': isDisabled(f) }"
                :title="isDisabled(f) ? disabledReason(f) : ''"
              >
                <input
                  type="checkbox"
                  :checked="selected.has(keyOf(f))"
                  :disabled="isDisabled(f)"
                  @change="toggleSelect(f)"
                />
                <div class="f-main">
                  <div class="f-line-1">
                    <span class="f-name"><code>{{ f.name }}</code></span>
                    <span class="f-ver">v{{ f.version }}</span>
                    <span v-if="existingNames.has(String(f.name).toLowerCase())" class="f-tag f-tag-exists">
                      <Icon icon="mdi:package-variant" width="11" height="11" />
                      {{ t('onboarding.phase2.tagExists') }}
                    </span>
                  </div>
                  <SkillTitle :source-path="f.source_path" :fetcher="fetchTitle" />
                  <div class="f-line-2">
                    <button
                      type="button"
                      class="f-path-btn"
                      :title="f.source_path"
                      @click.stop="revealInFileManager(f.source_path)"
                    >
                      <Icon icon="mdi:folder-outline" width="14" height="14" />
                    </button>
                    <span class="f-path-text">{{ f.source_path }}</span>
                  </div>
                </div>
              </label>
            </li>
          </ul>

          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')"
            class="cat-divider"
          >
            <span class="cat-divider-text">{{ t('onboarding.phase2.catSectionDivider') }}</span>
          </div>
          <div
            v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')"
            class="cat-label cat-system"
          >
            <Icon icon="mdi:lock-outline" width="14" height="14" />
            {{ t('onboarding.phase2.catSystem') }}
            <span class="cat-hint">— {{ t('onboarding.phase2.catSystemHint') }}</span>
          </div>
          <ul v-if="foundByTool[activeToolId].items.some((f) => f.category === 'system')" class="found-list found-list-system">
            <li v-for="f in foundByTool[activeToolId].items.filter((x) => x.category === 'system')"
                :key="keyOf(f)"
                class="system-item">
              <span class="found-item found-item-system">
                <input
                  type="checkbox"
                  disabled
                  aria-disabled="true"
                  :title="t('onboarding.phase2.catSystemHint')"
                />
                <div class="f-main">
                  <div class="f-line-1">
                    <span class="f-name"><code>{{ f.name }}</code></span>
                    <span class="f-ver">v{{ f.version }}</span>
                  </div>
                  <SkillTitle :source-path="f.source_path" :fetcher="fetchTitle" />
                  <div class="f-line-2">
                    <button
                      type="button"
                      class="f-path-btn"
                      :title="f.source_path"
                      @click.stop="revealInFileManager(f.source_path)"
                    >
                      <Icon icon="mdi:folder-outline" width="14" height="14" />
                    </button>
                    <span class="f-path-text">{{ f.source_path }}</span>
                  </div>
                </div>
                <Icon icon="mdi:lock-outline" width="12" height="12" class="lock-icon" />
              </span>
            </li>
          </ul>
        </div>

        <div class="card-footer">
          <button class="ghost" @click="doScan" :disabled="loading">
            <Icon icon="mdi:refresh" width="14" height="14" />
            {{ t('onboarding.btnRescan') }}
          </button>
          <button class="primary" :disabled="loading || selected.size === 0" @click="doImport">
            <span v-if="loading" class="spinner"></span>
            <Icon v-else icon="mdi:download" width="14" height="14" />
            {{ loading ? t('onboarding.phase2.importing') : t('onboarding.phase2.btnImport', { n: selected.size }) }}
          </button>
        </div>
      </div>
    </section>

    <!-- 阶段 3: 完成 -->
    <section v-else-if="phase === 'import'" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:check-circle" width="16" height="16" />
          {{ t('onboarding.phase3.title') }}
        </h3>
      </header>

      <div v-if="importResult" class="result-stats">
        <div class="stat-card stat-success">
          <span class="stat-number">{{ importResult.ok }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statOk') }}</span>
        </div>
        <div class="stat-card stat-error">
          <span class="stat-number">{{ importResult.failed }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statErr') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-number">{{ importResult.total }}</span>
          <span class="stat-label">{{ t('onboarding.phase3.statTotal') }}</span>
        </div>
      </div>

      <ul v-if="importResult?.results?.length" class="result-list">
        <li v-for="(r, i) in importResult.results" :key="i" :class="r.ok ? 'result-ok' : 'result-error'">
          <span class="r-tool">{{ r.tool_id || r.tool || t('common.dash') }}</span>
          <span class="r-name"><code>{{ r.name || r.canonical?.manifest?.name }}</code></span>
          <span class="r-msg">{{ r.error || r.message || (r.ok ? 'OK' : 'failed') }}</span>
        </li>
      </ul>

      <div class="card-footer">
        <button class="ghost" @click="reset">
          <Icon icon="mdi:refresh" width="14" height="14" />
          {{ t('onboarding.phase3.btnAgain') }}
        </button>
        <button class="primary" @click="goSkills">
          <Icon icon="mdi:arrow-right" width="14" height="14" />
          {{ t('onboarding.phase3.btnGoSkills') }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.onb {
  max-width: 980px;
  margin: 0 auto;
  color: var(--text);
  transition: color 0.3s ease;
}

/* 头部 action 区(header 右侧按钮) */
.view-header {
  margin-bottom: 24px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.header-actions {
  display: flex;
  gap: 8px;
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
  background: var(--accent-violet);
  color: #ffffff;
  flex-shrink: 0;
}

.view-icon-violet {
  background: var(--accent-violet);
  color: #ffffff;
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

/* 卡片 */
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
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.card-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.footer-info {
  font-size: 13px;
  color: var(--text-dim);
}

/* 表格 */
.grid {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.grid th, .grid td {
  padding: 12px 14px;
  text-align: left;
  border-bottom: 1px solid var(--border);
  transition: background-color 0.3s ease;
}

.grid th {
  background: var(--bg-subtle);
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

.icon-cell {
  color: var(--accent-blue);
}
.tool-icon {
  vertical-align: middle;
}

.td-path {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
  font-size: 12px;
}

/* 徽章 */
.badge {
  display: inline-flex;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
}

.badge-success {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border: 1px solid var(--accent-emerald-border);
}

.badge-warning {
  background: var(--accent-amber-bg);
  color: var(--accent-amber);
  border: 1px solid var(--accent-amber-border);
}

/* 批量操作 */
.bulk-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
}

.selection-info {
  font-size: 13px;
  color: var(--text-dim);
  margin-left: auto;
}

/* 工具 tab 栏:瑞士风,上划线 + 数字徽章 */
.tool-tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 16px;
  border-bottom: 1px solid var(--border);
  overflow-x: auto;
  scrollbar-width: thin;
}

.tool-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s ease, border-color 0.15s ease;
}

.tool-tab:hover:not(.active) {
  color: var(--text);
}

.tool-tab.active {
  color: var(--accent-blue);
  border-bottom-color: var(--accent-blue);
}

.tool-tab .tab-icon {
  flex-shrink: 0;
}

.tool-tab .tab-name {
  font-weight: 500;
}

.tool-tab .tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 18px;
  padding: 0 6px;
  border-radius: 9px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  font-size: 11px;
  font-weight: 600;
  font-feature-settings: 'tnum';
  transition: background 0.15s ease, color 0.15s ease;
}

.tool-tab.active .tab-count {
  background: var(--accent-blue-bg);
  color: var(--accent-blue);
}

.tool-tab .tab-count-sys {
  margin-left: 4px;
  color: var(--text-faint);
  font-weight: 500;
}

.tool-tab.active .tab-count-sys {
  color: var(--text-dim);
}

/* 分档小标题(user / system) */
.cat-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  margin-bottom: 10px;
}

.cat-user {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
  border: 1px solid var(--accent-emerald-border);
}

.cat-system {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.cat-hint {
  text-transform: none;
  letter-spacing: 0;
  font-weight: normal;
  color: var(--text-faint);
  margin-left: 6px;
}

.cat-divider {
  margin: 18px 0 12px;
  text-align: center;
  position: relative;
}

.cat-divider::before {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: 50%;
  height: 1px;
  background: var(--border);
  z-index: 0;
}

.cat-divider-text {
  position: relative;
  z-index: 1;
  background: var(--bg-card);
  padding: 0 12px;
  font-size: 11px;
  color: var(--text-faint);
  letter-spacing: 0.3px;
}

/* 系统级 skill 列表:灰色背景,checkbox 禁用 + 锁图标 */
.found-list-system {
  opacity: 0.78;
}

.found-list-system .found-item-system {
  cursor: not-allowed;
}

.found-list-system .found-item-system input[type="checkbox"]:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.found-list-system .f-name code,
.found-list-system .f-ver,
.found-list-system .f-path-text {
  color: var(--text-dim);
}

.lock-icon {
  color: var(--text-faint);
  flex-shrink: 0;
  margin-left: 4px;
}

/* 当前 tab 内容区(单工具的 skill 列表) */
.tool-panel {
  padding-top: 4px;
}

/* 发现列表 */
.found-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.found-list li {
  border-bottom: 1px solid var(--border);
  transition: background 0.15s ease;
}

.found-list li:last-child {
  border-bottom: none;
}

.found-list li.selected {
  background: var(--accent-blue-bg);
}

.found-list li.disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.found-item input[type="checkbox"]:checked {
  accent-color: var(--accent-blue);
}

/* found-item 是 label,内部用 .f-main 三行布局:
   - line-1: name + ver + tag
   - line-2: skill title (subtitle,选填)
   - line-3: 文件夹图标 + 路径(单行) */
.found-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  cursor: pointer;
}

.found-item.item-disabled {
  cursor: not-allowed;
}

.found-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  margin-top: 2px;
  flex-shrink: 0;
}

.found-item input[type="checkbox"]:disabled {
  cursor: not-allowed;
}

.f-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.f-line-1 {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.f-name {
  font-weight: 500;
  font-size: 13px;
  color: var(--text);
}

.f-name code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}

.f-ver {
  font-size: 11px;
  color: var(--text-dim);
  font-weight: 500;
}

/* 标签:客户端已存在 */
.f-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.2px;
  line-height: 1.5;
  white-space: nowrap;
}

.f-tag-exists {
  background: var(--accent-amber-bg);
  color: var(--accent-amber);
  border: 1px solid var(--accent-amber-border);
}

/* line-2: 文件夹图标 + 路径 */
.f-line-2 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 2px;
  min-width: 0;
}

.f-path-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: var(--bg-subtle);
  color: var(--text-dim);
  cursor: pointer;
  flex-shrink: 0;
  padding: 0;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.f-path-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--accent-blue-border);
}

.f-path-text {
  flex: 1;
  min-width: 0;
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 结果统计 */
.result-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  padding: 20px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  text-align: center;
  transition: all 0.3s ease;
}

.stat-success {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
}

.stat-error {
  background: var(--accent-rose-bg);
  border-color: var(--accent-rose-border);
}

.stat-number {
  display: block;
  font-size: 32px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
  margin-bottom: 8px;
}

.stat-success .stat-number { color: var(--accent-emerald); }
.stat-error .stat-number { color: var(--accent-rose); }

.stat-label {
  font-size: 12px;
  color: var(--text-dim);
}

/* 结果列表 */
.result-list {
  list-style: none;
  padding: 0;
  margin: 0;
  max-height: 300px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.result-list li {
  display: grid;
  grid-template-columns: 100px 180px 1fr;
  gap: 12px;
  padding: 10px 14px;
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}

.result-list li:last-child {
  border-bottom: none;
}

.result-ok {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
}

.result-error {
  background: var(--accent-rose-bg);
  color: var(--accent-rose);
}

.result-ok .r-name,
.result-error .r-name { color: var(--text); }

.result-ok .r-tool,
.result-error .r-tool { color: var(--text-dim); }

.r-tool {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
}

.r-name {
  color: var(--text);
}

.r-msg {
  color: inherit;
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
  color: var(--text-dim);
  margin: 6px 0 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .steps {
    flex-direction: column;
    align-items: stretch;
  }

  .step-connector {
    width: 2px;
    height: 12px;
    margin: 0;
    align-self: flex-start;
    margin-left: 30px;
  }

  .result-stats {
    grid-template-columns: 1fr;
  }

  .found-item {
    flex-wrap: wrap;
  }
}
</style>
