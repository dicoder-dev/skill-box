<script setup>
// AIRightPanel.vue — 右侧 AI 对话面板
//
// 设计要点:
//   - 替代原大纲区域,与 outline aside 互斥(由父级 CodeViewer 按 rightPanelMode 决定渲染哪个)
//   - 三段式:头部(标题 + 大纲切换 + 关闭) / 消息列表 / 底部输入区(标签栏 + 输入框 + 发送)
//   - 流式渲染复用 ai.js 的 chatStream(SSE),零后端改造
//   - 标签栏 2 个:翻译、检测
//     - 翻译:弹语言选择 Modal → 确认后把翻译提示词 + 原文填入输入框,用户再点发送
//     - 检测:同上,填入检测提示词 + 原文
//   - AI 消息支持"应用 / 拒绝":
//     - SKILL.md: emit apply-skill → 父级走 SkillsView.onAIApply 落盘
//     - 其它文件: emit apply-local → 父级 (SkillFileInlinePanel) 写 localFiles + 标 dirty
//   - 历史按 filePath 维度持久化到 sessionStorage,跨文件保留
//
// 复用:
//   - chatStream / listProviders from '@/api/skillbox/ai'
//   - Modal from '@/components/Modal.vue'
//   - IconPark + MDI_TO_ICONPARK(translate → Translate / robot-outline → RobotOne 等)
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import Modal from '@/components/Modal.vue'
import { chatStream, listProviders } from '@/api/skillbox/ai'
import { useToastStore } from '@/core/store/toast'

const props = defineProps({
  filePath: { type: String, required: true },
  fileContent: { type: String, default: '' },
  isSkillMd: { type: Boolean, default: false },
  readOnly: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'switch-outline', 'apply-skill', 'apply-local'])

const { t } = useI18n()
const toast = useToastStore()

// ===== 状态 =====
const messages = ref([])
const inputText = ref('')
const busy = ref(false)
const abort = ref(null)
const providers = ref([])
const hasProvider = ref(false)

