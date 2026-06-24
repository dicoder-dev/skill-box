<script setup>
// SkillsView - 技能首页(左右布局)
//
// 左侧:技能列表(顶部"新建 / 导入"按钮 + 搜索框 + 列表项,选中态高亮)
// 右侧:选中 skill 的详情
//   - 顶部 toolbar:技能名 + 版本 + 源徽章;右侧操作图标(测试 / 打标签 / 在文件夹打开 / 删除,hover 显示文字)
//   - scope chips:多选,默认"全局"必选;其他取自 listProjects
//   - 标签列表(横向 chips)
//   - 下方渲染 SKILL.md 的 body(markdown 简单自渲染)

import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill } from '@/api/skillbox/skills'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { listProjects } from '@/api/skillbox/projects'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
import AIPanel from '@/components/AIPanel.vue'
import Modal from '@/components/Modal.vue'
import { renderMarkdown } from '@/core/utils/markdown.js'
import { platform } from '@/platform'
import OnboardingImportDialog from '@/components/OnboardingImportDialog.vue'

const { t } = useI18n()

// ====== 列表 + 选中态 ======
const keyword = ref('')
const loading = ref(false)
const error = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 200
const selectedKey = ref(null) // 选中项的 key 字符串

// 当前选中的 skill 详情
const current = ref(null)         // 完整 skill 详情(loadSkill 后填充)
const currentMd = ref('')         // 原始 SKILL.md 全文
const currentBody = ref('')       // extractBody 后的正文
const currentMeta = reactive({ description: '', triggers: [] })
const currentTagList = ref([])    // 当前 skill 的 tag 列表
const currentLoading = ref(false)
const currentError = ref('')

// 内联编辑
const editing = ref(false)        // 是否处于内联编辑态
const editBody = ref('')          // 编辑器内的 body 文本
const editError = ref('')         // 校验错误
const editSaving = ref(false)     // 保存中

function startInlineEdit() {
  if (!current.value) return
  editBody.value = currentBody.value || ''
  editError.value = ''
  editing.value = true
}
function cancelInlineEdit() {
  editing.value = false
  editBody.value = ''
  editError.value = ''
}
async function saveInlineEdit() {
  if (!current.value) return
  editError.value = ''
  editSaving.value = true
  try {
    // 重新拼 SKILL.md(保留 frontmatter,只换 body)
    const newMd = rebuildSkillMd(editBody.value)
    await updateSkill({
      scope: current.value.scope,
      project_id: current.value.project_id,
      name: current.value.name,
      version: current.value.version,
      source: current.value.source || 'local',
      manifest: {
        name: current.value.name,
        version: current.value.version,
        description: currentMeta.description || '',
        triggers: currentMeta.triggers || [],
      },
      files: [{ path: 'SKILL.md', content: newMd }],
    })
    currentMd.value = newMd
    currentBody.value = extractBody(newMd)
    editing.value = false
  } catch (e) {
    editError.value = e?.message || String(e)
  } finally {
    editSaving.value = false
  }
}

// 用现有 frontmatter 重新拼一份 SKILL.md(只替换 body)
function rebuildSkillMd(newBody) {
  const fm = {
    name: current.value?.name || '',
    version: current.value?.version || '',
    description: currentMeta.description || '',
    triggers: currentMeta.triggers || [],
  }
  const yaml = Object.entries(fm)
    .map(([k, v]) => Array.isArray(v)
      ? `${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]`
      : `${k}: ${JSON.stringify(v)}`)
    .join('\n')
  return `---\n${yaml}\n---\n\n${newBody || ''}\n`
}

// 项目列表(scope 多选用)
const projects = ref([])

// scope 多选:默认 ["global"];其他按项目 ID 字符串
const activeScopes = ref(['global'])

// AI 侧栏
const aiOpen = ref(false)
function toggleAI() { aiOpen.value = !aiOpen.value }

function skillKey(p) { return `${p.scope}|${p.project_id || 0}|${p.name}|${p.version}` }

