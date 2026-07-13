<script setup>
// AIDialog.vue — 全局 AI 操作弹窗(替换旧 AIPanel 嵌入侧栏)。
//
// 触发:SkillsView 的工具栏 AI 图标 → v-model="open"。
// 当前内置操作:
//   - 翻译 Skill(translate_skill preset)
//   - 优化 Frontmatter(优化 frontmatter preset) — 暂用旧 preset 不重新设计 UI
//
// 设计要点:
//   - 复用已有 Modal.vue 做容器 + body 锁滚 + ESC 关闭
//   - 两栏 step 切换:
//     step 1 = action list
//     step 2 = 单个 action 的输入 + 输出(流式渲染 + 复制 + 应用到编辑器)
//   - 「应用」按钮:emit apply 把翻译结果回传 SkillsView,
//     父级直接调 updateSkill 落盘到 SKILL.md(2026-07-12 之前仅改 ref,
//     apply 不生效,根因是 onAIApply 没调 updateSkill)。
//
// 后端协议:
//   - 流式 chat: fetch /api/skillbox/ai/chat(SSE)
//   - 复用 ai.js 的 chatStream 客户端,带 abort
//   - 翻译的实际 system prompt 由后端 aiengine types.go 里的 translate_skill
//     preset 注入,前端这里的 raw prompt 文本仅作「预览 + 复制」用途
//     (用户可拿去别的工具跑,后端不受影响)
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Modal from '@/components/Modal.vue'
import IconPark from '@/components/IconPark.vue'
import { listProviders, chatStream } from '@/api/skillbox/ai.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // 当前 skill 全文(取自 SkillsView.currentSkillMd);用于 translate_skill 上下文
  contextText: { type: String, default: '' },
  // 当前 skill 名(只是显示,不参与逻辑)
  contextName: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'apply'])

const { t, locale } = useI18n()

const open = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

// providers 用于检测"还没配置 AI 模型"以及流式调用时显式指定 provider
const providers = ref([])
const hasProvider = ref(false)

async function loadProviders() {
  try {
    const r = await listProviders()
    providers.value = (r?.items || []).filter((p) => p && p.enabled && p.has_key)
    hasProvider.value = providers.value.length > 0
  } catch (e) {
    providers.value = []
    hasProvider.value = false
  }
}

// 弹窗打开时立即拉 providers(以及语言切换 — 让语言列表同步)
watch(open, (v) => {
  if (v) {
    loadProviders()
    activeView.value = 'list'
    resetTranslate()
    initPromptOnOpen()
  }
})
watch(locale, () => { /* 语言变化时 i18n 自动重渲,无需手动 */ })

const activeView = ref('list') // 'list' | 'translate'

// === 翻译子面板状态 ===
const targetLang = ref('en-US')
const translateBusy = ref(false)
const translateResult = ref('')
const translateErr = ref('')
const translateAbort = ref(null)
const translateAppliedHint = ref(false)
const promptCopied = ref(false)

// 用户对原始提示词的"当前编辑态"。
//   - 初次进入(以及点「重置为默认」)时,customPrompt = buildDefaultPrompt(targetLang)
//   - 用户在 textarea 里改了,customPrompt 就脱离默认值
//   - target_lang / skill_md 两个占位符由 buildEffectivePrompt() 在运行时
//     替换,然后作为 system prompt 发给后端。用户在 textarea 里看到的
//     `{skill_md}` / `{target_lang}` 是占位符本身的字面量,
//     当他们手动改动时,字面量会被保留(可读性最好),由前端在发请求前替换。
//
// 关键防御:为了避免 textarea 在 setup 渲染完成到 watch(open) 之前闪现 1 帧的
// 空字符串(然后才被默认值填上),这里直接初始化为默认值(不依赖 watch)。
const customPrompt = ref(buildDefaultPrompt(targetLang.value))

const langOptions = computed(() => {
  // 动态取 i18n 文件里 aiDialog.langs 的所有 key(value/title 都从 i18n 拿)
  const keys = ['zh-CN', 'zh-TW', 'en-US', 'ja-JP', 'ko-KR', 'fr-FR', 'de-DE', 'es-ES']
  return keys.map((k) => ({ value: k, label: t(`skills.aiDialog.langs.${k}`) }))
})

const effectiveProviderName = computed(() => providers.value[0]?.name || '')

