<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill } from '@/api/skillbox/skills'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { applySkill, undoApply, listApplies, checkUpdates } from '@/api/skillbox/skill_apply'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
import AIPanel from '@/components/AIPanel.vue'
import Modal from '@/components/Modal.vue'

const { t } = useI18n()

// 当前 scope 选择
const scope = ref('global')
const keyword = ref('')
const loading = ref(false)
const error = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 10

// 编辑器状态(弹窗)
const editorOpen = ref(false)
const draft = reactive({
  scope: 'global',
  project_id: 0,
  name: '',
  version: '0.1.0',
  description: '',
  triggersText: '',
  body: '',
})
const editingKey = ref(null)

// Apply / 撤销 / 更新检测
const TOOL_OPTIONS = ['codex', 'claude', 'opencode', 'cursor', 'trae']
const applyTool = ref('codex')
const applying = ref(false)
const applyMessage = ref('')
const applyError = ref('')
const lastApplies = ref([])
const undoing = ref(false)
const updating = ref(false)
const updateBadge = ref({ total: 0, updates: 0 })
const applyHistory = ref([])

// Tag 弹窗状态
const tagOpen = ref(false)
const tagList = ref([])
const tagLoading = ref(false)
const tagError = ref('')
const tagMessage = ref('')
const newTagName = ref('')
const newTagMessage = ref('')
const diffResult = ref(null)
const diffLeftTagID = ref(0)
const diffRightTagID = ref(0)
const rolling = ref(false)
const selectedSkill = ref(null)

// 测试结果弹窗
const testOpen = ref(false)
const testing = ref(false)
const testError = ref('')
const lastTest = ref(null)

// 通用确认弹窗(取代原生 confirm)
const confirmOpen = ref(false)
const confirmOpts = reactive({
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  variant: 'default', // default | danger
  resolve: null,
})
function openConfirm(opts) {
  confirmOpts.title = opts.title || t('common.confirm')
  confirmOpts.message = opts.message || ''
  confirmOpts.confirmText = opts.confirmText || t('common.confirm')
  confirmOpts.cancelText = opts.cancelText || t('common.cancel')
  confirmOpts.variant = opts.variant || 'default'
  confirmOpen.value = true
  return new Promise((resolve) => { confirmOpts.resolve = resolve })
}
function resolveConfirm(ok) {
  if (confirmOpts.resolve) confirmOpts.resolve(ok)
  confirmOpen.value = false
}

// AI 侧栏
const aiOpen = ref(false)
function toggleAI() { aiOpen.value = !aiOpen.value }

// 当前正在编辑的 SKILL.md 全文
const currentSkillMd = computed(() => {
  if (!editorOpen.value) return ''
  try { return buildSkillMd() } catch (_) { return '' }
})

function onAIApply(text) {
  const m = text.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  draft.body = m ? m[1].trim() : text.trim()
}

async function loadTags(row) {
  selectedSkill.value = row
  tagOpen.value = true
  tagList.value = []
  diffResult.value = null
  newTagName.value = ''
  newTagMessage.value = ''
  await loadTagList(row)
}

async function loadTagList(row) {
  if (!row) return
  tagLoading.value = true
  tagError.value = ''
  try {
    const out = await listTags({ skill_id: row.id })
    tagList.value = out?.items || []
  } catch (e) {
    tagError.value = e?.message || String(e)
  } finally {
    tagLoading.value = false
  }
}

async function doCreateTag() {
  if (!selectedSkill.value) { tagError.value = t('skills.tag.selectFirst'); return }
  if (!newTagName.value.trim()) { tagError.value = t('skills.tag.emptyName'); return }
  tagLoading.value = true
  tagError.value = ''
  try {
    await createTag({
      skill_id: selectedSkill.value.id,
      tag: newTagName.value.trim(),
      message: newTagMessage.value,
    })
    newTagName.value = ''
    newTagMessage.value = ''
    tagMessage.value = t('skills.tag.msgCreated')
    await loadTagList(selectedSkill.value)
  } catch (e) {
    tagError.value = e?.message || String(e)
  } finally {
    tagLoading.value = false
  }
}

async function doDeleteTag(tagID) {
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('skills.tag.confirmDelete', { id: tagID }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteTag({ tag_id: tagID })
    tagMessage.value = t('skills.tag.msgDeleted', { id: tagID })
    await loadTagList(selectedSkill.value)
  } catch (e) {
    tagError.value = e?.message || String(e)
  }
}

async function doDiff(leftID, rightID) {
  if (!selectedSkill.value) { tagError.value = t('skills.tag.selectFirst'); return }
  try {
    const out = await diffTag({
      skill_id: selectedSkill.value.id,
      left_tag_id: leftID || 0,
      right_tag_id: rightID || 0,
    })
    diffResult.value = out
    diffLeftTagID.value = leftID
    diffRightTagID.value = rightID
  } catch (e) {
    tagError.value = e?.message || String(e)
  }
}