// AI 输入的上下文 = 当前 skill 的 body
const currentSkillMd = computed(() => currentBody.value || '')
function onAIApply(text) {
  const m = text.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  currentBody.value = m ? m[1].trim() : text.trim()
  // 同时把 frontmatter 部分也同步到 currentMeta(若 AI 给了完整 frontmatter)
  const fm = text.match(/^---\n([\s\S]*?)\n---/)
  if (fm) {
    try {
      // 极简 frontmatter 解析:description / triggers
      const block = fm[1]
      const desc = block.match(/description:\s*(.+)/)?.[1]?.replace(/^["']|["']$/g, '')
      const trg = block.match(/triggers:\s*\[([^\]]*)\]/)?.[1]
        ?.split(',').map(s => s.trim().replace(/^["']|["']$/g, '')).filter(Boolean)
      if (desc) currentMeta.description = desc
      if (trg) currentMeta.triggers = trg
    } catch (_) { /* 忽略 AI 输出非标准 frontmatter */ }
  }
}

// ====== 数据加载 ======
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const resp = await listSkills({
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

async function loadProjects() {
  try {
    const resp = await listProjects({ page: 1, size: 200 })
    projects.value = resp?.items || []
  } catch (_) { /* 非关键失败,保持空数组 */ }
}

async function loadCurrent(row) {
  if (!row) return
  currentLoading.value = true
  currentError.value = ''
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
    currentMd.value = md
    currentBody.value = extractBody(md)
    currentMeta.description = c.description || ''
    currentMeta.triggers = c.triggers || []
    current.value = { ...row, _full: full }
    // 同步拉一次 tag 列表,让详情区"标签"chip 有数据
    try {
      const out = await listTags({ skill_id: row.id })
      currentTagList.value = out?.items || []
    } catch (_) { currentTagList.value = [] }
  } catch (e) {
    currentError.value = e?.message || String(e)
    current.value = { ...row }
    currentMd.value = ''
    currentBody.value = ''
  } finally {
    currentLoading.value = false
  }
}

function extractBody(skillmd) {
  const m = skillmd.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  return m ? m[1].trim() : skillmd
}

// 选中列表项
function selectItem(row) {
  // 切换 skill 时清掉内联编辑态,避免把旧 skill 的 editBody 带到新 skill
  if (editing.value) cancelInlineEdit()
  selectedKey.value = skillKey(row)
  loadCurrent(row)
}

// 监听选中 key 变化(支持按 Enter 在搜索结果中跳转)
watch(selectedKey, (k) => {
  if (!k) return
  const row = items.value.find((x) => skillKey(x) === k)
  if (row) loadCurrent(row)
})

// ====== 搜索 / 翻页 ======
function onSearchEnter() {
  page.value = 1
  reload()
}
function gotoPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  reload()
}

// 过滤后的列表(本地关键字过滤,后端只做弱匹配;本地二次过滤可让选中更稳定)
const filteredItems = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return items.value
  return items.value.filter((x) =>
    (x.name || '').toLowerCase().includes(kw) ||
    (x.version || '').toLowerCase().includes(kw))
})

// ====== 渲染后的 markdown HTML ======
const renderedHtml = computed(() => renderMarkdown(currentBody.value))

// ====== Scope chips ======
function toggleScope(key) {
  if (key === 'global') {
    // 全局必选,不允许取消
    if (!activeScopes.value.includes('global')) activeScopes.value.push('global')
    return
  }
  const i = activeScopes.value.indexOf(key)
  if (i >= 0) activeScopes.value.splice(i, 1)
  else activeScopes.value.push(key)
}
function isScopeActive(key) { return activeScopes.value.includes(key) }
function projectLabel(p) { return p.alias ? `${p.alias} · ${p.name}` : (p.name || p.alias || `#${p.ID}`) }

// ====== Tag 弹窗 ======
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

async function openTagDialog() {
  if (!current.value) return
  tagOpen.value = true
  tagList.value = []
  diffResult.value = null
  newTagName.value = ''
  newTagMessage.value = ''
  await loadTagList()
}
async function loadTagList() {
  if (!current.value) return
  tagLoading.value = true
  tagError.value = ''
  try {
    const out = await listTags({ skill_id: current.value.id })
    tagList.value = out?.items || []
    currentTagList.value = tagList.value
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { tagLoading.value = false }
}
async function doCreateTag() {
  if (!current.value) { tagError.value = t('skills.tag.selectFirst'); return }
  if (!newTagName.value.trim()) { tagError.value = t('skills.tag.emptyName'); return }
  tagLoading.value = true
  tagError.value = ''
  try {
    await createTag({ skill_id: current.value.id, tag: newTagName.value.trim(), message: newTagMessage.value })
    newTagName.value = ''
    newTagMessage.value = ''
    tagMessage.value = t('skills.tag.msgCreated')
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { tagLoading.value = false }
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
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
}
async function doDiff(leftID, rightID) {
  if (!current.value) { tagError.value = t('skills.tag.selectFirst'); return }
  try {
    const out = await diffTag({ skill_id: current.value.id, left_tag_id: leftID || 0, right_tag_id: rightID || 0 })
    diffResult.value = out
    diffLeftTagID.value = leftID
    diffRightTagID.value = rightID
  } catch (e) { tagError.value = e?.message || String(e) }
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
    const row = items.value.find((x) => skillKey(x) === selectedKey.value)
    if (row) await loadCurrent(row)
    await loadTagList()
  } catch (e) { tagError.value = e?.message || String(e) }
  finally { rolling.value = false }
}

// 标签 chip 列表(取自 currentTagList,与弹窗共用)
const currentTags = computed(() => currentTagList.value || [])

// ====== 测试弹窗 ======
const testOpen = ref(false)
const testing = ref(false)
const testError = ref('')
const lastTest = ref(null)
async function triggerTest() {
  if (!current.value) return
  const ok = await openConfirm({
    title: t('skills.test.title'),
    message: t('skills.test.confirmRun', { name: current.value.name, version: current.value.version }),
    confirmText: t('skills.list.btnTest'),
  })
  if (!ok) return
  testOpen.value = true
  testing.value = true
  testError.value = ''
  lastTest.value = null
  try {
    const out = await runSkillTest({
      scope: current.value.scope,
      project_id: current.value.project_id,
      name: current.value.name,
      version: current.value.version,
      trigger: 'manual',
    })
    lastTest.value = out
  } catch (e) { testError.value = e?.message || String(e) }
  finally { testing.value = false }
}

// ====== 在文件夹打开 ======
const openError = ref('')
async function openInFolder() {
  if (!current.value) return
  openError.value = ''
  try {
    // 优先用 getSkill 返回的 source_path
    const sp = current.value._full?.canonical?.source_path
      || current.value._full?.source_path
      || ''
    if (!sp) { openError.value = 'no source path'; return }
    // 桌面端用 platform.fs.reveal;Web 端也是同一个实现
    const r = await platform.fs.reveal(sp)
    if (r && r.ok === false && r.fallbackUrl) {
      // Web 端兜底:打开 file://
      platform.platform.openExternal(r.fallbackUrl)
    }
  } catch (e) {
    openError.value = t('skills.list.openFailed', { msg: e?.message || String(e) })
  }
}

// ====== 复制路径 ======
async function copySourcePath() {
  if (!current.value) return
  const sp = current.value._full?.canonical?.source_path
    || current.value._full?.source_path
    || ''
  if (!sp) return
  try {
    await platform.platform.setClipboardText(sp)
  } catch (_) {
    try { await navigator.clipboard.writeText(sp) } catch (_) {}
  }
}

// ====== 新建 / 编辑(简化版:用弹窗) ======
const editorOpen = ref(false)
const draft = reactive({
  scope: 'global', project_id: 0, name: '', version: '0.1.0',
  description: '', triggersText: '', body: '',
})
const editingKey = ref(null)
function startNew() {
  Object.assign(draft, {
    scope: 'global', project_id: 0, name: '', version: '0.1.0',
    description: '', triggersText: '', body: '',
  })
  editingKey.value = null
  error.value = ''
  editorOpen.value = true
}
function buildSkillMd() {
  const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
  const m = {
    name: draft.name, version: draft.version,
    description: draft.description, triggers,
  }
  const yaml = Object.entries(m)
    .map(([k, v]) => Array.isArray(v) ? `${k}: [${v.map((x) => JSON.stringify(x)).join(', ')}]` : `${k}: ${JSON.stringify(v)}`)
    .join('\n')
  return `---\n${yaml}\n---\n\n${draft.body || ''}\n`
}
async function submit() {
  error.value = ''
  if (!draft.name.trim()) { error.value = t('skills.editor.errNameEmpty'); return }
  if (draft.description.trim().length < 10) { error.value = t('skills.editor.errDescShort'); return }
  const triggers = draft.triggersText.split(/[\n,]/).map((s) => s.trim()).filter(Boolean)
  if (triggers.length === 0) { error.value = t('skills.editor.errTriggersEmpty'); return }
  const payload = {
    scope: draft.scope, project_id: draft.project_id,
    name: draft.name, version: draft.version,
    source: 'local',
    manifest: { name: draft.name, version: draft.version, description: draft.description, triggers },
    files: [{ path: 'SKILL.md', content: buildSkillMd() }],
  }
  try {
    if (editingKey.value) await updateSkill(payload)
    else await createSkill(payload)
    editorOpen.value = false
    await reload()
    // 选回刚保存的
    const row = items.value.find((x) => x.name === payload.name && x.version === payload.version)
    if (row) selectItem(row)
  } catch (e) { error.value = e?.message || String(e) }
}

// ====== 删除 ======
async function removeCurrent() {
  if (!current.value) return
  const row = current.value
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('skills.list.confirmDelete', { name: row.name, version: row.version }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteSkill({ scope: row.scope, project_id: row.project_id, name: row.name, version: row.version })
    if (editing.value) cancelInlineEdit()
    current.value = null
    selectedKey.value = null
    await reload()
  } catch (e) { error.value = e?.message || String(e) }
}

// ====== 通用确认弹窗 ======
const confirmOpen = ref(false)
const confirmOpts = reactive({
  title: '', message: '', confirmText: '', cancelText: '', variant: 'default', resolve: null,
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

// 跳转 Onboarding(以弹窗形式打开)
function goOnboarding() {
  importOpen.value = true
}

// 列表项键盘可达性
const listRefs = ref([])
function focusItem(i) {
  const el = listRefs.value[i]
  if (el) { el.focus() }
}

// 导入弹窗
const importOpen = ref(false)
function openImport() { importOpen.value = true }
function onImported() {
  // 导入完成后,刷新列表
  reload()
}

onMounted(() => {
  reload()
  loadProjects()
})
</script>

<template>
  <div class="skills-layout">
    <!-- 左侧:技能列表 -->
    <aside class="skills-pane">
      <!-- 顶部操作栏 -->
      <div class="left-topbar">
        <button class="left-action" :title="t('skills.list.btnNewSkillTitle')" @click="startNew">
          <Icon icon="mdi:plus" width="16" height="16" />
          <span>{{ t('skills.list.btnNewSkill') }}</span>
        </button>
        <button class="left-action" :title="t('skills.list.btnImportSkillTitle')" @click="goOnboarding">
          <Icon icon="mdi:tray-arrow-down" width="16" height="16" />
          <span>{{ t('skills.list.btnImportSkill') }}</span>
        </button>
      </div>

      <!-- 搜索框 -->
      <div class="left-search">
        <Icon icon="mdi:magnify" width="14" height="14" class="search-icon" />
        <input
          v-model="keyword"
          :placeholder="t('skills.searchPlaceholder')"
          class="search-input"
          :title="t('skills.list.searchTitle')"
          @keyup.enter="onSearchEnter"
        />
      </div>

      <p v-if="error" class="left-error">
        <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
        {{ error }}
      </p>

      <!-- 列表 -->
      <ul class="skill-list" role="listbox" :aria-label="t('skills.title')">
        <li
          v-for="(p, i) in filteredItems"
          :key="`${p.scope}-${p.project_id || 0}-${p.name}-${p.version}`"
          :ref="(el) => { if (el) listRefs[i] = el }"
          tabindex="0"
          role="option"
          :aria-selected="selectedKey === skillKey(p)"
          :class="['skill-item', { 'skill-item-active': selectedKey === skillKey(p) }]"
          @click="selectItem(p)"
          @keyup.enter="selectItem(p)"
        >
          <span class="skill-item-bar"></span>
          <div class="skill-item-main">
            <div class="skill-item-head">
              <span class="skill-item-name">{{ p.name }}</span>
              <span class="skill-item-version">@{{ p.version }}</span>
            </div>
            <div class="skill-item-meta">
              <span :class="['badge', p.source === 'market' ? 'blue' : 'gray']">{{ p.source || 'local' }}</span>
              <span v-if="p.scope === 'project'" class="badge violet">{{ p.project_id }}</span>
            </div>
          </div>
        </li>
      </ul>

      <div v-if="!loading && !filteredItems.length" class="skill-list-empty">
        <Icon icon="mdi:book-open-variant" width="28" height="28" />
        <p>{{ t('skills.list.emptyTitle') }}</p>
        <p class="hint">{{ t('skills.list.emptyHint') }}</p>
      </div>

      <div v-if="loading" class="skill-list-loading">
        <span class="spinner"></span>
        <span>{{ t('common.processing') }}</span>
      </div>

      <!-- 翻页 -->
      <footer v-if="totalPages > 1" class="left-pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">
          <Icon icon="mdi:chevron-left" width="12" height="12" />
          {{ t('common.prev') }}
        </button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="12" height="12" />
        </button>
      </footer>
    </aside>

    <!-- 右侧:技能详情 -->
    <section class="detail-pane">
      <!-- 空状态 -->
      <div v-if="!current" class="detail-empty">
        <Icon icon="mdi:cursor-default-click-outline" width="40" height="40" />
        <p class="empty-title">{{ t('skills.list.selectToView') }}</p>
      </div>

      <template v-else>
        <!-- 顶部 toolbar -->
        <header class="detail-toolbar">
          <div class="detail-title-block">
            <div class="detail-title-row">
              <h1 class="detail-name">{{ current.name }}</h1>
              <code class="detail-version">@{{ current.version }}</code>
              <span :class="['badge', current.source === 'market' ? 'blue' : 'gray']">{{ current.source || 'local' }}</span>
            </div>
            <p v-if="currentMeta.description" class="detail-desc">{{ currentMeta.description }}</p>
          </div>

          <div class="detail-actions">
            <button
              class="icon-btn"
              :title="t('skills.list.tooltipTest')"
              :aria-label="t('skills.list.tooltipTest')"
              :disabled="testing"
              @click="triggerTest"
            >
              <span v-if="testing" class="spinner spinner-sm"></span>
              <Icon v-else icon="mdi:test-tube" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :title="t('skills.list.tooltipTag')"
              :aria-label="t('skills.list.tooltipTag')"
              @click="openTagDialog"
            >
              <Icon icon="mdi:tag-outline" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :title="t('skills.list.tooltipOpenFolder')"
              :aria-label="t('skills.list.tooltipOpenFolder')"
              @click="openInFolder"
            >
              <Icon icon="mdi:folder-outline" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :title="t('skills.list.copyPath')"
              :aria-label="t('skills.list.copyPath')"
              @click="copySourcePath"
            >
              <Icon icon="mdi:content-copy" width="16" height="16" />
            </button>
            <button
              class="icon-btn"
              :title="t('skills.list.tooltipDelete')"
              :aria-label="t('skills.list.tooltipDelete')"
              @click="removeCurrent"
            >
              <Icon icon="mdi:delete" width="16" height="16" />
            </button>
            <button
              class="icon-btn ai-btn"
              :title="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
              :aria-label="aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen')"
              @click="toggleAI"
            >
              <Icon :icon="aiOpen ? 'mdi:robot' : 'mdi:robot-outline'" width="16" height="16" />
            </button>
          </div>
        </header>

        <p v-if="openError" class="message message-error">
          <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
          {{ openError }}
        </p>

        <!-- scope chips -->
        <section class="detail-section">
          <header class="section-header">
            <h3>
              <Icon icon="mdi:earth" width="14" height="14" />
              {{ t('skills.list.scopeLabel') }}
            </h3>
          </header>
          <div class="chip-row">
            <button
              :class="['chip', 'chip-global', isScopeActive('global') ? 'chip-active' : '']"
              @click="toggleScope('global')"
            >
              <Icon icon="mdi:earth" width="12" height="12" />
              {{ t('skills.list.scopeGlobalChip') }}
            </button>
            <button
              v-for="p in projects"
              :key="p.ID"
              :class="['chip', 'chip-project', isScopeActive(`p:${p.ID}`) ? 'chip-active' : '']"
              @click="toggleScope(`p:${p.ID}`)"
            >
              <Icon icon="mdi:folder-outline" width="12" height="12" />
              {{ projectLabel(p) }}
            </button>
            <span v-if="!projects.length" class="chip-empty">{{ t('common.dash') }}</span>
          </div>
        </section>

        <!-- 标签列表 -->
        <section class="detail-section">
          <header class="section-header">
            <h3>
              <Icon icon="mdi:tag-outline" width="14" height="14" />
              {{ t('skills.tag.titlePrefix') }}
            </h3>
            <button class="ghost-link" @click="openTagDialog">
              <Icon icon="mdi:plus" width="12" height="12" />
              {{ t('skills.tag.btnCreate') }}
            </button>
          </header>
          <div v-if="currentTags.length" class="chip-row">
            <span v-for="tg in currentTags" :key="tg.tag || tg" class="chip chip-tag">
              <Icon icon="mdi:tag" width="12" height="12" />
              {{ tg.tag || tg }}
            </span>
          </div>
          <p v-else class="section-empty">{{ t('skills.list.tagsEmpty') }}</p>
        </section>

        <!-- 触发词 + 更新时间 -->
        <section v-if="currentMeta.triggers?.length || current.updated_at" class="detail-section detail-meta-row">
          <div v-if="currentMeta.triggers?.length" class="meta-block">
            <span class="meta-label">{{ t('skills.editor.triggers') }}</span>
            <div class="chip-row">
              <span v-for="t in currentMeta.triggers" :key="t" class="chip chip-trigger">
                <Icon icon="mdi:lightning-bolt-outline" width="12" height="12" />
                {{ t }}
              </span>
            </div>
          </div>
          <div v-if="current.updated_at" class="meta-block meta-block-time">
            <span class="meta-label">{{ t('skills.list.colUpdated') }}</span>
            <span class="meta-value">{{ (current.updated_at || '').slice(0, 19) }}</span>
          </div>
        </section>

        <!-- 正文 -->
        <section class="detail-section detail-body">
          <header class="section-header">
            <h3>
              <Icon :icon="editing ? 'mdi:pencil-box-outline' : 'mdi:text-box-outline'" width="14" height="14" />
              {{ editing ? t('skills.list.bodyEditing') : t('skills.list.bodyTitle') }}
            </h3>
            <div v-if="!editing" class="body-actions">
              <button class="ghost-link" :title="t('common.edit')" @click="startInlineEdit">
                <Icon icon="mdi:pencil" width="12" height="12" />
                {{ t('common.edit') }}
              </button>
            </div>
            <div v-else class="body-actions">
              <button class="ghost-link" :disabled="editSaving" @click="cancelInlineEdit">
                <Icon icon="mdi:close" width="12" height="12" />
                {{ t('common.cancel') }}
              </button>
              <button class="ghost-link primary-link" :disabled="editSaving" @click="saveInlineEdit">
                <span v-if="editSaving" class="spinner spinner-sm"></span>
                <Icon v-else icon="mdi:content-save" width="12" height="12" />
                {{ editSaving ? t('common.processing') : t('common.save') }}
              </button>
            </div>
          </header>

          <p v-if="editError" class="message message-error">
            <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
            {{ editError }}
          </p>

          <!-- 编辑态:内联 textarea(Markdown 原文) -->
          <textarea
            v-if="editing"
            v-model="editBody"
            class="md-editor"
            spellcheck="false"
            :placeholder="t('skills.list.bodyEmpty')"
          ></textarea>

          <!-- 查看态:渲染 -->
          <template v-else>
            <div v-if="currentLoading" class="detail-loading">
              <span class="spinner"></span>
              <span>{{ t('common.processing') }}</span>
            </div>
            <p v-else-if="currentError" class="message message-error">
              <Icon icon="mdi:alert-circle-outline" width="12" height="12" />
              {{ currentError }}
            </p>
            <div v-else-if="currentBody" class="md-body" v-html="renderedHtml"></div>
            <p v-else class="section-empty">{{ t('skills.list.bodyEmpty') }}</p>
          </template>
        </section>
      </template>
    </section>

    <!-- AI 侧栏 -->
    <AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" />

    <!-- Tag 弹窗 -->
    <Modal
      v-model="tagOpen"
      size="xl"
      :title="current ? t('skills.tag.titlePrefix') + ' — ' + current.name + '@' + current.version : t('skills.tag.titlePrefix')"
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

    <!-- 编辑弹窗 -->
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

    <!-- 导入技能 弹窗 -->
    <OnboardingImportDialog v-model="importOpen" @imported="onImported" />
  </div>
</template>

<style scoped>
.skills-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  gap: 0;
  /* 占满父级(.content-area)高度,不要 height:100%(在 flex 父中脆弱) */
  flex: 1;
  min-height: 0;
  align-self: stretch;
  color: var(--text);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

/* ============================================
   左侧 - 技能列表面板
   ============================================ */
.skills-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
}

.left-topbar {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
}

.left-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 10px;
  font-size: 13px;
  font-weight: 500;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.left-action:hover {
  background: var(--bg-hover);
  border-color: var(--text-faint);
}

.left-search {
  position: relative;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.left-search .search-icon {
  position: absolute;
  left: 22px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-faint);
  pointer-events: none;
}

.left-search .search-input {
  width: 100%;
  height: 32px;
  padding-left: 30px;
  font-size: 13px;
  background: var(--bg-card);
}

.left-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 8px 12px;
  background: var(--danger-dim);
  color: var(--danger);
  font-size: 12px;
}

.skill-list {
  list-style: none;
  margin: 0;
  padding: 4px 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.skill-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0;
  cursor: pointer;
  transition: background-color 0.12s ease;
  outline: none;
}

.skill-item:hover { background: var(--bg-hover); }
.skill-item:focus-visible { background: var(--bg-hover); box-shadow: inset 0 0 0 1px var(--text-faint); }
.skill-item-active { background: var(--bg-subtle); }
.skill-item-active:hover { background: var(--bg-subtle); }

.skill-item-bar {
  flex-shrink: 0;
  width: 3px;
  align-self: stretch;
  background: transparent;
  margin-right: 8px;
}

.skill-item-active .skill-item-bar { background: var(--primary); }

.skill-item-main {
  flex: 1;
  min-width: 0;
  padding: 8px 12px 8px 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.skill-item-head {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.skill-item-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.skill-item-version {
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  flex-shrink: 0;
}

.skill-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.badge.gray {
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.skill-list-empty,
.skill-list-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--text-faint);
  text-align: center;
}

.skill-list-empty .hint {
  font-size: 12px;
  color: var(--text-faint);
  margin: 0;
}

.skill-list-loading {
  flex-direction: row;
  font-size: 12px;
  color: var(--text-dim);
}

.left-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
  font-size: 12px;
  color: var(--text-dim);
  flex-shrink: 0;
}

.left-pager button {
  padding: 4px 8px;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

/* ============================================
   右侧 - 详情面板
   ============================================ */
.detail-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow-y: auto;
  background: var(--bg);
}

.detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex: 1;
  color: var(--text-faint);
  padding: 60px 20px;
}

.detail-empty .empty-title {
  margin: 0;
  font-size: 14px;
  color: var(--text-dim);
}

.detail-toolbar {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.detail-title-block {
  min-width: 0;
  flex: 1;
}

.detail-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}

.detail-name {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.detail-version {
  font-size: 12px;
  color: var(--text-dim);
  font-family: 'JetBrains Mono', monospace;
  background: var(--primary-dim);
  padding: 2px 6px;
  border-radius: 4px;
}

.detail-desc {
  margin: 6px 0 0;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.detail-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.icon-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
}

.icon-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--border);
}

.icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.icon-btn.ai-btn { color: var(--accent-blue); }
.icon-btn.ai-btn:hover { background: var(--accent-blue-bg); border-color: var(--accent-blue-border); }

/* 让 danger hover 提示删除样式 - 用 :nth-last-child 单独标红 */
.detail-actions .icon-btn[aria-label="删除"]:hover:not(:disabled) {
  background: var(--danger-dim);
  color: var(--danger);
  border-color: var(--danger);
}

.spinner-sm {
  width: 12px;
  height: 12px;
  border-width: 2px;
}

.detail-section {
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-section.detail-meta-row {
  flex-direction: row;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.section-header h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.ghost-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  font-size: 12px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.12s ease;
}

.ghost-link:hover { background: var(--bg-hover); color: var(--text); border-color: var(--border); }
.ghost-link:disabled { opacity: 0.5; cursor: not-allowed; }
.ghost-link.primary-link { color: var(--primary); }
.ghost-link.primary-link:hover { background: var(--primary-dim); }

.body-actions { display: inline-flex; align-items: center; gap: 4px; }

.md-editor {
  display: block;
  width: 100%;
  min-height: 320px;
  padding: 12px 14px;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  resize: vertical;
  transition: border-color 0.12s ease, box-shadow 0.12s ease;
}

.md-editor:focus {
  border-color: var(--text);
  box-shadow: 0 0 0 3px var(--primary-dim);
}

.chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  background: var(--bg-card);
  color: var(--text-dim);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all 0.12s ease;
  user-select: none;
}

.chip:hover { background: var(--bg-hover); color: var(--text); }
.chip-active {
  background: var(--text);
  color: var(--bg-card);
  border-color: var(--text);
}
.chip-active:hover { background: var(--text); color: var(--bg-card); }

.chip-global.chip-active { background: var(--accent-blue); border-color: var(--accent-blue); color: #fff; }
.chip-project.chip-active { background: var(--accent-violet); border-color: var(--accent-violet); color: #fff; }
.chip-tag { cursor: default; }
.chip-tag:hover { background: var(--bg-card); color: var(--text-dim); }
.chip-trigger { background: var(--accent-amber-bg); color: var(--accent-amber); border-color: var(--accent-amber-border); }
.chip-trigger:hover { background: var(--accent-amber-bg); color: var(--accent-amber); }

.chip-empty {
  font-size: 12px;
  color: var(--text-faint);
}

.section-empty {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
}

.meta-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.meta-block-time { min-width: 180px; }
.meta-label { font-size: 11px; color: var(--text-faint); text-transform: uppercase; letter-spacing: 0.3px; }
.meta-value { font-size: 12px; color: var(--text-dim); font-family: 'JetBrains Mono', monospace; }

.detail-body { padding-bottom: 24px; }

.md-body {
  font-size: 13.5px;
  line-height: 1.7;
  color: var(--text);
  word-wrap: break-word;
}

.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3) {
  margin: 16px 0 8px;
  font-weight: 600;
  color: var(--text);
}
.md-body :deep(h1) { font-size: 18px; }
.md-body :deep(h2) { font-size: 16px; }
.md-body :deep(h3) { font-size: 14px; }

.md-body :deep(p) { margin: 8px 0; }
.md-body :deep(ul),
.md-body :deep(ol) { margin: 8px 0 8px 20px; padding: 0; }
.md-body :deep(li) { margin: 2px 0; }

.md-body :deep(code) {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.9em;
  background: var(--primary-dim);
  padding: 1px 5px;
  border-radius: 4px;
}

.md-body :deep(pre) {
  margin: 10px 0;
  padding: 12px 14px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12.5px;
  line-height: 1.6;
}

.md-body :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: inherit;
}

.md-body :deep(a) {
  color: var(--accent-blue);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.md-body :deep(blockquote) {
  margin: 8px 0;
  padding: 6px 12px;
  border-left: 3px solid var(--border);
  color: var(--text-dim);
  background: var(--bg-subtle);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.md-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--border);
  margin: 14px 0;
}

.md-body :deep(table) {
  border-collapse: collapse;
  margin: 10px 0;
  font-size: 12.5px;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid var(--border);
  padding: 6px 10px;
  text-align: left;
}
.md-body :deep(th) { background: var(--bg-subtle); font-weight: 600; }

.detail-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-dim);
  font-size: 13px;
  padding: 12px 0;
}