// 取目标语言的可读 label(发请求前的最后渲染会用到)
function langLabelOf(code) {
  const opt = langOptions.value.find((l) => l.value === code)
  return opt?.label || code
}

// 由 i18n 模板 + 当前 targetLang 拼出默认 prompt。
// {target_lang} 替换成 label 让用户看更顺眼;
// {skill_md} 占位保留(让用户知道"skill 全文会自动拼到那里")。
function buildDefaultPrompt(langCode) {
  const tmpl = t('skills.aiDialog.translate.promptTemplate')
  return tmpl.replace(/\{target_lang\}/g, () => langLabelOf(langCode))
}

// 用户是否修改过 prompt(对比当前 lang 下的默认值)。
// 用于显示「已自定义」徽标 + 是否给「重置为默认」按钮。
const promptIsCustomized = computed(() => {
  const dft = buildDefaultPrompt(targetLang.value)
  return (customPrompt.value || '') !== dft
})

// 重置为默认(覆盖当前语言版本)
function resetPromptToDefault() {
  customPrompt.value = buildDefaultPrompt(targetLang.value)
}

// 切语言时:如果用户没有手改过 prompt,跟着 default 同步;
// 如果手改过,保持用户的文本不动(让他手动改 {target_lang} 也行,不强制覆盖)。
watch(targetLang, (next) => {
  if (!promptIsCustomized.value) {
    customPrompt.value = buildDefaultPrompt(next)
  }
})

// 打开弹窗时初始化 prompt(以及重置状态)
function initPromptOnOpen() {
  customPrompt.value = buildDefaultPrompt(targetLang.value)
}

function resetTranslate() {
  if (translateAbort.value?.abort) translateAbort.value.abort()
  translateAbort.value = null
  translateBusy.value = false
  translateResult.value = ''
  translateErr.value = ''
  translateAppliedHint.value = false
  promptCopied.value = false
}

function startTranslate() {
  translateBusy.value = true
  translateResult.value = ''
  translateErr.value = ''
  translateAppliedHint.value = false

  const skillMd = props.contextText || ''
  if (!skillMd) {
    translateBusy.value = false
    translateErr.value = t('skills.aiDialog.translate.noContext')
    return
  }
  if (!(customPrompt.value || '').trim()) {
    translateBusy.value = false
    translateErr.value = t('skills.aiDialog.translate.promptEmpty')
    return
  }

  // 发请求前:把用户当前编辑的 customPrompt 里的两个占位符都实际替换掉。
  //   - {target_lang} → 语言 label
  //   - {skill_md} → 当前 skill 全文
  // 这一步非常关键 — 不替换的话 LLM 收到的 system 还在引用占位符字面量,
  // 反而把翻译责任推给模型自己替换(而且模型未必识别这种大括号语法)。
  const effectiveSystem = customPrompt.value
    .replace(/\{target_lang\}/g, () => langLabelOf(targetLang.value))
    .replace(/\{skill_md\}/g, () => skillMd)

  // 走 messages 路线(不进 preset):这样前端用户改 system 真的就能影响 LLM 行为。
  // user message 留个空内容兜底 — 一些模型对 only-system-no-user 的边界会
  // 加 EOS 后立刻截断,把"开始输出翻译"主动交给 user 触发更稳。
  let pendingBuf = ''
  translateAbort.value = chatStream(
    {
      provider: effectiveProviderName.value,
      messages: [
        { role: 'system', content: effectiveSystem },
        { role: 'user', content: t('skills.aiDialog.translate.startInstruction') },
      ],
    },
    {
      onEvent: (ev) => {
        if (ev.kind === 'chunk') {
          pendingBuf += ev.text || ''
          translateResult.value = pendingBuf
        } else if (ev.kind === 'error') {
          translateErr.value = ev.err || 'unknown'
        }
      },
      onDone: () => {
        translateBusy.value = false
      },
      onError: (err) => {
        translateBusy.value = false
        translateErr.value = err?.message || String(err)
      },
    },
  )
}

function stopTranslate() {
  if (translateAbort.value?.abort) translateAbort.value.abort()
  translateBusy.value = false
}

function copyResult() {
  if (!translateResult.value) return
  navigator.clipboard?.writeText(translateResult.value).catch(() => {})
}

