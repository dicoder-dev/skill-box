<script setup>
// AIPanel.vue - 可折叠 AI 助手侧栏
import { ref, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPresets, chatStream } from '@/api/skillbox/ai.js'

const { t } = useI18n()

const props = defineProps({
  contextText: { type: String, default: '' },
  provider: { type: String, default: '' },
})

const emit = defineEmits(['apply', 'error'])

const presets = ref([])
const activePreset = ref(null)
const messages = ref([])
const input = ref('')
const busy = ref(false)
const historyEl = ref(null)
const abortRef = ref(null)

async function loadPresets() {
  try {
    const resp = await listPresets()
    presets.value = resp?.items || []
  } catch (e) {
    presets.value = []
  }
}

onMounted(loadPresets)

function pushMsg(role, text, extra = {}) {
  messages.value.push({ role, text, ...extra })
  nextTick(scrollToBottom)
}

function scrollToBottom() {
  const el = historyEl.value
  if (el) el.scrollTop = el.scrollHeight
}

function pickPreset(p) {
  activePreset.value = p
  if (p.id === 'find_duplicates') {
    pushMsg('assistant', t('skills.ai.pickedDedupe'))
  } else {
    pushMsg('assistant', t('skills.ai.pickedPreset', { title: p.title, description: p.description }))
  }
}

function buildVars() {
  const vars = {}
  if (props.contextText) vars.skill_md = props.contextText
  if (activePreset.value?.id === 'find_duplicates') {
    vars.skill_list = input.value || props.contextText
  } else {
    vars.skill_md = props.contextText || input.value
  }
  return vars
}

async function send() {
  if (busy.value) return
  if (!activePreset.value) {
    pushMsg('assistant', t('skills.ai.pickFirst'))
    return
  }
  const userText = (activePreset.value.id === 'find_duplicates')
    ? (input.value || '')
    : (input.value || t('skills.ai.noExtraInput'))
  pushMsg('user', userText)
  const userInputSnapshot = input.value
  input.value = ''
  busy.value = true

  const placeholderIdx = messages.value.length
  pushMsg('assistant', '', { pending: true })

  let buf = ''
  let finished = false
  const onEvent = (ev) => {
    if (ev.kind === 'chunk') {
      buf += ev.text || ''
      messages.value[placeholderIdx].text = buf
      nextTick(scrollToBottom)
    } else if (ev.kind === 'error') {
      finished = true
      messages.value[placeholderIdx].text = buf + `\n\n` + t('skills.ai.errorTag', { msg: ev.err || 'unknown' })
      messages.value[placeholderIdx].pending = false
      busy.value = false
      emit('error', ev.err)
    } else if (ev.kind === 'done') {}
  }
  const onDone = () => {
    finished = true
    messages.value[placeholderIdx].pending = false
    busy.value = false
    if (buf && activePreset.value.id === 'optimize_frontmatter') {
      emit('apply', buf)
    }
  }
  const onError = (err) => {
    if (finished) return
    finished = true
    messages.value[placeholderIdx].text = (buf || '') + `\n\n` + t('skills.ai.errorTag', { msg: err?.message || err })
    messages.value[placeholderIdx].pending = false
    busy.value = false
    emit('error', err?.message || err)
  }

  abortRef.value = await chatStream(
    {
      provider: props.provider,
      preset_id: activePreset.value.id,
      vars: buildVars(),
    },
    { onEvent, onDone, onError },
  )
}

function stop() {
  if (abortRef.value?.abort) abortRef.value.abort()
  busy.value = false
}

function clear() {
  messages.value = []
  input.value = ''
}

function copy(text) {
  navigator.clipboard?.writeText(text || '')
}
</script>