.message {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin: 0;
}
.message-success { background: var(--success-dim); color: var(--success); }
.message-error { background: var(--danger-dim); color: var(--danger); }

/* ============================================
   Tag 弹窗(沿用原样)
   ============================================ */
.tag-create {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.tag-input { flex: 1; }
.tag-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  font-size: 13px;
  flex-wrap: wrap;
}
.diff-label { color: var(--text-dim); font-weight: 500; }
.diff-arrow { color: var(--text-faint); }

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
.tag-id { font-family: 'JetBrains Mono', monospace; color: var(--text-faint); }
.tag-name code { background: var(--primary-dim); color: var(--text); }
.tag-msg { color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag-time { color: var(--text-faint); font-size: 11px; }

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
.diff-header h4 { margin: 0; font-size: 14px; color: var(--text); }
.diff-stats { display: flex; gap: 8px; }
.stat { padding: 2px 8px; border-radius: 4px; font-size: 11px; font-weight: 600; }
.stat-added { background: var(--success-dim); color: var(--success); }
.stat-removed { background: var(--danger-dim); color: var(--danger); }
.stat-modified { background: var(--warning-dim); color: var(--warning); }
.stat-unchanged { background: var(--bg-card); color: var(--text-dim); }

.diff-file { margin: 8px 0; border: 1px solid var(--border); border-radius: 6px; overflow: hidden; }
.diff-kind-added .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--success); }
.diff-kind-removed .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--danger); }
.diff-kind-modified .diff-file-header { background: var(--bg-subtle); border-left: 3px solid var(--warning); }
.diff-kind-unchanged .diff-file-header { background: var(--bg-card); }
.diff-file-header { display: flex; align-items: center; gap: 10px; padding: 8px 12px; }
.diff-file-kind { font-size: 11px; padding: 2px 6px; border-radius: 4px; background: var(--bg-card); color: var(--text-dim); text-transform: uppercase; font-weight: 600; }
.diff-file-path { font-size: 12px; color: var(--text); }
.diff-content { padding: 8px 12px; margin: 0; font-family: 'JetBrains Mono', monospace; font-size: 12px; line-height: 1.6; background: var(--bg-card); max-height: 300px; overflow: auto; white-space: pre; }
.diff-line { display: block; }
.diff-line-added { background: var(--bg-subtle); color: var(--text); border-left: 3px solid var(--success); }
.diff-line-removed { background: var(--bg-subtle); color: var(--text-dim); border-left: 3px solid var(--danger); text-decoration: line-through; }
.diff-line-context { color: var(--text-dim); }
.diff-line-no { display: inline-block; min-width: 40px; padding-right: 10px; color: var(--text-faint); user-select: none; }

