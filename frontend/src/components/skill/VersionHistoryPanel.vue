<script setup>
// VersionHistoryPanel - VSCode 风格 commit 列表 + 独立 modal diff(2026-07-17 重构)
//
// 2026-07-17 v2 大改:
//   - 底部抽屉改成独立 modal(全屏居中 + 大尺寸,看清楚 diff)
//   - 文件列表只显示文件名(不显示目录路径),hover title 看完整路径
//   - 历史面板**只显示当前 skill 的 commits**(强制传 skillPath,
//     父级 SkillScopePanel 已经传,这里直接用 props.skillPath)
//
// 跟原 VersionHistoryModal 行为差异:无弹窗(嵌入面板内) + 弹窗(看 diff);
// commit 列表永远 inline,看具体文件差异才弹 modal。

import { ref, computed, watch, inject, onMounted, onUnmounted } from 'vue'
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
  // 当前 skill 在仓库内的路径(相对 repo root,例如 "frontend/code-review")
  // — 仅显示涉及该路径的 commit + 仅显示该路径下的文件变更。
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

// 2026-07-17:展开的文件列表(每个 commit 独立 toggle)
const expandedCommits = ref(new Set())

// 2026-07-17:diff modal — 不再是底部抽屉,是独立全屏 modal
const modalOpen = ref(false)
const modalCommitHash = ref('')
const modalFile = ref('')
const modalFileList = ref([]) // 当前 commit 的全部变更文件列表(过滤掉 skillPath 前缀)
const modalDiffText = ref('')
const modalDiffHint = ref('')
const modalDiffLoading = ref(false)

