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

import { ref, inject, computed, nextTick } from 'vue'
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

// 2026-07-18 增:三个市场源(本组件硬编码,跟 MarketView 的 sources 同款数据)。
// 2026-07-18 改:GitHub 卡不再只显示官网,改成显示"具体仓库列表"(参考 MarketView
// famousRepos 块),点击仓库直接跳到该 repo 的 skills 目录。
const sources = [
  {
    id: 'skillhub-cn',
    name: 'SkillHub-CN',
    url: 'https://skillhub.cn/skills',
    accent: '#0ea5e9',
    icon: 'mdi:earth',
    sourceType: 'skillhub-cn',
    descKey: 'onboarding.market.descSkillhub',
    example: 'https://skillhub.cn/skills/code-review',
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    url: 'https://www.skills.sh/hot',
    accent: '#10b981',
    icon: 'mdi:lightning-bolt',
    sourceType: 'skillssh',
    descKey: 'onboarding.market.descSkillssh',
    example: 'https://skills.sh/anthropics/skills/pdf',
  },
  {
    id: 'github',
    name: 'GitHub',
    // 2026-07-18 改:Github 主页 → GitHub skills 目录(更具体,直接能搜到主流仓库)
    url: 'https://github.com/topics/agent-skills',
    accent: '#1f2328',
    accentSoft: '#656d76',
    icon: 'mdi:github',
    sourceType: 'github',
    descKey: 'onboarding.market.descGithub',
    example: 'https://github.com/anthropics/skills/tree/main/skills/pdf',
    // 2026-07-18 增:GitHub 具体仓库快捷链接(从 MarketView 复用)
    repos: [
      { display: 'anthropics/skills', url: 'https://github.com/anthropics/skills/tree/main/skills' },
      { display: 'vercel-labs/agent-skills', url: 'https://github.com/vercel-labs/agent-skills/tree/main/skills' },
      { display: 'mattpocock/skills', url: 'https://github.com/mattpocock/skills/tree/main/skills' },
    ],
  },
]

const userInput = ref('')
const installing = ref(false)
const installError = ref('')
// 2026-07-18 增:input 元素 ref(供 onInputPaste / 主动 focus 用)
const inputEl = ref(null)
// 2026-07-09 增:同名 skill 冲突 Modal(沿用 MarketView 的 conflict 三选一)
const conflict = ref(null)
// 上次成功结果(用于 emit('done') 的 payload)
const lastResult = ref(null)

// 2026-07-18 增:容器粘贴兜底。用户焦点有时落在 mip-input-row 容器上
// (卡在 icon 旁边空白 / 焦点跑偏),CTRL+V 不会进入 input。这种情况
// 直接接住 paste 事件,把剪贴板文本写进 userInput。
function onContainerPaste(e) {
  if (installing.value) return
  const text = e?.clipboardData?.getData?.('text/plain') || e?.clipboardData?.getData?.('text') || ''
  if (!text) return
  e.preventDefault()
  // 文本是空(用户复制了非文本) → 静默忽略
  const next = (userInput.value || '') + text
  userInput.value = next
  installError.value = ''
  // 主动把焦点 + 光标移回 input 末尾
  if (inputEl.value) {
    inputEl.value.focus()
    try {
      const len = next.length
      inputEl.value.setSelectionRange(len, len)
    } catch (_) { /* setSelectionRange 失败时静默 */ }
  }
}

