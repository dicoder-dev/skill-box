<script setup>
// VersionHistoryPanel - VSCode 风格 commit 列表 + 抽屉 diff(2026-07-17 重构)
//
// 2026-07-17 大改:从原 Grid 左右分栏改成:
//   - 上方单列 commit 列表(紧凑,左圆点 + 连线 + scope 浅蓝 + 描述白字)
//   - 点 commit 后底部抽屉(drawer)显示文件列表 + diff 预览
//
// 这样长 commit message / 短 hash / 时间都能在一行展示,跟 VSCode Source
// Control 视图一致;选中态高亮,文件列表 hover 整行变浅灰。

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

// 2026-07-17 改:status 默认空对象(原 null)— 模板里有 status.remote_url
// / status.working_clean 等访问没全部加 guard,status=null 时访问属性
// 报 'object not found'(Vue 内部包装的 TypeError)。默认 {} 让所有
// 属性访问返 undefined,模板 v-if 会正确隐藏。
const status = ref({
  initialized: false,
  branch: '',
  remote_url: '',
  remote_branch: '',
  head_hash: '',
  head_short: '',
  head_message: '',
  working_clean: true,
  ahead: 0,
  behind: 0,
  has_token: false,
  pending_push: 0,
  last_push_error: '',
})

// 2026-07-17 改:展开的文件列表 — 跟每个 commit 独立,key 是 commit hash
// (不是 index,避免切换 commit 时复用错状态)。
const expandedCommits = ref(new Set())
// 抽屉显示的 commit hash(底部抽屉只显示一个 diff)
const drawerHash = ref('')
const drawerFile = ref('') // 单文件过滤(diff 只显示这一文件)
const diffText = ref('')
const diffLoading = ref(false)

// 2026-07-17:解析 conventional commit 头,提取 type(scope) + 描述 —
// 跟 VSCode 显示一致。"fix(scope): 描述" → { type: 'fix', scope: 'scope', rest: '描述' }
function parseCommitTitle(msg) {
  const firstLine = (msg || '').split('\n', 1)[0] || ''
  // 匹配 type(scope)?: ...
  const m = firstLine.match(/^([a-zA-Z]+)(\(([^)]+)\))?:\s*(.*)$/)
  if (m) {
    return {
      type: m[1],
      scope: m[3] || '',
      desc: (m[4] || '').trim(),
      full: firstLine,
    }
  }
  return { type: '', scope: '', desc: firstLine, full: firstLine }
}

// 推断单文件状态(M/A/D) — 走文件名启发式 + diff 文本头(后续 diff API 可
// 扩展 status 字段,这里先用 patch 第一行 "@@ +++ b/<file>" 之前的 +/-)。
function fileStatusHint(filePath, diffText) {
  if (!diffText) return 'M'
  // 简化版:统计 +/- 行数
  const lines = diffText.split('\n')
  let added = 0
  let removed = 0
  for (const line of lines) {
    if (line.startsWith('+') && !line.startsWith('+++')) added++
    else if (line.startsWith('-') && !line.startsWith('---')) removed++
  }
  if (added > 0 && removed === 0) return 'A'
  if (removed > 0 && added === 0) return 'D'
  return 'M'
}

// 文件路径拆成目录 + 文件名,渲染时目录暗灰 / 文件名亮。
function splitDirAndFile(filePath) {
  const idx = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'))
  if (idx < 0) return { dir: '', name: filePath }
  return { dir: filePath.slice(0, idx + 1), name: filePath.slice(idx + 1) }
}

