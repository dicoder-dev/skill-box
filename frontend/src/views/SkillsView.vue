<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill } from '@/api/skillbox/skills'
import { runSkillTest } from '@/api/skillbox/skill_test'
import { applySkill, undoApply, listApplies, checkUpdates } from '@/api/skillbox/skill_apply'
import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'
import AIPanel from '@/components/AIPanel.vue'

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
    <header class="bar">
      <h2>Skills</h2>
      <div class="tabs">
        <button :class="{ active: scope === 'global' }" @click="switchScope('global')">全局</button>
        <button :class="{ active: scope === 'project' }" @click="switchScope('project')">项目</button>
      </div>
      <div class="search">
        <input
          v-model="keyword"
          placeholder="按 name 过滤"
          @keyup.enter="() => { page = 1; reload() }"
        />
        <button @click="() => { page = 1; reload() }">搜索</button>
        <button @click="toggleAI">{{ aiOpen ? '关闭 AI' : '打开 AI' }}</button>
        <button class="primary" @click="startNew">新建 Skill</button>
      </div>
    </header>

    <div class="apply-bar">
      <span class="apply-label">Apply 目标工具:</span>
      <select v-model="applyTool">
        <option v-for="t in TOOL_OPTIONS" :key="t" :value="t">{{ t }}</option>
      </select>
      <button @click="checkUpdateBadge" :disabled="updating">
        {{ updating ? '检测中…' : '检测更新' }}
      </button>
      <span v-if="updateBadge.updates > 0" class="update-badge danger">
        {{ updateBadge.updates }} / {{ updateBadge.total }} 可更新
      </span>
      <span v-else-if="updateBadge.total > 0" class="update-badge ok">
        {{ updateBadge.total }} 个 skill 已是最新
      </span>
      <p v-if="applyMessage" class="apply-msg">{{ applyMessage }}</p>
      <p v-if="applyError" class="error">{{ applyError }}</p>
    </div>

    <form v-if="editing" class="editor" @submit.prevent="submit">
      <div class="row">
        <label>
          <span>Name</span>
          <input v-model="draft.name" placeholder="英文短名,如 review-pr" :disabled="!!editingKey" />
        </label>
        <label>
          <span>Version</span>
          <input v-model="draft.version" placeholder="0.1.0" :disabled="!!editingKey" />
        </label>
        <label>
          <span>Scope</span>
          <select v-model="draft.scope" :disabled="!!editingKey">
            <option value="global">global</option>
            <option value="project">project</option>
          </select>
        </label>
        <label v-if="draft.scope === 'project'">
          <span>Project ID</span>
          <input v-model.number="draft.project_id" type="number" min="0" :disabled="!!editingKey" />
        </label>
      </div>
      <label class="full">
        <span>Description <small>(≥ 10 字符)</small></span>
        <textarea v-model="draft.description" rows="2" />
      </label>
      <label class="full">
        <span>Triggers <small>(每行一个,或逗号分隔)</small></span>
        <textarea v-model="draft.triggersText" rows="2" placeholder="review pr&#10;code review" />
      </label>
      <label class="full">
        <span>Body (Markdown,frontmatter 会自动拼)</span>
        <textarea v-model="draft.body" rows="14" class="code" />
      </label>
      <div class="actions">
        <button type="button" @click="editing = false">取消</button>
        <button type="submit" class="primary">{{ editingKey ? '保存' : '创建' }}</button>
      </div>
    </form>

    <div v-if="applyHistory.length" class="apply-history">
      <header class="ah-head">
        <h4>最近 Apply 历史</h4>
        <span class="ah-count">{{ applyHistory.length }} 条</span>
      </header>
      <ul>
        <li v-for="h in applyHistory" :key="h.ID || h.id" :class="`status-${h.Status}`">
          <span class="ah-id">#{{ h.ID || h.id }}</span>
          <span class="ah-tool">{{ h.Tool }}</span>
          <span class="ah-status">{{ h.Status }}</span>
          <span class="ah-time">{{ h.AppliedAt?.slice(0, 19) || '—' }}</span>
          <button v-if="h.Status === 'applied'" class="link" :disabled="undoing" @click="doUndo(h.ID || h.id)">{{ undoing ? '撤销中…' : '撤销' }}</button>
        </li>
      </ul>
    </div>

    <div v-if="selectedSkill" class="tag-panel">
      <header class="tp-head">
        <h4>Tag 管理 — <code>{{ selectedSkill.Name }}@{{ selectedSkill.Version }}</code></h4>
        <span class="tp-count">{{ tagList.length }} 个 tag</span>
        <button class="link" @click="selectedSkill = null; tagList = []; diffResult = null">关闭</button>
      </header>
      <p v-if="tagMessage" class="tag-msg">{{ tagMessage }}</p>
      <p v-if="tagError" class="error">{{ tagError }}</p>

      <div class="tag-create">
        <input v-model="newTagName" placeholder="tag 名,如 v1.0.0" class="tag-input" />
        <input v-model="newTagMessage" placeholder="描述(可选)" class="tag-input" />
        <button class="primary" :disabled="tagLoading" @click="doCreateTag">{{ tagLoading ? '处理中…' : '打 Tag' }}</button>
      </div>

      <div v-if="tagList.length" class="tag-actions">
        <span class="tag-label">Diff:</span>
        <select v-model="diffLeftTagID">
          <option :value="0">current</option>
          <option v-for="t in tagList" :key="t.ID || t.id" :value="t.ID || t.id">{{ t.Tag }} ({{ (t.CreatedAt || '').slice(0, 16) }}){{ t.IsImplicit ? ' [implicit]' : '' }}</option>
        </select>
        <span>→</span>
        <select v-model="diffRightTagID">
          <option :value="0">current</option>
          <option v-for="t in tagList" :key="t.ID || t.id" :value="t.ID || t.id">{{ t.Tag }} ({{ (t.CreatedAt || '').slice(0, 16) }}){{ t.IsImplicit ? ' [implicit]' : '' }}</option>
        </select>
        <button @click="doDiff(diffLeftTagID, diffRightTagID)">看 Diff</button>
        <button @click="doDiff(0, 0)">清空</button>
      </div>

      <ul v-if="tagList.length" class="tag-list">
        <li v-for="t in tagList" :key="t.ID || t.id" :class="{ implicit: t.IsImplicit }">
          <span class="t-id">#{{ t.ID || t.id }}</span>
          <span class="t-name"><code>{{ t.Tag }}</code></span>
          <span class="t-msg">{{ t.Message || '—' }}</span>
          <span class="t-time">{{ (t.CreatedAt || '').slice(0, 19) }}</span>
          <button class="link" @click="doDiff(t.ID || t.id, 0)">vs current</button>
          <button class="link" :disabled="rolling" @click="doRollback(t.ID || t.id)">{{ rolling ? '回滚中…' : '回滚到此' }}</button>
          <button class="link danger" @click="doDeleteTag(t.ID || t.id)">删</button>
        </li>
      </ul>

      <div v-if="diffResult" class="diff-panel">
        <header class="dp-head">
          <h4>Diff 结果</h4>
          <span class="dp-stats">
            <span class="added">+{{ diffResult.added }}</span>
            <span class="removed">-{{ diffResult.removed }}</span>
            <span class="modified">~{{ diffResult.modified }}</span>
            <span class="unchanged">{{ diffResult.unchanged }} 不变</span>
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

    <div v-if="lastTest || testError" class="test-panel" :class="`status-${(lastTest?.run?.status || 'errored')}`">
      <header class="tp-head">
        <h4>最近测试结果</h4>
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

    <p v-if="error" class="error">{{ error }}</p>

    <table class="grid" v-if="items.length || !loading">
      <thead>
        <tr>
          <th>Name</th>
          <th>Version</th>
          <th>Source</th>
          <th>Project ID</th>
          <th>Updated</th>
          <th>更新</th>
          <th style="width: 220px">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="7" class="empty">该 scope 下还没有 skill,点右上角"新建 Skill"开始</td>
        </tr>
        <tr v-for="p in items" :key="`${p.Scope}-${p.ProjectID}-${p.Name}-${p.Version}`">
          <td><code>{{ p.Name }}</code></td>
          <td>{{ p.Version }}</td>
          <td>{{ p.Source }}</td>
          <td>{{ p.ProjectID || '—' }}</td>
          <td class="time">{{ p.UpdatedAt?.slice(0, 19) || '—' }}</td>
          <td>
            <span v-if="p.Source === 'market'" class="badge market">market</span>
            <span v-else class="badge local">{{ p.Source }}</span>
          </td>
          <td>
            <button class="link primary-link" :disabled="applying" @click="doApply(p)">{{ applying ? '应用中…' : '应用' }}</button>
            <button class="link" :disabled="testing" @click="triggerTest(p)">{{ testing ? '测试中…' : '测试' }}</button>
            <button class="link" @click="startEdit(p)">编辑</button>
            <button class="link" @click="loadTags(p)">Tag</button>
            <button class="link danger" @click="remove(p)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>

    <footer class="pager" v-if="totalPages > 1">
      <button :disabled="page <= 1" @click="gotoPage(page - 1)">上一页</button>
      <span>{{ page }} / {{ totalPages }} (共 {{ total }} 条)</span>
      <button :disabled="page >= totalPages" @click="gotoPage(page + 1)">下一页</button>
    </footer>
  </section><AIPanel v-if="aiOpen" :context-text="currentSkillMd" @apply="onAIApply" /></div>