async function doRollback(tagID) {
  const ok = await openConfirm({
    title: t('skills.tag.rollbackTo'),
    message: t('skills.tag.confirmRollback', { id: tagID }),
    confirmText: t('skills.tag.rollbackTo'),
    variant: 'danger',
  })
  if (!ok) return
  rolling.value = true
  tagError.value = ''
  try {
    const out = await rollbackTag({ tag_id: tagID })
    tagMessage.value = t('skills.tag.msgRolledBack', { pre: out.pre_rollback_tag, files: out.files_restored })
    diffResult.value = null
    await reload()
    if (selectedSkill.value) await loadTagList(selectedSkill.value)
  } catch (e) {
    tagError.value = e?.message || String(e)
  } finally {
    rolling.value = false
  }
}

async function doApply(row) {
  applying.value = true
  applyError.value = ''
  applyMessage.value = ''
  try {
    const out = await applySkill({
      skill_id: row.id,
      scope: row.scope,
      project_id: row.project_id,
      tools: [applyTool.value],
    })
    lastApplies.value = out?.applies || []
    applyMessage.value = out?.all_ok
      ? t('skills.applyBar.appliedOk', { name: row.name, version: row.version, tool: applyTool.value })
      : t('skills.applyBar.appliedPartial', { detail: (out?.applies || []).filter(a => a?.status !== 'applied').map(a => a?.error || a?.status).join('; ') })
    await loadApplyHistory(row)
    await checkUpdateBadge()
  } catch (e) {
    applyError.value = e?.message || String(e)
  } finally {
    applying.value = false
  }
}

async function doUndo(applyID) {
  const ok = await openConfirm({
    title: t('skills.applyHistory.undone'),
    message: t('skills.tag.confirmUndo', { id: applyID }),
    confirmText: t('skills.applyHistory.undone'),
    variant: 'danger',
  })
  if (!ok) return
  undoing.value = true
  applyError.value = ''
  try {
    await undoApply({ apply_id: applyID })
    applyMessage.value = t('skills.tag.undoMsg', { id: applyID })
    await reload()
    await checkUpdateBadge()
  } catch (e) {
    applyError.value = e?.message || String(e)
  } finally {
    undoing.value = false
  }
}

async function loadApplyHistory(row) {
  try {
    const out = await listApplies({ skill_id: row.id, page: 1, size: 5 })
    applyHistory.value = out?.items || []
  } catch (e) {}
}

