<script setup>
// GlobalImportPanel - 从 ~/.agents/skills 全局目录检索并批量导入 skill。
//
// 2026-07-10 增:首页"导入技能"弹窗新增 Tab「全局目录」。
//
// 行为:
//   - mount → GET /api/skillbox/onboarding/global-skills 拿候选列表(name/version/source_path/description)
//   - 顶部搜索框:本地按 name/version/description 过滤
//   - 列表项:checkbox 多选;底部"导入 N 个到 store"按钮 → POST /import-global-paths
//   - 后端响应的 LocalImportResult 与 LocalImportPanel 同构,done 阶段复用同一套
//     结果统计 UI(ok/failed/found + 结果列表 + 「再导一次」/「完成」按钮)
//
// 完成后 emit 'done' 通知父弹窗(OnboardingImportDialog)关闭,父视图刷新列表。
// 同 LocalImportPanel 一致,这里也 inject notifyImportDone / resetImportDoneSig 让
// 父组件在拿到响应那一刻就立即收到通知(不依赖用户点「完成」)。

import { ref, inject, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import {
  getOnboardingGlobalSkills,
  runOnboardingImportGlobalPaths,
} from '@/api/skillbox/onboarding'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const emit = defineEmits(['done'])

// 同 LocalImportPanel:provide 模式下,父组件立即收到通知;否则降级 emit('done')。
const notifyImportDone = inject('notifyImportDone', null)
const resetImportDoneSig = inject('resetImportDoneSig', null)

// phase: 'idle'(列表+搜索) | 'busy'(导入中) | 'done'(结果统计)
const phase = ref('idle')
const error = ref('')
const result = ref(null)

// 候选列表 + 选中态
const candidates = ref([])          // 原始列表
const rootPath = ref('')            // 实际扫描根
const rootExists = ref(false)
const selectedSet = ref(new Set())  // 选中的 source_path 集合
const searchKeyword = ref('')

// 过滤后的候选(本地搜索,name/version/description 任一命中)
const filteredCandidates = computed(() => {
  const kw = (searchKeyword.value || '').trim().toLowerCase()
  if (!kw) return candidates.value
  return candidates.value.filter((c) => {
    return (c.name || '').toLowerCase().includes(kw)
      || (c.version || '').toLowerCase().includes(kw)
      || (c.description || '').toLowerCase().includes(kw)
  })
})

// 选中的过滤后条目数量(底部按钮显示 + disabled 判定)
const selectedCount = computed(() => {
  let n = 0
  for (const it of filteredCandidates.value) {
    if (selectedSet.value.has(it.source_path)) n++
  }
  return n
})

// 总选中数(不受搜索过滤影响,给顶部 "已选 N/M" 用)
const totalSelectedCount = computed(() => selectedSet.value.size)

async function loadCandidates() {
  error.value = ''
  try {
    const data = await getOnboardingGlobalSkills()
    // 后端走标准信封 {code, msg, data},http 拦截器已剥离 data
    rootPath.value = data?.root || ''
    rootExists.value = !!data?.exists
    candidates.value = Array.isArray(data?.items) ? data.items : []
    // 重新加载时清掉旧的选中(避免目录扫描结果变了但选中态留着)
    selectedSet.value = new Set()
  } catch (e) {
    error.value = e?.message || String(e)
    toast.push({ type: 'error', message: t('onboarding.global.loadFailed', { msg: error.value }) })
  }
}

function toggleOne(c) {
  if (!c || !c.source_path) return
  const next = new Set(selectedSet.value)
  if (next.has(c.source_path)) {
    next.delete(c.source_path)
  } else {
    next.add(c.source_path)
  }
  selectedSet.value = next
}

function selectAllVisible() {
  const next = new Set(selectedSet.value)
  for (const c of filteredCandidates.value) {
    if (c?.source_path) next.add(c.source_path)
  }
  selectedSet.value = next
}

function clearVisible() {
  const next = new Set(selectedSet.value)
  for (const c of filteredCandidates.value) {
    if (c?.source_path) next.delete(c.source_path)
  }
  selectedSet.value = next
}

async function doImport() {
  if (phase.value === 'busy') return
  if (selectedCount.value === 0) return
  phase.value = 'busy'
  error.value = ''
  // 过滤顺序保留:用户先选的会先入 array(JS Set 保留插入顺序)
  const paths = Array.from(selectedSet.value)
  try {
    const r = await runOnboardingImportGlobalPaths(paths)
    onImportResult(r)
  } catch (e) {
    error.value = e?.message || String(e)
    phase.value = 'idle'
    toast.push({ type: 'error', message: t('onboarding.global.loadFailed', { msg: error.value }) })
  }
}

function onImportResult(r) {
  result.value = r
  phase.value = 'done'
  if (r?.ok > 0) {
    toast.push({
      type: 'success',
      message: t('onboarding.global.importOk', { ok: r.ok, failed: r.failed || 0 }),
    })
    if (notifyImportDone) {
      notifyImportDone(r)
    } else if (emit) {
      emit('done', r)
    }
  }
}

function reset() {
  phase.value = 'idle'
  result.value = null
  error.value = ''
  if (resetImportDoneSig) resetImportDoneSig()
  // 重新加载一次候选列表(导入过的 source_path 仍在,但后端 Save 内部会覆盖)
  loadCandidates()
}

function finish() {
  emit('done', result.value)
}

onMounted(() => {
  loadCandidates()
})
</script>

<template>
  <div class="gip">
    <!-- 阶段 1: 候选列表 + 搜索 + 多选 -->
    <section v-if="phase === 'idle'" class="gip-pane">
      <p class="gip-desc">{{ t('onboarding.global.desc') }}</p>

      <!-- 顶部信息条:扫描根 + 重新扫描按钮 -->
      <div class="gip-topbar">
        <div class="gip-root">
          <IconPark icon="mdi:folder-outline" width="13" height="13" />
          <span class="gip-root-label">{{ t('onboarding.global.rootLabel') }}:</span>
          <code class="gip-root-path" :title="rootPath">{{ rootPath || '—' }}</code>
          <span v-if="!rootExists" class="gip-root-tag gip-root-tag-missing">
            {{ t('onboarding.global.rootMissing') }}
          </span>
        </div>
        <button
          class="ghost gip-btn-rescan"
          :title="t('onboarding.global.btnRescan')"
          @click="loadCandidates"
        >
          <IconPark icon="mdi:refresh" width="13" height="13" />
          {{ t('onboarding.global.btnRescan') }}
        </button>
      </div>

      <!-- 根目录不存在时,给清晰提示(而不是空列表困惑) -->
      <div v-if="!rootExists" class="gip-missing">
        <IconPark icon="mdi:folder-alert-outline" width="20" height="20" />
        <p>{{ t('onboarding.global.rootMissingHint') }}</p>
      </div>

      <template v-else>
        <!-- 搜索框 + 全选/清空 -->
        <div class="gip-toolbar">
          <div class="gip-search">
            <IconPark icon="mdi:magnify" width="13" height="13" class="gip-search-icon" />
            <input
              v-model="searchKeyword"
              type="text"
              :placeholder="t('onboarding.global.searchPlaceholder')"
              class="gip-search-input"
              spellcheck="false"
            />
          </div>
          <div class="gip-bulk">
            <span class="gip-selected-tag">
              {{ t('onboarding.global.selected', { sel: totalSelectedCount, total: candidates.length }) }}
            </span>
            <button class="ghost gip-bulk-btn" @click="selectAllVisible">
              <IconPark icon="mdi:check-all" width="12" height="12" />
              {{ t('onboarding.global.selectAll') }}
            </button>
            <button class="ghost gip-bulk-btn" @click="clearVisible">
              <IconPark icon="mdi:close-circle-outline" width="12" height="12" />
              {{ t('onboarding.global.selectNone') }}
            </button>
          </div>
        </div>

        <!-- 候选列表 -->
        <ul v-if="filteredCandidates.length" class="gip-list">
          <li
            v-for="c in filteredCandidates"
            :key="c.source_path"
            :class="['gip-row', { 'gip-row-checked': selectedSet.has(c.source_path) }]"
            @click="toggleOne(c)"
          >
            <input
              type="checkbox"
              :checked="selectedSet.has(c.source_path)"
              class="gip-checkbox"
              :aria-label="c.name"
              @click.stop="toggleOne(c)"
            />
            <div class="gip-row-main">
              <div class="gip-row-head">
                <span class="gip-row-name">{{ c.name }}</span>
                <code v-if="c.version" class="gip-row-ver">v{{ c.version }}</code>
              </div>
              <p v-if="c.description" class="gip-row-desc">{{ c.description }}</p>
              <p class="gip-row-path">
                <IconPark icon="mdi:folder-outline" width="11" height="11" />
                <code>{{ c.source_path }}</code>
              </p>
            </div>
          </li>
        </ul>

        <!-- 空态:有目录但没 skill -->
        <div v-else-if="!error" class="gip-empty">
          <IconPark icon="mdi:package-variant-closed-remove" width="32" height="32" />
          <p class="gip-empty-title">{{ t('onboarding.global.empty') }}</p>
          <p class="gip-empty-hint">{{ t('onboarding.global.emptyHint') }}</p>
        </div>

        <p v-if="error" class="gip-error">
          <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ error }}
        </p>
      </template>

      <!-- 底部固定操作栏 -->
      <div v-if="rootExists" class="gip-footer">
        <span class="gip-footer-hint">
          {{ t('onboarding.global.selected', { sel: totalSelectedCount, total: candidates.length }) }}
        </span>
        <button
          class="primary"
          :disabled="selectedCount === 0 || phase === 'busy'"
          :title="t('onboarding.global.btnImportTitle')"
          @click="doImport"
        >
          <IconPark icon="mdi:tray-arrow-down" width="14" height="14" />
          {{ t('onboarding.global.btnImport', { n: selectedCount }) }}
        </button>
      </div>
    </section>

    <!-- 阶段 2: 导入中 -->
    <section v-else-if="phase === 'busy'" class="gip-pane gip-busy">
      <span class="spinner spinner-lg"></span>
      <p>{{ t('onboarding.global.importing') }}</p>
    </section>

    <!-- 阶段 3: 结果统计 — 复用 LocalImportPanel 的 done 阶段布局(对齐 UI 风格) -->
    <section v-else-if="phase === 'done' && result" class="gip-pane">
      <header class="gip-result-head">
        <IconPark icon="mdi:check-circle" width="18" height="18" />
        <h3>{{ t('onboarding.global.title') }}</h3>
      </header>

      <div class="gip-stats">
        <div class="gip-stat stat-ok">
          <span class="gip-stat-num">{{ result.ok || 0 }}</span>
          <span class="gip-stat-label">{{ t('onboarding.local.statOk') }}</span>
        </div>
        <div class="gip-stat stat-err">
          <span class="gip-stat-num">{{ result.failed || 0 }}</span>
          <span class="gip-stat-label">{{ t('onboarding.local.statErr') }}</span>
        </div>
        <div class="gip-stat">
          <span class="gip-stat-num">{{ result.found || 0 }}</span>
          <span class="gip-stat-label">{{ t('onboarding.local.statFound') }}</span>
        </div>
      </div>

      <ul v-if="result.results?.length" class="gip-result-list">
        <li
          v-for="(r, i) in result.results"
          :key="i"
          :class="r.ok ? 'gip-row-ok' : 'gip-row-err'"
        >
          <IconPark
            :icon="r.ok ? 'mdi:check' : 'mdi:close-circle-outline'"
            width="14"
            height="14"
            class="gip-row-icon"
          />
          <span class="gip-row-name"><code>{{ r.name }}</code></span>
          <span v-if="r.version" class="gip-row-ver">v{{ r.version }}</span>
          <span v-if="!r.ok && r.error" class="gip-row-msg">{{ r.error }}</span>
        </li>
      </ul>

      <div class="gip-footer gip-footer-result">
        <button class="ghost" @click="reset">
          <IconPark icon="mdi:refresh" width="14" height="14" />
          {{ t('onboarding.local.btnAgain') }}
        </button>
        <button class="primary" @click="finish">
          <IconPark icon="mdi:check" width="14" height="14" />
          {{ t('onboarding.local.btnDone') }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.gip {
  max-width: 760px;
  margin: 0 auto;
  padding: 4px 0;
  color: var(--text);
}

.gip-pane {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 22px 22px 14px;
  box-shadow: var(--shadow-card);
}

.gip-desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.6;
}