// 翻译弹窗
const translateDialogOpen = ref(false)
const targetLang = ref('en-US')
const langKeys = ['zh-CN', 'zh-TW', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR', 'de-DE', 'es-ES']
const langOptions = computed(() =>
  langKeys.map((k) => ({ value: k, label: t(`skills.aiDialog.langs.${k}`) })),
)
function langLabelOf(code) {
  return langOptions.value.find((l) => l.value === code)?.label || code
}

// sessionStorage key(按文件维度隔离)
function sessionKey(path) {
  return `skillbox.aiSession.${path || '__none__'}`
}

function loadSession() {
  try {
    const raw = sessionStorage.getItem(sessionKey(props.filePath))
    messages.value = raw ? JSON.parse(raw) : []
  } catch (_) {
    messages.value = []
  }
}

function persistSession() {
  try {
    sessionStorage.setItem(sessionKey(props.filePath), JSON.stringify(messages.value))
  } catch (_) { /* sessionStorage 不可用时静默 */ }
}

function clearHistory() {
  if (busy.value) {
    abort.value?.abort?.()
    busy.value = false
  }
  messages.value = []
  try { sessionStorage.removeItem(sessionKey(props.filePath)) } catch (_) {}
}

// ===== Providers =====
async function loadProviders() {
  try {
    const r = await listProviders()
    providers.value = (r?.items || []).filter((p) => p && p.enabled && p.has_key)
    hasProvider.value = providers.value.length > 0
  } catch (_) {
    providers.value = []
    hasProvider.value = false
  }
}

// ===== 翻译 / 检测标签 =====
function onTranslateClick() {
  if (!hasProvider.value) {
    toast.error(t('skills.aiPanel.noProvider'))
    return
  }
  if (!props.fileContent) {
    toast.error(t('skills.aiPanel.noContent'))
    return
  }
  translateDialogOpen.value = true
}

function confirmTranslate() {
  const label = langLabelOf(targetLang.value)
  const tmpl = t('skills.aiPanel.translatePromptTemplate')
  const prompt = tmpl
    .replace(/\{target_lang\}/g, label)
    .replace(/\{skill_md\}/g, props.fileContent || '')
  inputText.value = prompt
  translateDialogOpen.value = false
  nextTick(() => inputEl.value?.focus())
}

function onReviewClick() {
  if (!hasProvider.value) {
    toast.error(t('skills.aiPanel.noProvider'))
    return
  }
  if (!props.fileContent) {
    toast.error(t('skills.aiPanel.noContent'))
    return
  }
  const tmpl = t('skills.aiPanel.reviewPromptTemplate')
  inputText.value = tmpl.replace(/\{skill_md\}/g, props.fileContent || '')
  nextTick(() => inputEl.value?.focus())
}

// ===== 发送 / 取消 =====
let _uid = 0
function uid() { _uid += 1; return `m_${Date.now()}_${_uid}` }

function sendOrStop() {
  if (busy.value) {
    abort.value?.abort?.()
    return
  }
  sendMessage()
}

// 2026-07-13 增 v2:system prompt 中加入「customPromptHint」前缀,告诉 AI 自行判断 needs_apply。
//   翻译/检测标签:在 confirmTranslate/onReviewClick 里已经拼上了对应 promptTemplate(自带 JSON 格式要求)
//   自定义输入(用户直接打字):用这个 hint 包一层,让 AI 也能返回结构化 JSON
function buildUserPrompt(text) {
  // 如果用户在输入框里已经有内容(翻译/检测标签已经填好完整 prompt),直接发
  // 否则加上 customPromptHint 引导 AI 自行判断 needs_apply
  if (text.includes('```json') || text.includes('"needs_apply"')) {
    return text
  }
  return `${t('skills.aiPanel.customPromptHint')}\n\n${text}`
}

// ===== JSON 解析与 retry =====
//
// AI 返回结构化 JSON,前端剥离 ```json ... ``` 代码块标记后 parse。
// 解析失败 → 自动 retry(让 AI 重新生成),最多 3 次。3 次后兜底:
//   - needs_apply: false
//   - content: ''
//   - reason: 原始 AI 输出(让用户能看到 AI 说了什么)
// 这种情况下不显示"应用"按钮(纯展示,用户可手动复制)。
const MAX_PARSE_RETRIES = 3

function extractJsonBlock(raw) {
  if (!raw) return null
  // 匹配 ```json ... ``` 或 ``` ... ```
  const m = raw.match(/```(?:json)?\s*([\s\S]*?)```/)
  if (!m) {
    // 退而求其次:整段当 JSON 试
    const trimmed = raw.trim()
    if (trimmed.startsWith('{') && trimmed.endsWith('}')) return trimmed
    return null
  }
  return m[1].trim()
}

function parseAiJson(raw) {
  const blk = extractJsonBlock(raw)
  if (!blk) return { ok: false, raw }
  try {
    const obj = JSON.parse(blk)
    if (typeof obj !== 'object' || obj === null) return { ok: false, raw }
    const needsApply = obj.needs_apply === true
    const content = typeof obj.content === 'string' ? obj.content : ''
    const reason = typeof obj.reason === 'string' ? obj.reason : ''
    return { ok: true, needsApply, content, reason, raw }
  } catch (_) {
    return { ok: false, raw }
  }
}

// 兜底对象:解析失败时使用
function fallbackResult(raw) {
  return { ok: true, needsApply: false, content: '', reason: raw || '', raw }
}

// 真正发起一次 chat 流式调用,挂到指定 aiMsg 上,完成后回调 finalText
async function runChatStream({ provider, messages, onChunk, signal }) {
  return await chatStream(
    { provider, messages },
    {
      onEvent: (ev) => {
        if (ev.kind === 'chunk') onChunk(ev.text || '')
      },
      // onDone / onError 由外层 sendMessage 处理
    },
  )
}

async function sendMessage() {
  const text = (inputText.value || '').trim()
  if (!text || busy.value) return
  if (!hasProvider.value) {
    toast.error(t('skills.aiPanel.noProvider'))
    return
  }

  // 截断过长内容(避免单次请求 token 超限)
  const safeText = text.length > 16000 ? text.slice(0, 16000) + '\n\n...(已截断)' : text
  const userText = buildUserPrompt(safeText)

  const userMsg = { id: uid(), role: 'user', content: safeText, ts: Date.now() }
  messages.value.push(userMsg)
  inputText.value = ''
  busy.value = true
  let pendingBuf = ''
  let capped = false
  const MAX_AI_BYTES = 50 * 1024

  const provider = (providers.value.find((p) => p.enabled && p.has_key) || {}).name || ''
  // 第一轮:先用一个 placeholder aiMsg 让用户看到流式过程
  const aiMsg = { id: uid(), role: 'assistant', content: '', pending: true, ts: Date.now(), canApply: false, retriesLeft: 0 }
  messages.value.push(aiMsg)

  // 复用同一个 aiMsg,每次 retry 覆盖 content
  const baseMessages = [{ role: 'user', content: userText }]
  const systemHint = t('skills.aiPanel.customPromptHint')

  for (let attempt = 0; attempt <= MAX_PARSE_RETRIES; attempt++) {
    if (attempt > 0) {
      // retry 时:重置 pendingBuf,aiMsg 显示「重新生成中…」
      pendingBuf = ''
      capped = false
      aiMsg.content = ''
      aiMsg.pending = true
      aiMsg.retriesLeft = MAX_PARSE_RETRIES - attempt + 1
      aiMsg.retrying = true
      // 把上一轮失败的输出 + 修正指令追加到 messages,让 AI 自我纠正
      baseMessages.push({
        role: 'assistant',
        content: aiMsg.rawLast || '',
      })
      baseMessages.push({
        role: 'user',
        content: `你上一轮返回的数据无法被前端正确解析为 JSON。请严格按照下面的 schema 重新输出,且只输出一个 \`\`\`json 代码块:\n\`\`\`json\n{"needs_apply": boolean, "content": "string", "reason": "string"}\n\`\`\`\n注意:必须用 \`\`\`json 代码块包裹,布尔值是 true/false 不是字符串。`,
      })
    }

    let attemptBuf = ''
    let attemptErr = ''
    try {
      const ctrl = await runChatStream({
        provider,
        messages: baseMessages,
        onChunk: (chunk) => {
          attemptBuf += chunk
          if (attemptBuf.length > MAX_AI_BYTES) {
            attemptBuf = attemptBuf.slice(0, MAX_AI_BYTES) + '\n\n...(已截断)'
            capped = true
            abort.value?.abort?.()
          }
          aiMsg.content = attemptBuf
          aiMsg.pending = true
        },
      })
      abort.value = ctrl
    } catch (e) {
      attemptErr = e?.message || String(e)
    }

    // 流结束,检查解析结果
    aiMsg.pending = false
    aiMsg.retrying = false
    if (attemptErr) {
      aiMsg.error = attemptErr
      break
    }
    if (capped) {
      aiMsg.error = t('skills.aiPanel.truncated')
      break
    }

    const parsed = parseAiJson(attemptBuf)
    if (parsed.ok) {
      aiMsg.needsApply = parsed.needsApply
      aiMsg.content = parsed.content
      aiMsg.reason = parsed.reason
      aiMsg.canApply = parsed.needsApply && !!parsed.content.trim()
      aiMsg.parseFailed = false
      aiMsg.retriesLeft = 0
      break
    }
    aiMsg.rawLast = attemptBuf
    if (attempt >= MAX_PARSE_RETRIES) {
      // 3 次都失败 → 兜底:不显示应用按钮
      const fb = fallbackResult(attemptBuf)
      aiMsg.needsApply = false
      aiMsg.content = ''
      aiMsg.reason = fb.reason
      aiMsg.canApply = false
      aiMsg.parseFailed = true
      aiMsg.retriesLeft = 0
      break
    }
    // 继续下一轮 retry
  }

  busy.value = false
  abort.value = null
  persistSession()
}

// ===== 应用 / 拒绝 =====
function applyMessage(m) {
  if (m.applied || m.rejected) return
  if (!m.needsApply) return
  const text = m.content || ''
  if (!text.trim()) return
  m.applied = true
  if (props.isSkillMd) emit('apply-skill', text)
  else emit('apply-local', text)
  persistSession()
}

function rejectMessage(m) {
  if (m.applied || m.rejected) return
  m.rejected = true
  m.canApply = false
  persistSession()
}

// ===== 全屏编辑 =====
const fullscreenOpen = ref(false)
const fullscreenText = ref('')
function openFullscreen() {
  fullscreenText.value = inputText.value || ''
  fullscreenOpen.value = true
}
function saveFullscreen() {
  inputText.value = fullscreenText.value
  fullscreenOpen.value = false
  nextTick(() => inputEl.value?.focus())
}
function cancelFullscreen() {
  fullscreenOpen.value = false
}

// ===== 文件切换 =====
watch(() => props.filePath, () => {
  if (busy.value) abort.value?.abort?.()
  busy.value = false
  abort.value = null
  persistSession() // 旧 filePath 在切走前持久化一次(保险)
  loadSession()
})

// ===== 生命周期 =====
onMounted(async () => {
  loadSession()
  await loadProviders()
})

onBeforeUnmount(() => {
  if (busy.value) abort.value?.abort?.()
})

const inputEl = ref(null)
const listEl = ref(null)

watch(messages, async () => {
  // 新消息追加后,自动滚到底部
  await nextTick()
  if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
}, { deep: true })

const inputDisabled = computed(() => props.readOnly || !hasProvider.value || busy.value)
const sendDisabled = computed(() => props.readOnly || !hasProvider.value || busy.value || !(inputText.value || '').trim())
</script>

<template>
  <div class="airp">
    <!-- 头部 -->
    <header class="airp-header">
      <span class="airp-title">
        <IconPark icon="mdi:robot-outline" width="14" height="14" />
        {{ t('skills.aiPanel.roleAI') }}
      </span>
      <div class="airp-header-actions">
        <button
          class="airp-icon-btn"
          :data-tip="t('skills.aiPanel.switchToOutline')"
          :aria-label="t('skills.aiPanel.switchToOutline')"
          type="button"
          @click="emit('switch-outline')"
        >
          <IconPark icon="mdi:format-list-bulleted" width="13" height="13" />
        </button>
        <button
          v-if="messages.length"
          class="airp-icon-btn"
          :data-tip="t('skills.aiPanel.clearHistory')"
          :aria-label="t('skills.aiPanel.clearHistory')"
          type="button"
          @click="clearHistory"
        >
          <IconPark icon="mdi:delete-outline" width="13" height="13" />
        </button>
        <button
          class="airp-icon-btn"
          :data-tip="t('skills.aiPanel.closeBtn')"
          :aria-label="t('skills.aiPanel.closeBtn')"
          type="button"
          @click="emit('close')"
        >
          <IconPark icon="mdi:close" width="13" height="13" />
        </button>
      </div>
    </header>

    <!-- 未配置 provider 提示 -->
    <p v-if="!hasProvider" class="airp-warn">
      <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
      <span>{{ t('skills.aiPanel.noProvider') }}</span>
    </p>

    <!-- 消息列表 -->
    <div ref="listEl" class="airp-list">
      <p v-if="!messages.length" class="airp-empty">
        <img
          class="airp-empty-img"
          src="/images/skill-box-dog.png"
          alt="skill-box mascot"
          draggable="false"
        />
        <span class="airp-empty-text">{{ t('skills.aiPanel.emptyHint') }}</span>
      </p>
      <div
        v-for="m in messages"
        :key="m.id"
        :class="['airp-msg', m.role === 'user' ? 'airp-msg-user' : 'airp-msg-ai']"
      >
        <!-- 头像:AI 在左、用户在右 -->
        <img
          v-if="m.role === 'ai' || m.role === 'assistant'"
          class="airp-msg-avatar"
          src="/images/agent-dog-avatar.png"
          :alt="t('skills.aiPanel.roleAI')"
          draggable="false"
        />
        <div v-else class="airp-msg-avatar airp-msg-avatar-placeholder">
          <IconPark icon="mdi:account" width="18" height="18" />
        </div>

        <div class="airp-msg-body">
          <div class="airp-msg-meta">
            <span class="airp-msg-role">
              {{ m.role === 'user' ? t('skills.aiPanel.roleYou') : t('skills.aiPanel.roleAI') }}
            </span>
            <!-- retry 提示 -->
            <span v-if="m.retrying" class="airp-msg-retry">
              <IconPark icon="mdi:refresh" width="10" height="10" />
              {{ t('skills.aiPanel.retrying', { left: m.retriesLeft }) }}
            </span>
          </div>

          <!-- 用户消息 -->
          <div v-if="m.role === 'user'" class="airp-bubble airp-bubble-user">{{ m.content }}</div>

          <!-- AI 消息:reason 是给用户看的;content 仅在 needsApply 时是替换用的全文 -->
          <template v-else>
            <div v-if="m.parseFailed" class="airp-msg-warn">
              <IconPark icon="mdi:alert-circle-outline" width="11" height="11" />
              {{ t('skills.aiPanel.parseFailed') }}
            </div>
            <pre v-if="m.reason && !m.pending" class="airp-bubble airp-bubble-ai">{{ m.reason }}</pre>
            <pre v-else-if="m.pending || m.retrying" class="airp-bubble airp-bubble-ai">{{ m.content }}<span class="airp-cursor">▍</span></pre>
            <p v-if="m.error" class="airp-msg-error">[{{ t('skills.aiPanel.roleAI') }}] {{ m.error }}</p>

            <!-- 应用 / 拒绝:仅在 AI 明确返回 needs_apply=true 且未应用/拒绝时显示 -->
            <div v-if="!m.pending && !m.applied && !m.rejected && m.needsApply && m.canApply" class="airp-msg-actions">
              <button class="primary sm" type="button" @click="applyMessage(m)">
                <IconPark icon="mdi:check" width="12" height="12" />
                {{ t('skills.aiPanel.apply') }}
              </button>
              <button class="sm" type="button" @click="rejectMessage(m)">
                <IconPark icon="mdi:close" width="12" height="12" />
                {{ t('skills.aiPanel.reject') }}
              </button>
            </div>
            <p v-else-if="m.applied" class="airp-msg-hint airp-msg-hint-ok">
              <IconPark icon="mdi:check-circle-outline" width="11" height="11" />
              {{ t('skills.aiPanel.applied') }}
            </p>
            <p v-else-if="m.rejected" class="airp-msg-hint airp-msg-hint-mute">
              <IconPark icon="mdi:close-circle-outline" width="11" height="11" />
              {{ t('skills.aiPanel.rejected') }}
            </p>
          </template>
        </div>
      </div>
    </div>

    <!-- 底部输入区 -->
    <div class="airp-input-wrap">
      <div class="airp-tags">
        <button
          class="airp-tag-pill"
          type="button"
          :disabled="!hasProvider || !props.fileContent"
          @click="onTranslateClick"
        >
          <IconPark icon="mdi:translate" width="11" height="11" />
          {{ t('skills.aiPanel.tagTranslate') }}
        </button>
        <button
          class="airp-tag-pill"
          type="button"
          :disabled="!hasProvider || !props.fileContent"
          @click="onReviewClick"
        >
          <IconPark icon="mdi:magnify-scan" width="11" height="11" />
          {{ t('skills.aiPanel.tagReview') }}
        </button>
      </div>
      <div class="airp-input-row">
        <textarea
          ref="inputEl"
          v-model="inputText"
          class="airp-input"
          :placeholder="t('skills.aiPanel.inputPlaceholder')"
          :disabled="inputDisabled"
          rows="3"
          @keydown.enter.exact.prevent="sendOrStop"
        />
      </div>
      <div class="airp-actions">
        <!-- 全屏编辑按钮:放在发送按钮左侧,避免挤占输入框宽度 -->
        <!-- data-tip-top:tooltip 显示在按钮上方(底部边缘按钮下方易被裁切) -->
        <button
          class="airp-icon-btn airp-fullscreen-btn"
          type="button"
          :data-tip="t('skills.aiPanel.fullscreenEdit')"
          data-tip-top="true"
          :aria-label="t('skills.aiPanel.fullscreenEdit')"
          @click="openFullscreen"
        >
          <IconPark icon="mdi:arrow-expand" width="13" height="13" />
        </button>
        <button
          v-if="busy"
          class="airp-stop"
          type="button"
          data-tip-top="true"
          :data-tip="t('skills.aiPanel.stop')"
          :aria-label="t('skills.aiPanel.stop')"
          @click="abort?.abort?.()"
        >
          <IconPark icon="mdi:stop" width="12" height="12" />
          {{ t('skills.aiPanel.stop') }}
        </button>
        <button
          v-else
          class="airp-send"
          type="button"
          data-tip-top="true"
          :data-tip="t('skills.aiPanel.send')"
          :aria-label="t('skills.aiPanel.send')"
          :disabled="sendDisabled"
          @click="sendMessage"
        >
          <IconPark icon="mdi:send" width="12" height="12" />
          {{ t('skills.aiPanel.send') }}
        </button>
      </div>
    </div>

    <!-- 全屏编辑 Modal:让用户在大面板里编辑输入文本(避免 70-160px 限制) -->
    <Modal
      v-model="fullscreenOpen"
      size="full"
      :title="t('skills.aiPanel.fullscreenEditTitle')"
    >
      <textarea
        v-model="fullscreenText"
        class="airp-fullscreen-textarea"
        :placeholder="t('skills.aiPanel.inputPlaceholder')"
        spellcheck="false"
      />
      <div class="airp-dialog-actions">
        <button type="button" @click="cancelFullscreen">
          {{ t('skills.aiPanel.translateDialog.cancel') }}
        </button>
        <button class="primary" type="button" @click="saveFullscreen">
          {{ t('skills.aiPanel.fullscreenSave') }}
        </button>
      </div>
    </Modal>

    <!-- 翻译弹窗 -->
    <Modal
      v-model="translateDialogOpen"
      size="sm"
      :title="t('skills.aiPanel.translateDialog.title')"
    >
      <p class="airp-dialog-desc">{{ t('skills.aiPanel.translateDialog.desc') }}</p>
      <label class="airp-dialog-label">
        <span>目标语言</span>
        <select v-model="targetLang">
          <option v-for="l in langOptions" :key="l.value" :value="l.value">{{ l.label }}</option>
        </select>
      </label>
      <div class="airp-dialog-actions">
        <button type="button" @click="translateDialogOpen = false">
          {{ t('skills.aiPanel.translateDialog.cancel') }}
        </button>
        <button class="primary" type="button" @click="confirmTranslate">
          {{ t('skills.aiPanel.translateDialog.confirm') }}
        </button>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.airp {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-card);
  overflow: hidden;
}

