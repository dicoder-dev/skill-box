<script setup>
// AISettingsPanel.vue — 设置页"AI 模型"卡片。
//
// 功能:
//   - 列出已配置的 provider(name / kind / model / base_url / api_key 是否已配)
//   - 新建 / 编辑 provider(CRUD 调 ai.js 现成接口)
//   - 每个 provider 行 + 新建未保存行都有「测试连接」按钮
//     → 调 testConnection() → 成功显示绿色,失败显示完整错误文本
//   - 当前编辑态的字段(裸)允许"未保存就测试"
import { ref, reactive, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import {
  listProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  setProviderKey,
  testConnection,
} from '@/api/skillbox/ai.js'

const { t } = useI18n()

// 全部 provider 列表
const providers = ref([])
const loading = ref(false)

// 编辑态: null = 新建(用 createForm);数字 = 编辑某个 id(用 editForm)
const editId = ref(null)
const editForm = reactive({ name: '', kind: 'openai', base_url: '', model: '', priority: 100, enabled: true })
const editKey = ref('') // 仅发送 setProviderKey;不持久化在 editForm 里
const editTestResult = ref(null) // {ok, message, sample?, latency_ms?}
const editTestBusy = ref(false)

// 新建态
const creating = ref(false)
const createForm = reactive({ name: '', kind: 'openai', base_url: '', model: '', priority: 100, enabled: true })
const createKey = ref('')
const createTestResult = ref(null)
const createTestBusy = ref(false)

const baseErrMsg = ref('')
const createErrMsg = ref('')
const editErrMsg = ref('')

const KIND_OPTIONS = [
  { value: 'openai', labelKey: 'settings.ai.kindOpenAI' },
  { value: 'anthropic', labelKey: 'settings.ai.kindAnthropic' },
  { value: 'openai_compat', labelKey: 'settings.ai.kindOpenAICompat' },
]

async function refresh() {
  loading.value = true
  try {
    const r = await listProviders()
    providers.value = (r?.items || []).filter((p) => p && p.name)
  } catch (e) {
    baseErrMsg.value = t('settings.ai.errLoad', { msg: e?.message || String(e) })
  } finally {
    loading.value = false
  }
}
onMounted(refresh)

function startCreate() {
  creating.value = true
  Object.assign(createForm, { name: '', kind: 'openai', base_url: '', model: '', priority: 100, enabled: true })
  createKey.value = ''
  createTestResult.value = null
  createErrMsg.value = ''
}
function cancelCreate() {
  creating.value = false
}

function startEdit(p) {
  editId.value = p.id
  Object.assign(editForm, {
    name: p.name || '',
    kind: p.kind || 'openai',
    base_url: p.base_url || '',
    model: p.model || '',
    priority: p.priority ?? 100,
    enabled: !!p.enabled,
  })
  editKey.value = ''
  editTestResult.value = null
  editErrMsg.value = ''
}
function cancelEdit() {
  editId.value = null
}

async function saveCreate() {
  createErrMsg.value = ''
  try {
    const created = await createProvider({ ...createForm })
    if (createKey.value) {
      await setProviderKey(created.name, createKey.value)
    }
    creating.value = false
    createKey.value = ''
    await refresh()
  } catch (e) {
    createErrMsg.value = t('settings.ai.errSave', { msg: e?.message || String(e) })
  }
}

async function saveEdit() {
  editErrMsg.value = ''
  if (!editId.value) return
  try {
    await updateProvider({ id: editId.value, ...editForm })
    if (editKey.value) {
      await setProviderKey(editForm.name, editKey.value)
    }
    cancelEdit()
    await refresh()
  } catch (e) {
    editErrMsg.value = t('settings.ai.errSave', { msg: e?.message || String(e) })
  }
}

async function removeOne(p) {
  if (!window.confirm(t('settings.ai.confirmDelete', { name: p.name }))) return
  try {
    await deleteProvider(p.id)
    if (editId.value === p.id) cancelEdit()
    await refresh()
  } catch (e) {
    baseErrMsg.value = t('settings.ai.errDelete', { msg: e?.message || String(e) })
  }
}

async function runTest(payload, intoRef, busyRef, errRef) {
  intoRef.value = null
  errRef.value = ''
  busyRef.value = true
  try {
    const r = await testConnection(payload)
    intoRef.value = r
  } catch (e) {
    intoRef.value = { ok: false, message: e?.message || String(e), latency_ms: 0 }
  } finally {
    busyRef.value = false
  }
}

function testExisting(p) {
  // 已存 provider:优先把用户当前在 editKey 里刚输入的 key 带上 → 未保存也能测
  runTest(
    { provider_id: p.id, api_key: editId.value === p.id ? editKey.value : '' },
    editId.value === p.id ? editTestResult : ref(null), // 这里直接用一个临时 ref 没意义,所以改用 picker
    ref(false),
    ref(''),
  )
}

// 上面 testExisting 想用闭包的方式处理两态太绕,直接拆两个函数更直白
async function testExistingOnRow(p) {
  // 给列表行用 — 直接从 settings 已存 key 来探测
  editTestBusy.value = true // 暂时复用 editTestBusy(列表行只有一个,不会和 edit 表单并发)
  try {
    const r = await testConnection({ provider_id: p.id })
    editTestResult.value = { ...r, _rowName: p.name }
  } catch (e) {
    editTestResult.value = { ok: false, message: e?.message || String(e), _rowName: p.name }
  } finally {
    editTestBusy.value = false
  }
}

async function testEditForm() {
  await runTest(
    {
      provider_id: editId.value || 0,
      name: editForm.name,
      kind: editForm.kind,
      base_url: editForm.base_url,
      model: editForm.model,
      api_key: editKey.value,
    },
    editTestResult,
    editTestBusy,
    editErrMsg,
  )
}

async function testCreateForm() {
  await runTest(
    {
      name: createForm.name,
      kind: createForm.kind,
      base_url: createForm.base_url,
      model: createForm.model,
      api_key: createKey.value,
    },
    createTestResult,
    createTestBusy,
    createErrMsg,
  )
}

// 监听切换编辑 → 清掉旧结果
watch(editId, () => { editTestResult.value = null })
</script>

<template>
  <section class="card">
    <header class="card-header">
      <h3>
        <IconPark icon="mdi:robot-outline" width="18" height="18" />
        {{ t('settings.ai.title') }}
        <span class="card-sub">— {{ t('settings.ai.subtitle') }}</span>
      </h3>
      <div class="card-header-actions">
        <button class="primary" :disabled="creating || editId !== null" @click="startCreate">
          <IconPark icon="mdi:plus" width="14" height="14" />
          {{ t('settings.ai.btnNew') }}
        </button>
      </div>
    </header>

    <p v-if="baseErrMsg" class="hint-box error-hint">
      <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
      <span>{{ baseErrMsg }}</span>
    </p>

    <!-- 新建表单 -->
    <div v-if="creating" class="ai-form">
      <h4>
        <IconPark icon="mdi:plus-circle-outline" width="16" height="16" />
        {{ t('settings.ai.formNew') }}
      </h4>
      <div class="grid">
        <label>
          <span>{{ t('settings.ai.fieldName') }}</span>
          <input v-model="createForm.name" :placeholder="t('settings.ai.fieldNameHint')" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldKind') }}</span>
          <select v-model="createForm.kind">
            <option v-for="k in KIND_OPTIONS" :key="k.value" :value="k.value">{{ t(k.labelKey) }}</option>
          </select>
        </label>
        <label class="span2">
          <span>{{ t('settings.ai.fieldBaseURL') }}</span>
          <input v-model="createForm.base_url" :placeholder="t('settings.ai.fieldBaseURLHint')" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldModel') }}</span>
          <input v-model="createForm.model" :placeholder="t('settings.ai.fieldModelHint')" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldApiKey') }}</span>
          <input v-model="createKey" type="password" :placeholder="t('settings.ai.fieldApiKeyHint')" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldPriority') }}</span>
          <input v-model.number="createForm.priority" type="number" />
        </label>
        <label class="enable-toggle">
          <input type="checkbox" v-model="createForm.enabled" />
          <span>{{ t('settings.ai.fieldEnabled') }}</span>
        </label>
      </div>
      <p v-if="createErrMsg" class="hint-box error-hint">
        <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
        <span>{{ createErrMsg }}</span>
      </p>
      <div class="ai-test-row" v-if="createTestResult">
        <p v-if="createTestResult.ok" class="hint-box success-hint">
          <IconPark icon="mdi:check-circle-outline" width="14" height="14" />
          <span>
            {{ t('settings.ai.testOk') }}
            <span class="dim">({{ createTestResult.latency_ms }} ms)</span>
            <span v-if="createTestResult.sample" class="sample">「{{ createTestResult.sample }}」</span>
          </span>
        </p>
        <p v-else class="hint-box error-hint">
          <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
          <span class="error-msg">{{ createTestResult.message }}</span>
        </p>
      </div>
      <div class="ai-form-actions">
        <button class="primary" @click="testCreateForm" :disabled="createTestBusy">
          <span v-if="createTestBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:lan-connect" width="14" height="14" />
          {{ t('settings.ai.btnTest') }}
        </button>
        <button class="primary" @click="saveCreate">
          <IconPark icon="mdi:content-save" width="14" height="14" />
          {{ t('common.save') }}
        </button>
        <button @click="cancelCreate">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <!-- 编辑表单(单个) -->
    <div v-if="editId !== null" class="ai-form">
      <h4>
        <IconPark icon="mdi:pencil-outline" width="16" height="16" />
        {{ t('settings.ai.formEdit') }} — {{ editForm.name }}
      </h4>
      <div class="grid">
        <label>
          <span>{{ t('settings.ai.fieldName') }}</span>
          <input v-model="editForm.name" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldKind') }}</span>
          <select v-model="editForm.kind">
            <option v-for="k in KIND_OPTIONS" :key="k.value" :value="k.value">{{ t(k.labelKey) }}</option>
          </select>
        </label>
        <label class="span2">
          <span>{{ t('settings.ai.fieldBaseURL') }}</span>
          <input v-model="editForm.base_url" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldModel') }}</span>
          <input v-model="editForm.model" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldApiKey') }}</span>
          <input v-model="editKey" type="password" :placeholder="t('settings.ai.fieldApiKeyEditHint')" />
        </label>
        <label>
          <span>{{ t('settings.ai.fieldPriority') }}</span>
          <input v-model.number="editForm.priority" type="number" />
        </label>
        <label class="enable-toggle">
          <input type="checkbox" v-model="editForm.enabled" />
          <span>{{ t('settings.ai.fieldEnabled') }}</span>
        </label>
      </div>
      <p v-if="editErrMsg" class="hint-box error-hint">
        <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
        <span>{{ editErrMsg }}</span>
      </p>
      <div class="ai-test-row" v-if="editTestResult && !editTestResult._rowName">
        <p v-if="editTestResult.ok" class="hint-box success-hint">
          <IconPark icon="mdi:check-circle-outline" width="14" height="14" />
          <span>
            {{ t('settings.ai.testOk') }}
            <span class="dim">({{ editTestResult.latency_ms }} ms)</span>
            <span v-if="editTestResult.sample" class="sample">「{{ editTestResult.sample }}」</span>
          </span>
        </p>
        <p v-else class="hint-box error-hint">
          <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
          <span class="error-msg">{{ editTestResult.message }}</span>
        </p>
      </div>
      <div class="ai-form-actions">
        <button class="primary" @click="testEditForm" :disabled="editTestBusy">
          <span v-if="editTestBusy" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:lan-connect" width="14" height="14" />
          {{ t('settings.ai.btnTest') }}
        </button>
        <button class="primary" @click="saveEdit">
          <IconPark icon="mdi:content-save" width="14" height="14" />
          {{ t('common.save') }}
        </button>
        <button @click="cancelEdit">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <!-- 列表 -->
    <div v-if="loading" class="section-empty">{{ t('common.loading') }}</div>
    <div v-else-if="!providers.length && !creating" class="section-empty">{{ t('settings.ai.listEmpty') }}</div>
    <ul v-else class="ai-list">
      <li v-for="p in providers" :key="p.id" class="ai-row" :class="{ disabled: !p.enabled }">
        <div class="ai-row-main">
          <div class="ai-row-title">
            <strong>{{ p.name }}</strong>
            <span class="badge">{{ p.kind }}</span>
            <span v-if="!p.enabled" class="badge badge-warn">{{ t('settings.ai.badgeDisabled') }}</span>
          </div>
          <div class="ai-row-meta">
            <span v-if="p.model"><IconPark icon="mdi:tag-outline" width="12" height="12" /> {{ p.model }}</span>
            <span v-if="p.base_url"><IconPark icon="mdi:web" width="12" height="12" /> {{ p.base_url }}</span>
            <span :class="['has-key', p.has_key ? 'has-key-yes' : 'has-key-no']">
              <IconPark :icon="p.has_key ? 'mdi:key-variant' : 'mdi:key-remove'" width="12" height="12" />
              {{ p.has_key ? t('settings.ai.hasKey') : t('settings.ai.noKey') }}
            </span>
          </div>
        </div>
        <div class="ai-row-actions">
          <button :disabled="editId !== null" @click="testExistingOnRow(p)" :title="t('settings.ai.btnTestTitle')">
            <span v-if="editTestBusy && editTestResult?._rowName === p.name" class="spinner spinner-sm"></span>
            <IconPark v-else icon="mdi:lan-connect" width="14" height="14" />
          </button>
          <button :disabled="editId !== null" @click="startEdit(p)" :title="t('common.edit')">
            <IconPark icon="mdi:pencil-outline" width="14" height="14" />
          </button>
          <button :disabled="editId !== null" @click="removeOne(p)" :title="t('common.delete')">
            <IconPark icon="mdi:trash-can-outline" width="14" height="14" />
          </button>
        </div>
        <div v-if="editTestResult && editTestResult._rowName === p.name" class="ai-row-test">
          <p v-if="editTestResult.ok" class="hint-box success-hint">
            <IconPark icon="mdi:check-circle-outline" width="14" height="14" />
            <span>
              {{ t('settings.ai.testOk') }}
              <span class="dim">({{ editTestResult.latency_ms }} ms)</span>
              <span v-if="editTestResult.sample" class="sample">「{{ editTestResult.sample }}」</span>
            </span>
          </p>
          <p v-else class="hint-box error-hint">
            <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
            <span class="error-msg">{{ editTestResult.message }}</span>
          </p>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}
