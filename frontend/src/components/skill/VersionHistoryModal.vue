<script setup>
// VersionHistoryModal - 技能仓库 commit 历史弹窗(2026-07-17 增)
//
// 替代旧的 tag 弹窗,展示 go-git 仓库的 commit 时间线 + diff + checkout + push 状态。
// 嵌入 SkillsView 里,通过 props.open / props.skill 控制。
//
// 2026-07-17 设计决策:
//   - 只展示全局 git log(单仓),不做 per-skill 过滤(简单 — git log 加 --path
//     过滤在 go-git 上要走 commitFiles,慢)
//   - 用户在弹窗里可以:看 commit 列表 / 看 diff / reset 到某 commit / push 重试 / discard
//   - 选中 commit 后点击「Reset 到此」→ 二次确认 → 调 checkoutGit
//   - 错误信息顶部统一展示,操作级错误带行内 icon

import { ref, computed, watch } from 'vue'
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
import Modal from '@/components/Modal.vue'

const { t } = useI18n()

const props = defineProps({
  open: { type: Boolean, default: false },
  skill: { type: Object, default: () => ({}) },
  // 2026-07-17 增:可选 skillPath(相对 repo root,例如 "frontend/code-review"),
  // 非空时只显示涉及该路径的 commit(per-skill 修改历史)。
  // 不传则跟旧版一致,显示全仓 commit 历史。
  skillPath: { type: String, default: '' },
})
const emit = defineEmits(['update:open', 'checked-out'])

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

watch(() => props.open, (v) => {
  if (v) loadAll()
}, { immediate: true })

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    // 2026-07-17 改:per-skill 过滤 — props.skillPath 非空时传给 getGitLog 第二参,
    // 后端只返回涉及该路径的 commit。
    const [log, st] = await Promise.all([
      getGitLog(50, props.skillPath || undefined),
      getGitStatus(),
    ])
    items.value = log.items || []
    status.value = st
    // 默认选中 HEAD(第一个)
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
    // 关闭弹窗
    emit('update:open', false)
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

// 2026-07-17 增:Pull 按钮 — 拉取远端改动;工作区有未提交改动返 409。
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

function close() {
  emit('update:open', false)
  errorMsg.value = ''
}
</script>

<template>
  <Modal
    :model-value="open"
    size="xl"
    :title="props.skillPath ? `${t('git.history.title')} — ${props.skillPath}` : t('git.history.title')"
    @update:model-value="emit('update:open', $event)"
  >
    <div class="version-modal">
      <!-- 顶部状态条 -->
      <div v-if="status" class="version-status">
        <span v-if="status.initialized" class="status-pill ok">
          <IconPark type="check-one" :size="10" />
          {{ status.branch }} · {{ status.head_short }}
        </span>
        <span v-else class="status-pill warn">
          {{ t('git.notInit') }}
        </span>
        <span v-if="status.pending_push > 0" class="status-pill warn">
          <IconPark type="upload" :size="10" /> {{ status.pending_push }} pending
        </span>
        <span v-if="!status.working_clean" class="status-pill warn">
          <IconPark type="edit" :size="10" /> {{ t('git.dirty') }}
        </span>
      </div>

      <!-- 错误条 -->
      <div v-if="errorMsg" class="version-error">
        <IconPark type="warning" :size="12" />
        <span>{{ errorMsg }}</span>
      </div>

      <div v-if="!status || !status.initialized" class="version-empty">
        <IconPark type="github" :size="36" />
        <p>{{ t('git.history.initFirst') }}</p>
      </div>

      <div v-else class="version-body">
        <!-- 左:commit 列表 -->
        <div class="version-list">
          <div v-if="loading && !items.length" class="version-loading">
            {{ t('common.loading') }}
          </div>
          <ul v-else-if="items.length" class="version-items">
            <li
              v-for="it in items"
              :key="it.hash"
              :class="['version-item', { active: it.hash === selectedHash }]"
              @click="pickCommit(it.hash)"
            >
              <div class="version-item-row1">
                <code class="version-hash">{{ shortHash(it.hash) }}</code>
                <span class="version-msg">{{ it.message }}</span>
              </div>
              <div class="version-item-row2">
                <span class="version-author">{{ it.author }}</span>
                <span class="version-when">{{ (it.when || '').slice(0, 19) }}</span>
              </div>
            </li>
          </ul>
          <div v-else class="version-empty">
            <p>{{ props.skillPath ? t('git.history.emptySkill', '该技能暂无修改记录') : t('git.history.empty') }}</p>
          </div>
        </div>

        <!-- 右:diff 视图 -->
        <div class="version-diff">
          <div class="diff-header">
            <span class="diff-title">{{ t('git.history.diff') }}</span>
            <span class="diff-range">{{ shortHash(diffFrom) }} → {{ shortHash(diffTo) }}</span>
          </div>
          <div v-if="diffLoading" class="diff-loading">{{ t('common.loading') }}</div>
          <pre v-else-if="diffText" class="diff-pre">{{ diffText }}</pre>
          <div v-else class="diff-empty">
            {{ t('git.history.pickCommit') }}
          </div>
        </div>
      </div>

      <!-- 底部操作 -->
      <div v-if="status && status.initialized" class="version-actions">
        <button class="version-btn" :disabled="loading || !selected" @click="doCheckout">
          <IconPark type="undo" :size="12" />
          {{ t('git.history.checkout') }}
        </button>
        <button v-if="status.remote_url" class="version-btn" :disabled="loading" @click="doPush">
          <IconPark type="upload" :size="12" />
          {{ t('git.history.push') }}
        </button>
        <!-- 2026-07-17 增:Pull 按钮,跟 Push 平行;有 remote_url 才显示,
             工作区有未提交改动时仍可点(后端会返 409 + 友好错误)。 -->
        <button v-if="status.remote_url" class="version-btn" :disabled="loading" @click="doPull">
          <IconPark type="download" :size="12" />
          {{ t('git.history.pull') }}
        </button>
        <button v-if="!status.working_clean" class="version-btn warn" :disabled="loading" @click="doDiscard">
          <IconPark type="close" :size="12" />
          {{ t('git.discard') }}
        </button>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.version-modal {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 360px;
}

