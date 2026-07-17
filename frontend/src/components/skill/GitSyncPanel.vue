<script setup>
// GitSyncPanel - 技能仓库 Git 同步状态面板(2026-07-17 增)
//
// 2026-07-17 改:用通用 CollapsiblePanel 包裹,header 样式与作用域面板一致。
// 默认折叠,点 header 展开;accordion 模式:同 parent 内任一面板展开会收起其他。

import { ref, computed, onMounted, inject } from 'vue'
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
import CollapsiblePanel from '@/components/CollapsiblePanel.vue'

const { t } = useI18n()

// 2026-07-17 增:可选 skillPath,非空时"查看历史"按钮会 emit 这个路径,
// 让父组件打开 VersionHistoryModal 并传 skillPath 实现 per-skill 历史。
// 2026-07-17 改:历史面板改成 inline 自动展示,GitSyncPanel 不再需要
// emit show-history 事件;button 也删除,避免重复入口。
// (面板位置:Git 同步面板正下方,默认折叠)
const props = defineProps({
  skillPath: { type: String, default: '' },
})

// 2026-07-17 改:从本地 ref 改成走 AccordionGroup 协调器(注入式)。
// 没包 AccordionGroup 时退化为本地 ref,行为跟旧版一致。
const coordinator = inject('cpCoordinator', null)
const localExpanded = ref(false)
const gitExpanded = computed(() => {
  if (coordinator) {
    return coordinator.activeId === 'git'
  }
  return localExpanded.value
})
function onGitToggle(open) {
  if (coordinator) {
    coordinator.toggle('git', open)
  } else {
    localExpanded.value = open
  }
  if (open && !cfgLoaded.value) refresh()
}

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

// 2026-07-17 改:init 现在是 fire-and-forget(后端 202 + 异步跑),前端
// 轮询 status 看 init 是否完成。最多 8 秒(40 × 200ms),超时后停止轮询
// 让用户继续操作(后台 init 会跑完,下次 refresh 就能看到结果)。
async function doInit() {
  loading.value = true
  try {
    await initGit()
    // 轮询 status,等 initialized=true 出现就停
    for (let i = 0; i < 40; i++) {
      await new Promise((r) => setTimeout(r, 200))
      const st = await getGitStatus()
      if (st && st.initialized) {
        status.value = { ...status.value, ...st }
        break
      }
    }
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

onMounted(() => {
  // 不主动 refresh,等用户点开再拉(避免面板未显示就发请求)
})
</script>

<template>
  <CollapsiblePanel
    :expanded="gitExpanded"
    :title="t('git.title')"
    icon="github"
    panel-id="git"
    @update:expanded="onGitToggle"
  >
    <template #title-meta>
      <!-- 2026-07-17 改:zero head 显示 'no commits yet' 而非 raw hash;
           没 init 显示 '未初始化';都靠 status.head_hash 是否为空判断。 -->
      <span v-if="status.initialized && status.head_hash" class="gsp-badge ok">
        <IconPark type="check-one" :size="10" />
        {{ status.branch || 'main' }} · {{ status.head_short || '-' }}
      </span>
      <span v-else-if="status.initialized" class="gsp-badge warn">
        <IconPark type="warning" :size="10" />
        {{ t('git.noCommits', '无提交') }}
      </span>
      <span v-else class="gsp-badge warn">{{ t('git.notInit') }}</span>
    </template>

    <div v-if="!status.initialized" class="gsp-empty">
      <p class="gsp-empty-tip">{{ t('git.initTip') }}</p>
      <button class="gsp-btn primary" :disabled="loading" @click="doInit">
        <IconPark type="plus" :size="12" />
        {{ t('git.init') }}
      </button>
    </div>

    <div v-else class="gsp-content">
      <div class="gsp-row">
        <div class="gsp-row-label">{{ t('git.remote') }}</div>
        <div class="gsp-row-value">
          <span v-if="hasRemote" class="gsp-remote-url" :title="status.remote_url">{{ status.remote_url }}</span>
          <span v-else class="gsp-remote-missing">{{ t('git.remoteMissing') }}</span>
          <span v-if="status.has_token" class="gsp-token-ok">
            <IconPark type="lock" :size="10" /> Token
          </span>
          <span v-else class="gsp-token-warn">
            <IconPark type="unlock" :size="10" /> {{ t('git.noToken') }}
          </span>
          <button class="gsp-icon-btn" :title="t('git.config')" @click="configOpen = !configOpen">
            <IconPark type="config" :size="12" />
          </button>
        </div>
      </div>

      <div class="gsp-row">
        <div class="gsp-row-label">{{ t('git.head') }}</div>
        <div class="gsp-row-value">
          <code class="gsp-head-hash">{{ status.head_short || (status.initialized ? t('git.noCommits', '无提交') : '-') }}</code>
          <span class="gsp-head-msg" :title="status.head_message">{{ status.head_message || (status.initialized ? t('git.noCommits', '无提交') : t('git.notInit', '未初始化')) }}</span>
        </div>
      </div>

      <div class="gsp-row">
        <div class="gsp-row-label">{{ t('git.workingTree') }}</div>
        <div class="gsp-row-value">
          <span v-if="status.working_clean" class="gsp-clean">
            <IconPark type="check-one" :size="10" /> {{ t('git.clean') }}
          </span>
          <span v-else class="gsp-dirty">
            <IconPark type="edit" :size="10" /> {{ t('git.dirty') }}
          </span>
          <span v-if="status.pending_push > 0" class="gsp-pending" :title="t('git.pendingTip')">
            <IconPark type="upload" :size="10" /> {{ t('git.pending', { n: status.pending_push }) }}
          </span>
        </div>
      </div>

      <div v-if="status.last_push_error" class="gsp-error">
        <IconPark type="warning" :size="12" />
        <span class="gsp-error-msg">{{ status.last_push_error }}</span>
      </div>

      <div class="gsp-actions">
        <button v-if="hasRemote" class="gsp-btn" :disabled="loading" @click="doPush">
          <IconPark type="upload" :size="12" />
          {{ t('git.push') }}
        </button>
        <button v-if="!status.working_clean" class="gsp-btn warn" :disabled="loading" @click="doDiscard">
          <IconPark type="undo" :size="12" />
          {{ t('git.discard') }}
        </button>
        <!-- 2026-07-17 删:'查看历史'按钮 — 历史面板改成 inline,直接
             在 Git 同步下方展示,不需要按钮触发 -->
      </div>

      <div v-if="configOpen" class="gsp-config-form">
        <label class="gsp-form-row">
          <span class="gsp-form-label">{{ t('git.formRemoteURL') }}</span>
          <input v-model="formRemoteURL" type="text" placeholder="https://github.com/user/repo.git" class="gsp-input" />
        </label>
        <label class="gsp-form-row">
          <span class="gsp-form-label">{{ t('git.formBranch') }}</span>
          <input v-model="formBranch" type="text" placeholder="main" class="gsp-input" />
        </label>
        <label class="gsp-form-row">
          <span class="gsp-form-label">{{ t('git.formToken') }}</span>
          <input v-model="formToken" type="password" :placeholder="t('git.formToken')" class="gsp-input" />
        </label>
        <label class="gsp-form-row">
          <span class="gsp-form-label">{{ t('git.formUserName') }}</span>
          <input v-model="formUserName" type="text" class="gsp-input" />
        </label>
        <label class="gsp-form-row">
          <span class="gsp-form-label">{{ t('git.formUserEmail') }}</span>
          <input v-model="formUserEmail" type="text" class="gsp-input" />
        </label>
        <div class="gsp-form-actions">
          <button class="gsp-btn" :disabled="formSaving" @click="configOpen = false">{{ t('git.cancel') }}</button>
          <button class="gsp-btn primary" :disabled="formSaving" @click="saveConfig">{{ t('git.save') }}</button>
        </div>
      </div>

      <div v-if="errorMsg" class="gsp-error">
        <IconPark type="close" :size="12" />
        <span class="gsp-error-msg">{{ errorMsg }}</span>
      </div>
    </div>
  </CollapsiblePanel>
</template>

<style scoped>
/* 2026-07-17 改:header 全部用 .cp-* (CollapsiblePanel 内置),本组件只负责
   body 内的内容样式(.gsp- 前缀)。所有 header 相关的旧 .ssp-git-* 样式
   全部移除,避免跟 .cp-* 重复/冲突。 */

.gsp-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  font-family: var(--font-mono, monospace);
}
.gsp-badge.ok { background: rgba(34, 197, 94, 0.15); color: rgb(34, 197, 94); }
.gsp-badge.warn { background: rgba(245, 158, 11, 0.15); color: rgb(245, 158, 11); }

.gsp-empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}
.gsp-empty-tip {
  margin: 0;
  font-size: 11px;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}

