<!--
  AutoCommitPanel - 设置面板的"自动 commit message"配置块(2026-07-18 增)。
  嵌在 SettingsView 里,可独立测试 LLM 后启用 + 选择模板。

  设计要点:
  - LLMEnabled 必须后端判定 LLM 可用(enabled provider + api_key)才允许勾选;
    不可用时 checkbox 自动 disabled + 显示原因 + "测试 LLM"按钮
  - 模板 3 选 1(timestamp / filename / op_files)— 即使 LLM 不可用也能选,
    LLM 失败时降级到选中的模板
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
      testResult.value = t('git.autoCommit.testOk', { model: r.model, output: r.output })
      // 测试通过后刷新可用性 — 后端可能没返回 reason,所以拉一次 status
      await load()
    } else {
      testResult.value = t('git.autoCommit.testFail', { msg: r.reason || 'unknown' })
    }
  } catch (e) {
    testResult.value = t('git.autoCommit.testFail', { msg: e?.message || e })
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
      <!-- LLM 开关 + 可用性提示 -->
      <div class="pref-item">
        <div class="pref-label">
          <label>{{ t('git.autoCommit.llmEnabled.label') }}</label>
          <p class="pref-desc">{{ t('git.autoCommit.llmEnabled.desc') }}</p>
          <p v-if="!available" class="pref-warn">
            ⚠️ {{ t('git.autoCommit.llmEnabled.unavailableReason') }}{{ reason ? ` (${reason})` : '' }}
          </p>
          <p v-else class="pref-meta">
            <code>{{ providerName }}</code>
          </p>
        </div>
        <div class="pref-control">
          <label class="switch">
            <input
              type="checkbox"
              :checked="llmEnabled"
              :disabled="saving || !available"
              @change="(e) => onToggleLLM(e.target.checked)"
            />
            <span class="slider" />
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

      <!-- 模板单选 -->
      <div class="pref-item">
        <div class="pref-label">
          <label>{{ t('git.autoCommit.template.label') }}</label>
          <p class="pref-desc">{{ t('git.autoCommit.template.desc') }}</p>
        </div>
        <div class="pref-control acp-tpl-row">
          <label
            v-for="opt in [
              { v: 'filename',  d: t('git.autoCommit.template.filename') },
              { v: 'op_files',  d: t('git.autoCommit.template.opFiles') },
              { v: 'timestamp', d: t('git.autoCommit.template.timestamp') },
            ]"
            :key="opt.v"
            class="acp-radio"
            :class="{ active: template === opt.v }"
          >
            <input
              type="radio"
              name="acp-template"
              :value="opt.v"
              :checked="template === opt.v"
              :disabled="saving"
              @change="onTemplateChange(opt.v)"
            />
            <span class="acp-radio-dot" />
            <span class="acp-radio-text">{{ opt.d }}</span>
          </label>
        </div>
      </div>

      <!-- status 提示 -->
      <div v-if="saveHint || testResult" class="pref-status">
        <span v-if="saveHint">{{ saveHint }}</span>
        <span v-if="testResult">{{ testResult }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.acp-icon {
  font-size: 18px;
  margin-right: 2px;
}
.pref-warn { color: rgb(245, 158, 11); }
.pref-meta { color: var(--text-faint, rgba(127,127,127,0.7)); }
.pref-meta code {
  background: var(--bg-card, rgba(127,127,127,0.08));
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--font-mono, monospace);
  font-size: 11px;
}
.pref-control {
  display: flex;
  align-items: center;
  gap: 10px;
}
.acp-test-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border-color, rgba(127,127,127,0.2));
  background: var(--bg-card, #fff);
  color: var(--text);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: background 100ms ease;
}
.acp-test-btn:hover:not(:disabled) {
  background: var(--bg-hover, rgba(127,127,127,0.05));
}
.acp-test-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.acp-tpl-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}
.acp-radio {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  user-select: none;
}
.acp-radio.active { background: rgba(59, 130, 246, 0.1); color: rgb(59, 130, 246); }
.acp-radio input { display: none; }
.acp-radio-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid rgba(127,127,127,0.4);
  position: relative;
  flex-shrink: 0;
}
.acp-radio.active .acp-radio-dot { border-color: rgb(59, 130, 246); }
.acp-radio.active .acp-radio-dot::after {
  content: '';
  position: absolute;
  inset: 3px;
  border-radius: 50%;
  background: rgb(59, 130, 246);
}
.pref-status {
  margin-top: 8px;
  padding: 6px 10px;
  background: var(--bg-card, rgba(127,127,127,0.04));
  border-left: 2px solid rgb(59, 130, 246);
  border-radius: 3px;
  font-size: 11px;
  color: var(--text-dim);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
</style>