/* ===== Header ===== */
.airp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 8px 8px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg-card);
  flex-shrink: 0;
}
.airp-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.airp-header-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.airp-icon-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--text-faint);
  cursor: pointer;
  transition: background 100ms ease, color 100ms ease, border-color 100ms ease;
}
.airp-icon-btn:hover {
  background: var(--bg-hover);
  color: var(--accent-blue);
  border-color: var(--border);
}

/* ===== Warn hint ===== */
.airp-warn {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 10px;
  font-size: 11.5px;
  background: rgba(245, 158, 11, 0.10);
  color: #b45309;
  border-bottom: 1px solid var(--border);
}

/* ===== 消息列表 ===== */
.airp-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 12px 12px 6px;
  scrollbar-width: thin;
  scrollbar-color: #d4d4d8 transparent;
  /* 空态需要在面板中央显示;把 list 当成 flex column,空态 flex:1 撑满高度再内部居中 */
  display: flex;
  flex-direction: column;
}
.airp-list::-webkit-scrollbar { width: 6px; height: 6px; }
.airp-list::-webkit-scrollbar-track { background: transparent; }
.airp-list::-webkit-scrollbar-thumb { background: #d4d4d8; border-radius: 3px; }

.airp-empty {
  /* 撑满 .airp-list 整个高度,让 flex 内容(图片+文字)相对面板中央居中 */
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin: 0 8px;
  text-align: center;
  color: var(--text-faint);
  font-size: 12.5px;
  line-height: 1.6;
}
.airp-empty-img {
  display: block;
  width: 96px;
  height: 96px;
  object-fit: contain;
  /* 黑色背景图配面板的浅底:加柔和投影 + 透明黑边,让狗狗不至于融在背景里 */
  border-radius: 12px;
  user-select: none;
  -webkit-user-drag: none;
}
.airp-empty-text {
  display: block;
  align-self: center;
}

/* ===== 聊天风格消息列表 ===== */
.airp-msg {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 14px;
  font-size: 12.5px;
  line-height: 1.55;
  word-break: break-word;
}
/* AI 在左(默认),用户在右(整列反转) */
.airp-msg-user {
  flex-direction: row-reverse;
}
.airp-msg-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-subtle);
  /* 头像自带黑色背景,加 1px 浅边让它在浅色面板里不突兀 */
  border: 1px solid var(--border);
  user-select: none;
  -webkit-user-drag: none;
}
.airp-msg-avatar-placeholder {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--text-faint);
  background: var(--primary-dim, rgba(59, 130, 246, 0.08));
  border: 1px solid var(--border);
}
.airp-msg-body {
  display: flex;
  flex-direction: column;
  /* body 自适应宽度,但不超过容器;max-width 85% 避免贴边 */
  max-width: calc(100% - 32px - 8px);
  min-width: 0;
}
.airp-msg-user .airp-msg-body {
  align-items: flex-end;
}