/* ============================================
   测试 / 编辑 / 确认弹窗(沿用原样)
   ============================================ */
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
.test-status-badge { padding: 3px 10px; border-radius: var(--radius-full); font-size: 11px; font-weight: 600; text-transform: uppercase; background: var(--text); color: var(--bg-card); }
.test-summary { color: var(--text-dim); font-size: 13px; margin: 0; flex: 1; min-width: 0; }
.test-list { list-style: none; padding: 0; margin: 0; }
.test-list li { display: grid; grid-template-columns: 140px 90px 1fr; gap: 12px; padding: 8px 0; border-bottom: 1px dashed var(--border); font-size: 13px; align-items: center; }
.test-check-name { font-family: 'JetBrains Mono', monospace; color: var(--text); }
.test-check-status { padding: 2px 8px; border-radius: var(--radius-full); font-size: 11px; font-weight: 600; text-align: center; }
.status-passed { background: var(--success-dim); color: var(--success); }
.status-failed { background: var(--danger-dim); color: var(--danger); }
.status-errored { background: var(--warning-dim); color: var(--warning); }
.status-skipped { background: var(--bg-subtle); color: var(--text-dim); }
.test-check-msg { color: var(--text-dim); }
.test-detail { margin-top: 8px; }
.test-detail summary { cursor: pointer; font-size: 12px; color: var(--text-dim); padding: 4px 0; }
.test-detail pre { background: var(--bg-subtle); padding: 12px; border-radius: var(--radius-sm); font-size: 11px; max-height: 200px; overflow: auto; margin: 8px 0 0; }
.test-loading { display: flex; align-items: center; gap: 10px; padding: 16px 0; color: var(--text-dim); }

