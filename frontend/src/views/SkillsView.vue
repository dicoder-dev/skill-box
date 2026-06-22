<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill } from '@/api/skillbox/skills'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { applySkill, undoApply, listApplies, checkUpdates } from '@/api/skillbox/skill_apply'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
import AIPanel from '@/components/AIPanel.vue'

const { t } = useI18n()

// 当前 scope 选择
const scope = ref('global') // global | project
const keyword = ref('')
const loading = ref(false)
const error = ref('')
const items = ref([])
const total = ref(0)
const page = ref(1)
const size = 10

// 编辑器状态
const editing = ref(false)
const draft = reactive({
  scope: 'global',
  project_id: 0,
  name: '',
  version: '0.1.0',
  description: '',
  triggersText: '', // 用换行 / 逗号分隔,提交时转数组
  body: '', // SKILL.md body
})
const editingKey = ref(null) // {scope, name, version, project_id}

// Apply / 撤销 / 更新检测
const TOOL_OPTIONS = ['codex', 'claude', 'opencode', 'cursor', 'trae']
const applyTool = ref('codex') // 当前要 apply 的目标工具
const applying = ref(false)
const applyMessage = ref('')
const applyError = ref('')
const lastApplies = ref([]) // 最近一次 apply 的结果
const undoing = ref(false)
const updating = ref(false)
const updateBadge = ref({ total: 0, updates: 0 }) // 来自 checkUpdates 的概览
const applyHistory = ref([]) // apply/list 数据

// Tag / Diff / Rollback
const tagList = ref([])
const tagLoading = ref(false)
const tagError = ref('')
const tagMessage = ref('')
const newTagName = ref('')
const newTagMessage = ref('')
const diffResult = ref(null) // { files: [], added, removed, modified, unchanged }
const diffLeftTagID = ref(0)
const diffRightTagID = ref(0)
const rolling = ref(false)
const selectedSkill = ref(null) // 当前查看 tag 的 skill

async function loadTags(row) {
  selectedSkill.value = row
  tagLoading.value = true
  tagError.value = ''
  try {
    const out = await listTags({ skill_id: row.ID })
    tagList.value = out?.items || []
  } catch (e) {
    tagError.value = e?.message || String(e)
  } finally {
    tagLoading.value = false
  }
}

async function doCreateTag() {
  if (!selectedSkill.value) { tagError.value = '先选一个 skill'; return }
  if (!newTagName.value.trim()) { tagError.value = 'tag 名不能为空'; return }
  tagLoading.value = true
  tagError.value = ''
  try {
    await createTag({
      skill_id: selectedSkill.value.ID,
      tag: newTagName.value.trim(),
      message: newTagMessage.value,
    })
    newTagName.value = ''
    newTagMessage.value = ''
    tagMessage.value = `已打 tag`
    await loadTags(selectedSkill.value)
  } catch (e) {
    tagError.value = e?.message || String(e)
  } finally {
    tagLoading.value = false
  }
}

async function doDeleteTag(tagID) {
  if (!confirm(`删除 tag #${tagID}?file_snapshots 也会一起删。`)) return
  try {
    await deleteTag({ tag_id: tagID })
    tagMessage.value = `已删除 tag #${tagID}`
    await loadTags(selectedSkill.value)
  } catch (e) {
    tagError.value = e?.message || String(e)
  }
}