async function checkUpdateBadge() {
  updating.value = true
  try {
    const out = await checkUpdates({})
    updateBadge.value = { total: out?.total || 0, updates: out?.updates || 0 }
  } catch (e) {} finally {
    updating.value = false
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const resp = await listSkills({
      scope: scope.value,
      keyword: keyword.value || undefined,
      page: page.value,
      size,
    })
    items.value = resp?.items || []
    total.value = resp?.total || 0
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

function startNew() {
  Object.assign(draft, {
    scope: scope.value,
    project_id: 0,
    name: '',
    version: '0.1.0',
    description: '',
    triggersText: '',
    body: '',
  })
  editingKey.value = null
  error.value = ''
  editorOpen.value = true
}

async function startEdit(row) {
  error.value = ''
  editorOpen.value = true
  // 先给一个空 draft,再异步加载详情
  Object.assign(draft, {
    scope: row.scope,
    project_id: row.project_id,
    name: row.name,
    version: row.version,
    description: '',
    triggersText: '',
    body: '',
  })
  editingKey.value = null
  try {
    const full = await getSkill({
      scope: row.scope,
      project_id: row.project_id,
      name: row.name,
      version: row.version,
      full: true,
    })
    const c = full?.canonical?.manifest || {}
    const files = full?.canonical?.files || []
    const md = files.find((f) => f.path === 'SKILL.md')?.content || ''
    Object.assign(draft, {
      scope: row.scope,
      project_id: row.project_id,
      name: row.name,
      version: row.version,
      description: c.description || '',
      triggersText: (c.triggers || []).join('\n'),
      body: extractBody(md),
    })
    editingKey.value = { scope: row.scope, name: row.name, version: row.version, project_id: row.project_id }
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

function extractBody(skillmd) {
  const m = skillmd.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  return m ? m[1].trim() : skillmd
}

function buildSkillMd() {
  const triggers = draft.triggersText
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  const m = {
    name: draft.name,
    version: draft.version,
    description: draft.description,
    triggers,
  }
  const yaml = Object.entries(m)
    .map(([k, v]) => Array.isArray(v) ? `${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]` : `${k}: ${JSON.stringify(v)}`)
    .join('\n')
  return `---\n${yaml}\n---\n\n${draft.body || ''}\n`
}

async function submit() {
  error.value = ''
  if (!draft.name.trim()) {
    error.value = t('skills.editor.errNameEmpty')
    return
  }
  if (draft.description.trim().length < 10) {
    error.value = t('skills.editor.errDescShort')
    return
  }
  const triggers = draft.triggersText
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (triggers.length === 0) {
    error.value = t('skills.editor.errTriggersEmpty')
    return
  }
  const payload = {
    scope: draft.scope,
    project_id: draft.project_id,
    name: draft.name,
    version: draft.version,
    source: 'local',
    manifest: {
      name: draft.name,
      version: draft.version,
      description: draft.description,
      triggers,
    },
    files: [{ path: 'SKILL.md', content: buildSkillMd() }],
  }
  try {
    if (editingKey.value) {
      await updateSkill(payload)
    } else {
      await createSkill(payload)
    }
    editorOpen.value = false
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

async function triggerTest(row) {
  const ok = await openConfirm({
    title: t('skills.test.title'),
    message: t('skills.test.confirmRun', { name: row.name, version: row.version }),
    confirmText: t('skills.list.btnTest'),
  })
  if (!ok) return
  testOpen.value = true
  testing.value = true
  testError.value = ''
  lastTest.value = null
  try {
    const out = await runSkillTest({
      scope: row.scope,
      project_id: row.project_id,
      name: row.name,
      version: row.version,
      trigger: 'manual',
    })
    lastTest.value = out
  } catch (e) {
    testError.value = e?.message || String(e)
  } finally {
    testing.value = false
  }
}

async function remove(row) {
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('skills.list.confirmDelete', { name: row.name, version: row.version }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteSkill({
      scope: row.scope,
      project_id: row.project_id,
      name: row.name,
      version: row.version,
    })
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

function gotoPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  reload()
}

function switchScope(s) {
  scope.value = s
  page.value = 1
  reload()
}

onMounted(() => { reload(); checkUpdateBadge() })
</script>

<template>
  <div class="skills-layout">
    <section class="skills-view" :class="{ 'with-ai': aiOpen }">
      <!-- 页面头部 -->
      <header class="view-header">
        <div class="view-title">
          <div class="view-icon">
            <Icon icon="mdi:book-open-variant" width="24" height="24" />
          </div>
          <div>
            <h1>{{ t('skills.title') }}</h1>
            <p>{{ t('skills.subtitle') }}</p>
          </div>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <div class="scope-tabs">
            <button
              :class="['scope-tab', scope === 'global' ? 'scope-tab-active' : '']"
              @click="switchScope('global')"
            >
              <Icon icon="mdi:earth" width="16" height="16" />
              {{ t('skills.scopeGlobal') }}
            </button>
            <button
              :class="['scope-tab', scope === 'project' ? 'scope-tab-active' : '']"
              @click="switchScope('project')"
            >
              <Icon icon="mdi:folder-outline" width="16" height="16" />
              {{ t('skills.scopeProject') }}
            </button>
          </div>
        </div>

        <div class="toolbar-right">
          <div class="search-box">
            <Icon icon="mdi:magnify" width="16" height="16" class="search-icon" />
            <input
              v-model="keyword"
              :placeholder="t('skills.searchPlaceholder')"
              class="search-input"
              @keyup.enter="() => { page = 1; reload() }"
            />
          </div>
          <button class="ai-btn" @click="toggleAI">
            <Icon icon="mdi:robot-outline" width="16" height="16" />
            {{ aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen') }}
          </button>
          <button class="primary" @click="startNew">
            <Icon icon="mdi:plus" width="16" height="16" />
            {{ t('skills.btnNew') }}
          </button>
        </div>
      </div>

      <!-- Apply 工具栏 -->
      <div class="apply-toolbar">
        <div class="apply-left">
          <span class="apply-label">{{ t('skills.applyBar.target') }}</span>
          <select v-model="applyTool" class="apply-select">
            <option v-for="t in TOOL_OPTIONS" :key="t" :value="t">{{ t }}</option>
          </select>
        </div>
        <div class="apply-right">
          <button class="check-updates-btn" @click="checkUpdateBadge" :disabled="updating">
            <span v-if="updating" class="spinner"></span>
            <Icon v-else icon="mdi:refresh" width="14" height="14" />
            {{ updating ? t('skills.applyBar.checking') : t('skills.applyBar.checkUpdates') }}
          </button>
          <span v-if="updateBadge.updates > 0" class="update-badge update-badge-danger">
            <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ t('skills.applyBar.updatesAvailable', { updates: updateBadge.updates, total: updateBadge.total }) }}
          </span>
          <span v-else-if="updateBadge.total > 0" class="update-badge update-badge-success">
            <Icon icon="mdi:check-circle-outline" width="12" height="12" />
            {{ t('skills.applyBar.allUpToDate', { total: updateBadge.total }) }}
          </span>
        </div>
      </div>

      <p v-if="applyMessage" class="message message-success">{{ applyMessage }}</p>
      <p v-if="applyError" class="message message-error">{{ applyError }}</p>

      <!-- Apply 历史(只显示最近 5 条,这里保持内嵌卡片) -->
      <div v-if="applyHistory.length" class="card">
        <header class="card-header">
          <h3>{{ t('skills.applyHistory.title') }}</h3>
          <span class="card-sub">{{ t('skills.applyHistory.count', { count: applyHistory.length }) }}</span>
        </header>
        <ul class="apply-list">
          <li v-for="h in applyHistory" :key="h.apply_id || h.ID || h.id" :class="`apply-status-${h.status}`">
            <span class="apply-id">#{{ h.apply_id || h.ID || h.id }}</span>
            <span class="apply-tool-name">{{ h.tool }}</span>
            <span class="apply-status-badge" :class="`badge-${h.status}`">{{ h.status }}</span>
            <span class="apply-time">{{ h.applied_at?.slice(0, 19) || t('common.dash') }}</span>
            <button v-if="h.status === 'applied'" class="link danger" :disabled="undoing" @click="doUndo(h.apply_id || h.ID || h.id)">
              {{ undoing ? t('skills.applyHistory.undoing') : t('skills.applyHistory.undone') }}
            </button>
          </li>
        </ul>
      </div>

      <!-- 错误提示 -->
      <p v-if="error" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ error }}
      </p>

      <!-- 列表卡片 -->
      <div class="card">
        <header class="card-header">
          <h3>
            <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
            {{ t('skills.list.title') }}
            <span class="card-sub">— {{ t('common.totalCount', { count: total }) }}</span>
          </h3>
          <span v-if="loading" class="spinner"></span>
        </header>

        <div class="table-container">
          <table v-if="items.length" class="grid">
            <!-- 列宽分配:内联 style 写在 col 上最稳(不受 Vue scoped 影响)。
                 name 18% / version 10% / source 12% / project 10% /
                 updated 22% / actions 28% = 100%。
                 之前是 <th style="width:280px"> 固定 280px,前 5 列内容较短,
                 浏览器按内容自动分配时把 actions 列撑到 280px、其他列挤 1/3,
                 看起来像"标题没对齐"。改 fixed 布局 + colgroup 显式分配。 -->
            <colgroup>
              <col style="width: 18%" />
              <col style="width: 10%" />
              <col style="width: 12%" />
              <col style="width: 10%" />
              <col style="width: 22%" />
              <col style="width: 28%" />
            </colgroup>
            <thead>
              <tr>
                <th>{{ t('skills.list.colName') }}</th>
                <th>{{ t('skills.list.colVersion') }}</th>
                <th>{{ t('skills.list.colSource') }}</th>
                <th>{{ t('skills.list.colProject') }}</th>
                <th>{{ t('skills.list.colUpdated') }}</th>
                <th>{{ t('skills.list.colActions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in items" :key="`${p.scope}-${p.project_id}-${p.name}-${p.version}`">
                <td><code class="skill-name">{{ p.name }}</code></td>
                <td><code class="skill-version">{{ p.version }}</code></td>
                <td>
                  <span v-if="p.source === 'market'" class="badge badge-blue">{{ p.source }}</span>
                  <span v-else class="badge badge-gray">{{ p.source }}</span>
                </td>
                <td class="td-dim">{{ p.project_id || t('common.dash') }}</td>
                <td class="td-time">{{ p.updated_at?.slice(0, 19) || t('common.dash') }}</td>
                <td class="row-actions">
                  <button class="action-btn action-btn-apply" :disabled="applying" @click="doApply(p)">
                    <Icon icon="mdi:download" width="12" height="12" />
                    {{ applying ? t('skills.list.applying') : t('skills.list.btnApply') }}
                  </button>
                  <button class="action-btn" :disabled="testing" @click="triggerTest(p)">
                    <Icon icon="mdi:test-tube" width="12" height="12" />
                    {{ testing ? t('skills.list.testing') : t('skills.list.btnTest') }}
                  </button>
                  <button class="action-btn" @click="startEdit(p)">
                    <Icon icon="mdi:pencil" width="12" height="12" />
                    {{ t('common.edit') }}
                  </button>
                  <button class="action-btn" @click="loadTags(p)">
                    <Icon icon="mdi:tag-outline" width="12" height="12" />
                    {{ t('skills.list.btnTag') }}
                  </button>
                  <button class="action-btn action-btn-danger" @click="remove(p)">
                    <Icon icon="mdi:delete" width="12" height="12" />
                    {{ t('common.delete') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-else-if="!loading" class="empty-state">
            <Icon icon="mdi:book-open-variant" width="48" height="48" />
            <p class="empty-title">{{ t('skills.list.emptyTitle') }}</p>
            <p class="empty-hint">{{ t('skills.list.emptyHint') }}</p>
          </div>
        </div>

        <footer v-if="totalPages > 1" class="pager">
          <button :disabled="page <= 1" @click="gotoPage(page - 1)">
            <Icon icon="mdi:chevron-left" width="14" height="14" />
            {{ t('common.prev') }}
          </button>
          <span class="pager-info">{{ page }} / {{ totalPages }} ({{ t('common.totalCount', { count: total }) }})</span>
          <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">
            {{ t('common.next') }}
            <Icon icon="mdi:chevron-right" width="14" height="14" />
          </button>
        </footer>
      </div>
    </section>

    <!-- AI 面板 -->
    <AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" />

    <!-- 技能编辑器弹窗 -->
    <Modal
      v-model="editorOpen"
      size="xl"
      :title="editingKey ? t('skills.editor.titleEdit') : t('skills.editor.titleNew')"
    >
      <template #title-icon>
        <Icon :icon="editingKey ? 'mdi:pencil' : 'mdi:plus'" width="18" height="18" />
      </template>
      <form class="editor-form" @submit.prevent="submit">
        <div v-if="editingKey" class="editor-hint-bar">
          <code>{{ editingKey.name }}@{{ editingKey.version }}</code>
        </div>
        <div class="editor-grid">
          <div class="editor-field">
            <label>{{ t('skills.editor.name') }}</label>
            <input v-model="draft.name" :placeholder="t('skills.editor.nameHint')" :disabled="!!editingKey" />
          </div>
          <div class="editor-field">
            <label>{{ t('skills.editor.version') }}</label>
            <input v-model="draft.version" :placeholder="t('skills.editor.versionHint')" :disabled="!!editingKey" />
          </div>
          <div class="editor-field">
            <label>{{ t('skills.editor.scope') }}</label>
            <select v-model="draft.scope" :disabled="!!editingKey">
              <option value="global">global</option>
              <option value="project">project</option>
            </select>
          </div>
          <div class="editor-field" v-if="draft.scope === 'project'">
            <label>{{ t('skills.editor.projectId') }}</label>
            <input v-model.number="draft.project_id" type="number" min="0" :disabled="!!editingKey" />
          </div>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.description') }} <small>({{ t('skills.editor.descriptionHint') }})</small></label>
          <textarea v-model="draft.description" rows="2"></textarea>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.triggers') }} <small>({{ t('skills.editor.triggersHint') }})</small></label>
          <textarea v-model="draft.triggersText" rows="2" placeholder="review pr&#10;code review"></textarea>
        </div>

        <div class="editor-field-full">
          <label>{{ t('skills.editor.body') }}</label>
          <textarea v-model="draft.body" rows="14" class="code"></textarea>
        </div>

        <p v-if="error" class="message message-error" style="margin: 0 0 12px">
          <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ error }}
        </p>
      </form>
      <template #footer>
        <button type="button" class="ghost" @click="editorOpen = false">
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="primary" @click="submit">
          <Icon :icon="editingKey ? 'mdi:content-save' : 'mdi:plus'" width="14" height="14" />
          {{ editingKey ? t('common.save') : t('common.create') }}
        </button>
      </template>
    </Modal>

    <!-- Tag 管理弹窗 -->
    <Modal
      v-model="tagOpen"
      size="xl"
      :title="selectedSkill ? t('skills.tag.titlePrefix') + ' — ' + selectedSkill.name + '@' + selectedSkill.version : t('skills.tag.titlePrefix')"
    >
      <template #title-icon>
        <Icon icon="mdi:tag-outline" width="18" height="18" />
      </template>

      <p v-if="tagMessage" class="message message-success">
        <Icon icon="mdi:check-circle-outline" width="14" height="14" />
        {{ tagMessage }}
      </p>
      <p v-if="tagError" class="message message-error">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ tagError }}
      </p>

      <div class="tag-create">
        <input v-model="newTagName" :placeholder="t('skills.tag.createPlaceholder')" class="tag-input" />
        <input v-model="newTagMessage" :placeholder="t('skills.tag.msgPlaceholder')" class="tag-input" />
        <button class="primary" :disabled="tagLoading" @click="doCreateTag">
          {{ tagLoading ? t('common.processing') : t('skills.tag.btnCreate') }}
        </button>
      </div>

      <div v-if="tagList.length" class="tag-actions">
        <span class="diff-label">{{ t('skills.tag.diff') }}:</span>
        <select v-model="diffLeftTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :value="tg.tag_id || tg.ID || tg.id">
            {{ tg.tag }} ({{ (tg.created_at || '').slice(0, 16) }}){{ tg.is_implicit ? t('skills.tag.implicit') : '' }}
          </option>
        </select>
        <Icon icon="mdi:arrow-right" width="14" height="14" class="diff-arrow" />
        <select v-model="diffRightTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :value="tg.tag_id || tg.ID || tg.id">
            {{ tg.tag }} ({{ (tg.created_at || '').slice(0, 16) }}){{ tg.is_implicit ? t('skills.tag.implicit') : '' }}
          </option>
        </select>
        <button @click="doDiff(diffLeftTagID, diffRightTagID)">{{ t('skills.tag.seeDiff') }}</button>
        <button @click="doDiff(0, 0)">{{ t('skills.tag.clear') }}</button>
      </div>

      <ul v-if="tagList.length" class="tag-list">
        <li v-for="tg in tagList" :key="tg.tag_id || tg.ID || tg.id" :class="{ 'tag-implicit': tg.is_implicit }">
          <span class="tag-id">#{{ tg.tag_id || tg.ID || tg.id }}</span>
          <span class="tag-name"><code>{{ tg.tag }}</code></span>
          <span class="tag-msg">{{ tg.message || t('common.dash') }}</span>
          <span class="tag-time">{{ (tg.created_at || '').slice(0, 19) }}</span>
          <button class="link" @click="doDiff(tg.tag_id || tg.ID || tg.id, 0)">{{ t('skills.tag.vsCurrent') }}</button>
          <button class="link" :disabled="rolling" @click="doRollback(tg.tag_id || tg.ID || tg.id)">
            {{ rolling ? t('skills.tag.rollingBack') : t('skills.tag.rollbackTo') }}
          </button>
          <button class="link danger" @click="doDeleteTag(tg.tag_id || tg.ID || tg.id)">{{ t('common.delete') }}</button>
        </li>
      </ul>

      <div v-else-if="!tagLoading" class="empty-state empty-state-sm">
        <Icon icon="mdi:tag-off-outline" width="36" height="36" />
        <p class="empty-title">{{ t('common.dash') }}</p>
      </div>

      <!-- Diff 结果 -->
      <div v-if="diffResult" class="diff-panel">
        <header class="diff-header">
          <h4>{{ t('skills.tag.resultTitle') }}</h4>
          <div class="diff-stats">
            <span class="stat stat-added">+{{ t('skills.tag.added', { n: diffResult.added }) }}</span>
            <span class="stat stat-removed">-{{ t('skills.tag.removed', { n: diffResult.removed }) }}</span>
            <span class="stat stat-modified">~{{ t('skills.tag.modified', { n: diffResult.modified }) }}</span>
            <span class="stat stat-unchanged">={{ t('skills.tag.unchanged', { n: diffResult.unchanged }) }}</span>
          </div>
        </header>
        <div v-for="f in diffResult.files" :key="f.path" :class="['diff-file', `diff-kind-${f.kind}`]">
          <div class="diff-file-header">
            <span class="diff-file-kind">{{ f.kind }}</span>
            <code class="diff-file-path">{{ f.path }}</code>
          </div>
          <pre v-if="f.lines?.length" class="diff-content"><span v-for="(l, i) in f.lines" :key="i" :class="`diff-line diff-line-${l.kind}`"><span class="diff-line-no">{{ l.left_no || '' }}|{{ l.right_no || '' }}</span>{{ l.text }}
</span></pre>
        </div>
      </div>
    </Modal>

    <!-- 测试结果弹窗 -->
    <Modal
      v-model="testOpen"
      size="lg"
      :title="t('skills.test.title')"
    >
      <template #title-icon>
        <Icon icon="mdi:test-tube" width="18" height="18" />
      </template>

      <div :class="['test-status-row', `test-status-${lastTest?.run?.status || 'errored'}`]">
        <span v-if="lastTest?.run" class="test-status-badge">{{ lastTest.run.status }}</span>
        <p v-if="testError" class="message message-error" style="margin: 0">
          <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
          {{ t('skills.test.errPrefix') }} {{ testError }}
        </p>
        <p v-else-if="lastTest?.run?.summary" class="test-summary">{{ lastTest.run.summary }}</p>
      </div>

      <ul v-if="lastTest?.results?.length" class="test-list">
        <li v-for="r in lastTest.results" :key="r.id || r.ID" :class="`test-check test-check-${r.status}`">
          <span class="test-check-name">{{ r.check }}</span>
          <span class="test-check-status" :class="`status-${r.status}`">{{ r.status }}</span>
          <span class="test-check-msg">{{ r.message }}</span>
        </li>
      </ul>

      <details v-for="r in lastTest?.results || []" :key="`d-${r.id || r.ID}`" class="test-detail">
        <summary>{{ r.check }} detail</summary>
        <pre>{{ r.detail }}</pre>
      </details>

      <div v-if="testing" class="test-loading">
        <span class="spinner"></span>
        <span>{{ t('common.processing') }}</span>
      </div>
    </Modal>

    <!-- 通用确认弹窗 -->
    <Modal
      v-model="confirmOpen"
      size="sm"
      :title="confirmOpts.title"
      :close-on-mask="false"
    >
      <p class="confirm-message">{{ confirmOpts.message }}</p>
      <template #footer>
        <button type="button" class="ghost" @click="resolveConfirm(false)">
          {{ confirmOpts.cancelText }}
        </button>
        <button
          type="button"
          :class="confirmOpts.variant === 'danger' ? 'danger' : 'primary'"
          @click="resolveConfirm(true)"
        >
          {{ confirmOpts.confirmText }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.skills-layout {
  display: flex;
  gap: 20px;
  height: 100%;
}

.skills-view {
  padding: 0;
  max-width: 1100px;
  margin: 0 auto;
  color: var(--text);
  flex: 1;
  min-width: 0;
  transition: color 0.3s ease;
}

.skills-view.with-ai {
  max-width: none;
}

/* 页面头部 */
.view-header {
  margin-bottom: 24px;
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
  background: var(--text);
  color: var(--bg-card);
  flex-shrink: 0;
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

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.scope-tabs {
  display: flex;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 4px;
  gap: 4px;
}

.scope-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  background: transparent;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s ease;
}

.scope-tab:hover {
  color: var(--text);
  background: var(--bg-hover);
}

.scope-tab-active {
  background: var(--text);
  color: var(--bg-card);
}

.scope-tab-active:hover {
  background: var(--primary-hover);
  color: var(--bg-card);
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-faint);
  pointer-events: none;
}

.search-input {
  padding-left: 36px;
  width: 240px;
  height: 38px;
}

.ai-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.ai-btn:hover {
  background: var(--bg-hover);
  border-color: var(--text-faint);
}

/* Apply 工具栏 */
.apply-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.apply-left, .apply-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.apply-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
}

.apply-select {
  padding: 6px 12px;
  min-width: 100px;
}

.check-updates-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
}

.update-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
}

.update-badge-danger {
  background: var(--danger-dim);
  color: var(--danger);
}

.update-badge-success {
  background: var(--success-dim);
  color: var(--success);
}

/* 消息提示 */
.message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 12px;
}

.message-success {
  background: var(--success-dim);
  color: var(--success);
}

.message-error {
  background: var(--danger-dim);
  color: var(--danger);
}

/* 卡片样式 */
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
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.card-header h3, .card-header h4 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

/* 编辑器表单(弹窗内) */
.editor-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.editor-hint-bar {
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  font-size: 12px;
  color: var(--text-dim);
}

.editor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}

.editor-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.editor-field label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
}