.airp-msg-meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 3px;
  font-size: 10.5px;
}
.airp-msg-role {
  font-weight: 600;
  color: var(--text-faint);
  letter-spacing: 0.02em;
}
.airp-msg-user .airp-msg-role { color: var(--accent-blue); }
.airp-msg-ai .airp-msg-role { color: var(--primary); }

.airp-bubble {
  margin: 0;
  padding: 8px 12px;
  border-radius: 10px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 12.5px;
  line-height: 1.55;
  max-width: 100%;
}
/* 用户气泡:蓝底白字,右上角略圆(模拟对话框尖角) */
.airp-bubble-user {
  background: var(--primary);
  color: var(--bg-card);
  border-top-right-radius: 4px;
}
/* AI 气泡:浅底深字,左上角略圆 */
.airp-bubble-ai {
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text);
  border-top-left-radius: 4px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
}

.airp-cursor {
  display: inline-block;
  animation: airp-blink 1s steps(1) infinite;
  color: var(--primary);
}
@keyframes airp-blink { 50% { opacity: 0; } }

.airp-msg-error {
  margin: 4px 0 0;
  padding: 6px 8px;
  font-size: 11.5px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  color: var(--danger, #dc2626);
  background: rgba(239, 68, 68, 0.10);
  border-radius: 4px;
}

.airp-msg-warn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 4px 0;
  padding: 4px 8px;
  font-size: 11px;
  color: #b45309;
  background: rgba(245, 158, 11, 0.10);
  border-radius: 4px;
}