<template>
  <aside class="ai-panel">
    <header class="ai-header">
      <strong>
        <Icon icon="mdi:robot" width="14" height="14" class="ai-icon" />
        {{ t('skills.ai.header') }}
      </strong>
      <button class="link" @click="clear" :title="t('skills.ai.clear')">{{ t('skills.ai.clear') }}</button>
    </header>

    <div class="presets">
      <button
        v-for="p in presets"
        :key="p.id"
        class="chip"
        :class="{ active: activePreset?.id === p.id }"
        :title="p.description"
        @click="pickPreset(p)"
      >
        {{ p.title }}
      </button>
      <span v-if="!presets.length" class="hint">{{ t('skills.ai.hintNoProvider') }}</span>
    </div>

    <div class="history" ref="historyEl">
      <p v-if="!messages.length" class="empty">
        <Icon icon="mdi:chat-outline" width="24" height="24" />
        <span>{{ t('skills.ai.empty') }}</span>
      </p>
      <article
        v-for="(m, i) in messages"
        :key="i"
        class="msg"
        :class="['role-' + m.role, { pending: m.pending }]"
      >
        <div class="meta">{{ m.role === 'user' ? t('skills.ai.roleUser') : t('skills.ai.roleAssistant') }}</div>
        <pre class="body">{{ m.text }}<span v-if="m.pending" class="cursor">▍</span></pre>
        <button v-if="!m.pending && m.text" class="link small" @click="copy(m.text)">{{ t('skills.ai.copy') }}</button>
      </article>
    </div>

    <footer class="composer">
      <textarea
        v-model="input"
        :placeholder="activePreset ? t('skills.ai.inputPlaceholderHint') : t('skills.ai.inputPlaceholderNoPreset')"
        :disabled="!activePreset"
        rows="3"
        @keydown.meta.enter.prevent="send"
        @keydown.ctrl.enter.prevent="send"
      />
      <div class="actions">
        <button v-if="busy" class="danger" @click="stop">
          <Icon icon="mdi:stop" width="12" height="12" />
          {{ t('skills.ai.stop') }}
        </button>
        <button v-else class="primary" :disabled="!activePreset" @click="send">
          <Icon icon="mdi:send" width="12" height="12" />
          {{ t('skills.ai.send') }}
        </button>
      </div>
    </footer>
  </aside>
</template>

<style scoped>
.ai-panel {
  display: flex;
  flex-direction: column;
  width: 380px;
  min-width: 320px;
  max-width: 420px;
  height: 100%;
  background: var(--bg-card);
  border-left: 1px solid var(--border);
  font-size: 13px;
  color: var(--text);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  transition: all 0.3s ease;
}

.ai-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.ai-header strong {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: var(--text);
}

.ai-icon {
  color: var(--text);
}

.presets {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.presets .chip {
  font-size: 12px;
  padding: 5px 12px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s ease;
}

.presets .chip:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-dim);
}

.presets .chip.active {
  background: var(--primary);
  color: var(--bg-card);
  border-color: var(--primary);
}

.presets .hint {
  color: var(--text-faint);
  font-size: 12px;
  padding: 4px 0;
}

.history {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.history .empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--text-faint);
  font-size: 12px;
  margin: auto 0;
  text-align: center;
}

.msg {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 14px;
  border-radius: var(--radius);
  transition: all 0.3s ease;
}

.msg.role-user {
  background: var(--primary-dim);
  color: var(--text);
  align-self: flex-end;
  max-width: 90%;
}

.msg.role-assistant {
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text);
  align-self: flex-start;
  max-width: 100%;
}

.msg .meta {
  font-size: 11px;
  color: var(--text-dim);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.msg .body {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--text);
}

.msg.pending .body {
  color: var(--text-dim);
}

.cursor {
  display: inline-block;
  animation: blink 1s steps(1) infinite;
  color: var(--primary);
}

@keyframes blink { 50% { opacity: 0; } }

.link {
  background: none;
  border: none;
  color: var(--primary);
  cursor: pointer;
  font-size: 12px;
  padding: 0;
  transition: color 0.15s ease;
}

.link:hover {
  color: var(--primary-hover);
}

.link.small {
  align-self: flex-end;
}

.composer {
  border-top: 1px solid var(--border);
  padding: 12px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.composer textarea {
  width: 100%;
  resize: vertical;
  font-family: inherit;
  font-size: 13px;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: all 0.15s ease;
}

.composer textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}

.composer textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.composer .actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.composer button {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.composer button.primary {
  background: var(--primary);
  color: var(--bg-card);
  border-color: var(--primary);
}

.composer button.primary:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}

.composer button.danger {
  background: var(--danger);
  color: var(--bg-card);
  border-color: var(--danger);
}

.composer button.danger:hover:not(:disabled) {
  background: #b91c1c;
  border-color: #b91c1c;
}

.composer button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
</style>