async function copyPromptTemplate() {
  // 复制:用户当前在 textarea 里看到的内容(含自定义)。{skill_md}/{target_lang} 占位符
  // 保留字面量,让用户拿走去别处也能看清原模板结构。
  const text = (customPrompt.value || '').trim()
  if (!text) return
  try {
    await navigator.clipboard?.writeText(text)
    promptCopied.value = true
    setTimeout(() => (promptCopied.value = false), 1500)
  } catch (_) {}
}

function applyToEditor() {
  if (!translateResult.value) return
  emit('apply', translateResult.value)
  translateAppliedHint.value = true
  setTimeout(() => (translateAppliedHint.value = false), 2200)
}
</script>

<template>
  <Modal
    v-model="open"
    size="lg"
    :title="activeView === 'list' ? t('skills.aiDialog.title') : t('skills.aiDialog.translate.title')"
  >
    <template #title-icon>
      <IconPark icon="mdi:robot" width="18" height="18" />
    </template>

    <!-- step 1:Action list -->
    <div v-if="activeView === 'list'" class="ai-actions">
      <p class="ai-desc">{{ t('skills.aiDialog.subtitle') }}</p>
      <p v-if="!hasProvider" class="hint-box warn-hint">
        <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
        <span>{{ t('skills.aiDialog.providerMissing') }}</span>
      </p>
      <ul class="action-list">
        <li>
          <button class="action-card" :disabled="!hasProvider" @click="activeView = 'translate'">
            <IconPark icon="mdi:translate" width="20" height="20" class="action-icon" />
            <div class="action-meta">
              <strong>{{ t('skills.aiDialog.actions.translate') }}</strong>
              <span>{{ t('skills.aiDialog.actions.translateDesc') }}</span>
            </div>
            <IconPark icon="mdi:chevron-right" width="16" height="16" class="action-arrow" />
          </button>
        </li>
        <li>
          <button class="action-card" disabled>
            <IconPark icon="mdi:auto-fix" width="20" height="20" class="action-icon" />
            <div class="action-meta">
              <strong>{{ t('skills.aiDialog.actions.optimize') }}</strong>
              <span>{{ t('skills.aiDialog.actions.optimizeDesc') }}</span>
            </div>
            <span class="badge">{{ t('skills.aiDialog.actions.comingSoon') }}</span>
          </button>
        </li>
      </ul>
    </div>

    <!-- step 2:Translate sub-panel -->
    <div v-else class="ai-translate">
      <button class="back-link" @click="activeView = 'list'">
        <IconPark icon="mdi:chevron-left" width="14" height="14" />
        {{ t('skills.aiDialog.actionsTitle') }}
      </button>

      <p class="ai-desc">{{ t('skills.aiDialog.translate.desc') }}</p>

      <div class="form-grid">
        <label>
          <span>{{ t('skills.aiDialog.translate.targetLang') }}</span>
          <select v-model="targetLang">
            <option v-for="l in langOptions" :key="l.value" :value="l.value">{{ l.label }}</option>
          </select>
        </label>
      </div>

      <label class="full-row">
        <span class="row-between">
          <span>
            {{ t('skills.aiDialog.translate.promptLabel') }}
            <span v-if="promptIsCustomized" class="badge badge-warn">
              {{ t('skills.aiDialog.translate.promptCustomized') }}
            </span>
          </span>
          <span class="row-actions">
            <button
              v-if="promptIsCustomized"
              class="link-btn"
              type="button"
              @click="resetPromptToDefault"
              :title="t('skills.aiDialog.translate.promptReset')"
            >
              <IconPark icon="mdi:restore" width="12" height="12" />
              {{ t('skills.aiDialog.translate.promptReset') }}
            </button>
            <button class="link-btn" type="button" @click="copyPromptTemplate">
              <IconPark :icon="promptCopied ? 'mdi:check' : 'mdi:content-copy'" width="12" height="12" />
              {{ promptCopied
                  ? t('skills.aiDialog.translate.promptCopied')
                  : t('skills.aiDialog.translate.promptCopy') }}
            </button>
          </span>
        </span>
        <textarea
          v-model="customPrompt"
          class="prompt-edit"
          rows="14"
          spellcheck="false"
        />
        <span class="hint-mute">{{ t('skills.aiDialog.translate.promptHint') }}</span>
      </label>

      <div v-if="translateErr" class="hint-box error-hint">
        <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
        <span class="error-msg">{{ translateErr }}</span>
      </div>

      <div class="result-area" v-if="translateResult || translateBusy">
        <div class="result-header">
          <IconPark icon="mdi:text-box-outline" width="14" height="14" />
          <strong>{{ t('skills.aiDialog.translate.resultTitle') }}</strong>
          <span v-if="translateBusy" class="spinner spinner-sm" />
        </div>
        <pre class="result-body">{{ translateResult }}<span v-if="translateBusy" class="cursor">▍</span></pre>
      </div>

      <div class="actions-row">
        <button v-if="translateBusy" class="danger" @click="stopTranslate">
          <IconPark icon="mdi:stop" width="14" height="14" />
          {{ t('skills.aiDialog.translate.stop') }}
        </button>
        <button v-else class="primary" :disabled="!hasProvider || !props.contextText" @click="startTranslate">
          <IconPark icon="mdi:send" width="14" height="14" />
          {{ t('skills.aiDialog.translate.submit') }}
        </button>
        <button v-if="translateResult" :disabled="translateBusy" @click="copyResult">
          <IconPark icon="mdi:content-copy" width="14" height="14" />
          {{ t('skills.aiDialog.translate.copyResult') }}
        </button>
        <button v-if="translateResult" :disabled="translateBusy" class="primary" @click="applyToEditor">
          <IconPark icon="mdi:check" width="14" height="14" />
          {{ translateAppliedHint
              ? t('skills.aiDialog.translate.applied')
              : t('skills.aiDialog.translate.applyToEditor') }}
        </button>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.ai-actions { display: flex; flex-direction: column; gap: 12px; }