// 2026-07-18 增:input 自身 paste 兜底 — 显式 setData 写回 v-model。
// 原生 <input> 默认支持 paste,但某些 webview / 桌面 wails 场景下
// v-model 同步可能丢文本,显式补一刀最稳。
function onInputPaste(e) {
  const text = e?.clipboardData?.getData?.('text/plain') || e?.clipboardData?.getData?.('text') || ''
  if (!text) return
  // 不 preventDefault,让 input 自然处理;这里只是同步一下 v-model,
  // 防止 v-model 与 DOM value 不一致。
  nextTick(() => {
    userInput.value = inputEl.value?.value ?? userInput.value
    installError.value = ''
  })
}

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
    <!-- 2026-07-18 改:输入示例区(三个来源各一条,放在输入框上方,让用户知道怎么输入)
         点击示例 = 填入输入框。比卡片底部的 example 更显眼。 -->
    <div class="mip-examples">
      <div class="mip-examples-label">
        <IconPark icon="mdi:lightbulb-on-outline" width="12" height="12" />
        {{ t('onboarding.market.examplesLabel') }}
      </div>
      <div class="mip-examples-list">
        <button
          v-for="s in sources"
          :key="`ex-${s.id}`"
          type="button"
          class="mip-example"
          :style="{ '--accent': s.accent }"
          :title="t('onboarding.market.fillExample')"
          @click="fillExample(s.example)"
        >
          <IconPark :icon="s.icon || 'mdi:link-variant'" width="11" height="11" />
          <code class="mip-example-url">{{ s.example }}</code>
        </button>
      </div>
    </div>

    <!-- 2026-07-18 改:三个市场源卡片 — 提高标识度(加深底色 + 更大 padding + 显眼图标 +
         GitHub 卡片显示具体仓库)。 -->
    <div class="mip-sources">
      <div
        v-for="s in sources"
        :key="s.id"
        class="mip-source"
        :style="{
          '--accent': s.accent,
          '--accent-soft': s.accentSoft || s.accent,
        }"
      >
        <div class="mip-source-head">
          <div class="mip-source-icon" :style="{ background: s.accent, color: '#fff' }">
            <IconPark :icon="s.icon || 'mdi:link-variant'" width="16" height="16" />
          </div>
          <div class="mip-source-meta">
            <div class="mip-source-name">{{ s.name }}</div>
            <div class="mip-source-desc">{{ t(s.descKey) }}</div>
          </div>
          <button
            type="button"
            class="mip-source-open"
            :title="t('onboarding.market.gotoSite', { name: s.name })"
            @click="openInExternal(s.url)"
          >
            <IconPark icon="mdi:open-in-new" width="12" height="12" />
          </button>
        </div>
        <!-- 2026-07-18 增:GitHub 卡显示具体仓库快捷链接 -->
        <div v-if="s.repos && s.repos.length" class="mip-source-repos">
          <button
            v-for="r in s.repos"
            :key="r.url"
            type="button"
            class="mip-source-repo"
            :title="r.url"
            @click="openInExternal(r.url)"
          >
            <IconPark icon="mdi:github" width="10" height="10" />
            <code>{{ r.display }}</code>
          </button>
        </div>
      </div>
    </div>

    <!-- URL 输入框 + 导入按钮 -->
    <!-- 2026-07-18 改:在 input-row 容器挂 @paste,容许用户在容器内任意位置(包
         括左侧 icon / 右侧 clear 按钮之间的空白)粘贴;同时 input 也挂 @paste
         兜底,显式把 e.clipboardData 写回 userInput。解决"焦点在卡片按钮上时
         CTRL+V 失效"的问题。 -->
    <div
      class="mip-input-row"
      tabindex="0"
      @paste="onContainerPaste"
    >
      <div class="mip-input-wrap">
        <IconPark icon="mdi:link-variant" width="14" height="14" class="mip-input-icon" />
        <input
          ref="inputEl"
          v-model="userInput"
          type="text"
          class="mip-input"
          :placeholder="t('onboarding.market.inputPlaceholder')"
          :disabled="installing"
          @keyup.enter="onImport"
          @paste="onInputPaste"
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

