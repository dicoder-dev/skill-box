<script setup>
// LocalImportPanel - 从本地文件夹 / zip 压缩包导入 skill。
//
// 行为:
//   - 点"选择文件夹" → platform.fs.pickFolder() → POST /api/skillbox/onboarding/import-local {mode:'folder', path}
//   - 点"选择 zip" →
//       桌面端:platform.fs.pickFile() → POST /api/skillbox/onboarding/import-local {mode:'zip_path', path}
//       Web 端:<input type=file> → POST /api/skillbox/onboarding/import-zip-bytes (octet-stream)
//   - 后端统一在 store pkg 校验 SKILL.md,0 命中返 ErrNoSkillMD,前端 toast 提示。
//
// 完成后 emit 'done' 通知父弹窗(OnboardingImportDialog)关闭,父视图刷新列表。
//
// 2026-07-01 新增,跟 OnboardingView(扫工具)并列放在 OnboardingImportDialog tab 容器里。

import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { platform } from '@/platform'
import { runOnboardingImportLocal, runOnboardingImportZipBytes } from '@/api/skillbox/onboarding'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const emit = defineEmits(['done'])

// phase: 'idle' | 'busy' | 'done'
const phase = ref('idle')
const error = ref('')
const result = ref(null)

// Web 端:用 <input type="file"> 兜底桌面端 pickFile
const fileInputRef = ref(null)

const isWeb = !platform.isDesktop

async function pickFolder() {
  if (phase.value === 'busy') return
  error.value = ''
  let path = ''
  try {
    path = await platform.fs.pickFolder()
  } catch (e) {
    toast.push({ type: 'error', message: t('onboarding.local.errNoPick') + ': ' + (e?.message || e) })
    return
  }
  if (!path) return // 用户取消
  await doImport({ mode: 'folder', path })
}

async function pickZipViaDialog() {
  if (phase.value === 'busy') return
  error.value = ''
  let path = ''
  try {
    path = await platform.fs.pickFile()
  } catch (e) {
    // 桌面端 pickFile 未实现时降级到 file input
    if (fileInputRef.value) fileInputRef.value.click()
    return
  }
  if (!path) return
  await doImport({ mode: 'zip_path', path })
}

function pickZipViaInput() {
  if (phase.value === 'busy') return
  if (fileInputRef.value) fileInputRef.value.click()
}

async function onZipFileChosen(e) {
  const file = e.target.files?.[0]
  // 同一文件能再次选:reset
  e.target.value = ''
  if (!file) return
  error.value = ''
  try {
    const buf = await file.arrayBuffer()
    await doImportZipBytes(buf)
  } catch (err) {
    error.value = err?.message || String(err)
    toast.push({ type: 'error', message: t('onboarding.local.errImport', { msg: error.value }) })
  }
}

async function doImport(payload) {
  phase.value = 'busy'
  error.value = ''
  try {
    const r = await runOnboardingImportLocal(payload)
    onImportResult(r)
  } catch (e) {
    error.value = e?.message || e
    phase.value = 'idle'
    if (/no SKILL\.md/i.test(error.value)) {
      toast.push({ type: 'error', message: t('onboarding.local.errNoSKILLMD') })
    } else {
      toast.push({ type: 'error', message: t('onboarding.local.errImport', { msg: error.value }) })
    }
  }
}

async function doImportZipBytes(buf) {
  phase.value = 'busy'
  error.value = ''
  try {
    const r = await runOnboardingImportZipBytes(buf)
    onImportResult(r)
  } catch (e) {
    error.value = e?.message || e
    phase.value = 'idle'
    if (/no SKILL\.md/i.test(error.value)) {
      toast.push({ type: 'error', message: t('onboarding.local.errNoSKILLMD') })
    } else {
      toast.push({ type: 'error', message: t('onboarding.local.errImport', { msg: error.value }) })
    }
  }
}

function onImportResult(r) {
  result.value = r
  phase.value = 'done'
  if (r?.ok > 0) {
    toast.push({ type: 'success', message: t('onboarding.local.okImport', { ok: r.ok, failed: r.failed || 0 }) })
  }
}

function reset() {
  phase.value = 'idle'
  result.value = null
  error.value = ''
}

function finish() {
  emit('done', result.value)
}
</script>

