<!--
  AutoCommitPanel - 设置面板的"自动 commit message"配置块(2026-07-18 重构)。

  设计要点:
  - LLMEnabled 必须后端判定 LLM 可用(enabled provider + api_key)才允许勾选;
    不可用时 checkbox 自动 disabled + 显示原因 + "测试 LLM"按钮
  - 模板 3 选 1(timestamp / filename / op_files)— 即使 LLM 不可用也能选,
    LLM 失败时降级到选中的模板
  - 2026-07-18 重构样式:
    * LLM 开关与测试按钮放一行(参考 settings.desktop.toggle 范式)
    * 模板区单独占一整行,分段式按钮组(参考 settings.applyMode.mode-segmented),
      左对齐紧跟引导语下方,选中态主色填充 + check 图标(跟 copy/symlink 一致)
    * status hint 用 .hint-box 全局组件,不自己画左 border
  - 保存成功 + 失败都有 hint message(setTimeout 清)
  - 与桌面 / web 均可见(无 platform 相关代码)
-->
<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import {
  getAutoCommitStatus,
  saveAutoCommit,
  testLLM,
} from '@/api/skillbox/git.js'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const llmEnabled = ref(false)
const template = ref('filename')
const available = ref(false)
const reason = ref('')
const providerName = ref('')
const testing = ref(false)
const testResult = ref('')
const saving = ref(false)
const saveHint = ref('')

async function load() {
  try {
    const r = await getAutoCommitStatus()
    llmEnabled.value = !!r.llm_enabled
    template.value = r.template || 'filename'
    available.value = !!r.available
    reason.value = r.reason || ''
    providerName.value = r.provider_name || ''
  } catch (e) {
    saveHint.value = t('git.autoCommit.loadFail', { msg: e?.message || e })
  }
}

async function onToggleLLM(v) {
  // 切到 true 时立刻本地乐观,但保存失败会再回滚
  llmEnabled.value = v
  await persist()
}

async function onTemplateChange(v) {
  if (v === template.value) return
  template.value = v
  await persist()
}

async function persist() {
  saving.value = true
  saveHint.value = ''
  try {
    await saveAutoCommit({
      llm_enabled: llmEnabled.value,
      template: template.value,
    })
    saveHint.value = t('git.autoCommit.saved')
    toast.success(saveHint.value)
  } catch (e) {
    // 后端校验失败时回滚本地勾选
    await load()
    const msg = e?.message || String(e)
    saveHint.value = t('git.autoCommit.saveFail', { msg })
    toast.error(saveHint.value)
    // 2026-07-18 增:LLM 不可用(后端 reason=unavailable)时,弹 toast 带
    // "跳到 AI 设置"action 按钮。后端 SaveAutoCommit handler 在 LLMEnabled=true
    // 但 provider/api_key 缺失时返 400 + reason=unavailable,这里识别这个状态
    // 给用户一个一键修复路径,而不是只看到红字"启用失败"。
    // axios/ofetch 风格的错误对象 e.response.data 拿不到时,fallback 看 error msg。
    const errData = e?.response?.data || e?.data || null
    const reason = errData?.reason || (typeof msg === 'string' && msg.includes('unavailable') ? 'unavailable' : '')
    if (reason === 'unavailable') {
      toast.action(
        t('git.autoCommit.goAiSetup', '跳到 AI 设置'),
        () => {
          if (typeof window !== 'undefined') {
            window.dispatchEvent(new CustomEvent('skillbox:tab-change', { detail: 'settings' }))
            // settings tab 内有 AI 设置子项,emit 子事件让 SettingsView 切过去
            window.dispatchEvent(new CustomEvent('skillbox:settings-subtab', { detail: 'ai' }))
          }
        },
        { type: 'error', message: msg, duration: 8000 },
      )
    }
  } finally {
    saving.value = false
    setTimeout(() => (saveHint.value = ''), 5000)
  }
}

