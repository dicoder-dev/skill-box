<script setup>
// AIPanel.vue - 可折叠 AI 助手侧栏。
//
// 顶部 preset 横排 chips,中间对话历史(用户/助手气泡),底部输入框 + 发送。
// 用法:
//   <AIPanel :context-text="someSkillMd" @apply="onApply" />
//
// 暴露事件:
//   - apply(text) 助手产出"最终"内容时触发(让父组件把改写后的内容回填到表单)
import { ref, onMounted, nextTick } from 'vue'
import { listPresets, chatStream } from '@/api/skillbox/ai.js'

const props = defineProps({
  // 当前编辑器里选中的 skill 全文(可空)。preset 在渲染 prompt 时会塞到 vars.skill_md。
  contextText: { type: String, default: '' },
  // 选定的 provider 名字;空 = 后端按 priority 自动选
  provider: { type: String, default: '' },
})

const emit = defineEmits(['apply', 'error'])

// 内部状态
const presets = ref([])
const activePreset = ref(null) // 当前选中的 preset 对象
const messages = ref([])       // 对话历史 [{role, text, pending?}]
const input = ref('')
const busy = ref(false)
const historyEl = ref(null)
const abortRef = ref(null)

async function loadPresets() {
  try {
    const resp = await listPresets()
    presets.value = resp?.items || []
  } catch (e) {
    // 后端暂未启 AI 时静默降级
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
    // 查重:多 skill 对比;给个轻提示
    pushMsg('assistant', '请在输入框里把要对比的若干 skill 全文贴进来(每个用 \n\n---\n\n 分隔),我会给出重叠度评分。')
  } else {
    pushMsg('assistant', `已选择 preset:「${p.title}」。${p.description}\n把上下文(可空)和额外要求贴到下方,点发送即可。`)
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
    pushMsg('assistant', '请先在上方选一个 preset。')
    return
  }
  const userText = (activePreset.value.id === 'find_duplicates')
    ? (input.value || '')
    : (input.value || '(无额外输入,只基于上下文)')
  pushMsg('user', userText)
  const userInputSnapshot = input.value
  input.value = ''
  busy.value = true

  // 占位 assistant 消息,等待流式追加
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
      messages.value[placeholderIdx].text = buf + `\n\n[error] ${ev.err || 'unknown'}`
      messages.value[placeholderIdx].pending = false
      busy.value = false
      emit('error', ev.err)
    } else if (ev.kind === 'done') {
      // 流式结束由 [DONE] 触发 onDone,这里不重复处理
    }
  }
  const onDone = () => {
    finished = true
    messages.value[placeholderIdx].pending = false
    busy.value = false
    if (buf && activePreset.value.id === 'optimize_frontmatter') {
      // 给父组件一个"应用"的钩子(让父组件把改写后的 markdown 写回表单)
      emit('apply', buf)
    }
  }
  const onError = (err) => {
    if (finished) return
    finished = true
    messages.value[placeholderIdx].text = (buf || '') + `\n\n[error] ${err?.message || err}`
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
      <strong>AI 助手</strong>
      <button class="link" @click="clear" title="清空对话">清空</button>
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
      <span v-if="!presets.length" class="hint">暂未配置 AI provider 或内置 preset</span>
    </div>

    <div class="history" ref="historyEl">
      <p v-if="!messages.length" class="empty">
        先选一个 preset(优化 frontmatter / 检验 description / 润色正文 / 查重复 / 安全检查),再发问。
      </p>
      <article
        v-for="(m, i) in messages"
        :key="i"
        class="msg"
        :class="['role-' + m.role, { pending: m.pending }]"
      >
        <div class="meta">{{ m.role === 'user' ? '你' : 'AI' }}</div>
        <pre class="body">{{ m.text }}<span v-if="m.pending" class="cursor">▍</span></pre>
        <button v-if="!m.pending && m.text" class="link small" @click="copy(m.text)">复制</button>
      </article>
    </div>

    <footer class="composer">
      <textarea
        v-model="input"
        :placeholder="activePreset ? '补充说明(可空)' : '先选 preset'"
        :disabled="!activePreset"
        rows="3"
        @keydown.meta.enter.prevent="send"
        @keydown.ctrl.enter.prevent="send"
      />
      <div class="actions">
        <button v-if="busy" class="danger" @click="stop">停止</button>
        <button v-else class="primary" :disabled="!activePreset" @click="send">发送</button>
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
  background: #fbfbfd;
  border-left: 1px solid #e5e7eb;
  font-size: 13px;
  color: #1a1a1a;
}
.ai-header { display: flex; align-items: center; justify-content: space-between; padding: 10px 12px; border-bottom: 1px solid #eef0f3; }
.presets { display: flex; flex-wrap: wrap; gap: 6px; padding: 10px 12px; border-bottom: 1px solid #eef0f3; }
.presets .chip {
  font-size: 12px; padding: 4px 8px; border-radius: 999px;
  border: 1px solid #d0d0d0; background: #fff; cursor: pointer; color: #4b5563;
}
.presets .chip:hover { border-color: #2563eb; color: #2563eb; }
.presets .chip.active { background: #2563eb; color: #fff; border-color: #2563eb; }
.presets .hint { color: #9ca3af; font-size: 12px; }
.history { flex: 1; overflow-y: auto; padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; }
.history .empty { color: #9ca3af; font-size: 12px; margin: auto 0; text-align: center; }
.msg { display: flex; flex-direction: column; gap: 4px; padding: 8px 10px; border-radius: 6px; }
.msg.role-user { background: #eef2ff; align-self: flex-end; max-width: 90%; }
.msg.role-assistant { background: #fff; border: 1px solid #e5e7eb; align-self: flex-start; max-width: 100%; }
.msg .meta { font-size: 11px; color: #6b7280; }
.msg .body { margin: 0; white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12.5px; line-height: 1.55; }
.msg.pending .body { color: #4b5563; }
.cursor { display: inline-block; animation: blink 1s steps(1) infinite; }
@keyframes blink { 50% { opacity: 0; } }
.link { background: none; border: none; color: #2563eb; cursor: pointer; font-size: 12px; padding: 0; }
.link.small { align-self: flex-end; }
.composer { border-top: 1px solid #eef0f3; padding: 10px 12px; display: flex; flex-direction: column; gap: 6px; }
.composer textarea { width: 100%; resize: vertical; font-family: inherit; font-size: 13px; padding: 6px 8px; border: 1px solid #d0d0d0; border-radius: 4px; }
.composer .actions { display: flex; justify-content: flex-end; gap: 8px; }
.composer button { font-size: 13px; padding: 5px 12px; border-radius: 4px; border: 1px solid #d0d0d0; background: #fff; cursor: pointer; }
.composer button.primary { background: #2563eb; color: #fff; border-color: #2563eb; }
.composer button.danger { background: #b91c1c; color: #fff; border-color: #b91c1c; }
.composer button:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