.gip-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 12px;
  font-size: 12px;
}

.gip-root {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
  color: var(--text-dim);
}

.gip-root-label {
  flex-shrink: 0;
}

.gip-root-path {
  color: var(--text);
  font-family: 'JetBrains Mono', monospace;
  font-size: 11px;
  background: transparent;
  padding: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 380px;
}

.gip-root-tag {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 500;
}

.gip-root-tag-missing {
  background: var(--accent-rose-bg);
  color: var(--accent-rose);
}

.gip-btn-rescan {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
}

.gip-missing {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 14px;
  margin: 4px 0 12px;
  background: var(--accent-rose-bg);
  border: 1px solid var(--accent-rose-border);
  border-radius: var(--radius-sm);
  color: var(--accent-rose);
  font-size: 12px;
}

.gip-missing p {
  margin: 0;
  color: var(--text);
}

/* 搜索 + 批量操作行 */
.gip-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}

.gip-search {
  position: relative;
  flex: 1;
  min-width: 0;
}

.gip-search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-faint);
  pointer-events: none;
}

.gip-search-input {
  width: 100%;
  height: 32px;
  padding: 0 10px 0 30px;
  font-size: 13px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
}

.gip-bulk {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.gip-selected-tag {
  font-size: 12px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
}

.gip-bulk-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
}