.editor-field-full {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.editor-field-full label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
}

.editor-field-full small {
  color: var(--text-faint);
}

.editor-field-full textarea {
  min-height: 100px;
}

/* 表格 */
.table-container {
  /* 不再做 margin:0 -20px / padding:0 20px 横向溢出;
     由 .card 自身内边距负责,table 在 card 内自然铺满。 */
  overflow-x: auto;
}

.grid {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  /* table-layout: fixed 让 thead 和 tbody 共用 colgroup 给的列宽,
     不会被 <td> 长内容(如 updated_at 19 字符)撑开,标题列和数据列
     永远起始位置一致。 */
  table-layout: fixed;
}

.grid th, .grid td {
  text-align: left;
  padding: 12px 14px;
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

.skill-name {
  font-weight: 600;
  color: var(--text);
}

.skill-version {
  color: var(--text-dim);
}

.td-dim {
  color: var(--text-dim);
}

.td-time {
  color: var(--text-faint);
  font-size: 12px;
}

/* 徽章 */
.badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.badge-blue {
  background: var(--text);
  color: var(--bg-card);
}

.badge-gray {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

/* 操作按钮 */
.row-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  border-color: var(--text-faint);
  color: var(--text);
}

.action-btn-apply {
  background: var(--text);
  border-color: var(--text);
  color: var(--bg-card);
}

.action-btn-apply:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
  color: var(--bg-card);
}

