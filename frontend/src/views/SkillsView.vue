<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { listSkills, getSkill, createSkill, updateSkill, deleteSkill } from '@/api/skillbox/skills'
import { runSkillTest } from '@/api/skillbox/skill_test'
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

onMounted(reload)
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
          <th style="width: 140px">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!items.length">
          <td colspan="6" class="empty">该 scope 下还没有 skill,点右上角"新建 Skill"开始</td>
        </tr>
        <tr v-for="p in items" :key="`${p.Scope}-${p.ProjectID}-${p.Name}-${p.Version}`">
          <td><code>{{ p.Name }}</code></td>
          <td>{{ p.Version }}</td>
          <td>{{ p.Source }}</td>
          <td>{{ p.ProjectID || '—' }}</td>
          <td class="time">{{ p.UpdatedAt?.slice(0, 19) || '—' }}</td>
          <td>
            <button class="link" @click="startEdit(p)">编辑</button>
            <button class="link" :disabled="testing" @click="triggerTest(p)">{{ testing ? '测试中…' : '测试' }}</button>
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
</style>