.airp-msg-retry {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 6px;
  font-size: 10.5px;
  color: var(--accent-blue);
  font-weight: 500;
}

.airp-msg-reason {
  margin: 0;
  padding: 8px 10px;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 12.5px;
  line-height: 1.55;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text);
}

.airp-msg-actions {
  display: flex;
  gap: 6px;
  margin-top: 6px;
}
.airp-msg-actions button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  font-size: 11.5px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.airp-msg-actions button.primary {
  background: var(--primary);
  color: var(--bg-card);
  border-color: var(--primary);
}
.airp-msg-actions button.primary:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}
.airp-msg-actions button:hover:not(:disabled) {
  border-color: var(--primary);
  color: var(--primary);
}
.airp-msg-actions button:disabled { opacity: 0.5; cursor: not-allowed; }

.airp-msg-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 6px 0 0;
  font-size: 11px;
}
.airp-msg-hint-ok { color: var(--primary); }
.airp-msg-hint-mute { color: var(--text-faint); }

/* ===== 底部输入区 ===== */
.airp-input-wrap {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px 10px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
}
.airp-tags {
  display: flex;
  gap: 6px;
}
.airp-tag-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border-radius: 999px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 11.5px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.airp-tag-pill:hover:not(:disabled) {
  background: var(--primary-dim, rgba(59, 130, 246, 0.08));
  border-color: var(--primary);
  color: var(--primary);
}
.airp-tag-pill:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.airp-input-row {
  display: flex;
  gap: 4px;
  align-items: stretch;
}