.version-status {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-family: var(--font-mono, monospace);
}
.status-pill.ok {
  background: rgba(34, 197, 94, 0.15);
  color: rgb(34, 197, 94);
}
.status-pill.warn {
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.version-error {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(239, 68, 68, 0.1);
  border-left: 2px solid rgb(239, 68, 68);
  padding: 4px 8px;
  border-radius: 3px;
  font-size: 12px;
  color: rgb(239, 68, 68);
}

.version-body {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(280px, 1.5fr);
  gap: 8px;
  min-height: 360px;
}

.version-list {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: 6px;
  overflow: auto;
  max-height: 480px;
}
.version-items {
  list-style: none;
  margin: 0;
  padding: 0;
}
.version-item {
  padding: 6px 8px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  cursor: pointer;
  font-size: 12px;
}
.version-item:hover { background: rgba(127, 127, 127, 0.05); }
.version-item.active { background: rgba(59, 130, 246, 0.1); }
.version-item-row1 {
  display: flex;
  gap: 6px;
  align-items: baseline;
}
.version-hash {
  font-family: var(--font-mono, monospace);
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  flex: 0 0 auto;
}
.version-msg {
  flex: 1 1 auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.version-item-row2 {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--text-faint, rgba(127, 127, 127, 0.5));
  margin-top: 2px;
}

.version-diff {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.diff-header {
  display: flex;
  justify-content: space-between;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  font-size: 12px;
  font-weight: 600;
}
.diff-range {
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.diff-pre {
  flex: 1 1 auto;
  margin: 0;
  padding: 8px 10px;
  overflow: auto;
  max-height: 440px;
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  line-height: 1.5;
  background: var(--bg-elevated, rgba(127, 127, 127, 0.03));
  white-space: pre;
}
.diff-empty,
.diff-loading,
.version-empty,
.version-loading {
  flex: 1 1 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  font-size: 12px;
  padding: 32px;
  text-align: center;
}
.version-empty { min-height: 200px; }

.version-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  padding-top: 4px;
  border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
}
.version-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 12px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  background: transparent;
  color: var(--text-primary, currentColor);
  border-radius: 4px;
  cursor: pointer;
}
.version-btn:hover:not(:disabled) { background: rgba(127, 127, 127, 0.08); }
.version-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.version-btn.warn {
  border-color: rgb(245, 158, 11);
  color: rgb(245, 158, 11);
}
</style>