<script setup>
// GitSyncPanel - 技能仓库 Git 同步状态面板(2026-07-17 增)
//
// 嵌入 SkillScopePanel 内部,默认折叠,点击 header 展开。
// 跟作用域面板同款交互(可点击 header 展开/收起,折叠态只显示 header 行)。
//
// 2026-07-17 设计决策:
//   - 用 vue-i18n 的 t(key) 而非 plainT(key, fallback) — plainT 的第二参是
//     插值变量,不是 fallback 字符串,所有 .vue 里 plainT('git.xxx', 'fallback')
//     都把 fallback 当成 {key: 'fallback'} 插值对象,显示成字面量
//   - 默认折叠 — 跟作用域面板一致,不抢首页视觉重点
//   - 位置:在 SkillScopePanel 内部,作用域区下方(改 <section> 渲染顺序)

import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  getGitStatus,
  getGitConfig,
  saveGitConfig,
  initGit,
  pushGit,
  discardGit,
} from '@/api/skillbox/git.js'
import IconPark from '@/components/IconPark.vue'

const { t } = useI18n()

// 2026-07-17 改:默认折叠,跟作用域面板一致
const expanded = ref(false)
// 是否展示配置表单(默认收起,有 remote_url 才展开)
const configOpen = ref(false)

const formRemoteURL = ref('')
const formBranch = ref('main')
const formToken = ref('')
const formUserName = ref('')
const formUserEmail = ref('')
const formSaving = ref(false)

const status = ref({
  initialized: false,
  branch: '',
  remote_url: '',
  remote_branch: '',
  head_hash: '',
  head_short: '',
  head_message: '',
  working_clean: true,
  ahead: 0,
  behind: 0,
  has_token: false,
  pending_push: 0,
  last_push_error: '',
})
const cfgLoaded = ref(false)
const loading = ref(false)
const errorMsg = ref('')

const hasRemote = computed(() => !!status.value.remote_url)

