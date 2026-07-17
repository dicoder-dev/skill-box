<script setup>
// MarketImportPanel - 从三方市场导入 skill(2026-07-18 增)。
//
// 放 OnboardingImportDialog 第一位 tab,只一个 URL 输入框 + sources 列表 +
// 操作提示。后端走 POST /api/skillbox/market/install-from-input,后端会
// 按 URL 域名识别市场源(skillhub.cn / skills.sh / github.com),用户
// 无需关心。
//
// 数据流:
//   1) 用户粘详情页 URL → 点「导入」
//   2) installFromInput({input, source_hint, group_path, conflict_mode})
//   3) 成功 → toast + emit('done') + notifyImportDone(provide 链路)
//   4) 409 冲突 → 弹「覆盖/另存为/取消」Modal
//   5) 400/404/422 → toast 错误条
//
// 参考 MarketView.vue L49-138 (sources 硬编码) + L345-432 (install + conflict)
// + L699-729 (conflict 弹窗模板) + L1357-1480 (conflict CSS)。

import { ref, inject, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { installFromInput } from '@/api/skillbox/market'
import { platform } from '@/platform'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

const emit = defineEmits(['done'])

// OnboardingImportDialog 注入的通知回调 — 父组件在拿到响应那一刻就刷新列表。
const notifyImportDone = inject('notifyImportDone', null)
const resetImportDoneSig = inject('resetImportDoneSig', null)

// OnboardingImportDialog provide 的目标分组(用户在顶部选择)。
// 空 = 走后端默认派生。MarketImportPanel 始终透传这个值给后端。
const targetGroupPath = inject('targetGroupPath', ref(''))

// 三个市场源(2026-07-18 增:本组件硬编码,跟 MarketView 的 sources 数组同款来源,
// 区别是这里每个源只展示"复制链接粘贴进来"的引导,不再做"装到 skill-box"按钮)。
const sources = [
  {
    id: 'skillhub-cn',
    name: 'SkillHub-CN',
    url: 'https://skillhub.cn/skills',
    accent: '#0ea5e9',
    sourceType: 'skillhub-cn',
    descKey: 'onboarding.market.descSkillhub',
    example: 'https://skillhub.cn/skills/code-review',
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    url: 'https://www.skills.sh/hot',
    accent: '#10b981',
    sourceType: 'skillssh',
    descKey: 'onboarding.market.descSkillssh',
    example: 'https://skills.sh/anthropics/skills/pdf',
  },
  {
    id: 'github',
    name: 'GitHub',
    url: 'https://github.com',
    accent: '#1f2328',
    accentSoft: '#656d76',
    sourceType: 'github',
    descKey: 'onboarding.market.descGithub',
    example: 'https://github.com/anthropics/skills/tree/main/skills/pdf',
  },
]

const userInput = ref('')
const installing = ref(false)
const installError = ref('')
// 2026-07-09 增:同名 skill 冲突 Modal(沿用 MarketView 的 conflict 三选一)
const conflict = ref(null)
// 上次成功结果(用于 emit('done') 的 payload)
const lastResult = ref(null)

async function openInExternal(url) {
  try {
    await platform.platform.openExternal(url)
  } catch (e) {
    // 跨平台兼容:web 端 window.open 被拦截也算异常,这里静默吞掉
  }
}

// 点示例条目 → 自动填入输入框(让用户点"导入"才走安装)
function fillExample(text) {
  if (installing.value) return
  userInput.value = String(text || '')
  installError.value = ''
}

async function doInstall(input, conflictMode) {
  installing.value = true
  installError.value = ''
  try {
    const out = await installFromInput({
      // 2026-07-18:source_hint 不传 — 后端按 URL 域名 auto 识别,支持任意源
      input,
      scope: 'global',
      conflict_mode: conflictMode,
      // 2026-07-18 增:目标分组(从 OnboardingImportDialog 注入的 ref 取)
      group_path: targetGroupPath.value || '',
    })
    lastResult.value = out
    toast.success(t('market.success.msg', {
      name: out.skill_name,
      version: out.skill_version || '0.1.0',
    }))
    installing.value = false
    // 通知 OnboardingImportDialog → SkillsView 立即 reload
    const payload = { ok: 1, failed: 0, results: [{ ok: true, name: out.skill_name }] }
    if (notifyImportDone) notifyImportDone(payload)
    if (emit) emit('done', payload)
  } catch (e) {
    const status = e?.response?.status || e?.status
    const data = e?.response?.data || e?.data || {}
    if (status === 409) {
      installing.value = false
      conflict.value = {
        name: data.skill_name || userInput.value,
        existingVersion: data.conflict_existing_version || '?',
        existingPath: data.conflict_existing_path || '',
        input,
      }
      return
    }
    installing.value = false
    const msg = String(data?.error || e?.message || e)
    if (status === 400) {
      installError.value = t('market.input.errInvalidInput') + (msg ? ` · ${msg}` : '')
    } else if (status === 404) {
      installError.value = t('market.input.errSkillNotFound', { msg })
    } else if (status === 422) {
      installError.value = t('market.input.errSkillMalformed', { msg })
    } else if (/timeout/i.test(msg)) {
      installError.value = t('market.input.errTimeout', { msg })
    } else {
      installError.value = t('market.input.errGeneric', { msg })
    }
    toast.error(installError.value)
  }
}

function onImport() {
  if (installing.value) return
  const input = userInput.value.trim()
  if (!input) {
    installError.value = t('onboarding.market.errEmpty')
    return
  }
  doInstall(input, '')
}

async function resolveConflict(mode) {
  if (mode === 'cancel' || !conflict.value) {
    conflict.value = null
    return
  }
  const c = conflict.value
  conflict.value = null
  if (mode === 'rename') {
    await doInstall(c.input, 'rename')
  } else {
    // overwrite
    await doInstall(c.input, 'overwrite')
  }
}

function reset() {
  if (installing.value) return
  userInput.value = ''
  installError.value = ''
  conflict.value = null
  if (resetImportDoneSig) resetImportDoneSig()
}

// 计算属性:当前用户输入解析后会走哪个 source(纯前端提示用,不传给后端)
const matchedSource = computed(() => {
  const v = userInput.value.trim().toLowerCase()
  if (!v.includes('://')) return null
  if (v.includes('skillhub.cn')) return sources[0]
  if (v.includes('skills.sh')) return sources[1]
  if (v.includes('github.com')) return sources[2]
  return null
})
</script>

<template>
  <div class="mip">
    <!-- 三方源列表:让用户知道有哪些源,带跳转官网 -->
    <div class="mip-sources">
      <button
        v-for="s in sources"
        :key="s.id"
        type="button"
        class="mip-source"
        :style="{
          '--accent': s.accent,
          '--accent-soft': s.accentSoft || s.accent,
        }"
        :title="t('onboarding.market.gotoSite', { name: s.name })"
        @click="openInExternal(s.url)"
      >
        <div class="mip-source-name">
          <IconPark icon="mdi:open-in-new" width="11" height="11" />
          {{ s.name }}
        </div>
        <div class="mip-source-desc">{{ t(s.descKey) }}</div>
        <div class="mip-source-example">
          <code>{{ s.example }}</code>
          <button
            type="button"
            class="mip-fill"
            :title="t('onboarding.market.fillExample')"
            @click.stop="fillExample(s.example)"
          >
            <IconPark icon="mdi:content-copy" width="11" height="11" />
          </button>
        </div>
      </button>
    </div>

    <!-- URL 输入框 + 导入按钮 -->
    <div class="mip-input-row">
      <div class="mip-input-wrap">
        <IconPark icon="mdi:link-variant" width="14" height="14" class="mip-input-icon" />
        <input
          v-model="userInput"
          type="text"
          class="mip-input"
          :placeholder="t('onboarding.market.inputPlaceholder')"
          :disabled="installing"
          @keyup.enter="onImport"
        />
        <button
          v-if="userInput"
          type="button"
          class="mip-clear"
          :title="t('onboarding.market.clear')"
          @click="userInput = ''; installError = ''"
        >
          <IconPark icon="mdi:close-circle" width="12" height="12" />
        </button>
      </div>
      <button
        type="button"
        class="mip-btn"
        :disabled="installing || !userInput.trim()"
        @click="onImport"
      >
        <IconPark v-if="!installing" icon="mdi:download" width="13" height="13" />
        <span v-else class="spinner-inline"></span>
        {{ installing ? t('onboarding.market.btnImporting') : t('onboarding.market.btnImport') }}
      </button>
    </div>

    <!-- 错误条 -->
    <p v-if="installError" class="mip-error">
      <IconPark icon="mdi:alert-circle-outline" width="12" height="12" />
      {{ installError }}
    </p>

    <!-- 简单操作提示 -->
    <p class="mip-tip">
      <IconPark icon="mdi:information-outline" width="12" height="12" />
      {{ t('onboarding.market.tip') }}
      <span v-if="matchedSource" class="mip-tip-source">
        · {{ t('onboarding.market.detectedSource', { name: matchedSource.name }) }}
      </span>
    </p>

    <!-- 2026-07-09 增:同名 skill 冲突三选一 Modal(复用 market.conflict.* 文案) -->
    <div v-if="conflict" class="mip-conflict-overlay" @click.self="resolveConflict('cancel')">
      <div class="mip-conflict-modal">
        <h3 class="mip-conflict-title">{{ t('market.conflict.title') }}</h3>
        <p class="mip-conflict-desc">
          {{ t('market.conflict.desc', {
            name: conflict.name,
            existingVersion: conflict.existingVersion,
            existingPath: conflict.existingPath,
          }) }}
        </p>
        <div class="mip-conflict-actions">
          <button class="mip-conflict-btn mip-conflict-overwrite" @click="resolveConflict('overwrite')">
            <IconPark icon="mdi:content-save-outline" width="13" height="13" />
            {{ t('market.conflict.overwrite') }}
          </button>
          <button class="mip-conflict-btn mip-conflict-rename" @click="resolveConflict('rename')">
            <IconPark icon="mdi:content-duplicate" width="13" height="13" />
            {{ t('market.conflict.rename') }}
          </button>
          <button class="mip-conflict-btn mip-conflict-cancel" @click="resolveConflict('cancel')">
            <IconPark icon="mdi:close" width="13" height="13" />
            {{ t('market.conflict.cancel') }}
          </button>
        </div>
        <p class="mip-conflict-hint">
          <span class="mip-conflict-hint-row">
            <strong>{{ t('market.conflict.overwrite') }}:</strong>
            {{ t('market.conflict.overwriteTip') }}
          </span>
          <span class="mip-conflict-hint-row">
            <strong>{{ t('market.conflict.rename') }}:</strong>
            {{ t('market.conflict.renameTip') }}
          </span>
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mip {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* 三个源卡片(并排) */
.mip-sources {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.mip-source {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  text-align: left;
  background: var(--surface-2, transparent);
  border: 1px solid var(--border, #2a2a2a);
  border-left: 3px solid var(--accent, #3b82f6);
  border-radius: 6px;
  color: inherit;
  font: inherit;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}
.mip-source:hover {
  background: var(--hover, rgba(255, 255, 255, 0.04));
}

.mip-source-name {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--accent);
}
.mip-source-desc {
  font-size: 11px;
  color: var(--text-dim, #999);
  line-height: 1.4;
}
.mip-source-example {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10.5px;
  color: var(--text-dim, #777);
  margin-top: 2px;
}
.mip-source-example code {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 10.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  background: rgba(255, 255, 255, 0.04);
  padding: 2px 5px;
  border-radius: 3px;
}
.mip-fill {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--text-dim, #999);
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
}
.mip-fill:hover {
  color: var(--accent);
  background: rgba(255, 255, 255, 0.06);
}

/* URL 输入框 + 按钮 */
.mip-input-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
}
.mip-input-wrap {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
}
.mip-input-icon {
  position: absolute;
  left: 10px;
  color: var(--text-dim, #999);
  pointer-events: none;
}
.mip-input {
  flex: 1;
  width: 100%;
  padding: 9px 28px 9px 32px;
  background: var(--surface-2, transparent);
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 6px;
  color: inherit;
  font: inherit;
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s ease;
}
.mip-input:focus {
  border-color: var(--accent-blue, #3b82f6);
}
.mip-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.mip-clear {
  position: absolute;
  right: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: var(--text-dim, #999);
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
}
.mip-clear:hover {
  color: var(--accent-red, #ef4444);
}
.mip-btn {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 16px;
  background: var(--accent-blue, #3b82f6);
  border: none;
  border-radius: 6px;
  color: #fff;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: opacity 0.15s ease;
}
.mip-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.mip-btn:hover:not(:disabled) {
  opacity: 0.85;
}

.spinner-inline {
  display: inline-block;
  width: 11px;
  height: 11px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 错误条 / 提示 */
.mip-error {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0;
  padding: 8px 10px;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 5px;
  color: var(--accent-red, #ef4444);
  font-size: 12px;
  line-height: 1.4;
}

.mip-tip {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 0;
  font-size: 11.5px;
  color: var(--text-dim, #999);
  line-height: 1.5;
}
.mip-tip-source {
  color: var(--accent-blue, #3b82f6);
  font-weight: 500;
}

/* 冲突 Modal */
.mip-conflict-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  animation: mip-fade 0.15s ease;
}
@keyframes mip-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}
.mip-conflict-modal {
  background: var(--bg, #1a1a1a);
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 8px;
  padding: 20px;
  width: 460px;
  max-width: 90vw;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}
.mip-conflict-title {
  margin: 0 0 8px;
  font-size: 15px;
  font-weight: 600;
}
.mip-conflict-desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--text-dim, #999);
  line-height: 1.5;
}
.mip-conflict-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.mip-conflict-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 8px 10px;
  background: transparent;
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 5px;
  color: inherit;
  font: inherit;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}
.mip-conflict-btn:hover {
  background: var(--hover, rgba(255, 255, 255, 0.04));
}
.mip-conflict-overwrite {
  border-color: var(--accent-red, #ef4444);
  color: var(--accent-red, #ef4444);
}
.mip-conflict-rename {
  border-color: var(--accent-blue, #3b82f6);
  color: var(--accent-blue, #3b82f6);
}
.mip-conflict-hint {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding-top: 10px;
  border-top: 1px solid var(--border, #2a2a2a);
  font-size: 11px;
  color: var(--text-dim, #999);
  line-height: 1.4;
}
.mip-conflict-hint-row strong {
  color: var(--text, #ddd);
  font-weight: 600;
  margin-right: 4px;
}
</style>