async function runTest() {
  testing.value = true
  testResult.value = ''
  try {
    const r = await testLLM()
    if (r.ok) {
      // 2026-07-18 改:测试通过用 strongSuccess — 绿色背景醒目变体,跟
      // 普通 success(浅底)区分开,用户一眼能识别"通过 / 一般成功"。
      const msg = t('git.autoCommit.testOk', { model: r.model, output: r.output })
      testResult.value = msg
      toast.strongSuccess(msg)
      // 测试通过后刷新可用性 — 后端可能没返回 reason,所以拉一次 status
      await load()
    } else {
      testResult.value = t('git.autoCommit.testFail', { msg: r.reason || 'unknown' })
      toast.error(testResult.value)
    }
  } catch (e) {
    testResult.value = t('git.autoCommit.testFail', { msg: e?.message || e })
    toast.error(testResult.value)
  } finally {
    testing.value = false
    setTimeout(() => (testResult.value = ''), 8000)
  }
}

onMounted(load)
// 切回 settings tab 时自动刷 — 通过 window 事件兜底(其他类似 prefs component 同款)
if (typeof window !== 'undefined') {
  window.addEventListener('skillbox:tab-change', (e) => {
    if (e?.detail === 'settings') load()
  })
}
</script>

<template>
  <section class="card">
    <header class="card-header">
      <h3>
        <IconPark icon="mdi:history" width="18" height="18" />
        {{ t('git.autoCommit.title') }}
        <span class="card-sub">— {{ t('git.autoCommit.subtitle') }}</span>
      </h3>
    </header>

    <div class="pref-list">
      <!-- LLM 开关 + 可用性提示 + 测试按钮(右对齐控件组) -->
      <div class="pref-item">
        <div class="pref-info">
          <div class="pref-label">{{ t('git.autoCommit.llmEnabled.label') }}</div>
          <div class="pref-hint">{{ t('git.autoCommit.llmEnabled.desc') }}</div>
          <div v-if="!available" class="pref-warn">
            <IconPark icon="mdi:alert-circle-outline" width="13" height="13" />
            {{ t('git.autoCommit.llmEnabled.unavailableReason') }}{{ reason ? ` (${reason})` : '' }}
          </div>
          <div v-else class="pref-meta">
            <code>{{ providerName }}</code>
          </div>
        </div>
        <div class="acp-control">
          <label class="toggle">
            <input
              type="checkbox"
              :checked="llmEnabled"
              :disabled="saving"
              @change="(e) => onToggleLLM(e.target.checked)"
            />
            <span class="toggle-slider"></span>
          </label>
          <button
            class="acp-test-btn"
            :disabled="testing || saving"
            @click="runTest"
          >
            <span v-if="testing" class="spinner spinner-xs"></span>
            <IconPark v-else icon="mdi:test-tube" width="13" height="13" />
            {{ t('git.autoCommit.testBtn') }}
          </button>
        </div>
      </div>

      <!-- 模板单选 — 单独占一整行,分段式按钮组,左对齐紧跟引导语下方 -->
      <div class="pref-item pref-item-block">
        <div class="pref-info">
          <div class="pref-label">{{ t('git.autoCommit.template.label') }}</div>
          <div class="pref-hint">{{ t('git.autoCommit.template.desc') }}</div>
        </div>
        <div class="tpl-segmented">
          <button
            v-for="opt in [
              { v: 'filename',  d: t('git.autoCommit.template.filename') },
              { v: 'op_files',  d: t('git.autoCommit.template.opFiles') },
              { v: 'timestamp', d: t('git.autoCommit.template.timestamp') },
            ]"
            :key="opt.v"
            type="button"
            :class="['tpl-btn', template === opt.v ? 'tpl-btn-active' : '']"
            :disabled="saving"
            @click="onTemplateChange(opt.v)"
          >
            <IconPark
              v-if="template === opt.v"
              icon="mdi:check-circle"
              width="14"
              height="14"
              class="tpl-btn-icon"
            />
            <IconPark
              v-else
              icon="mdi:radiobox-blank"
              width="14"
              height="14"
              class="tpl-btn-icon"
            />
            <span class="tpl-btn-label">{{ opt.d }}</span>
          </button>
        </div>
      </div>

      <!-- status 提示(用全局 .hint-box 风格,跟其他 settings 卡片统一) -->
      <div v-if="saveHint || testResult" class="pref-status">
        <IconPark
          v-if="saveHint && !saveHint.startsWith('×')"
          icon="mdi:check-circle"
          width="14"
          height="14"
          class="hint-icon hint-success"
        />
        <IconPark
          v-if="saveHint && saveHint.startsWith('×')"
          icon="mdi:close-circle"
          width="14"
          height="14"
          class="hint-icon hint-error"
        />
        <IconPark
          v-if="!saveHint && testResult"
          icon="mdi:information"
          width="14"
          height="14"
          class="hint-icon"
        />
        <span>{{ saveHint || testResult }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* ──────────────────────────────────────────────────────────────
 * 2026-07-18:AutoCommitPanel 是独立组件,<style scoped> 隔离后
 * SettingsView 里的 .pref-item / .pref-info / .toggle 等样式不生效
 * (scoped 后选择器带 [data-v-xxx],跨组件匹配不上)。这里必须自带
 * 一份"pref 范式"样式,跟 SettingsView 保持视觉一致。
 * ────────────────────────────────────────────────────────────── */

/* 列表 + 项(沿用 SettingsView 的 pref-list / pref-item 范式) */
.pref-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}
.pref-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
  transition: background 0.15s ease;
}
.pref-item:first-child {
  padding-top: 0;
}
.pref-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}
/* 模板区:占完整宽度,控件在右侧贴齐(对齐策略跟语言切换器一致) */
.pref-item-block {
  align-items: center;
}