.action-btn-danger:hover:not(:disabled) {
  background: var(--danger-dim);
  border-color: var(--danger);
  color: var(--danger);
}

/* 分页器 */
.pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.pager button {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
}

.pager-info {
  font-size: 13px;
  color: var(--text-dim);
}

/* Apply 历史 */
.apply-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.apply-list li {
  display: grid;
  grid-template-columns: 60px 80px 100px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--border);
  font-size: 13px;
}

.apply-list li:last-child {
  border-bottom: none;
}

.apply-id {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-faint);
}

.apply-tool-name {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text);
}

.apply-status-badge {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  text-align: center;
}

.badge-applied {
  background: var(--success-dim);
  color: var(--success);
}

.badge-rolled_back {
  background: var(--bg-subtle);
  color: var(--text-dim);
}

.badge-failed {
  background: var(--danger-dim);
  color: var(--danger);
}

.apply-time {
  color: var(--text-dim);
  font-size: 12px;
}

/* Tag 面板(弹窗内) */
.tag-count {
  font-size: 12px;
  color: var(--text-dim);
}

.tag-create {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.tag-input {
  flex: 1;
}

.tag-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 13px;
  flex-wrap: wrap;
}

.diff-label {
  color: var(--text-dim);
  font-weight: 500;
}

.diff-arrow {
  color: var(--text-faint);
}