/* 候选列表 */
.gip-list {
  list-style: none;
  padding: 0;
  margin: 0 0 12px;
  max-height: 380px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}

.gip-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border);
  transition: background-color 0.12s ease;
}

.gip-row:last-child {
  border-bottom: none;
}

.gip-row:hover {
  background: var(--bg-hover);
}

.gip-row-checked {
  background: var(--accent-blue-bg);
}

.gip-row-checked:hover {
  background: var(--accent-blue-bg);
}

.gip-checkbox {
  flex-shrink: 0;
  margin-top: 3px;
  cursor: pointer;
  width: 14px;
  height: 14px;
}

.gip-row-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.gip-row-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.gip-row-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gip-row-ver {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
}

.gip-row-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.gip-row-path {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gip-row-path code {
  background: transparent;
  padding: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 空态 + 错误 */
.gip-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 20px;
  color: var(--text-faint);
  text-align: center;
}

.gip-empty-title {
  margin: 8px 0 4px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
}

.gip-empty-hint {
  margin: 0;
  font-size: 12px;
}

.gip-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 12px;
  padding: 8px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
}

/* 阶段 2: 导入中 */
.gip-busy {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 60px 24px;
  color: var(--text-dim);
  font-size: 14px;
}

.spinner-lg {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 底部操作栏 */
.gip-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.gip-footer-hint {
  font-size: 12px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
}

.gip-footer-result {
  margin-top: 14px;
}

/* 阶段 3: 结果统计 — 与 LocalImportPanel .lip-* 字段同名同款样式 */
.gip-result-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}