/* 三个源卡片(并排) — 2026-07-18 改:加深底色 + 加大 padding + 显眼图标 + 整卡 hover */
.mip-sources {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.mip-source {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 14px 12px;
  text-align: left;
  /* 2026-07-18 改:用更明显的渐变底色 + 加深边框,跟"工具"/"全局目录"/"本地" tab
     的卡片形成明显区分,提升三方导入 tab 在弹窗里的视觉权重 */
  background: linear-gradient(135deg,
    color-mix(in srgb, var(--accent) 8%, var(--surface-2, transparent)),
    var(--surface-2, rgba(255, 255, 255, 0.03))
  );
  border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border, #2a2a2a));
  border-left: 4px solid var(--accent, #3b82f6);
  border-radius: 8px;
  color: inherit;
  font: inherit;
  transition: background 0.15s ease, border-color 0.15s ease, transform 0.15s ease;
}
.mip-source:hover {
  border-color: color-mix(in srgb, var(--accent) 60%, var(--border, #2a2a2a));
  background: linear-gradient(135deg,
    color-mix(in srgb, var(--accent) 14%, var(--surface-2, transparent)),
    var(--surface-2, rgba(255, 255, 255, 0.03))
  );
}

.mip-source-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.mip-source-icon {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 7px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}
.mip-source-meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.mip-source-name {
  font-size: 13.5px;
  font-weight: 700;
  color: var(--text, #f0f0f0);
  letter-spacing: 0.01em;
}
.mip-source-desc {
  font-size: 11.5px;
  color: var(--text-dim, #999);
  line-height: 1.4;
}
.mip-source-open {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: transparent;
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 5px;
  color: var(--text-dim, #aaa);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease;
}
.mip-source-open:hover {
  color: var(--accent);
  border-color: var(--accent);
}

/* 2026-07-18 增:GitHub 卡显示的具体仓库快捷链接 */
.mip-source-repos {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding-top: 6px;
  border-top: 1px dashed var(--border, #2a2a2a);
}
.mip-source-repo {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 7px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 4px;
  color: var(--text, #ddd);
  font: inherit;
  font-size: 10.5px;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.mip-source-repo:hover {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}
.mip-source-repo code {
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 10.5px;
  font-weight: 500;
}

/* 2026-07-18 增:输入示例区(三个来源各一条,放在输入框上方) */
.mip-examples {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  background: rgba(59, 130, 246, 0.04);
  border: 1px dashed color-mix(in srgb, var(--accent-blue, #3b82f6) 40%, var(--border, #2a2a2a));
  border-radius: 6px;
}
.mip-examples-label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-dim, #999);
}
.mip-examples-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mip-example {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--text, #ddd);
  font: inherit;
  font-size: 11.5px;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}
.mip-example:hover {
  background: var(--surface-2, rgba(255, 255, 255, 0.04));
  border-color: var(--accent);
  color: var(--accent);
}
.mip-example-url {
  flex: 1;
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 11.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  background: transparent;
  padding: 0;
  border-radius: 0;
}

/* URL 输入框 + 按钮 — 2026-07-18 改:加深边框,看起来更像输入框 */
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
  left: 12px;
  color: var(--accent-blue, #3b82f6);
  pointer-events: none;
}
.mip-input {
  flex: 1;
  width: 100%;
  padding: 11px 32px 11px 34px;
  /* 2026-07-18 改:加深背景 + 加深边框 + 加深 placeholder,看起来明显是输入框 */
  background: var(--bg, #141414);
  border: 1.5px solid var(--border-strong, #4a4a4a);
  border-radius: 6px;
  color: var(--text, #f0f0f0);
  font: inherit;
  font-size: 13.5px;
  font-weight: 500;
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.mip-input::placeholder {
  color: var(--text-dim, #888);
  font-weight: 400;
}
.mip-input:hover {
  border-color: color-mix(in srgb, var(--accent-blue, #3b82f6) 50%, var(--border-strong, #4a4a4a));
}
.mip-input:focus {
  border-color: var(--accent-blue, #3b82f6);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.2);
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
  padding: 0 18px;
  background: var(--accent-blue, #3b82f6);
  border: none;
  border-radius: 6px;
  color: #fff;
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
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