.tag-list {
  list-style: none;
  padding: 0;
  margin: 0;
  border-top: 1px dashed var(--border);
}

.tag-list li {
  display: grid;
  grid-template-columns: 50px 160px 1fr 160px auto auto auto;
  gap: 10px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px dashed var(--border);
  font-size: 13px;
}

.tag-list li.tag-implicit {
  background: var(--bg-subtle);
  margin: 0 -20px;
  padding: 10px 20px;
  border-radius: var(--radius-sm);
  border: 1px dashed var(--border);
  border-bottom: 1px dashed var(--border);
}

.tag-id {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-faint);
}

.tag-name code {
  background: var(--primary-dim);
  color: var(--text);
}

.tag-msg {
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-time {
  color: var(--text-faint);
  font-size: 11px;
}

/* Diff 面板 */
.diff-panel {
  margin-top: 20px;
  padding: 16px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.diff-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 12px;
}

.diff-header h4 {
  margin: 0;
  font-size: 14px;
  color: var(--text);
}

.diff-stats {
  display: flex;
  gap: 8px;
}

.stat {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.stat-added {
  background: var(--success-dim);
  color: var(--success);
}

.stat-removed {
  background: var(--danger-dim);
  color: var(--danger);
}

.stat-modified {
  background: var(--warning-dim);
  color: var(--warning);
}

.stat-unchanged {
  background: var(--bg-card);
  color: var(--text-dim);
}

.diff-file {
  margin: 8px 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}

.diff-kind-added .diff-file-header {
  background: var(--bg-subtle);
  border-left: 3px solid var(--success);
}

.diff-kind-removed .diff-file-header {
  background: var(--bg-subtle);
  border-left: 3px solid var(--danger);
}

.diff-kind-modified .diff-file-header {
  background: var(--bg-subtle);
  border-left: 3px solid var(--warning);
}

.diff-kind-unchanged .diff-file-header {
  background: var(--bg-card);
}

.diff-file-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
}

.diff-file-kind {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--bg-card);
  color: var(--text-dim);
  text-transform: uppercase;
  font-weight: 600;
}