// 2026-07-17 改:同时 watch isExpanded 和 skillPath,都强制刷新。
watch(() => props.skillPath, () => {
  // 切 skill 时清掉展开 / 抽屉状态,避免上一 skill 的状态泄露。
  expandedCommits.value = new Set()
  drawerHash.value = ''
  drawerFile.value = ''
  diffText.value = ''
  loadAll()
})
watch(isExpanded, (open) => {
  if (open) loadAll()
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [log, st] = await Promise.all([
      getGitLog(50, props.skillPath || undefined),
      getGitStatus(),
    ])
    items.value = (log.items || []).map((it) => ({
      ...it,
      _title: parseCommitTitle(it.message),
    }))
    status.value = st
    if (drawerHash.value) {
      // 重新拉后保留抽屉选中,重新拉 diff
      const exists = items.value.some((it) => it.hash === drawerHash.value)
      if (!exists) {
        drawerHash.value = ''
        drawerFile.value = ''
        diffText.value = ''
      }
    }
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

function toggleCommitFiles(hash) {
  const set = new Set(expandedCommits.value)
  if (set.has(hash)) {
    set.delete(hash)
  } else {
    set.add(hash)
    // 选中并打开抽屉显示该 commit 的文件列表(抽屉默认不开,等点文件再开)
    drawerHash.value = hash
    drawerFile.value = ''
    diffText.value = ''
  }
  expandedCommits.value = set
}

async function openFileDiff(commitHash, filePath) {
  drawerHash.value = commitHash
  drawerFile.value = filePath
  diffLoading.value = true
  diffText.value = ''
  try {
    // 2026-07-17:用 commit 的第一个 parent hash(若 root commit 则用空 tree)
    // 作为 from,这样能拿到 commit 引入的全部变更。getGitDiff
    // 内部走 git rev-parse,后端 resolveCommit 会把空 tree / 零
    // hash 退化成 Tree().Patch 兜底,不会卡死。
    const r = await getGitDiff(commitHash + '^', commitHash)
    // 过滤只保留该文件的 diff 块(简化:按文件路径分行)
    if (filePath) {
      diffText.value = filterDiffByFile(r.diff || '', filePath)
    } else {
      diffText.value = r.diff || ''
    }
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
    diffText.value = ''
  } finally {
    diffLoading.value = false
  }
}

function closeDrawer() {
  drawerHash.value = ''
  drawerFile.value = ''
  diffText.value = ''
}

// 从完整 diff 里抽出指定文件的块(diff 格式以 "diff --git a/x b/x" 分段)
//
// 2026-07-17 改:用 startsWith + 包含 " b/<path>" 判断,不再死磕
// "a/<path> " 字符串。git diff 的标准格式:
//   diff --git a/<path> b/<path>
// 目标文件: 路径名出现在 diff --git 行内(完整出现,不被空白拆分)。
function filterDiffByFile(diff, filePath) {
  if (!diff || !filePath) return diff
  const lines = diff.split('\n')
  const out = []
  let inTarget = false
  for (const line of lines) {
    if (line.startsWith('diff --git ')) {
      // 判定是否目标文件:命中 a/<path> 或 b/<path> 子串
      // (git 在 rename 时会把 b/ 写成 b/旧名,所以也要匹)
      inTarget =
        line.includes(' a/' + filePath) ||
        line.includes(' b/' + filePath)
    }
    if (inTarget) out.push(line)
  }
  return out.join('\n')
}

const drawerCommit = computed(() =>
  items.value.find((it) => it.hash === drawerHash.value) || null,
)

async function doCheckout() {
  if (!drawerHash.value) return
  if (!confirm(t('git.checkoutConfirm', { hash: drawerHash.value.slice(0, 7) }))) {
    return
  }
  loading.value = true
  try {
    await checkoutGit(drawerHash.value)
    errorMsg.value = ''
    emit('checked-out', drawerHash.value)
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

function shortHash(h) {
  return (h || '').slice(0, 7)
}

function formatTime(when) {
  if (!when) return ''
  // 2026-07-17:简化显示 — 只取日期(YYYY-MM-DD),更紧凑;hover 时看完整时间。
  return when.slice(0, 10)
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
      <!-- per-skill 模式标识 — 让用户知道当前看的是哪个 skill 的历史 -->
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

    <div v-else class="vhp-shell">
      <!-- 顶部 commit 列表(单列紧凑,VSCode 风格) -->
      <div class="vhp-list">
        <div v-if="loading && !items.length" class="vhp-loading">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="items.length" class="vhp-commits">
          <div
            v-for="it in items"
            :key="it.hash"
            class="vhp-commit"
          >
            <!-- commit 行:圆点 + 连线 + scope + 描述 + 时间 -->
            <div
              :class="['vhp-commit-row', {
                active: it.hash === drawerHash,
              }]"
              @click="toggleCommitFiles(it.hash)"
            >
              <!-- 左侧圆点 + 连线 (VSCode 节点风格) -->
              <div class="vhp-node">
                <div class="vhp-node-line vhp-node-line-top" />
                <div class="vhp-node-dot" />
                <div class="vhp-node-line vhp-node-line-bot" />
              </div>
              <!-- commit 主体 -->
              <div class="vhp-commit-body">
                <div class="vhp-commit-msg">
                  <span v-if="it._title.type" class="vhp-commit-type">{{ it._title.type }}</span>
                  <span v-if="it._title.scope" class="vhp-commit-scope">({{ it._title.scope }})</span>
                  <span class="vhp-commit-sep">:</span>
                  <span class="vhp-commit-desc" :title="it._title.full">{{ it._title.desc || it._title.full }}</span>
                </div>
                <div class="vhp-commit-meta">
                  <code class="vhp-commit-hash">{{ shortHash(it.hash) }}</code>
                  <span class="vhp-commit-when" :title="it.when">{{ formatTime(it.when) }}</span>
                  <span class="vhp-commit-author">{{ it.author }}</span>
                </div>
              </div>
              <!-- 右侧展开箭头 -->
              <IconPark
                :type="expandedCommits.has(it.hash) ? 'down' : 'right'"
                :size="10"
                class="vhp-commit-arrow"
              />
            </div>

            <!-- 展开的文件列表 -->
            <div v-if="expandedCommits.has(it.hash)" class="vhp-files">
              <div v-if="!it.files || !it.files.length" class="vhp-files-empty">
                {{ t('git.history.noFiles') }}
              </div>
              <div
                v-for="f in (it.files || [])"
                :key="f"
                class="vhp-file-row"
                @click.stop="openFileDiff(it.hash, f)"
              >
                <IconPark type="right" :size="10" class="vhp-file-arrow" />
                <span class="vhp-file-status" :data-status="fileStatusHint(f, '')">·</span>
                <span class="vhp-file-path" :title="f">
                  <span v-if="splitDirAndFile(f).dir" class="vhp-file-dir">{{ splitDirAndFile(f).dir }}</span><span class="vhp-file-name">{{ splitDirAndFile(f).name }}</span>
                </span>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="vhp-empty">
          <p>{{ skillPath ? t('git.history.emptySkill') : t('git.history.empty') }}</p>
        </div>
      </div>

      <!-- 底部抽屉:选中 commit 的 diff 预览 -->
      <transition name="vhp-drawer">
        <div v-if="drawerHash" class="vhp-drawer">
          <div class="vhp-drawer-header">
            <IconPark type="code" :size="12" />
            <span class="vhp-drawer-title">
              {{ drawerFile || (drawerCommit && drawerCommit._title.full) || shortHash(drawerHash) }}
            </span>
            <span class="vhp-drawer-range">
              {{ shortHash(drawerHash + '^') }} → {{ shortHash(drawerHash) }}
            </span>
            <button class="vhp-drawer-close" :title="t('common.close', '关闭')" @click="closeDrawer">
              <IconPark type="close" :size="12" />
            </button>
          </div>
          <div class="vhp-drawer-body">
            <div v-if="diffLoading" class="vhp-drawer-loading">{{ t('common.loading') }}</div>
            <pre v-else-if="diffText" class="vhp-drawer-pre">{{ diffText }}</pre>
            <div v-else class="vhp-drawer-empty">
              {{ t('git.history.pickCommit') }}
            </div>
          </div>
          <div class="vhp-drawer-actions">
            <button class="vhp-btn" :disabled="loading" @click="doCheckout">
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
      </transition>
    </div>
  </CollapsiblePanel>
</template>

<style scoped>
/* 2026-07-17 大改:VSCode 风格重写整个面板 — 单列 commit 列表 + 抽屉 diff */

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

/* shell = 列表 + 抽屉,纵向 flex */
.vhp-shell {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* commit 列表区 */
.vhp-list {
  overflow: auto;
  max-height: 360px;
}
.vhp-commits { display: flex; flex-direction: column; }

.vhp-commit { display: flex; flex-direction: column; }

/* commit 行:左圆点 + 连线 + 主体 + 箭头 */
.vhp-commit-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px 4px 0;
  cursor: pointer;
  font-size: 12px;
  line-height: 1.4;
  transition: background 80ms;
}
.vhp-commit-row:hover { background: rgba(127, 127, 127, 0.05); }
.vhp-commit-row.active {
  background: rgba(59, 130, 246, 0.12);
}

/* 左侧节点(圆点 + 上下连线) */
.vhp-node {
  flex: 0 0 14px;
  display: flex;
  flex-direction: column;
  align-items: center;
  align-self: stretch;
  position: relative;
}
.vhp-node-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: rgb(59, 130, 246);
  margin-top: 8px;
  flex-shrink: 0;
  z-index: 1;
  box-shadow: 0 0 0 2px var(--bg-primary, transparent);
}
.vhp-commit-row.active .vhp-node-dot {
  background: rgb(96, 165, 250);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}
.vhp-node-line {
  width: 1px;
  flex: 1;
  background: rgba(127, 127, 127, 0.25);
}
.vhp-node-line-top { margin-bottom: -3.5px; }
.vhp-node-line-bot { margin-top: -3.5px; }

/* commit 主体 */
.vhp-commit-body {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.vhp-commit-msg {
  display: flex;
  align-items: baseline;
  gap: 0;
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  overflow: hidden;
}
.vhp-commit-type {
  color: #9cdcfe;
  flex-shrink: 0;
}
.vhp-commit-scope {
  color: #9cdcfe;
  flex-shrink: 0;
}
.vhp-commit-sep {
  color: rgba(127, 127, 127, 0.7);
  margin: 0 1px;
  flex-shrink: 0;
}
.vhp-commit-desc {
  color: var(--text-primary, currentColor);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}

.vhp-commit-meta {
  display: flex;
  gap: 8px;
  font-size: 10px;
  color: rgba(127, 127, 127, 0.6);
  font-family: var(--font-mono, monospace);
}
.vhp-commit-hash { color: rgba(127, 127, 127, 0.7); }
.vhp-commit-when { }
.vhp-commit-author { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.vhp-commit-arrow {
  flex-shrink: 0;
  color: rgba(127, 127, 127, 0.5);
  margin-right: 4px;
}

/* 展开的文件列表 */
.vhp-files {
  margin-left: 14px;
  border-left: 1px solid rgba(127, 127, 127, 0.15);
  padding: 2px 0 4px 6px;
  display: flex;
  flex-direction: column;
}
.vhp-files-empty {
  padding: 4px 8px;
  font-size: 10px;
  color: rgba(127, 127, 127, 0.5);
  font-style: italic;
}
.vhp-file-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  font-size: 11px;
  font-family: var(--font-mono, monospace);
  cursor: pointer;
  border-radius: 3px;
  transition: background 80ms;
}
.vhp-file-row:hover { background: rgba(127, 127, 127, 0.08); }
.vhp-file-arrow {
  flex-shrink: 0;
  color: rgba(127, 127, 127, 0.4);
}
.vhp-file-status {
  flex-shrink: 0;
  font-weight: 700;
  width: 10px;
  text-align: center;
}
.vhp-file-status[data-status="M"] { color: rgb(245, 158, 11); }
.vhp-file-status[data-status="A"] { color: rgb(34, 197, 94); }
.vhp-file-status[data-status="D"] { color: rgb(239, 68, 68); }
.vhp-file-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}
.vhp-file-dir { color: rgba(127, 127, 127, 0.55); }
.vhp-file-name { color: var(--text-primary, currentColor); }

/* 底部抽屉 */
.vhp-drawer-enter-active,
.vhp-drawer-leave-active {
  transition: transform 200ms ease, opacity 150ms ease;
  overflow: hidden;
}
.vhp-drawer-enter-from,
.vhp-drawer-leave-to {
  transform: translateY(20px);
  opacity: 0;
}

.vhp-drawer {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.2));
  background: var(--bg-elevated, rgba(127, 127, 127, 0.02));
  max-height: 320px;
  margin-top: 4px;
}
.vhp-drawer-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
  font-size: 11px;
  font-weight: 600;
}
.vhp-drawer-title {
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.vhp-drawer-range {
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  flex-shrink: 0;
}
.vhp-drawer-close {
  background: transparent;
  border: 0;
  padding: 2px;
  cursor: pointer;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  border-radius: 3px;
}
.vhp-drawer-close:hover {
  background: rgba(127, 127, 127, 0.08);
  color: var(--text-primary, currentColor);
}

.vhp-drawer-body {
  flex: 1 1 auto;
  overflow: auto;
  min-height: 100px;
  max-height: 220px;
}
.vhp-drawer-loading,
.vhp-drawer-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 16px;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  text-align: center;
}
.vhp-drawer-pre {
  margin: 0;
  padding: 6px 8px;
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  line-height: 1.45;
  white-space: pre;
  background: transparent;
}

.vhp-drawer-actions {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
  padding: 4px 8px;
  border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.1));
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