.gip-result-head h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.gip-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 14px;
}

.gip-stat {
  padding: 14px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  text-align: center;
}

.gip-stat-num {
  display: block;
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
  margin-bottom: 4px;
}

.gip-stat-label {
  font-size: 12px;
  color: var(--text-dim);
}

.stat-ok {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
}
.stat-ok .gip-stat-num { color: var(--accent-emerald); }

.stat-err {
  background: var(--accent-rose-bg);
  border-color: var(--accent-rose-border);
}
.stat-err .gip-stat-num { color: var(--accent-rose); }

.gip-result-list {
  list-style: none;
  padding: 0;
  margin: 0 0 14px;
  max-height: 280px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.gip-result-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  font-size: 13px;
  border-bottom: 1px solid var(--border);
}

.gip-result-list li:last-child {
  border-bottom: none;
}

.gip-row-ok {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
}

.gip-row-err {
  background: var(--accent-rose-bg);
  color: var(--accent-rose);
}

.gip-row-icon {
  flex-shrink: 0;
}

.gip-row-msg {
  font-size: 12px;
  color: inherit;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60%;
}

@media (max-width: 600px) {
  .gip-toolbar { flex-direction: column; align-items: stretch; }
  .gip-bulk { justify-content: flex-end; }
  .gip-stats { grid-template-columns: 1fr; }
}
</style>