// 2026-07-17:解析 conventional commit 头
function parseCommitTitle(msg) {
  const firstLine = (msg || '').split('\n', 1)[0] || ''
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

// 2026-07-17:只取文件名(去掉当前 skill 路径前缀 + 全部目录)。
function shortFileName(filePath, skillPath) {
  if (!filePath) return ''
  let rest = filePath
  if (skillPath && filePath.startsWith(skillPath + '/')) {
    rest = filePath.slice(skillPath.length + 1)
  }
  const idx = Math.max(rest.lastIndexOf('/'), rest.lastIndexOf('\\'))
  return idx < 0 ? rest : rest.slice(idx + 1)
}

// 过滤掉当前 skillPath 前缀,得到内部相对路径
function relativeFilePath(filePath, skillPath) {
  if (skillPath && filePath.startsWith(skillPath + '/')) {
    return filePath.slice(skillPath.length + 1)
  }
  return filePath
}

watch(() => props.skillPath, () => {
  // 切 skill 时清掉展开 / modal
  expandedCommits.value = new Set()
  modalOpen.value = false
  modalFile.value = ''
  modalCommitHash.value = ''
  modalFileList.value = []
  modalDiffText.value = ''
  modalDiffHint.value = ''
  loadAll()
})
watch(isExpanded, (open) => {
  if (open) loadAll()
})

async function loadAll() {
  loading.value = true
  errorMsg.value = ''
  try {
    // 2026-07-17:强制传 skillPath — 只显示当前 skill 范围内的 commit
    const log = await getGitLog(50, props.skillPath || undefined)
    items.value = (log.items || []).map((it) => ({
      ...it,
      _title: parseCommitTitle(it.message),
    }))
    const st = await getGitStatus()
    status.value = st
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

// 2026-07-18 增:仅刷新 status(轻量,不拉 log)。折叠态也用这个 —
// 否则 title meta 默认 "initialized:false" 一直显示 "未初始化"。
async function refreshStatus() {
  try {
    const st = await getGitStatus()
    status.value = st
  } catch (_) {}
}

function toggleCommitFiles(hash) {
  const set = new Set(expandedCommits.value)
  if (set.has(hash)) {
    set.delete(hash)
  } else {
    set.add(hash)
  }
  expandedCommits.value = set
}

// 2026-07-17:点文件弹 modal。打开 modal 时拉取该 commit 的全量 diff,
// 前端按文件路径切分渲染(避免反复拉 API)。
// 2026-07-17 改:用 commit.parent_hash 作为 from(避免发 "<hash>^"
// 让 go-git ResolveRevision 卡 15s);root commit 没 parent → from=""
// 后端会退化到空 tree,生成"全文件新增"diff。
async function openFileModal(commitHash, filePath) {
  const commit = items.value.find((it) => it.hash === commitHash)
  modalCommitHash.value = commitHash
  modalFile.value = filePath
  modalDiffLoading.value = true
  modalDiffText.value = ''
  modalDiffHint.value = ''
  modalOpen.value = true
  try {
    // 文件列表 = 该 commit 的所有变更文件(已过滤 skillPath 前缀)
    modalFileList.value = (commit?.files || []).map((f) => relativeFilePath(f, props.skillPath))
    const fromRef = commit?.parent_hash || ''
    const r = await getGitDiff(fromRef, commitHash)
    modalDiffText.value = r.diff || ''
    modalDiffHint.value = r.hint || ''
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
    modalDiffText.value = ''
  } finally {
    modalDiffLoading.value = false
  }
}

function closeModal() {
  modalOpen.value = false
  modalFile.value = ''
  modalCommitHash.value = ''
  modalFileList.value = []
  modalDiffText.value = ''
  modalDiffHint.value = ''
}

// 2026-07-17:把 hint 里嵌入的 git diff 命令复制到剪贴板,方便用户
// 直接粘贴到终端跑。
async function copyDiffCmd() {
  if (!modalDiffHint.value) return
  // 从 hint 文本里抠出 `git diff A B` 这一段(在反引号里)
  const m = modalDiffHint.value.match(/`([^`]+)`/)
  const cmd = m ? m[1] : modalDiffHint.value
  try {
    await navigator.clipboard.writeText(cmd)
    errorMsg.value = t('git.copied', { cmd })
  } catch (e) {
    errorMsg.value = t('git.copyFailed', { msg: String(e) })
  }
}

// 2026-07-17:modal 内点文件名 → 切 modalFile;不重新拉 API(diff 已存在)
function pickModalFile(filePath) {
  modalFile.value = filePath
}

// 2026-07-17:从全量 diff 里抽出指定文件的块(已用相对路径)
function filterDiffByFile(diff, relPath) {
  if (!diff || !relPath) return diff
  const lines = diff.split('\n')
  const out = []
  let inTarget = false
  for (const line of lines) {
    if (line.startsWith('diff --git ')) {
      inTarget =
        line.includes(' a/' + relPath) ||
        line.includes(' b/' + relPath)
    }
    if (inTarget) out.push(line)
  }
  return out.join('\n')
}

const modalCommit = computed(() =>
  items.value.find((it) => it.hash === modalCommitHash.value) || null,
)

const modalFilteredDiff = computed(() => {
  if (!modalFile.value) return modalDiffText.value
  return filterDiffByFile(modalDiffText.value, modalFile.value)
})

// 2026-07-17:diff 行级拆 + 染色
// 2026-07-18 大改:从"每行一个 vnode"改成"合并相邻同色行成一个 segment"。
// 调研结论:
//   - VSCode / GitHub / JetBrains 全部行级(line-level)染色,不做 word-level
//   - context 行不加任何背景(只靠 +/- 符号提示)
//   - hunk header 与 context 行用不同灰度色分段
//   - 性能:3000 行 diff 合并后只剩 ~100 segments,vnode 量级降 10x
// segment 形态: { cls: 'diff-add' | 'diff-del' | 'diff-ctx' | 'diff-hunk'
//                | 'diff-meta', lines: string[] }
// 渲染:<pre><span v-for="seg in modalDiffSegments" :class="seg.cls">...</span></pre>
function classifyDiffLine(line) {
  if (!line) return 'diff-ctx' // 空行当 context(行首空 = 空白行,染色不必要)
  if (line.startsWith('@@')) return 'diff-hunk'
  if (line.startsWith('+++') || line.startsWith('---')) return 'diff-meta'
  if (line.startsWith('diff --git ')) return 'diff-meta'
  if (line.startsWith('+')) return 'diff-add'
  if (line.startsWith('-')) return 'diff-del'
  if (line.startsWith(' ')) return 'diff-ctx'
  return 'diff-ctx' // 兜底
}

const modalDiffSegments = computed(() => {
  if (!modalFilteredDiff.value) return []
  const raw = modalFilteredDiff.value.split('\n')
  const out = []
  let cur = null
  // 末尾 \n split 会产生一个空字符串尾巴 — 直接丢
  for (let i = 0; i < raw.length; i++) {
    const line = raw[i]
    // 跳过末尾空行 + 纯空白尾,但保留行内的有效 \n
    if (i === raw.length - 1 && line === '') continue
    const cls = classifyDiffLine(line)
    if (cur && cur.cls === cls) {
      cur.lines.push(line)
    } else {
      if (cur) out.push(cur)
      cur = { cls, lines: [line] }
    }
  }
  if (cur) out.push(cur)
  return out
})

// 保留函数给旧引用(如果别处还在用)
const modalDiffLines = computed(() => {
  if (!modalFilteredDiff.value) return []
  return modalFilteredDiff.value.split('\n')
})
function diffLineClass(line) {
  return classifyDiffLine(line)
}

// 2026-07-18 增:跨组件事件 — 保存后通知本 panel 重拉 log。
// store.Save 末尾走 goroutine 异步 commit,前端 save 调用方 dispatch
// 'skillbox:git-refresh'。本 panel 仅在已展开 + 绑定了 skillPath 时
// 才重新拉,折叠态不触发 IO。
function onGitRefresh() {
  if (!props.skillPath) return
  if (!isExpanded.value) return
  loadAll()
}

// 2026-07-17:ESC 关 modal
function onKeydown(e) {
  if (e.key === 'Escape' && modalOpen.value) {
    closeModal()
  }
}
onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('skillbox:git-refresh', onGitRefresh)
  // 2026-07-18 增:折叠态也刷新 status — 否则 title meta 默认
  // "initialized:false" 一直显示 "未初始化" 徽章,误以为仓库没 init。
  // 不拉 log(避免折叠态偷偷吃 IO),只刷 status 让 badge 文案正确。
  refreshStatus()
})

// 2026-07-18 增:切 skill 时刷 status(不依赖 isExpanded)。
// 同 GitSyncPanel 策略 — status 是轻量,折叠态也得正确。
watch(
  () => props.skillPath,
  () => {
    refreshStatus()
  },
)
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('skillbox:git-refresh', onGitRefresh)
})

async function doCheckout() {
  if (!modalCommitHash.value) return
  if (!confirm(t('git.checkoutConfirm', { hash: modalCommitHash.value.slice(0, 7) }))) {
    return
  }
  loading.value = true
  try {
    await checkoutGit(modalCommitHash.value)
    errorMsg.value = ''
    emit('checked-out', modalCommitHash.value)
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
      <!-- 单列 commit 列表(只显示当前 skill 范围) -->
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
            <!-- commit 行 -->
            <div
              class="vhp-commit-row"
              @click="toggleCommitFiles(it.hash)"
            >
              <div class="vhp-node">
                <div class="vhp-node-line vhp-node-line-top" />
                <div class="vhp-node-dot" />
                <div class="vhp-node-line vhp-node-line-bot" />
              </div>
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
              <IconPark
                :type="expandedCommits.has(it.hash) ? 'down' : 'right'"
                :size="10"
                class="vhp-commit-arrow"
              />
            </div>

            <!-- 展开的文件列表(只显示文件名) -->
            <div v-if="expandedCommits.has(it.hash)" class="vhp-files">
              <div v-if="!it.files || !it.files.length" class="vhp-files-empty">
                {{ t('git.history.noFiles') }}
              </div>
              <div
                v-for="f in (it.files || [])"
                :key="f"
                class="vhp-file-row"
                :title="f"
                @click.stop="openFileModal(it.hash, relativeFilePath(f, skillPath))"
              >
                <IconPark type="right" :size="10" class="vhp-file-arrow" />
                <span class="vhp-file-name">{{ shortFileName(f, skillPath) }}</span>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="vhp-empty">
          <p>{{ t('git.history.emptySkill') }}</p>
        </div>
      </div>
    </div>
  </CollapsiblePanel>

  <!-- 2026-07-17:diff 用独立 modal 全屏显示 — 抽屉位置太小看不清 -->
  <teleport to="body">
    <transition name="vhp-modal">
      <div
        v-if="modalOpen"
        class="vhp-modal-mask"
        @click.self="closeModal"
      >
        <div class="vhp-modal" role="dialog">
          <!-- modal header -->
          <div class="vhp-modal-header">
            <div class="vhp-modal-header-left">
              <IconPark type="code" :size="14" />
              <span class="vhp-modal-title">
                {{ modalFile || (modalCommit && modalCommit._title.full) || shortHash(modalCommitHash) }}
              </span>
              <span v-if="modalCommitHash" class="vhp-modal-range">
                {{ shortHash(modalCommitHash) }}
              </span>
            </div>
            <div class="vhp-modal-header-right">
              <button
                class="vhp-btn"
                :disabled="loading"
                @click="doCheckout"
              >
                <IconPark type="undo" :size="11" />
                {{ t('git.history.checkout') }}
              </button>
              <button
                v-if="status.remote_url"
                class="vhp-btn"
                :disabled="loading"
                @click="doPush"
              >
                <IconPark type="upload" :size="11" />
                {{ t('git.history.push') }}
              </button>
              <button
                v-if="status.remote_url"
                class="vhp-btn"
                :disabled="loading"
                @click="doPull"
              >
                <IconPark type="download" :size="11" />
                {{ t('git.history.pull') }}
              </button>
              <button
                v-if="!status.working_clean"
                class="vhp-btn warn"
                :disabled="loading"
                @click="doDiscard"
              >
                <IconPark type="undo" :size="11" />
                {{ t('git.discard') }}
              </button>
              <button
                class="vhp-modal-close"
                :title="t('common.close')"
                @click="closeModal"
              >
                <IconPark type="close" :size="14" />
              </button>
            </div>
          </div>

          <!-- modal body:左侧文件列表 + 右侧 diff -->
          <div class="vhp-modal-body">
            <!-- 左:文件列表(只文件名,选中的高亮) -->
            <div class="vhp-modal-files">
              <div
                :class="['vhp-modal-file', { active: !modalFile }]"
                @click="pickModalFile('')"
              >
                <IconPark type="file" :size="11" />
                <span class="vhp-modal-file-name">{{ t('git.history.allFiles') }}</span>
              </div>
              <div
                v-for="f in modalFileList"
                :key="f"
                :class="['vhp-modal-file', { active: f === modalFile }]"
                @click="pickModalFile(f)"
              >
                <IconPark type="file" :size="11" />
                <span class="vhp-modal-file-name" :title="f">{{ f }}</span>
              </div>
              <div v-if="!modalFileList.length" class="vhp-modal-files-empty">
                {{ t('git.history.noFiles') }}
              </div>
            </div>

            <!-- 右:diff 内容 -->
            <div class="vhp-modal-diff">
              <div v-if="modalDiffLoading" class="vhp-modal-diff-loading">
                {{ t('common.loading') }}
              </div>
              <div v-else-if="modalDiffHint" class="vhp-modal-diff-empty">
                <IconPark type="warning" :size="14" />
                <p>{{ modalDiffHint }}</p>
                <button class="vhp-btn" @click="copyDiffCmd">
                  <IconPark type="copy" :size="11" />
                  {{ t('git.copyCmd') }}
                </button>
              </div>
              <!--
                2026-07-18 重写:行级染色 diff viewer。
                - segments 合并策略:相邻同色行合成一个 <span>,vnode 量级 ~segments 数
                - 不在 <pre> 内部用 v-for span,改成一段一段渲染,但写起来还是
                  <pre> 顶层 + 顶层 v-for span,本质上 Vue 3 不会丢 children
                - 行级配色见 .vhp-modal-diff-pre .diff-add / .diff-del
              -->
              <pre
                v-else-if="modalDiffText"
                class="vhp-modal-diff-pre"
              ><span
                v-for="(seg, i) in modalDiffSegments"
                :key="i"
                :class="seg.cls"
              >{{ seg.lines.join('\n') + '\n' }}</span></pre>
              <div v-else class="vhp-modal-diff-empty">
                {{ t('git.history.pickCommit') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<style scoped>
/* 2026-07-17 v2:VSCode 风格 + 独立 modal */

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

.vhp-shell { display: flex; flex-direction: column; min-height: 0; }

.vhp-list {
  overflow: auto;
  max-height: 480px;
}
.vhp-commits { display: flex; flex-direction: column; }
.vhp-commit { display: flex; flex-direction: column; }

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
.vhp-node-line { width: 1px; flex: 1; background: rgba(127, 127, 127, 0.25); }
.vhp-node-line-top { margin-bottom: -3.5px; }
.vhp-node-line-bot { margin-top: -3.5px; }

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
.vhp-commit-type { color: #9cdcfe; flex-shrink: 0; }
.vhp-commit-scope { color: #9cdcfe; flex-shrink: 0; }
.vhp-commit-sep { color: rgba(127, 127, 127, 0.7); margin: 0 1px; flex-shrink: 0; }
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

.vhp-commit-arrow {
  flex-shrink: 0;
  color: rgba(127, 127, 127, 0.5);
  margin-right: 4px;
}

/* 文件列表 — 只显示文件名 */
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
.vhp-file-arrow { flex-shrink: 0; color: rgba(127, 127, 127, 0.4); }
.vhp-file-name { color: var(--text-primary, currentColor); }

/* =========================================================================
   Diff Modal — 全屏居中独立弹窗,看清楚差异
   ========================================================================= */

.vhp-modal-enter-active,
.vhp-modal-leave-active {
  transition: opacity 150ms ease;
}
.vhp-modal-enter-from,
.vhp-modal-leave-to {
  opacity: 0;
}
.vhp-modal-enter-active .vhp-modal,
.vhp-modal-leave-active .vhp-modal {
  transition: transform 200ms ease;
}
.vhp-modal-enter-from .vhp-modal,
.vhp-modal-leave-to .vhp-modal {
  transform: scale(0.96) translateY(8px);
}

.vhp-modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9000;
  padding: 32px;
}

.vhp-modal {
  width: min(1100px, calc(100vw - 64px));
  height: min(720px, calc(100vh - 64px));
  /* 2026-07-18 改:diff 区域是深底容器 + 亮色 diff 字体才不"一片空白"
     (亮底 + 浅字 = 看不清)。用 --bg-card 跟主面板一致,字色 inherit var(--text)
     在 dark theme 下 = #fafafa(亮白),在 light theme 下 = #171717(深色)。
     diff 行级染色保留原有 CSS 类对 addon。 */
  background: var(--bg-card, #fff);
  color: var(--text, #171717);
  border-radius: 8px;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.35);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.vhp-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.2));
  background: var(--bg-elevated, rgba(127, 127, 127, 0.03));
  flex-shrink: 0;
}
.vhp-modal-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.vhp-modal-title {
  font-family: var(--font-mono, monospace);
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}
.vhp-modal-range {
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  flex-shrink: 0;
}
.vhp-modal-header-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.vhp-modal-close {
  background: transparent;
  border: 0;
  padding: 4px;
  cursor: pointer;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.vhp-modal-close:hover {
  background: rgba(127, 127, 127, 0.08);
  color: var(--text-primary, currentColor);
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
.vhp-btn.warn { border-color: rgb(245, 158, 11); color: rgb(245, 158, 11); }

/* modal body:左文件列表 + 右 diff */
.vhp-modal-body {
  flex: 1 1 auto;
  display: flex;
  min-height: 0;
  overflow: hidden;
}

.vhp-modal-files {
  flex: 0 0 220px;
  border-right: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  overflow: auto;
  padding: 6px 4px;
  background: var(--bg-elevated, rgba(127, 127, 127, 0.02));
}
.vhp-modal-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  cursor: pointer;
  border-radius: 3px;
  transition: background 80ms;
}
.vhp-modal-file:hover { background: rgba(127, 127, 127, 0.08); }
.vhp-modal-file.active {
  background: rgba(59, 130, 246, 0.15);
  color: rgb(59, 130, 246);
}
.vhp-modal-file-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}
.vhp-modal-files-empty {
  padding: 12px 8px;
  font-size: 10px;
  color: rgba(127, 127, 127, 0.5);
  font-style: italic;
  text-align: center;
}

.vhp-modal-diff {
  flex: 1 1 auto;
  overflow: auto;
  /* 2026-07-18 改:跟 modal 背景同 var(--bg-card),避免 modal 卡体深色 +
     diff 区白底卡中开白底的怪相。 */
  background: var(--bg-elevated, var(--bg-card, #fff));
  color: var(--text);
  position: relative;
}
.vhp-modal-diff-loading,
.vhp-modal-diff-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 12px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
/* 2026-07-18 升级:diff 行级染色 — 参考 VSCode / GitHub / JetBrains 三家
   一致的 line-level 配色(行内 word-level 不做,先求稳定)。
   context 行不加任何背景(只靠 +/- 符号提示),行业规范。 */
.vhp-modal-diff-pre {
  margin: 0;
  padding: 12px 16px;
  font-family: var(--font-mono, 'SF Mono', Menlo, Consolas, monospace);
  font-size: 12px;
  line-height: 1.55;
  /* white-space: pre(pre 是 HTML5 raw-text 元素,空白字符会被保留),
     不要再加 pre-wrap/pre+wrap — pre 元素语义已经是保留空白。 */
  white-space: pre;
  /* 横向滚动让长行(markdown 段落、JSON 行)能展开 — pre 自带 overflow:auto 也行,
     这里显式给浏览器 sticky 提示。 */
  overflow-x: auto;
  overflow-y: hidden;
  color: var(--text, #24292f);
}
.vhp-modal-diff-pre .diff-ctx {
  /* 上下文不染色 — VSCode / GitHub / IDEA 一致选择,只靠 +/- 符号区分。 */
  background: transparent;
  color: inherit;
}
.vhp-modal-diff-pre .diff-add {
  /* 新增行:GitHub primer 浅绿 #e6ffec。这里用 rgba 半透明,与 modal 底色叠加。 */
  background: rgba(46, 160, 67, 0.15);
  color: #1a7f37;
}
.vhp-modal-diff-pre .diff-del {
  /* 删除行:浅红 #ffebe9 同款 rgba 半透明。 */
  background: rgba(248, 81, 73, 0.15);
  color: #cf222e;
}
.vhp-modal-diff-pre .diff-hunk {
  /* @@ -10,5 +12,7 @@ — GitHub 蓝灰分段背景,深色下稍深。 */
  background: rgba(56, 139, 253, 0.12);
  color: #6e7781;
}
.vhp-modal-diff-pre .diff-meta {
  /* diff --git / --- / +++ 文件头 — 中性灰,跟 hunk 区分但同色系。 */
  background: transparent;
  color: #6e7781;
}
/* dark theme override — :root[data-theme="dark"] 与 html.dark 都罩住。
   现代浏览器 useDark + wails 都给 <html> 加 data-theme 或 class。 */
:root[data-theme="dark"] .vhp-modal-diff-pre,
html.dark .vhp-modal-diff-pre {
  color: #c9d1d9;
}
:root[data-theme="dark"] .vhp-modal-diff-pre .diff-add,
html.dark .vhp-modal-diff-pre .diff-add {
  background: rgba(46, 160, 67, 0.28);
  color: #3fb950;
}
:root[data-theme="dark"] .vhp-modal-diff-pre .diff-del,
html.dark .vhp-modal-diff-pre .diff-del {
  background: rgba(248, 81, 73, 0.28);
  color: #f85149;
}
:root[data-theme="dark"] .vhp-modal-diff-pre .diff-hunk,
html.dark .vhp-modal-diff-pre .diff-hunk {
  background: rgba(56, 139, 253, 0.22);
  color: #8b949e;
}
:root[data-theme="dark"] .vhp-modal-diff-pre .diff-meta,
html.dark .vhp-modal-diff-pre .diff-meta {
  color: #8b949e;
}
</style>