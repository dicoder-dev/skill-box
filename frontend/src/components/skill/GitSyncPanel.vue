<script setup>
// GitSyncPanel - 技能仓库 Git 同步状态面板(2026-07-17 增)
//
// 嵌入 SkillScopePanel 顶部,展示:
//   - 仓库 init 状态 + branch + HEAD 短码 + HEAD message
//   - 远端 URL / branch / has_token
//   - Push 失败队列长度 + 最后错误(可展开)
//   - 操作:初始化 / 配置远端 / Push / Discard / 查看历史
//
// 2026-07-17 设计决策:
//   - 不内嵌 version 弹窗(那个在 SkillsView 里),这里只做"同步状态 + 操作入口"
//   - "配置远端"按钮打开 inline form(本组件内),避免依赖全局 Settings Tab
//   - token 字段用 type="password",本地不持久化明文,提交后由后端落 0600 文件
//   - 远端 URL 必须在客户端简单校验 https://(后端会再校验一次)
//   - 任何 push 失败 → 显示红条 + 最后错误,点击可重试

import { ref, computed, onMounted } from 'vue'
import { plainT } from '@/core/i18n/index.js'
import {
  getGitStatus,
  getGitConfig,
  saveGitConfig,
  initGit,
  pushGit,
  discardGit,
} from '@/api/skillbox/git.js'
import IconPark from '@/components/IconPark.vue'

// 是否展示配置表单(默认收起,有 remote_url 才展开)
const configOpen = ref(false)
// 表单字段
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
  if (!confirm(plainT('git.discardConfirm', '丢弃所有未提交改动?此操作不可撤销。'))) return
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
  // 客户端校验 https://
  const url = formRemoteURL.value.trim()
  if (url && !/^https:\/\//i.test(url)) {
    errorMsg.value = plainT('git.invalidUrl', '远端 URL 必须以 https:// 开头')
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
    formToken.value = '' // 写盘后清空,避免浏览器 cache 残留
    configOpen.value = false
    await refresh()
  } catch (e) {
    errorMsg.value = (e && e.message) || String(e)
  } finally {
    formSaving.value = false
  }
}

function openConfig() {
  configOpen.value = !configOpen.value
}

onMounted(refresh)

// 暴露给父组件(用于外部触发刷新,例如"查看历史"关闭后)
defineExpose({ refresh })
</script>