.airp-fullscreen-textarea {
  width: 100%;
  min-height: 70vh;
  padding: 14px 16px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 6px;
  outline: none;
  resize: vertical;
}
.airp-fullscreen-textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-dim, rgba(59, 130, 246, 0.15));
}

.airp-input {
  width: 100%;
  resize: none;
  padding: 8px 10px;
  font-family: inherit;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--text);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 6px;
  outline: none;
  min-height: 70px;
  max-height: 160px;
  transition: border-color 100ms ease, box-shadow 100ms ease;
}
.airp-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-dim, rgba(59, 130, 246, 0.15));
}
.airp-input:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.airp-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}
.airp-fullscreen-btn {
  flex-shrink: 0;
  width: 28px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s ease;
}
.airp-fullscreen-btn:hover {
  background: var(--primary-dim, rgba(59, 130, 246, 0.08));
  border-color: var(--primary);
  color: var(--primary);
}
.airp-send,
.airp-stop {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 14px;
  border-radius: 5px;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}
.airp-send {
  background: var(--primary);
  color: var(--bg-card);
  border: 1px solid var(--primary);
}
.airp-send:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}
.airp-send:disabled { opacity: 0.5; cursor: not-allowed; }
.airp-stop {
  background: var(--danger, #dc2626);
  color: var(--bg-card);
  border: 1px solid var(--danger, #dc2626);
}

/* ===== 翻译弹窗 ===== */
.airp-dialog-desc {
  margin: 0 0 12px;
  font-size: 12.5px;
  color: var(--text-dim);
  line-height: 1.6;
}
.airp-dialog-label {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12.5px;
  color: var(--text-dim);
}
.airp-dialog-label select {
  padding: 6px 10px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--bg-card);
  color: var(--text);
  outline: none;
}
.airp-dialog-label select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-dim, rgba(59, 130, 246, 0.15));
}
.airp-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
.airp-dialog-actions button {
  padding: 5px 14px;
  border-radius: 5px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  font-size: 12.5px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.airp-dialog-actions button:hover {
  border-color: var(--primary);
  color: var(--primary);
}
.airp-dialog-actions button.primary {
  background: var(--primary);
  color: var(--bg-card);
  border-color: var(--primary);
}
.airp-dialog-actions button.primary:hover {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}
</style>