.pref-info {
  flex: 1;
  min-width: 0;
}
.pref-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  margin-bottom: 4px;
}
.pref-hint {
  font-size: 12px;
  color: var(--text-dim);
  max-width: 480px;
}

/* 开关(跟 SettingsView .toggle 完全一致) */
.toggle {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
  flex-shrink: 0;
}
.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}
.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--border);
  transition: 0.3s;
  border-radius: 26px;
}
.toggle-slider::before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: var(--bg-card);
  transition: 0.3s;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}
.toggle input:checked + .toggle-slider {
  background-color: var(--primary);
}
.toggle input:checked + .toggle-slider::before {
  transform: translateX(22px);
}

/* 右侧控件组:开关 + 测试按钮 横排 */
.acp-control {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

/* 警告文字 — 用 IconPark 替代 emoji ⚠️ */
.pref-warn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  padding: 6px 10px;
  background: rgba(245, 158, 11, 0.08);
  border-left: 2px solid rgb(245, 158, 11);
  border-radius: 3px;
  color: rgb(245, 158, 11);
  font-size: 12px;
  max-width: 480px;
}
.pref-meta {
  margin-top: 6px;
  color: var(--text-faint, rgba(127,127,127,0.7));
  font-size: 12px;
}
.pref-meta code {
  background: var(--bg-subtle);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--font-mono, monospace);
  font-size: 11px;
}

/* 测试按钮 */
.acp-test-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.12s ease;
  white-space: nowrap;
}
.acp-test-btn:hover:not(:disabled) {
  background: var(--bg-subtle);
  border-color: var(--primary);
  color: var(--primary);
}
.acp-test-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 模板分段式按钮 — 复用 settings.applyMode.mode-segmented 范式:
   浅底容器 + 选中态主色填充 + check 图标 + 阴影 */
.tpl-segmented {
  display: inline-flex;
  align-items: stretch;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
  flex-shrink: 0;
}
.tpl-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
  background: transparent;
  border: 1px solid transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  cursor: pointer;
  transition: all 0.12s ease;
  white-space: nowrap;
}
.tpl-btn:hover:not(:disabled):not(.tpl-btn-active) {
  color: var(--text);
  background: var(--bg-card);
}
.tpl-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.tpl-btn-icon {
  flex-shrink: 0;
}
.tpl-btn-label {
  line-height: 1;
}
.tpl-btn.tpl-btn-active {
  background: var(--primary);
  color: var(--primary-contrast, #fff);
  border-color: var(--primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.12);
  font-weight: 600;
}
.tpl-btn.tpl-btn-active .tpl-btn-icon {
  color: var(--primary-contrast, #fff);
}

/* status 提示 */
.pref-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 10px 14px;
  background: var(--bg-subtle);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text-dim);
}
.pref-status .hint-icon {
  flex-shrink: 0;
}
.pref-status .hint-success {
  color: var(--success);
}
.pref-status .hint-error {
  color: var(--danger);
}

/* 响应式 */
@media (max-width: 768px) {
  .pref-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .acp-control,
  .tpl-segmented {
    align-self: flex-start;
  }
}
</style>