.ai-desc {
  margin: 0 0 4px;
  color: var(--text-dim);
  font-size: 12.5px;
}
.action-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.action-card {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  cursor: pointer;
  text-align: left;
  transition: all 0.15s ease;
}
.action-card:hover:not(:disabled) {
  border-color: var(--primary);
  background: var(--primary-dim);
}
.action-card:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.action-icon { color: var(--primary); }
.action-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.action-meta strong { font-size: 14px; }
.action-meta span { font-size: 12px; color: var(--text-dim); }
.action-arrow { color: var(--text-faint); }
.badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}
.ai-translate {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.back-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: var(--text-dim);
  font-size: 12.5px;
  cursor: pointer;
  padding: 0;
}
.back-link:hover { color: var(--primary); }
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
label.full-row, .ai-translate label {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12.5px;
}
label.full-row > span, .ai-translate label > span { color: var(--text-dim); }
.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.row-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.badge {
  display: inline-block;
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 999px;
  margin-left: 6px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}
.badge-warn {
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
  border-color: rgba(245, 158, 11, 0.30);
}
.link-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  font-size: 11.5px;
  color: var(--primary);
  transition: color 0.15s ease;
}
.link-btn:hover { color: var(--primary-hover); }
.prompt-edit {
  width: 100%;
  padding: 10px 12px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12px;
  line-height: 1.55;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  resize: vertical;
  min-height: 200px;
  max-height: 360px;
}
.prompt-edit:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.ai-translate select,
.ai-translate textarea {
  padding: 6px 10px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  resize: vertical;
  font-family: inherit;
}
.ai-translate textarea { font-family: 'JetBrains Mono', ui-monospace, monospace; }
.ai-translate select:focus,
.ai-translate textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.hint-mute {
  font-size: 11.5px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  word-break: break-word;
  margin-top: 2px;
}
.hint-box {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  line-height: 1.5;
}
.warn-hint { background: rgba(245, 158, 11, 0.10); color: #b45309; }
.error-hint { background: rgba(239, 68, 68, 0.12); color: var(--danger, #dc2626); }
.error-msg {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 11.5px;
}
.result-area {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
  display: flex;
  flex-direction: column;
  max-height: 320px;
}
.result-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  font-size: 12.5px;
}
.result-body {
  margin: 0;
  padding: 10px 12px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  font-size: 12.5px;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
  flex: 1;
}
.cursor {
  display: inline-block;
  animation: blink 1s steps(1) infinite;
  color: var(--primary);
}
@keyframes blink { 50% { opacity: 0; } }
.actions-row {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
.actions-row button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.actions-row button.primary {
  background: var(--primary);
  color: var(--bg-card);
  border-color: var(--primary);
}
.actions-row button.primary:hover:not(:disabled) {
  background: var(--primary-hover);
  border-color: var(--primary-hover);
}
.actions-row button.danger {
  background: var(--danger);
  color: var(--bg-card);
  border-color: var(--danger);
}
.actions-row button:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