.card-header-actions {
  display: flex;
  gap: 8px;
}
.ai-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.ai-row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}
.ai-row.disabled { opacity: 0.55; }
.ai-row-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.ai-row-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}
.badge-warn {
  background: var(--warn-dim);
  color: var(--warn, #b45309);
}
.ai-row-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 12px;
  color: var(--text-dim);
}
.ai-row-meta span { display: inline-flex; align-items: center; gap: 4px; }
.has-key-yes { color: var(--success, #16a34a); }
.has-key-no { color: var(--text-faint); }
.ai-row-actions {
  display: flex;
  gap: 4px;
}
.ai-row-actions button {
  width: 28px; height: 28px;
  padding: 0;
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  cursor: pointer;
  transition: all 0.15s ease;
}
.ai-row-actions button:hover:not(:disabled) {
  border-color: var(--primary);
  color: var(--primary);
}
.ai-row-actions button:disabled { opacity: 0.5; cursor: not-allowed; }
.ai-row-test {
  grid-column: 1 / -1;
}
.ai-form {
  margin: 12px 0;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
}
.ai-form h4 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 10px;
  font-size: 13px;
}
.ai-form .grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 14px;
}
.ai-form .grid .span2 { grid-column: span 2; }
.ai-form label {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}
.ai-form label > span { color: var(--text-dim); }
.ai-form input,
.ai-form select {
  padding: 6px 10px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: all 0.15s ease;
}
.ai-form input:focus,
.ai-form select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.ai-form .enable-toggle {
  flex-direction: row;
  align-items: center;
  gap: 6px;
  align-self: end;
}
.ai-form-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}
.ai-test-row {
  margin-top: 10px;
}
.hint-box {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  line-height: 1.5;
  margin: 6px 0 0;
}
.success-hint { background: rgba(34, 197, 94, 0.12); color: #16a34a; }
.error-hint { background: rgba(239, 68, 68, 0.12); color: var(--danger, #dc2626); }
.error-msg {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 11.5px;
}
.dim { color: var(--text-faint); margin: 0 6px; }
.sample { color: var(--text-dim); margin-left: 6px; }
.section-empty {
  padding: 20px;
  text-align: center;
  color: var(--text-faint);
  font-size: 13px;
}
</style>