.gsp-content { display: flex; flex-direction: column; gap: 4px; }

.gsp-row { display: flex; align-items: baseline; gap: 6px; }
.gsp-row-label { flex: 0 0 40px; font-size: 11px; color: var(--text-muted, rgba(127, 127, 127, 0.7)); }
.gsp-row-value { flex: 1 1 auto; display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 11px; }

.gsp-remote-url {
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.gsp-remote-missing { color: var(--text-muted, rgba(127, 127, 127, 0.5)); }
.gsp-token-ok { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.gsp-token-warn { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.gsp-head-hash { font-family: var(--font-mono, monospace); }
.gsp-head-msg { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }
.gsp-clean { color: rgb(34, 197, 94); display: inline-flex; align-items: center; gap: 2px; }
.gsp-dirty { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }
.gsp-pending { color: rgb(245, 158, 11); display: inline-flex; align-items: center; gap: 2px; }

.gsp-icon-btn {
  flex: 0 0 auto;
  background: transparent;
  border: 0;
  padding: 2px 4px;
  cursor: pointer;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
}
.gsp-icon-btn:hover { color: var(--text-primary, currentColor); }

.gsp-error {
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
.gsp-error-msg { word-break: break-all; }

.gsp-actions { display: flex; gap: 6px; margin-top: 4px; }

.gsp-btn {
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
.gsp-btn:hover:not(:disabled) { background: rgba(127, 127, 127, 0.08); }
.gsp-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.gsp-btn.primary {
  border-color: rgb(59, 130, 246);
  background: rgba(59, 130, 246, 0.15);
  color: rgb(59, 130, 246);
}
.gsp-btn.warn {
  border-color: rgb(245, 158, 11);
  background: rgba(245, 158, 11, 0.15);
  color: rgb(245, 158, 11);
}

.gsp-config-form {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  background: rgba(127, 127, 127, 0.04);
  border-radius: 4px;
}
.gsp-form-row { display: flex; flex-direction: column; gap: 2px; }
.gsp-form-label { font-size: 10px; color: var(--text-muted, rgba(127, 127, 127, 0.7)); }
.gsp-input {
  padding: 3px 6px;
  font-size: 11px;
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
  border-radius: 3px;
  background: var(--bg-primary, transparent);
  color: var(--text-primary, currentColor);
  font-family: var(--font-mono, monospace);
}
.gsp-form-actions { display: flex; gap: 6px; justify-content: flex-end; margin-top: 4px; }
</style>