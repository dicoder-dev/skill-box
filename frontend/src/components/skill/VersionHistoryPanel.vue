<script setup>
// VersionHistoryPanel - 技能仓库 commit 历史 inline 面板(2026-07-17 增)
//
// 2026-07-17 改:从 Modal 改成 inline panel — 跟作用域/Git 同步面板同款
// 折叠交互,通过 AccordionGroup 协调互斥展开。
// 嵌入 SkillScopePanel 内部,位置:作用域下方、Git 同步上方。
//
// Props:
//   - skillPath: 非空时只显示该路径的 commit(per-skill 历史)
//
// 跟 VersionHistoryModal 行为差异:无弹窗,固定位置,AccordionGroup 互斥。

import { ref, computed, watch, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getGitLog,
  getGitDiff,
  checkoutGit,
  pushGit,
  pullGit,
  discardGit,
  getGitStatus,
} from '@/api/skillbox/git.js'
import IconPark from '@/components/IconPark.vue'
import CollapsiblePanel from '@/components/CollapsiblePanel.vue'

const props = defineProps({
  // 2026-07-17 增:可选 skillPath(相对 repo root,例如 "frontend/code-review"),
  // 非空时只显示涉及该路径的 commit(per-skill 修改历史)。
  skillPath: { type: String, default: '' },
})
const emit = defineEmits(['checked-out'])

const { t } = useI18n()

// 2026-07-17:AccordionGroup 协调器 — 跟 Git 同步面板互斥展开。
const coordinator = inject('cpCoordinator', null)
const localExpanded = ref(false)
const isExpanded = computed(() => {
  if (coordinator) return coordinator.activeId === 'history'
  return localExpanded.value
})
function onHistoryToggle(open) {
  if (coordinator) {
    coordinator.toggle('history', open)
  } else {
    localExpanded.value = open
  }
  if (open) loadAll()
}

const loading = ref(false)
const errorMsg = ref('')
const items = ref([])
const status = ref(null)

const selectedHash = ref('')
const diffFrom = ref('')
const diffTo = ref('HEAD')
const diffText = ref('')
const diffLoading = ref(false)

const selected = computed(() => items.value.find((it) => it.hash === selectedHash.value) || null)