async function doDiff(leftID, rightID) {
  if (!selectedSkill.value) { tagError.value = '先选一个 skill'; return }
  try {
    const out = await diffTag({
      skill_id: selectedSkill.value.ID,
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
  if (!confirm(`回滚到 tag #${tagID}?会自动打一个 _pre_rollback 隐式 tag,当前状态不会丢失。`)) return
  rolling.value = true
  tagError.value = ''
  try {
    const out = await rollbackTag({ tag_id: tagID })
    tagMessage.value = `已回滚(自动打 ${out.pre_rollback_tag},恢复 ${out.files_restored} 个文件)`
    diffResult.value = null
    await reload()
    if (selectedSkill.value) await loadTags(selectedSkill.value)
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
      skill_id: row.ID,
      scope: row.Scope,
      project_id: row.ProjectID,
      tools: [applyTool.value],
    })
    lastApplies.value = out?.applies || []
    applyMessage.value = out?.all_ok
      ? `已把 ${row.Name}@${row.Version} 落到 ${applyTool.value}`
      : `部分失败: ${(out?.applies || []).filter(a => a?.status !== 'applied').map(a => a?.error || a?.status).join('; ')}`
    await loadApplyHistory(row)
    await checkUpdateBadge()
  } catch (e) {
    applyError.value = e?.message || String(e)
  } finally {
    applying.value = false
  }
}

async function doUndo(applyID) {
  if (!confirm(`撤销 apply #${applyID}?将恢复目标目录到 apply 之前的状态。`)) return
  undoing.value = true
  applyError.value = ''
  try {
    await undoApply({ apply_id: applyID })
    applyMessage.value = `已撤销 apply #${applyID}`
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
    const out = await listApplies({ skill_id: row.ID, page: 1, size: 5 })
    applyHistory.value = out?.items || []
  } catch (e) {
    // 静默失败,不影响主流程
  }
}

async function checkUpdateBadge() {
  updating.value = true
  try {
    const out = await checkUpdates({})
    updateBadge.value = { total: out?.total || 0, updates: out?.updates || 0 }
  } catch (e) {
    // 静默
  } finally {
    updating.value = false
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

// Skill 测试
const testing = ref(false)
const testError = ref('')
const lastTest = ref(null) // { run, results }

// AI 侧栏
const aiOpen = ref(false)
function toggleAI() { aiOpen.value = !aiOpen.value }

// 当前正在编辑的 SKILL.md 全文(供 AIPanel 的 contextText 用)
const currentSkillMd = computed(() => {
  if (!editing.value) return ''
  // 用 buildSkillMd 拼一份(仅在需要时)
  try { return buildSkillMd() } catch (_) { return '' }
})

function onAIApply(text) {
  // 把 AI 改写后的 markdown 提取 body 回填
  const m = text.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  draft.body = m ? m[1].trim() : text.trim()
}

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
  editing.value = true
}

async function startEdit(row) {
  error.value = ''
  try {
    const full = await getSkill({
      scope: row.Scope,
      project_id: row.ProjectID,
      name: row.Name,
      version: row.Version,
      full: true,
    })
    const c = full?.canonical?.manifest || {}
    const files = full?.canonical?.files || []
    const md = files.find((f) => f.path === 'SKILL.md')?.content || ''
    Object.assign(draft, {
      scope: row.Scope,
      project_id: row.ProjectID,
      name: row.Name,
      version: row.Version,
      description: c.description || '',
      triggersText: (c.triggers || []).join('\n'),
      body: extractBody(md),
    })
    editingKey.value = { scope: row.Scope, name: row.Name, version: row.Version, project_id: row.ProjectID }
    editing.value = true
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

function extractBody(skillmd) {
  // 去掉 frontmatter,只留 body
  const m = skillmd.match(/^---\n[\s\S]*?\n---\n?([\s\S]*)$/)
  return m ? m[1].trim() : skillmd
}

function buildSkillMd() {
  // 简单拼:frontmatter 用 manifest 渲染,body 留用户原文
  // 注:Step 4 暂不接 RenderSkillMD,只在本视图里手动拼。
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
    error.value = 'name 不能为空'
    return
  }
  if (draft.description.trim().length < 10) {
    error.value = 'description 至少 10 个字符'
    return
  }
  const triggers = draft.triggersText
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (triggers.length === 0) {
    error.value = 'triggers 至少填一个'
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
    editing.value = false
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

async function triggerTest(row) {
  if (!confirm(`对 skill "${row.Name}@${row.Version}" 跑一次测试?(static + script + ai)`)) return
  testing.value = true
  testError.value = ''
  try {
    const out = await runSkillTest({
      scope: row.Scope,
      project_id: row.ProjectID,
      name: row.Name,
      version: row.Version,
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
  if (!confirm(`确定删除 skill "${row.Name}@${row.Version}" ?`)) return
  try {
    await deleteSkill({
      scope: row.Scope,
      project_id: row.ProjectID,
      name: row.Name,
      version: row.Version,
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
  <div class="skills-layout"><section class="skills-view" :class="{ 'with-ai': aiOpen }">
    <header class="head">
      <h2 class="flex items-center gap-2">
        <Icon icon="mdi:book-open-variant" width="20" height="20" class="text-sb-primary" />
        {{ t('skills.title') }}
      </h2>
      <p class="muted">{{ t('skills.subtitle') }}</p>
    </header>

    <div class="bar">
      <div class="tabs flex">
        <button :class="['px-4 py-1.5 border border-sb-border text-[13px] transition-colors flex items-center gap-1.5', scope === 'global' ? 'bg-sb-primary text-white border-sb-primary' : 'bg-white text-sb-dim hover:bg-gray-50']" @click="switchScope('global')">
          <Icon icon="mdi:earth" width="14" height="14" />{{ t('skills.scopeGlobal') }}
        </button>
        <button :class="['px-4 py-1.5 border border-sb-border border-l-0 text-[13px] transition-colors rounded-r flex items-center gap-1.5', scope === 'project' ? 'bg-sb-primary text-white border-sb-primary' : 'bg-white text-sb-dim hover:bg-gray-50']" @click="switchScope('project')">
          <Icon icon="mdi:folder-outline" width="14" height="14" />{{ t('skills.scopeProject') }}
        </button>
      </div>
      <div class="search flex gap-1.5 md:ml-auto">
        <input
          v-model="keyword"
          :placeholder="t('skills.searchPlaceholder')"
          class="w-32 md:w-56"
          @keyup.enter="() => { page = 1; reload() }"
        />
        <button @click="() => { page = 1; reload() }">{{ t('common.search') }}</button>
      </div>
      <div class="actions flex gap-1.5">
        <button @click="toggleAI" class="flex items-center gap-1.5">
          <Icon icon="mdi:robot-outline" width="14" height="14" />
          {{ aiOpen ? t('skills.btnAiClose') : t('skills.btnAiOpen') }}
        </button>
        <button class="primary" @click="startNew">{{ t('skills.btnNew') }}</button>
      </div>
    </div>

    <div class="apply-bar flex flex-wrap items-center gap-2.5 mb-3.5 px-3.5 py-2 bg-white border border-sb-border rounded text-[13px]">
      <span class="text-sb-dim font-medium">{{ t('skills.applyBar.target') }}</span>
      <select v-model="applyTool" class="!py-1">
        <option v-for="t in TOOL_OPTIONS" :key="t" :value="t">{{ t }}</option>
      </select>
      <button class="sm flex items-center gap-1.5" @click="checkUpdateBadge" :disabled="updating">
        <span v-if="updating" class="spinner"></span>
        <Icon v-else icon="mdi:refresh" width="14" height="14" />
        {{ updating ? t('skills.applyBar.checking') : t('skills.applyBar.checkUpdates') }}
      </button>
      <span v-if="updateBadge.updates > 0" class="px-2 py-0.5 rounded-full text-[12px] font-medium bg-sb-danger-dim text-sb-danger inline-flex items-center gap-1">
        <Icon icon="mdi:alert-circle-outline" width="12" height="12" />{{ t('skills.applyBar.updatesAvailable', { updates: updateBadge.updates, total: updateBadge.total }) }}
      </span>
      <span v-else-if="updateBadge.total > 0" class="px-2 py-0.5 rounded-full text-[12px] font-medium bg-sb-success-dim text-sb-success inline-flex items-center gap-1">
        <Icon icon="mdi:check-circle-outline" width="12" height="12" />{{ t('skills.applyBar.allUpToDate', { total: updateBadge.total }) }}
      </span>
      <p v-if="applyMessage" class="text-sb-success m-0 text-[12px] basis-full">{{ applyMessage }}</p>
      <p v-if="applyError" class="text-sb-danger m-0 text-[12px] basis-full">{{ applyError }}</p>
    </div>

    <form v-if="editing" class="card editor" @submit.prevent="submit">
      <h3>{{ editingKey ? t('skills.editor.titleEdit') : t('skills.editor.titleNew') }}
        <span v-if="editingKey" class="card-sub"><code>{{ editingKey.name }}@{{ editingKey.version }}</code></span>
      </h3>
      <div class="row">
        <label>
          <span>{{ t('skills.editor.name') }}</span>
          <input v-model="draft.name" :placeholder="t('skills.editor.nameHint')" :disabled="!!editingKey" />
        </label>
        <label>
          <span>{{ t('skills.editor.version') }}</span>
          <input v-model="draft.version" :placeholder="t('skills.editor.versionHint')" :disabled="!!editingKey" />
        </label>
        <label>
          <span>{{ t('skills.editor.scope') }}</span>
          <select v-model="draft.scope" :disabled="!!editingKey">
            <option value="global">global</option>
            <option value="project">project</option>
          </select>
        </label>
        <label v-if="draft.scope === 'project'">
          <span>{{ t('skills.editor.projectId') }}</span>
          <input v-model.number="draft.project_id" type="number" min="0" :disabled="!!editingKey" />
        </label>
      </div>
      <label class="full">
        <span>{{ t('skills.editor.description') }} <small>({{ t('skills.editor.descriptionHint') }})</small></span>
        <textarea v-model="draft.description" rows="2" />
      </label>
      <label class="full">
        <span>{{ t('skills.editor.triggers') }} <small>({{ t('skills.editor.triggersHint') }})</small></span>
        <textarea v-model="draft.triggersText" rows="2" placeholder="review pr&#10;code review" />
      </label>
      <label class="full">
        <span>{{ t('skills.editor.body') }}</span>
        <textarea v-model="draft.body" rows="14" class="code" />
      </label>
      <div class="actions">
        <button type="button" class="ghost" @click="editing = false">{{ t('common.cancel') }}</button>
        <button type="submit" class="primary">{{ editingKey ? t('common.save') : t('common.create') }}</button>
      </div>
    </form>

    <div v-if="applyHistory.length" class="card apply-history">
      <header class="ah-head">
        <h3>{{ t('skills.applyHistory.title') }}</h3>
        <span class="card-sub">{{ t('skills.applyHistory.count', { count: applyHistory.length }) }}</span>
      </header>
      <ul>
        <li v-for="h in applyHistory" :key="h.ID || h.id" :class="`status-${h.Status}`">
          <span class="ah-id">#{{ h.ID || h.id }}</span>
          <span class="ah-tool">{{ h.Tool }}</span>
          <span class="ah-status">{{ h.Status }}</span>
          <span class="ah-time">{{ h.AppliedAt?.slice(0, 19) || t('common.dash') }}</span>
          <button v-if="h.Status === 'applied'" class="link danger" :disabled="undoing" @click="doUndo(h.ID || h.id)">{{ undoing ? t('skills.applyHistory.undoing') : t('skills.applyHistory.undone') }}</button>
        </li>
      </ul>
    </div>

    <div v-if="selectedSkill" class="card tag-panel">
      <header class="tp-head">
        <h4>{{ t('skills.tag.titlePrefix') }} — <code>{{ selectedSkill.Name }}@{{ selectedSkill.Version }}</code></h4>
        <span class="tp-count">{{ t('skills.tag.count', { count: tagList.length }) }}</span>
        <button class="link" @click="selectedSkill = null; tagList = []; diffResult = null">{{ t('common.close') }}</button>
      </header>
      <p v-if="tagMessage" class="tag-msg">{{ tagMessage }}</p>
      <p v-if="tagError" class="error">{{ tagError }}</p>

      <div class="tag-create">
        <input v-model="newTagName" :placeholder="t('skills.tag.createPlaceholder')" class="tag-input" />
        <input v-model="newTagMessage" :placeholder="t('skills.tag.msgPlaceholder')" class="tag-input" />
        <button class="primary" :disabled="tagLoading" @click="doCreateTag">{{ tagLoading ? t('common.processing') : t('skills.tag.btnCreate') }}</button>
      </div>

      <div v-if="tagList.length" class="tag-actions">
        <span class="tag-label">{{ t('skills.tag.diff') }}:</span>
        <select v-model="diffLeftTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="t in tagList" :key="t.ID || t.id" :value="t.ID || t.id">{{ t.Tag }} ({{ (t.CreatedAt || '').slice(0, 16) }}){{ t.IsImplicit ? t('skills.tag.implicit') : '' }}</option>
        </select>
        <span>→</span>
        <select v-model="diffRightTagID">
          <option :value="0">{{ t('skills.tag.current') }}</option>
          <option v-for="t in tagList" :key="t.ID || t.id" :value="t.ID || t.id">{{ t.Tag }} ({{ (t.CreatedAt || '').slice(0, 16) }}){{ t.IsImplicit ? t('skills.tag.implicit') : '' }}</option>
        </select>
        <button @click="doDiff(diffLeftTagID, diffRightTagID)">{{ t('skills.tag.seeDiff') }}</button>
        <button @click="doDiff(0, 0)">{{ t('skills.tag.clear') }}</button>
      </div>

      <ul v-if="tagList.length" class="tag-list">
        <li v-for="t in tagList" :key="t.ID || t.id" :class="{ implicit: t.IsImplicit }">
          <span class="t-id">#{{ t.ID || t.id }}</span>
          <span class="t-name"><code>{{ t.Tag }}</code></span>
          <span class="t-msg">{{ t.Message || t('common.dash') }}</span>
          <span class="t-time">{{ (t.CreatedAt || '').slice(0, 19) }}</span>
          <button class="link" @click="doDiff(t.ID || t.id, 0)">{{ t('skills.tag.vsCurrent') }}</button>
          <button class="link" :disabled="rolling" @click="doRollback(t.ID || t.id)">{{ rolling ? t('skills.tag.rollingBack') : t('skills.tag.rollbackTo') }}</button>
          <button class="link danger" @click="doDeleteTag(t.ID || t.id)">{{ t('common.delete') }}</button>
        </li>
      </ul>

      <div v-if="diffResult" class="diff-panel">
        <header class="dp-head">
          <h4>{{ t('skills.tag.resultTitle') }}</h4>
          <span class="dp-stats">
            <span class="added">{{ t('skills.tag.added', { n: diffResult.added }) }}</span>
            <span class="removed">{{ t('skills.tag.removed', { n: diffResult.removed }) }}</span>
            <span class="modified">{{ t('skills.tag.modified', { n: diffResult.modified }) }}</span>
            <span class="unchanged">{{ t('skills.tag.unchanged', { n: diffResult.unchanged }) }}</span>
          </span>
        </header>
        <div v-for="f in diffResult.files" :key="f.path" class="diff-file" :class="`kind-${f.kind}`">
          <div class="df-head">
            <span class="df-kind">{{ f.kind }}</span>
            <code class="df-path">{{ f.path }}</code>
          </div>
          <pre v-if="f.lines?.length"><span v-for="(l, i) in f.lines" :key="i" :class="`ln-${l.kind}`"><span class="ln-no">{{ l.left_no || '' }}|{{ l.right_no || '' }}</span>{{ l.text }}
</span></pre>
        </div>
      </div>
    </div>

    <div v-if="lastTest || testError" class="card test-panel" :class="`status-${(lastTest?.run?.status || 'errored')}`">
      <header class="tp-head">
        <h3>最近测试结果</h3>
        <span v-if="lastTest?.run" class="tp-status">{{ lastTest.run.status }}</span>
      </header>
      <p v-if="testError" class="error">测试失败: {{ testError }}</p>
      <p v-else-if="lastTest?.run?.summary" class="tp-summary">{{ lastTest.run.summary }}</p>
      <ul v-if="lastTest?.results?.length" class="tp-list">
        <li v-for="r in lastTest.results" :key="r.ID || r.id" :class="`check-${r.Status}`">
          <span class="check-name">{{ r.Check }}</span>
          <span class="check-status">{{ r.Status }}</span>
          <span class="check-msg">{{ r.Message }}</span>
        </li>
      </ul>
      <details v-for="r in lastTest?.results || []" :key="`d-${r.ID || r.id}`" class="tp-detail">
        <summary>{{ r.Check }} 详情</summary>
        <pre>{{ r.Detail }}</pre>
      </details>
    </div>

    <p v-if="error" class="error inline-flex items-center gap-1.5">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />{{ error }}
    </p>

    <div class="card">
      <h3>技能列表
        <span class="card-sub">— 共 {{ total }} 条</span>
        <span v-if="loading" class="spinner ml-auto"></span>
      </h3>

      <div class="overflow-x-auto -mx-4 px-4">
      <table v-if="items.length" class="grid">
        <thead>
          <tr>
            <th>Name</th>
            <th>Version</th>
            <th>Source</th>
            <th>Project</th>
            <th>Updated</th>
            <th style="width: 260px">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in items" :key="`${p.Scope}-${p.ProjectID}-${p.Name}-${p.Version}`">
            <td><code>{{ p.Name }}</code></td>
            <td><code>{{ p.Version }}</code></td>
            <td>
              <span v-if="p.Source === 'market'" class="badge market">market</span>
              <span v-else class="badge local">{{ p.Source }}</span>
            </td>
            <td>{{ p.ProjectID || '—' }}</td>
            <td class="time">{{ p.UpdatedAt?.slice(0, 19) || '—' }}</td>
            <td class="row-actions">
              <button class="link primary-link" :disabled="applying" @click="doApply(p)">{{ applying ? '应用中…' : '应用' }}</button>
              <button class="link" :disabled="testing" @click="triggerTest(p)">{{ testing ? '测试中…' : '测试' }}</button>
              <button class="link" @click="startEdit(p)">编辑</button>
              <button class="link" @click="loadTags(p)">Tag</button>
              <button class="link danger" @click="remove(p)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else-if="!loading" class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:book-open-variant" width="36" height="36" />
        </span>
        <p style="margin: 8px 0 4px">该 scope 下还没有 skill</p>
        <p class="muted" style="margin: 0">点右上角"+ 新建 Skill"开始,或去 Onboarding 从已装工具导入</p>
      </div>

      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="page <= 1" @click="gotoPage(page - 1)">上一页</button>
        <span>{{ page }} / {{ totalPages }} (共 {{ total }} 条)</span>
        <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">下一页</button>
      </footer>
    </div>
  </section><AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" /></div>
</template>

<style scoped>
.skills-layout { display: flex; height: 100%; }
.skills-view { padding: 0; max-width: 1100px; margin: 0 auto; color: var(--text); flex: 1; min-width: 0; }
.skills-view.with-ai { max-width: none; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }
.bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.tabs { display: flex; gap: 0; }
.tabs button { border-radius: 0; }
.tabs button:first-child { border-top-left-radius: 4px; border-bottom-left-radius: 4px; }
.tabs button:last-child { border-top-right-radius: 4px; border-bottom-right-radius: 4px; border-left: none; }
.tabs button.active { background: var(--primary); color: #fff; border-color: var(--primary); }
.search { margin-left: auto; display: flex; gap: 6px; }
.search input { width: 220px; }
.bar .actions { display: flex; gap: 6px; }
.editor { display: flex; flex-direction: column; gap: 10px; }
.editor .row { display: grid; grid-template-columns: 1.4fr 0.8fr 0.8fr 0.8fr; gap: 10px 14px; }
.editor label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--text-dim); }
.editor label.full { width: 100%; }
.editor label small { color: var(--text-faint); }
.editor .actions { display: flex; gap: 8px; justify-content: flex-end; }

.test-panel.status-passed { border-color: #bbf7d0; background: #f0fdf4; }
.test-panel.status-failed { border-color: #fecaca; background: #fef2f2; }
.test-panel.status-errored { border-color: #fde68a; background: #fffbeb; }
.test-panel.status-skipped { background: #f9fafb; }
.tp-head { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.tp-head h3 { margin: 0; font-size: 14px; }
.tp-status { font-size: 11px; padding: 2px 8px; border-radius: 10px; background: #1f2937; color: #fff; text-transform: uppercase; font-weight: 500; }
.tp-summary { color: var(--text-dim); font-size: 13px; margin: 4px 0 8px; }
.tp-list { list-style: none; padding: 0; margin: 0; }
.tp-list li { display: grid; grid-template-columns: 100px 90px 1fr; gap: 8px; padding: 4px 0; border-bottom: 1px dashed var(--border); font-size: 13px; }
.tp-list .check-name { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text); }
.tp-list .check-status { font-size: 11px; padding: 1px 6px; border-radius: 3px; text-align: center; }
.tp-list .check-passed .check-status { background: var(--success-dim); color: var(--success); }
.tp-list .check-failed .check-status { background: var(--danger-dim); color: var(--danger); }
.tp-list .check-errored .check-status { background: var(--warning-dim); color: var(--warning); }
.tp-list .check-skipped .check-status { background: #f3f4f6; color: var(--text-dim); }
.tp-list .check-msg { color: var(--text-dim); }
.tp-detail { margin-top: 6px; }
.tp-detail summary { cursor: pointer; font-size: 12px; color: var(--text-dim); }
.tp-detail pre { background: #f3f4f6; padding: 6px 8px; border-radius: var(--radius-sm); font-size: 11px; max-height: 200px; overflow: auto; }

.apply-bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-bottom: 14px; padding: 8px 14px; background: #fff; border: 1px solid var(--border); border-radius: var(--radius); font-size: 13px; }
.apply-label { color: var(--text-dim); font-weight: 500; }
.apply-bar select { padding: 3px 6px; }
.update-badge { padding: 3px 9px; border-radius: 10px; font-size: 12px; font-weight: 500; }
.update-badge.danger { background: var(--danger-dim); color: var(--danger); }
.update-badge.ok { background: var(--success-dim); color: var(--success); }
.apply-msg { color: var(--success); margin: 0; font-size: 12px; width: 100%; }
.apply-bar p.error { margin: 0; font-size: 12px; width: 100%; }

.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { text-align: left; padding: 8px 10px; border-bottom: 1px solid #f3f4f6; }
.grid th { background: #f9fafb; color: var(--text-dim); font-weight: 600; }
.grid .time { color: var(--text-dim); font-size: 12px; }
.row-actions { white-space: nowrap; }
.badge { display: inline-block; padding: 1px 8px; border-radius: 10px; font-size: 11px; }
.badge.market { background: #dbeafe; color: #1e40af; }
.badge.local { background: #f3f4f6; color: var(--text-dim); }

.pager { display: flex; align-items: center; gap: 12px; margin-top: 12px; font-size: 13px; color: var(--text-dim); justify-content: flex-end; }

.apply-history .ah-head { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.apply-history .ah-head h3 { margin: 0; }
.apply-history ul { list-style: none; padding: 0; margin: 0; }
.apply-history li { display: grid; grid-template-columns: 60px 80px 100px 1fr auto; gap: 8px; align-items: center; padding: 5px 0; border-bottom: 1px dashed var(--border); font-size: 13px; }
.ah-id { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text-dim); }
.ah-tool { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text); }
.ah-status { font-size: 11px; padding: 1px 8px; border-radius: 10px; text-align: center; font-weight: 500; }
.status-applied .ah-status { background: var(--success-dim); color: var(--success); }
.status-rolled_back .ah-status { background: #f3f4f6; color: var(--text-dim); }
.status-failed .ah-status { background: var(--danger-dim); color: var(--danger); }
.ah-time { color: var(--text-dim); font-size: 12px; }

.tag-panel .tp-head { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.tag-panel .tp-head h3 { margin: 0; font-size: 14px; }
.tag-panel .tp-count { color: var(--text-dim); font-size: 12px; }
.tag-msg { color: var(--success); font-size: 12px; margin: 4px 0; }
.tag-create { display: flex; gap: 8px; margin-bottom: 8px; }
.tag-input { flex: 1; }
.tag-actions { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; font-size: 13px; }
.tag-actions .tag-label { color: var(--text-dim); }
.tag-list { list-style: none; padding: 0; margin: 0; border-top: 1px dashed var(--border); }
.tag-list li { display: grid; grid-template-columns: 50px 160px 1fr 160px auto auto auto; gap: 8px; align-items: center; padding: 6px 0; border-bottom: 1px dashed var(--border); font-size: 13px; }
.tag-list li.implicit { background: var(--warning-dim); }
.tag-list .t-id { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: var(--text-dim); }
.tag-list .t-msg { color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag-list .t-time { color: var(--text-dim); font-size: 11px; }
.diff-panel { margin-top: 12px; padding: 10px 12px; border: 1px solid var(--border); border-radius: var(--radius-sm); background: #fff; }
.dp-head { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.dp-head h4 { margin: 0; font-size: 13px; }
.dp-stats { display: flex; gap: 8px; font-size: 12px; }
.dp-stats .added { color: var(--success); background: var(--success-dim); padding: 1px 6px; border-radius: 3px; }
.dp-stats .removed { color: var(--danger); background: var(--danger-dim); padding: 1px 6px; border-radius: 3px; }
.dp-stats .modified { color: var(--warning); background: var(--warning-dim); padding: 1px 6px; border-radius: 3px; }
.dp-stats .unchanged { color: var(--text-dim); }
.diff-file { margin: 6px 0; border: 1px solid #f3f4f6; border-radius: 4px; overflow: hidden; }
.diff-file.kind-added .df-head { background: var(--success-dim); }
.diff-file.kind-removed .df-head { background: var(--danger-dim); }
.diff-file.kind-modified .df-head { background: var(--warning-dim); }
.diff-file.kind-unchanged .df-head { background: #f3f4f6; }
.df-head { padding: 4px 8px; display: flex; gap: 8px; align-items: center; }
.df-kind { font-size: 11px; padding: 1px 6px; border-radius: 3px; background: #fff; color: var(--text-dim); }
.diff-file pre { padding: 4px 8px; margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.5; background: #fafafa; max-height: 300px; overflow: auto; white-space: pre; }
.ln-added { display: block; background: #dcfce7; color: #14532d; }
.ln-removed { display: block; background: #fee2e2; color: #7f1d1d; }
.ln-context { display: block; color: var(--text-dim); }
.ln-no { display: inline-block; min-width: 50px; color: var(--text-faint); padding-right: 6px; user-select: none; }
</style>