</template>

<style scoped>
.test-panel { margin: 8px 0 12px; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 6px; background: #fafafa; }
.test-panel.status-passed { border-color: #bbf7d0; background: #f0fdf4; }
.test-panel.status-failed { border-color: #fecaca; background: #fef2f2; }
.test-panel.status-errored { border-color: #fde68a; background: #fffbeb; }
.test-panel.status-skipped { border-color: #e5e7eb; background: #f9fafb; }
.tp-head { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.tp-head h4 { margin: 0; font-size: 14px; }
.tp-status { font-size: 12px; padding: 2px 6px; border-radius: 4px; background: #1f2937; color: #fff; text-transform: uppercase; }
.tp-summary { color: #4b5563; font-size: 13px; margin: 4px 0 8px; }
.tp-list { list-style: none; padding: 0; margin: 0; }
.tp-list li { display: grid; grid-template-columns: 80px 90px 1fr; gap: 8px; padding: 4px 0; border-bottom: 1px dashed #e5e7eb; font-size: 13px; }
.tp-list .check-name { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #1f2937; }
.tp-list .check-status { font-size: 12px; padding: 1px 6px; border-radius: 3px; text-align: center; }
.tp-list .check-passed .check-status { background: #bbf7d0; color: #065f46; }
.tp-list .check-failed .check-status { background: #fecaca; color: #991b1b; }
.tp-list .check-errored .check-status { background: #fde68a; color: #92400e; }
.tp-list .check-skipped .check-status { background: #e5e7eb; color: #4b5563; }
.tp-list .check-msg { color: #4b5563; }
.tp-detail { margin-top: 6px; }
.tp-detail summary { cursor: pointer; font-size: 12px; color: #6b7280; }
.tp-detail pre { background: #f3f4f6; padding: 6px 8px; border-radius: 4px; font-size: 11px; max-height: 200px; overflow: auto; }
</style>
<style scoped>
.skills-layout { display: flex; height: 100%; }
.skills-view { padding: 16px 20px; max-width: 1100px; margin: 0 auto; color: #1a1a1a; flex: 1; min-width: 0; }
.skills-view.with-ai { max-width: none; }
.bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.bar h2 { margin: 0; font-size: 18px; }
.tabs { display: flex; gap: 0; }
.tabs button { border-radius: 0; }
.tabs button:first-child { border-top-left-radius: 4px; border-bottom-left-radius: 4px; }
.tabs button:last-child { border-top-right-radius: 4px; border-bottom-right-radius: 4px; border-left: none; }
.tabs button.active { background: #2563eb; color: #fff; border-color: #2563eb; }
.search { margin-left: auto; display: flex; gap: 6px; }
.search input { width: 200px; }
input, select, button, textarea {
  font-size: 14px; padding: 5px 9px; border: 1px solid #d0d0d0; border-radius: 4px; background: #fff; color: #1a1a1a;
}
button { cursor: pointer; }
button.primary { background: #2563eb; color: #fff; border-color: #2563eb; }
button.link { border: none; background: none; padding: 2px 4px; color: #2563eb; }
button.link.danger { color: #b91c1c; }
button:disabled { opacity: 0.45; cursor: not-allowed; }
textarea { width: 100%; resize: vertical; font-family: inherit; }
textarea.code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; }
.editor { display: flex; flex-direction: column; gap: 10px; padding: 12px; border: 1px solid #e5e7eb; border-radius: 6px; background: #fafafa; margin-bottom: 12px; }
.editor .row { display: grid; grid-template-columns: 1.4fr 0.8fr 0.8fr 0.8fr; gap: 10px 14px; }
.editor label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: #4b5563; }
.editor label.full { width: 100%; }
.editor label small { color: #9ca3af; }
.actions { display: flex; gap: 8px; justify-content: flex-end; }
.error { color: #b91c1c; margin: 6px 0; }
.grid { width: 100%; border-collapse: collapse; font-size: 13px; }
.grid th, .grid td { text-align: left; padding: 7px 9px; border-bottom: 1px solid #eef0f3; }
.grid th { background: #f7f8fa; color: #4b5563; font-weight: 600; }
.grid .empty { text-align: center; color: #9ca3af; padding: 18px; }
.grid .time { color: #6b7280; font-size: 12px; }
.pager { display: flex; align-items: center; gap: 12px; margin-top: 12px; font-size: 13px; color: #4b5563; }
.apply-bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-bottom: 12px; padding: 8px 12px; background: #f5f7fa; border: 1px solid #e5e7eb; border-radius: 6px; font-size: 13px; }
.apply-label { color: #4b5563; font-weight: 500; }
.apply-bar select { padding: 3px 6px; }
.update-badge { padding: 3px 8px; border-radius: 10px; font-size: 12px; font-weight: 500; }
.update-badge.danger { background: #fee2e2; color: #991b1b; }
.update-badge.ok { background: #d1fae5; color: #065f46; }
.apply-msg { color: #047857; margin: 0; font-size: 12px; }
.apply-bar p.error { margin: 0; font-size: 12px; }
.badge { display: inline-block; padding: 1px 6px; border-radius: 8px; font-size: 11px; }
.badge.market { background: #dbeafe; color: #1e40af; }
.badge.local { background: #f3f4f6; color: #4b5563; }
.link.primary-link { color: #047857; font-weight: 500; }
.apply-history { margin: 8px 0 12px; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 6px; background: #fafafa; }
.ah-head { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.ah-head h4 { margin: 0; font-size: 14px; }
.ah-count { font-size: 12px; color: #6b7280; }
.apply-history ul { list-style: none; padding: 0; margin: 0; }
.apply-history li { display: grid; grid-template-columns: 60px 80px 100px 1fr auto; gap: 8px; align-items: center; padding: 4px 0; border-bottom: 1px dashed #e5e7eb; font-size: 13px; }
.ah-id { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #4b5563; }
.ah-tool { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #1f2937; }
.ah-status { font-size: 11px; padding: 1px 6px; border-radius: 3px; text-align: center; }
.status-applied .ah-status { background: #bbf7d0; color: #065f46; }
.status-rolled_back .ah-status { background: #e5e7eb; color: #4b5563; }
.status-failed .ah-status { background: #fecaca; color: #991b1b; }
.ah-time { color: #6b7280; font-size: 12px; }
.tag-panel { margin: 8px 0 12px; padding: 12px 14px; border: 1px solid #e5e7eb; border-radius: 6px; background: #fafafa; }
.tag-panel .tp-head { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.tag-panel .tp-head h4 { margin: 0; font-size: 14px; }
.tag-panel .tp-count { color: #6b7280; font-size: 12px; }
.tag-msg { color: #047857; font-size: 12px; margin: 4px 0; }
.tag-create { display: flex; gap: 8px; margin-bottom: 8px; }
.tag-input { flex: 1; }
.tag-actions { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; font-size: 13px; }
.tag-actions .tag-label { color: #4b5563; }
.tag-list { list-style: none; padding: 0; margin: 0; border-top: 1px dashed #e5e7eb; }
.tag-list li { display: grid; grid-template-columns: 50px 140px 1fr 150px auto auto auto; gap: 8px; align-items: center; padding: 5px 0; border-bottom: 1px dashed #e5e7eb; font-size: 13px; }
.tag-list li.implicit { background: #fef3c7; }
.tag-list .t-id { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #4b5563; }
.tag-list .t-name code { background: #f3f4f6; padding: 1px 5px; border-radius: 3px; }
.tag-list .t-msg { color: #4b5563; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag-list .t-time { color: #6b7280; font-size: 11px; }
.diff-panel { margin-top: 12px; padding: 10px 12px; border: 1px solid #e5e7eb; border-radius: 6px; background: #fff; }
.dp-head { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.dp-head h4 { margin: 0; font-size: 13px; }
.dp-stats { display: flex; gap: 8px; font-size: 12px; }
.dp-stats .added { color: #065f46; background: #d1fae5; padding: 1px 6px; border-radius: 3px; }
.dp-stats .removed { color: #991b1b; background: #fee2e2; padding: 1px 6px; border-radius: 3px; }
.dp-stats .modified { color: #92400e; background: #fef3c7; padding: 1px 6px; border-radius: 3px; }
.dp-stats .unchanged { color: #4b5563; }
.diff-file { margin: 6px 0; border: 1px solid #f3f4f6; border-radius: 4px; overflow: hidden; }
.diff-file.kind-added .df-head { background: #d1fae5; }
.diff-file.kind-removed .df-head { background: #fee2e2; }
.diff-file.kind-modified .df-head { background: #fef3c7; }
.diff-file.kind-unchanged .df-head { background: #f3f4f6; }
.df-head { padding: 4px 8px; display: flex; gap: 8px; align-items: center; }
.df-kind { font-size: 11px; padding: 1px 6px; border-radius: 3px; background: #fff; color: #4b5563; }
.df-path { font-size: 12px; color: #1f2937; }
.diff-file pre { padding: 4px 8px; margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 1.5; background: #fafafa; max-height: 300px; overflow: auto; white-space: pre; }
.ln-added { display: block; background: #dcfce7; color: #14532d; }
.ln-removed { display: block; background: #fee2e2; color: #7f1d1d; }
.ln-context { display: block; color: #4b5563; }
.ln-no { display: inline-block; min-width: 50px; color: #9ca3af; padding-right: 6px; user-select: none; }
</style>