// watch skillPath 变化(切 skill 时)重新拉
watch(() => props.skillPath, () => {
  if (isExpanded.value) loadAll()
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [log, st] = await Promise.all([
      getGitLog(50, props.skillPath || undefined),
      getGitStatus(),
    ])
    items.value = log.items || []
    status.value = st
    if (!selectedHash.value && items.value.length) {
      selectedHash.value = items.value[0].hash
    }
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function loadDiff() {
  if (!diffFrom.value) {
    diffText.value = ''
    return
  }
  diffLoading.value = true
  try {
    const r = await getGitDiff(diffFrom.value, diffTo.value || 'HEAD')
    diffText.value = r.diff || ''
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
    diffText.value = ''
  } finally {
    diffLoading.value = false
  }
}

async function doCheckout() {
  if (!selectedHash.value) return
  if (!confirm(t('git.checkoutConfirm', { hash: selectedHash.value.slice(0, 7) }))) {
    return
  }
  loading.value = true
  try {
    await checkoutGit(selectedHash.value)
    errorMsg.value = ''
    emit('checked-out', selectedHash.value)
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doPush() {
  loading.value = true
  try {
    await pushGit()
    await loadAll()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doPull() {
  loading.value = true
  try {
    await pullGit()
    await loadAll()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doDiscard() {
  if (!confirm(t('git.discardConfirm'))) return
  loading.value = true
  try {
    await discardGit()
    await loadAll()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

function pickCommit(h) {
  selectedHash.value = h
  diffFrom.value = h
  loadDiff()
}

function shortHash(h) {
  return (h || '').slice(0, 7)
}
</script>

<template>
  <CollapsiblePanel
    :expanded="isExpanded"
    :title="t('git.history.title')"
    icon="history"
    panel-id="history"
    @update:expanded="onHistoryToggle"
  >
    <template #title-meta>
      <span v-if="status && status.initialized" class="vhp-badge ok">
        {{ status.head_short || t('git.noCommits') }}
      </span>
      <span v-else-if="status" class="vhp-badge warn">{{ t('git.notInit') }}</span>
      <!-- 2026-07-17 增:per-skill 模式标识 — 让用户知道当前看的是
           哪个 skill 的历史,空 = 全仓 -->
      <span v-if="skillPath" class="vhp-skill-path" :title="skillPath">
        {{ skillPath }}
      </span>
    </template>

    <div v-if="errorMsg" class="vhp-error">
      <IconPark type="warning" :size="12" />
      <span>{{ errorMsg }}</span>
    </div>

    <div v-if="!status || !status.initialized" class="vhp-empty">
      <IconPark type="github" :size="32" />
      <p>{{ t('git.history.initFirst') }}</p>
    </div>

    <div v-else class="vhp-body">
      <div class="vhp-list">
        <div v-if="loading && !items.length" class="vhp-loading">
          {{ t('common.loading') }}
        </div>
        <ul v-else-if="items.length" class="vhp-items">
          <li
            v-for="it in items"
            :key="it.hash"
            :class="['vhp-item', { active: it.hash === selectedHash }]"
            @click="pickCommit(it.hash)"
          >
            <div class="vhp-item-row1">
              <code class="vhp-hash">{{ shortHash(it.hash) }}</code>
              <span class="vhp-msg">{{ it.message }}</span>
            </div>
            <div class="vhp-item-row2">
              <span class="vhp-author">{{ it.author }}</span>
              <span class="vhp-when">{{ (it.when || '').slice(0, 19) }}</span>
            </div>
          </li>
        </ul>
        <div v-else class="vhp-empty">
          <p>{{ skillPath ? t('git.history.emptySkill') : t('git.history.empty') }}</p>
        </div>
      </div>

      <div class="vhp-diff">
        <div class="vhp-diff-header">
          <span class="vhp-diff-title">{{ t('git.history.diff') }}</span>
          <span class="vhp-diff-range">{{ shortHash(diffFrom) }} → {{ shortHash(diffTo) }}</span>
        </div>
        <div v-if="diffLoading" class="vhp-diff-loading">{{ t('common.loading') }}</div>
        <pre v-else-if="diffText" class="vhp-diff-pre">{{ diffText }}</pre>
        <div v-else class="vhp-diff-empty">
          {{ t('git.history.pickCommit') }}
        </div>
      </div>

      <div class="vhp-actions">
        <button class="vhp-btn" :disabled="loading || !selected" @click="doCheckout">
          <IconPark type="undo" :size="12" />
          {{ t('git.history.checkout') }}
        </button>
        <button v-if="status.remote_url" class="vhp-btn" :disabled="loading" @click="doPush">
          <IconPark type="upload" :size="12" />
          {{ t('git.history.push') }}
        </button>
        <button v-if="status.remote_url" class="vhp-btn" :disabled="loading" @click="doPull">
          <IconPark type="download" :size="12" />
          {{ t('git.history.pull') }}
        </button>
        <button v-if="!status.working_clean" class="vhp-btn warn" :disabled="loading" @click="doDiscard">
          <IconPark type="undo" :size="12" />
          {{ t('git.discard') }}
        </button>
      </div>
    </div>
  </CollapsiblePanel>
</template>

<style scoped>
.vhp-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-family: var(--font-mono, monospace);
  font-weight: 500;
}
.vhp-badge.ok { background: rgba(34, 197, 94, 0.15); color: rgb(34, 197, 94); }
.vhp-badge.warn { background: rgba(245, 158, 11, 0.15); color: rgb(245, 158, 11); }

.vhp-skill-path {
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  background: rgba(127, 127, 127, 0.06);
  padding: 1px 6px;
  border-radius: 3px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.vhp-error {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(239, 68, 68, 0.1);
  border-left: 2px solid rgb(239, 68, 68);
  padding: 4px 6px;
  border-radius: 3px;
  font-size: 11px;
  color: rgb(239, 68, 68);
}

.vhp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 24px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  font-size: 12px;
  text-align: center;
}
.vhp-loading {
  text-align: center;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  padding: 12px;
  font-size: 12px;
}

.vhp-body {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(220px, 1.4fr);
  gap: 6px;
}

.vhp-list {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: 4px;
  overflow: auto;
  max-height: 320px;
  min-width: 0;
}
.vhp-items {
  list-style: none;
  margin: 0;
  padding: 0;
}
.vhp-item {
  padding: 4px 6px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  cursor: pointer;
  font-size: 11px;
}
.vhp-item:hover { background: rgba(127, 127, 127, 0.05); }
.vhp-item.active { background: rgba(59, 130, 246, 0.1); }
.vhp-item-row1 { display: flex; gap: 4px; align-items: baseline; }
.vhp-item-row2 {
  display: flex;
  justify-content: space-between;
  font-size: 9px;
  color: var(--text-faint, rgba(127, 127, 127, 0.5));
  margin-top: 1px;
}
.vhp-hash { font-family: var(--font-mono, monospace); color: var(--text-muted, rgba(127, 127, 127, 0.7)); }
.vhp-msg {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.vhp-diff {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.vhp-diff-header {
  display: flex;
  justify-content: space-between;
  padding: 4px 6px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  font-size: 11px;
  font-weight: 600;
}
.vhp-diff-title { font-weight: 600; }
.vhp-diff-range {
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.vhp-diff-loading, .vhp-diff-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  padding: 16px;
  text-align: center;
}
.vhp-diff-pre {
  flex: 1;
  margin: 0;
  padding: 6px 8px;
  overflow: auto;
  max-height: 280px;
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  line-height: 1.4;
  background: var(--bg-elevated, rgba(127, 127, 127, 0.03));
  white-space: pre;
}

.vhp-actions {
  grid-column: 1 / -1;
  display: flex;
  gap: 4px;
  justify-content: flex-end;
  padding-top: 4px;
  border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  margin-top: 2px;
}
.vhp-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 8px;
  font-size: 11px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  background: transparent;
  color: var(--text-primary, currentColor);
  border-radius: 3px;
  cursor: pointer;
}
.vhp-btn:hover:not(:disabled) { background: rgba(127, 127, 127, 0.08); }
.vhp-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.vhp-btn.warn {
  border-color: rgb(245, 158, 11);
  color: rgb(245, 158, 11);
}
</style>