.diff-file-path {
  font-size: 12px;
  color: var(--text);
}

.diff-content {
  padding: 8px 12px;
  margin: 0;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  background: var(--bg-card);
  max-height: 300px;
  overflow: auto;
  white-space: pre;
}

.diff-line {
  display: block;
}

.diff-line-added {
  background: var(--bg-subtle);
  color: var(--text);
  border-left: 3px solid var(--success);
}

.diff-line-removed {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border-left: 3px solid var(--danger);
  text-decoration: line-through;
}

.diff-line-context {
  color: var(--text-dim);
}

.diff-line-no {
  display: inline-block;
  min-width: 40px;
  padding-right: 10px;
  color: var(--text-faint);
  user-select: none;
}

/* 测试结果(弹窗内) */
.test-status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.test-status-row.test-status-passed { color: var(--success); }
.test-status-row.test-status-failed { color: var(--danger); }
.test-status-row.test-status-errored { color: var(--warning); }

.test-status-badge {
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  background: var(--text);
  color: var(--bg-card);
}

.test-summary {
  color: var(--text-dim);
  font-size: 13px;
  margin: 0;
  flex: 1;
  min-width: 0;
}

.test-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.test-list li {
  display: grid;
  grid-template-columns: 140px 90px 1fr;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
  font-size: 13px;
  align-items: center;
}