async function refresh() {
  loading.value = true
  errorMsg.value = ''
  try {
    const [st, cfg] = await Promise.all([getGitStatus(), getGitConfig()])
    status.value = { ...status.value, ...st }
    if (!cfgLoaded.value) {
      formRemoteURL.value = cfg.remote_url || ''
      formBranch.value = cfg.branch || 'main'
      formUserName.value = cfg.user_name || ''
      formUserEmail.value = cfg.user_email || ''
      cfgLoaded.value = true
    }
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doInit() {
  loading.value = true
  try {
    await initGit()
    await refresh()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doPush() {
  loading.value = true
  try {
    await pushGit()
    await refresh()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function doDiscard() {
  if (!confirm(t('git.discardConfirm'))) return
  loading.value = true
  try {
    await discardGit()
    await refresh()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  const url = formRemoteURL.value.trim()
  if (url && !/^https:\/\//i.test(url)) {
    errorMsg.value = t('git.invalidUrl')
    return
  }
  formSaving.value = true
  errorMsg.value = ''
  try {
    await saveGitConfig({
      remote_url: url,
      branch: formBranch.value.trim() || 'main',
      token: formToken.value,
      user_name: formUserName.value.trim(),
      user_email: formUserEmail.value.trim(),
    })
    formToken.value = ''
    configOpen.value = false
    await refresh()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    formSaving.value = false
  }
}

function toggle() {
  expanded.value = !expanded.value
  if (expanded.value) refresh()
}

onMounted(refresh)
</script>

<template>
  <section class="ssp-git" :class="{ 'is-expanded': expanded }">
    <!-- Header 跟作用域面板同款(button 切换展开,chevron 提示状态) -->
    <button
      type="button"
      class="ssp-git-header ssp-git-header-toggle"
      :aria-expanded="expanded"
      @click="toggle"
    >
      <IconPark type="github" :size="14" />
      <span class="ssp-git-title">{{ t('git.title') }}</span>
      <span v-if="status.initialized" class="ssp-git-badge ok">
        <IconPark type="check-one" :size="10" />
        {{ status.branch || 'main' }} · {{ status.head_short || '-' }}
      </span>
      <span v-else class="ssp-git-badge warn">{{ t('git.notInit') }}</span>
      <span class="ssp-git-chevron">
        <IconPark :type="expanded ? 'up' : 'down'" :size="12" />
      </span>
    </button>

    <!-- 折叠态下不渲染 body,跟作用域面板一致 -->
    <div v-if="expanded" class="ssp-git-body">
      <!-- 未 init 状态 -->
      <div v-if="!status.initialized" class="ssp-git-empty">
        <p class="ssp-git-empty-tip">{{ t('git.initTip') }}</p>
        <button class="ssp-git-btn primary" :disabled="loading" @click="doInit">
          <IconPark type="plus" :size="12" />
          {{ t('git.init') }}
        </button>
      </div>

      <!-- 已 init 状态 -->
      <div v-else class="ssp-git-content">
        <div class="ssp-git-row">
          <div class="ssp-git-row-label">{{ t('git.remote') }}</div>
          <div class="ssp-git-row-value">
            <span v-if="hasRemote" class="ssp-git-remote-url" :title="status.remote_url">{{ status.remote_url }}</span>
            <span v-else class="ssp-git-remote-missing">{{ t('git.remoteMissing') }}</span>
            <span v-if="status.has_token" class="ssp-git-token-ok">
              <IconPark type="lock" :size="10" /> Token
            </span>
            <span v-else class="ssp-git-token-warn">
              <IconPark type="unlock" :size="10" /> {{ t('git.noToken') }}
            </span>
            <button class="ssp-git-icon-btn" :title="t('git.config')" @click="configOpen = !configOpen">
              <IconPark type="config" :size="12" />
            </button>
          </div>
        </div>

        <div class="ssp-git-row">
          <div class="ssp-git-row-label">{{ t('git.head') }}</div>
          <div class="ssp-git-row-value">
            <code class="ssp-git-head-hash">{{ status.head_short || '-' }}</code>
            <span class="ssp-git-head-msg" :title="status.head_message">{{ status.head_message || '(无)' }}</span>
          </div>
        </div>

        <div class="ssp-git-row">
          <div class="ssp-git-row-label">{{ t('git.workingTree') }}</div>
          <div class="ssp-git-row-value">
            <span v-if="status.working_clean" class="ssp-git-clean">
              <IconPark type="check-one" :size="10" /> {{ t('git.clean') }}
            </span>
            <span v-else class="ssp-git-dirty">
              <IconPark type="edit" :size="10" /> {{ t('git.dirty') }}
            </span>
            <span v-if="status.pending_push > 0" class="ssp-git-pending" :title="t('git.pendingTip')">
              <IconPark type="upload" :size="10" /> {{ t('git.pending', { n: status.pending_push }) }}
            </span>
          </div>
        </div>

        <div v-if="status.last_push_error" class="ssp-git-error">
          <IconPark type="warning" :size="12" />
          <span class="ssp-git-error-msg">{{ status.last_push_error }}</span>
        </div>

        <div class="ssp-git-actions">
          <button v-if="hasRemote" class="ssp-git-btn" :disabled="loading" @click="doPush">
            <IconPark type="upload" :size="12" />
            {{ t('git.push') }}
          </button>
          <button v-if="!status.working_clean" class="ssp-git-btn warn" :disabled="loading" @click="doDiscard">
            <IconPark type="undo" :size="12" />
            {{ t('git.discard') }}
          </button>
        </div>

        <div v-if="configOpen" class="ssp-git-config-form">
          <label class="ssp-git-form-row">
            <span class="ssp-git-form-label">{{ t('git.formRemoteURL') }}</span>
            <input v-model="formRemoteURL" type="text" placeholder="https://github.com/user/repo.git" class="ssp-git-input" />
          </label>
          <label class="ssp-git-form-row">
            <span class="ssp-git-form-label">{{ t('git.formBranch') }}</span>
            <input v-model="formBranch" type="text" placeholder="main" class="ssp-git-input" />
          </label>
          <label class="ssp-git-form-row">
            <span class="ssp-git-form-label">{{ t('git.formToken') }}</span>
            <input v-model="formToken" type="password" :placeholder="t('git.formToken')" class="ssp-git-input" />
          </label>
          <label class="ssp-git-form-row">
            <span class="ssp-git-form-label">{{ t('git.formUserName') }}</span>
            <input v-model="formUserName" type="text" class="ssp-git-input" />
          </label>
          <label class="ssp-git-form-row">
            <span class="ssp-git-form-label">{{ t('git.formUserEmail') }}</span>
            <input v-model="formUserEmail" type="text" class="ssp-git-input" />
          </label>
          <div class="ssp-git-form-actions">
            <button class="ssp-git-btn" :disabled="formSaving" @click="configOpen = false">{{ t('git.cancel') }}</button>
            <button class="ssp-git-btn primary" :disabled="formSaving" @click="saveConfig">{{ t('git.save') }}</button>
          </div>
        </div>

        <div v-if="errorMsg" class="ssp-git-error">
          <IconPark type="close" :size="12" />
          <span class="ssp-git-error-msg">{{ errorMsg }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* 2026-07-17 改:跟作用域面板样式对齐(.ssp- 前缀),用 section 包裹。
   折叠态只显示 header,展开态显示 body。 */

.ssp-git {
  margin-top: 8px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: var(--radius, 6px);
  background: var(--bg-elevated, rgba(255, 255, 255, 0.02));
}

.ssp-git-header {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 10px;
  background: transparent;
  border: 0;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary, currentColor);
  text-align: left;
}
.ssp-git-header:hover { background: rgba(127, 127, 127, 0.04); }
.ssp-git-title { flex: 0 0 auto; }

.ssp-git-badge {
  flex: 1 1 auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  font-family: var(--font-mono, monospace);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssp-git-badge.ok {
  background: rgba(34, 197, 94, 0.15);
  color: rgb(34, 197, 94);
}
.ssp-git-badge.warn {
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.ssp-git-chevron {
  flex: 0 0 auto;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  display: inline-flex;
  align-items: center;
}

.ssp-git-body {
  padding: 8px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ssp-git-empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}
.ssp-git-empty-tip {
  margin: 0;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}

.ssp-git-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ssp-git-row {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.ssp-git-row-label {
  flex: 0 0 40px;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.ssp-git-row-value {
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 11px;
}
.ssp-git-remote-url {
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-git-remote-missing { color: var(--text-muted, rgba(127, 127, 127, 0.5)); }
.ssp-git-token-ok { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.ssp-git-token-warn { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.ssp-git-head-hash { font-family: var(--font-mono, monospace); }
.ssp-git-head-msg {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.ssp-git-clean { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.ssp-git-dirty { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.ssp-git-pending { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }

.ssp-git-icon-btn {
  flex: 0 0 auto;
  background: transparent;
  border: 0;
  padding: 2px 4px;
  cursor: pointer;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.ssp-git-icon-btn:hover { color: var(--text-primary, currentColor); }

.ssp-git-error {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(239, 68, 68, 0.1);
  border-left: 2px solid rgb(239, 68, 68);
  padding: 4px 6px;
  border-radius: 3px;
  font-size: 11px;
  color: rgb(239, 68, 68);
}
.ssp-git-error-msg { word-break: break-all; }

.ssp-git-actions {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}

.ssp-git-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  font-size: 11px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  background: transparent;
  color: var(--text-primary, currentColor);
  border-radius: 4px;
  cursor: pointer;
}
.ssp-git-btn:hover:not(:disabled) { background: rgba(127, 127, 127, 0.08); }
.ssp-git-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ssp-git-btn.primary {
  border-color: rgb(59, 130, 246);
  background: rgba(59, 130, 246, 0.15);
  color: rgb(59, 130, 246);
}
.ssp-git-btn.warn {
  border-color: rgb(245, 158, 11);
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.ssp-git-config-form {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  background: rgba(127, 127, 127, 0.04);
  border-radius: 4px;
}
.ssp-git-form-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ssp-git-form-label {
  font-size: 10px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.ssp-git-input {
  padding: 3px 6px;
  font-size: 11px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  border-radius: 3px;
  background: var(--bg-primary, transparent);
  color: var(--text-primary, currentColor);
  font-family: var(--font-mono, monospace);
}
.ssp-git-form-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  margin-top: 4px;
}
</style>