<template>
  <div class="lip">
    <!-- 阶段 1: 选择来源 -->
    <section v-if="phase === 'idle'" class="lip-pane">
      <p class="lip-desc">{{ t('onboarding.local.desc') }}</p>

      <div class="lip-actions">
        <button
          v-if="!isWeb"
          class="lip-action"
          :title="t('onboarding.local.btnPickFolderTitle')"
          @click="pickFolder"
        >
          <Icon icon="mdi:folder-open-outline" width="28" height="28" />
          <span class="lip-action-name">{{ t('onboarding.local.btnPickFolder') }}</span>
        </button>
        <div v-else class="lip-disabled" :title="t('onboarding.local.webNoFolderTitle', { default: '' }) || t('onboarding.local.webNoFolder')">
          <button class="lip-action" disabled :title="t('onboarding.local.webNoFolder')">
            <Icon icon="mdi:folder-open-outline" width="28" height="28" />
            <span class="lip-action-name">{{ t('onboarding.local.btnPickFolder') }}</span>
          </button>
          <p class="lip-hint">{{ t('onboarding.local.webNoFolderHint') }}</p>
        </div>

        <button
          class="lip-action"
          :title="t('onboarding.local.btnPickZipTitle')"
          @click="isWeb ? pickZipViaInput() : pickZipViaDialog()"
        >
          <Icon icon="mdi:folder-zip-outline" width="28" height="28" />
          <span class="lip-action-name">{{ t('onboarding.local.btnPickZip') }}</span>
        </button>
      </div>

      <!-- Web 端隐藏 file input,触发选 zip -->
      <input
        v-if="isWeb"
        ref="fileInputRef"
        type="file"
        accept=".zip"
        style="display: none"
        @change="onZipFileChosen"
      />
    </section>

    <!-- 阶段 2: 导入中 -->
    <section v-else-if="phase === 'busy'" class="lip-pane lip-busy">
      <span class="spinner spinner-lg"></span>
      <p>{{ t('onboarding.local.importing') }}</p>
    </section>

    <!-- 阶段 3: 结果统计 -->
    <section v-else-if="phase === 'done' && result" class="lip-pane">
      <header class="lip-result-head">
        <Icon icon="mdi:check-circle" width="18" height="18" />
        <h3>{{ t('onboarding.local.resultTitle') }}</h3>
      </header>

      <div class="lip-stats">
        <div class="lip-stat stat-ok">
          <span class="lip-stat-num">{{ result.ok || 0 }}</span>
          <span class="lip-stat-label">{{ t('onboarding.local.statOk') }}</span>
        </div>
        <div class="lip-stat stat-err">
          <span class="lip-stat-num">{{ result.failed || 0 }}</span>
          <span class="lip-stat-label">{{ t('onboarding.local.statErr') }}</span>
        </div>
        <div class="lip-stat">
          <span class="lip-stat-num">{{ result.found || 0 }}</span>
          <span class="lip-stat-label">{{ t('onboarding.local.statFound') }}</span>
        </div>
      </div>

      <ul v-if="result.results?.length" class="lip-result-list">
        <li
          v-for="(r, i) in result.results"
          :key="i"
          :class="r.ok ? 'lip-row-ok' : 'lip-row-err'"
        >
          <Icon
            :icon="r.ok ? 'mdi:check' : 'mdi:close-circle-outline'"
            width="14"
            height="14"
            class="lip-row-icon"
          />
          <span class="lip-row-name"><code>{{ r.name }}</code></span>
          <span v-if="r.version" class="lip-row-ver">v{{ r.version }}</span>
          <span v-if="!r.ok && r.error" class="lip-row-msg">{{ r.error }}</span>
        </li>
      </ul>

      <div class="lip-footer">
        <button class="ghost" @click="reset">
          <Icon icon="mdi:refresh" width="14" height="14" />
          {{ t('onboarding.local.btnAgain') }}
        </button>
        <button class="primary" @click="finish">
          <Icon icon="mdi:check" width="14" height="14" />
          {{ t('onboarding.local.btnDone') }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.lip {
  max-width: 720px;
  margin: 0 auto;
  padding: 4px 0;
  color: var(--text);
}

.lip-pane {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 28px 24px;
  box-shadow: var(--shadow-card);
}

.lip-desc {
  margin: 0 0 20px;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.6;
}

.lip-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.lip-action {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px 16px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  cursor: pointer;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.lip-action:hover:not(:disabled) {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue-border);
  color: var(--accent-blue);
  transform: translateY(-1px);
}

.lip-action:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.lip-action-name {
  font-size: 14px;
  font-weight: 500;
}

.lip-disabled {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.lip-hint {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
  text-align: center;
}

/* 阶段 2: 导入中 */
.lip-busy {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 60px 24px;
  color: var(--text-dim);
  font-size: 14px;
}

.spinner-lg {
  width: 28px;
  height: 28px;
  border: 3px solid var(--border);
  border-top-color: var(--accent-blue);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 阶段 3: 结果 */
.lip-result-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
}

.lip-result-head h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.lip-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 18px;
}

.lip-stat {
  padding: 16px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  text-align: center;
}

.lip-stat-num {
  display: block;
  font-size: 28px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
  margin-bottom: 6px;
}

.lip-stat-label {
  font-size: 12px;
  color: var(--text-dim);
}

.stat-ok {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
}
.stat-ok .lip-stat-num { color: var(--accent-emerald); }

.stat-err {
  background: var(--accent-rose-bg);
  border-color: var(--accent-rose-border);
}
.stat-err .lip-stat-num { color: var(--accent-rose); }

.lip-result-list {
  list-style: none;
  padding: 0;
  margin: 0 0 18px;
  max-height: 280px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.lip-result-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  font-size: 13px;
  border-bottom: 1px solid var(--border);
}

.lip-result-list li:last-child {
  border-bottom: none;
}

.lip-row-ok {
  background: var(--accent-emerald-bg);
  color: var(--accent-emerald);
}

.lip-row-err {
  background: var(--accent-rose-bg);
  color: var(--accent-rose);
}

.lip-row-icon {
  flex-shrink: 0;
}

.lip-row-name {
  font-weight: 500;
  color: var(--text);
}

.lip-row-name code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
}

.lip-row-ver {
  font-size: 11px;
  color: var(--text-dim);
  font-weight: 500;
}

.lip-row-msg {
  font-size: 12px;
  color: inherit;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 60%;
}

.lip-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 14px;
  border-top: 1px solid var(--border);
}

@media (max-width: 600px) {
  .lip-actions { grid-template-columns: 1fr; }
  .lip-stats { grid-template-columns: 1fr; }
}
</style>