<template>
  <div class="git-sync-panel">
    <div class="git-header">
      <IconPark type="github" :size="14" />
      <span class="git-title">{{ plainT('git.title', 'Git 同步') }}</span>
      <span v-if="status.initialized" class="git-badge ok">
        <IconPark type="check-one" :size="10" />
        {{ status.branch || 'main' }} · {{ status.head_short || '-' }}
      </span>
      <span v-else class="git-badge warn">
        {{ plainT('git.notInit', '未初始化') }}
      </span>
      <button v-if="status.initialized && !configOpen" class="git-icon-btn" :title="plainT('git.config', '配置远端')" @click="openConfig">
        <IconPark type="config" :size="12" />
      </button>
      <button v-else-if="status.initialized && configOpen" class="git-icon-btn" :title="plainT('git.close', '收起')" @click="openConfig">
        <IconPark type="up" :size="12" />
      </button>
    </div>

    <!-- 未 init 状态 -->
    <div v-if="!status.initialized" class="git-empty">
      <p class="git-empty-tip">{{ plainT('git.initTip', '首次启用 Git 版本管理,点击下面按钮初始化本地仓库。') }}</p>
      <button class="git-btn primary" :disabled="loading" @click="doInit">
        <IconPark type="plus" :size="12" />
        {{ plainT('git.init', '初始化仓库') }}
      </button>
    </div>

    <!-- 已 init 状态 -->
    <div v-else class="git-body">
      <!-- 远端 / 同步状态 -->
      <div class="git-row">
        <div class="git-row-label">{{ plainT('git.remote', '远端') }}</div>
        <div class="git-row-value">
          <span v-if="hasRemote" class="git-remote-url" :title="status.remote_url">{{ status.remote_url }}</span>
          <span v-else class="git-remote-missing">{{ plainT('git.remoteMissing', '未配置') }}</span>
          <span v-if="status.has_token" class="git-token-ok">
            <IconPark type="lock" :size="10" /> Token
          </span>
          <span v-else class="git-token-warn">
            <IconPark type="unlock" :size="10" /> {{ plainT('git.noToken', '无 Token') }}
          </span>
        </div>
      </div>

      <div class="git-row">
        <div class="git-row-label">{{ plainT('git.head', '当前') }}</div>
        <div class="git-row-value">
          <code class="git-head-hash">{{ status.head_short || '-' }}</code>
          <span class="git-head-msg" :title="status.head_message">{{ status.head_message || '(无)' }}</span>
        </div>
      </div>

      <div class="git-row">
        <div class="git-row-label">{{ plainT('git.workingTree', '工作区') }}</div>
        <div class="git-row-value">
          <span v-if="status.working_clean" class="git-clean">
            <IconPark type="check-one" :size="10" /> {{ plainT('git.clean', '干净') }}
          </span>
          <span v-else class="git-dirty">
            <IconPark type="edit" :size="10" /> {{ plainT('git.dirty', '有改动') }}
          </span>
          <span v-if="status.pending_push > 0" class="git-pending" :title="plainT('git.pendingTip', '等待重试的 push 任务数')">
            <IconPark type="upload" :size="10" /> {{ status.pending_push }} pending
          </span>
        </div>
      </div>

      <!-- 失败错误 -->
      <div v-if="status.last_push_error" class="git-error">
        <IconPark type="warning" :size="12" />
        <span class="git-error-msg">{{ status.last_push_error }}</span>
      </div>

      <!-- 操作按钮 -->
      <div class="git-actions">
        <button v-if="hasRemote" class="git-btn" :disabled="loading" @click="doPush">
          <IconPark type="upload" :size="12" />
          {{ plainT('git.push', 'Push') }}
        </button>
        <button v-if="!status.working_clean" class="git-btn warn" :disabled="loading" @click="doDiscard">
          <IconPark type="undo" :size="12" />
          {{ plainT('git.discard', 'Discard') }}
        </button>
      </div>

      <!-- 配置表单(折叠) -->
      <div v-if="configOpen" class="git-config-form">
        <label class="git-form-row">
          <span class="git-form-label">{{ plainT('git.formRemoteURL', '远端 URL') }}</span>
          <input v-model="formRemoteURL" type="text" placeholder="https://github.com/user/repo.git" class="git-input" />
        </label>
        <label class="git-form-row">
          <span class="git-form-label">{{ plainT('git.formBranch', '分支') }}</span>
          <input v-model="formBranch" type="text" placeholder="main" class="git-input" />
        </label>
        <label class="git-form-row">
          <span class="git-form-label">{{ plainT('git.formToken', 'Token') }}</span>
          <input v-model="formToken" type="password" placeholder="github_pat_xxx (留空保留现有)" class="git-input" />
        </label>
        <label class="git-form-row">
          <span class="git-form-label">{{ plainT('git.formUserName', 'Author 名') }}</span>
          <input v-model="formUserName" type="text" placeholder="留空用环境变量" class="git-input" />
        </label>
        <label class="git-form-row">
          <span class="git-form-label">{{ plainT('git.formUserEmail', 'Author 邮箱') }}</span>
          <input v-model="formUserEmail" type="text" placeholder="留空用环境变量" class="git-input" />
        </label>
        <div class="git-form-actions">
          <button class="git-btn" :disabled="formSaving" @click="configOpen = false">{{ plainT('git.cancel', '取消') }}</button>
          <button class="git-btn primary" :disabled="formSaving" @click="saveConfig">
            {{ plainT('git.save', '保存') }}
          </button>
        </div>
      </div>

      <!-- 错误信息(操作级,不是 push 错误) -->
      <div v-if="errorMsg" class="git-error">
        <IconPark type="close" :size="12" />
        <span class="git-error-msg">{{ errorMsg }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.git-sync-panel {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: 6px;
  padding: 8px 10px;
  margin-bottom: 8px;
  background: var(--bg-elevated, rgba(255, 255, 255, 0.02));
  font-size: 12px;
}

.git-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}

.git-title {
  flex: 0 0 auto;
}

.git-badge {
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
.git-badge.ok {
  background: rgba(34, 197, 94, 0.15);
  color: rgb(34, 197, 94);
}
.git-badge.warn {
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.git-icon-btn {
  flex: 0 0 auto;
  background: transparent;
  border: 0;
  padding: 2px 4px;
  cursor: pointer;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.git-icon-btn:hover { color: var(--text-primary, currentColor); }

.git-empty {
  margin-top: 8px;
}
.git-empty-tip {
  margin: 0 0 6px;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}

.git-body {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.git-row {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.git-row-label {
  flex: 0 0 40px;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.git-row-value {
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 11px;
}
.git-remote-url {
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.git-remote-missing { color: var(--text-muted, rgba(127, 127, 127, 0.5)); }
.git-token-ok { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.git-token-warn { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.git-head-hash { font-family: var(--font-mono, monospace); }
.git-head-msg {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.git-clean { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.git-dirty { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.git-pending { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }

.git-error {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(239, 68, 68, 0.1);
  border-left: 2px solid rgb(239, 68, 68);
  padding: 4px 6px;
  border-radius: 3px;
  margin-top: 6px;
  font-size: 11px;
  color: rgb(239, 68, 68);
}
.git-error-msg {
  word-break: break-all;
}

.git-actions {
  display: flex;
  gap: 6px;
  margin-top: 6px;
}

.git-btn {
  flex: 0 0 auto;
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
.git-btn:hover:not(:disabled) { background: rgba(127, 127, 127, 0.08); }
.git-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.git-btn.primary {
  border-color: rgb(59, 130, 246);
  background: rgba(59, 130, 246, 0.15);
  color: rgb(59, 130, 246);
}
.git-btn.warn {
  border-color: rgb(245, 158, 11);
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.git-config-form {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  background: rgba(127, 127, 127, 0.04);
  border-radius: 4px;
}
.git-form-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.git-form-label {
  font-size: 10px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.git-input {
  padding: 3px 6px;
  font-size: 11px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  border-radius: 3px;
  background: var(--bg-primary, transparent);
  color: var(--text-primary, currentColor);
  font-family: var(--font-mono, monospace);
}
.git-form-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  margin-top: 4px;
}
</style>