.test-check-name {
  font-family: 'JetBrains Mono', monospace;
  color: var(--text);
}

.test-check-status {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 600;
  text-align: center;
}

.status-passed {
  background: var(--success-dim);
  color: var(--success);
}

.status-failed {
  background: var(--danger-dim);
  color: var(--danger);
}

.status-errored {
  background: var(--warning-dim);
  color: var(--warning);
}

.status-skipped {
  background: var(--bg-subtle);
  color: var(--text-dim);
}

.test-check-msg {
  color: var(--text-dim);
}

.test-detail {
  margin-top: 8px;
}

.test-detail summary {
  cursor: pointer;
  font-size: 12px;
  color: var(--text-dim);
  padding: 4px 0;
}

.test-detail pre {
  background: var(--bg-subtle);
  padding: 12px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  max-height: 200px;
  overflow: auto;
  margin: 8px 0 0;
}

.test-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 0;
  color: var(--text-dim);
}

/* 确认弹窗 */
.confirm-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
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

.empty-state-sm {
  padding: 24px 16px;
}

.empty-state .empty-icon {
  opacity: 0.5;
  margin-bottom: 12px;
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--text);
  margin: 0 0 4px;
}

.empty-hint {
  font-size: 13px;
  color: var(--text-dim);
  margin: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left, .toolbar-right {
    justify-content: center;
    flex-wrap: wrap;
  }

  .search-input {
    width: 100%;
  }

  .apply-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .table-container {
    margin: 0 -16px;
    padding: 0 16px;
  }

  .grid th, .grid td {
    padding: 10px 8px;
  }

  .row-actions {
    flex-direction: column;
    gap: 4px;
  }
}
</style>