.editor-form { display: flex; flex-direction: column; gap: 14px; }
.editor-hint-bar { background: var(--bg-subtle); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 8px 12px; font-size: 12px; color: var(--text-dim); }
.editor-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 14px; }
.editor-field, .editor-field-full { display: flex; flex-direction: column; gap: 6px; }
.editor-field-full small { color: var(--text-faint); }
.editor-field label, .editor-field-full label { font-size: 12px; font-weight: 500; color: var(--text-dim); }
.editor-field-full textarea { min-height: 100px; }

.confirm-message { margin: 0; font-size: 14px; line-height: 1.6; color: var(--text); white-space: pre-line; }

.empty-state { padding: 48px 24px; text-align: center; color: var(--text-faint); background: var(--bg-subtle); border: 1px dashed var(--border); border-radius: var(--radius); }
.empty-state-sm { padding: 24px 16px; }

/* ============================================
   响应式
   ============================================ */
@media (max-width: 900px) {
  .skills-layout { grid-template-columns: 280px minmax(0, 1fr); }
}

@media (max-width: 720px) {
  .skills-layout {
    grid-template-columns: 1fr;
    grid-template-rows: 240px minmax(0, 1fr);
  }
  .skills-pane { border-right: none; border-bottom: 1px solid var(--border); }
}
</style>
