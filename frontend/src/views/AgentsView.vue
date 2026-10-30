<script setup>
// AgentsView - Agent元数据管理视图。
//
// 2026-07-01 新建。对应后端 7 个 cagent 接口(2026-06-30 上线)——
// 用户可在此浏览 / 启停 / 编辑 / 增删 / 调 reload 的Agent表。
//
// 布局结构(从上到下):
//   1. 页面头:标题 + 副标题(展示总数 / 系统 / 用户)
//   2. toolbar:搜索 + 三选一过滤 + 新建 + Reload
//   3. 错误提示条(只在 store.error 非空时)
//   4. 卡片网格:每张卡 1 个Agent
//      - 顶部:icon + display_name + agent_id + 系统/用户徽章
//      - 中部:maturity chip + path 数 + note
//      - 底部:enabled switch(主交互) + 编辑 / 删除操作图标(hover 显)
//   5. 新建 / 编辑 Modal(size="lg",含 paths 子表) — 由 store.formOpen 控
//   6. 删除确认 Modal(size="sm")          — 由 store.confirmOpen 控
//
// 复用模式:
//   - HTTP:    @/api/skillbox/agents.js
//   - store:   @/core/store/agents.js
//   - 弹窗:    @/components/Modal.vue(已封 size / 标题图标 / 滚动锁定)
//   - 时间:    @/core/utils/time.js#formatRelative
//   - 文件夹:  platform.fs.pickFolder()
//   - i18n:    useI18n() -> t('agents.*'),共命名空间

import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { useAgentsStore, PATH_SLOTS } from '@/core/store/agents'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'
import AgentIcon from '@/components/AgentIcon.vue'
import { uploadAgentIcon, uploadAgentIconByPath } from '@/api/skillbox/agents'
import { formatRelative } from '@/core/utils/time.js'
import { platform } from '@/platform'

const { t } = useI18n()
const agents = useAgentsStore()
const toast = useToastStore()

// 搜索框 draft:输入实时回显,Enter / @input 触发写入 store.setKeyword
const keywordDraft = ref('')
watch(() => agents.filter.keyword, (v) => {
  if (v !== keywordDraft.value) keywordDraft.value = v
})
function applyKeyword() {
  agents.setKeyword(keywordDraft.value)
}

// 过滤按钮互斥
function selectSource(src) {
  agents.setSource(src)
}

// reload:成功后弹 toast(toast 单独抽出来,失败也提示)
async function onReload() {
  try {
    await agents.reloadRegistry()
    toast.success(t('agents.reloadedOk'))
  } catch (e) {
    toast.error(t('agents.reloadFailed', { msg: e?.message || e }))
  }
}

// 新建 / 编辑 / 删除 / 启停 由 store 内部集中处理;view 只负责 toast 反馈
async function onSubmitForm() {
  try {
    await agents.submitForm()
    toast.success(t('agents.savedOk'))
  } catch (e) {
    toast.error(t('agents.saveFailed', { msg: e?.message || e }))
  }
}

async function onConfirmDelete() {
  try {
    await agents.confirmDelete()
    toast.success(t('agents.deletedOk'))
  } catch (e) {
    toast.error(t('agents.deleteFailed', { msg: e?.message || e }))
  }
}

async function onToggleEnabled(t_item) {
  try {
    // store.toggleEnabled 内部顺序:update -> load -> reloadRegistry
    // load 后列表中 t_item 引用会被替换,这里按"调用前的状态"取反提示
    const willEnable = !t_item.enabled
    await agents.toggleEnabled(t_item)
    toast.success(willEnable ? t('agents.enabledOk') : t('agents.disabledOk'))
  } catch (err) {
    toast.error(t('agents.toggleFailed', { msg: err?.message || err }))
  }
}

// 2026-07-06/07 改:打开Agent对应的 skills 目录按钮。
//
// 2026-07-07 修语义:之前"取第一个非空 path",实际上拿到的是 global|system
// (即 ~/.claude/plugins/marketplaces/claude-plugins-official 之类Agent
// 自带的目录),用户期望打开的是"用户配置的全局 skills 目录",即
// (scope=global, category=user) 那条。所以这里**只取 global|user**。
// 2026-07-07 行为:目录不存在时按钮仍可用(走 mkdir 兜底流程),只有当该
// agent 完全没有 (global,user) 这条 path 时才置灰。
function firstSkillsPath(t_item) {
  const list = t_item?.paths || []
  const slot = list.find((p) => p && p.scope === 'global' && p.category === 'user')
  if (!slot) return ''
  return (slot.path || '').trim()
}

async function openSkillsDir(t_item) {
  const p = firstSkillsPath(t_item)
  if (!p) {
    toast.info(t('agents.openNoPath'))
    return
  }
  try {
    await platform.fs.reveal(p)
  } catch (e) {
    // 2026-07-07 改:reveal 失败时(通常是目录不存在)弹确认,用户同意就
    // mkdir -p 后再 reveal。失败信息可能含 "(resolved from " 这类我们
    // 自己加的提示,判断时忽略它,只看是否指向"不存在"。
    const msg = String(e?.message || e)
    const notExist = /does not exist|ENOENT|no such file/i.test(msg)
    if (!notExist) {
      toast.error(t('agents.openFailed', { msg }))
      return
    }
    const ok = window.confirm(t('agents.openCreateConfirm', { path: p }))
    if (!ok) return
    try {
      const r = await platform.fs.mkdir(p)
      toast.success(r.created ? t('agents.openCreateOk') : t('agents.openExistedOpen'))
      // 创建后再 reveal 一次
      await platform.fs.reveal(p)
    } catch (mkErr) {
      toast.error(t('agents.openCreateFailed', { msg: mkErr?.message || mkErr }))
    }
  }
}

// pickFolder 辅助函数:用户取消时静默(返空串不报错)
// 2026-07-04 改:4 格单 path 模型,接收 slot 对象而不是 path 数组行。
async function pickPath(slot) {
  try {
    const v = await platform.fs.pickFolder()
    if (v) slot.path = v
  } catch (e) {
    toast.error(t('agents.pickFolderFailed', { msg: e?.message || e }))
  }
}

// maturity 在三个不同位置用到,集中一个 helper 返回 mdi 图标
function maturityIcon(m) {
  if (m === 'stable') return 'mdi:check-decagram-outline'
  if (m === 'experimental') return 'mdi:flask-outline'
  if (m === 'deprecated') return 'mdi:archive-arrow-down-outline'
  return 'mdi:help-circle-outline'
}

// 选择图标文件:
// 2026-07-03 v3:回退到 <label> 包裹原生 <input>。
// label 的 click 是 HTML 原生事件转发到内嵌 input,触发文件选择器,完全不依赖
// Vue 模板指令(@click / @change 之前都因 Vue patch 丢失 listener 而失效)。
// 改用 onMounted 里给 document 装一个捕获式 change 监听 + class 匹配,保证 input
// 即便被 Vue patch 漏掉 change handler 也能正常处理。
const iconUploadInputClass = 'icon-upload-input'
const uploadingAgentFlag = ref(false)

// 2026-07-03 v4 修复:桌面端走 wails3 原生 OpenFileDialog,web 端走 <input type=file>。
// 桌面 WKWebView 下 <label> 包裹 <input> 触发文件选择器会被静默吞(根因:webview
// 的合成 click 事件流与 label-for 转发冲突),完全绕开 webkit file picker。
// isDesktop 从 window.__APP_RUNTIME__.runMode 解析,跟 platform.isDesktop 一致。
const isDesktopRun = typeof window !== 'undefined' && window.__APP_RUNTIME__?.runMode === 'desktop'

// 桌面端:点按钮 → 调 wails3 OpenFileDialog → 拿到 path → 调新接口 upload-by-path。
// 完全不依赖 <input type="file">,跟 ProjectsView 的"导入项目"(pickFolder)走同套机制。
async function pickIconFileByDesktop() {
  uploadingAgentFlag.value = true
  try {
    const path = await platform.fs.pickFile()
    if (!path) return // 用户取消
    const res = await uploadAgentIconByPath(path)
    if (res && res.name) {
      agents.form.icon_file = res.name
      toast.success(t('agents.uploadIconOk'))
    }
  } catch (err) {
    toast.error(t('agents.uploadIconFailed', { msg: err?.message || err }))
  } finally {
    uploadingAgentFlag.value = false
  }
}

// web 端:点 label → 浏览器原生转发 click 到 <input type=file> → 触发文件选择器。
// label+input 是 HTML 原生最稳的写法,跨浏览器 100% 兼容。
function pickIconFileByWeb() {
  // 这个函数挂到 label 的 click 上,label 的 click 会自动转发到内嵌 input;
  // 这里其实只是占位,真正的触发由浏览器原生完成。
  // 但为了避免 label click 在某些环境下不转发,这里也提供 JS 兜底(找 input click)。
  const input = document.querySelector(`.${iconUploadInputClass}`)
  if (input) input.click()
}

async function onIconFileChosen(input) {
  const file = input.files && input.files[0]
  // 允许清空(重新选) — 只看是否拿到 file
  if (file) {
    uploadingAgentFlag.value = true
    try {
      const res = await uploadAgentIcon(file)
      if (res && res.name) {
        agents.form.icon_file = res.name
        toast.success(t('agents.uploadIconOk'))
      }
    } catch (err) {
      toast.error(t('agents.uploadIconFailed', { msg: err?.message || err }))
    } finally {
      uploadingAgentFlag.value = false
      // 重置 input,允许选同一文件(浏览器同一 file 不会触发 change 事件)
      input.value = ''
    }
  } else {
    input.value = ''
  }
}

function clearIconFile() {
  agents.form.icon_file = ''
}

// 兜底:onMounted 里给 document 装 capture-phase change 监听,匹配 .icon-upload-input
// 元素。这是最后一道保险:即使 Vue patch 阶段漏掉 @change handler,这里也能正常处理。
// 用 class 匹配而不是 id,避免与模板 ref 冲突。
let iconUploadDocListenerBound = false
function bindIconUploadDocListener() {
  if (iconUploadDocListenerBound) return
  iconUploadDocListenerBound = true
  document.addEventListener(
    'change',
    (e) => {
      const t = e.target
      if (t && t instanceof HTMLInputElement && t.type === 'file' && t.classList.contains(iconUploadInputClass)) {
        onIconFileChosen(t)
      }
    },
    true,
  )
}

// 给预览图拼基础 URL
function resolveBaseURL() {
  if (typeof window === 'undefined') return ''
  const cfg = window.__APP_CONFIG__
  if (cfg && typeof cfg.baseURL === 'string') return cfg.baseURL.replace(/\/$/, '')
  if (window.location) return `${window.location.protocol}//${window.location.host}`
  return ''
}
function iconPreviewURL(name) {
  if (!name) return ''
  return `${resolveBaseURL()}/api/files/agent-icons/${name}`
}

// 取 store 计算结果(用 computed 是为了响应式跟随 state 变化)
const items = computed(() => agents.filteredItems)
const total = computed(() => agents.totalCount)
const systemCount = computed(() => agents.systemCount)
const userCount = computed(() => agents.userCount)
const loading = computed(() => agents.loading)
const error = computed(() => agents.error)

const ALLOWED_MATURITY = ['stable', 'experimental', 'deprecated']

// 2026-07-18 改:弹窗表格只展示 category=user 的格子。
// system 格子在 PATH_SLOTS 里依然存在(对应 store.form.slots 中固定位置),
// 后续编辑带 system 路径的Agent时,后端真实数据会被原封不动保留,不会丢路径。
// 我们只是不在 UI 上让用户配/看/改 system 路径(用户的诉求:"重点是用户路径")。
const visiblePathSlots = computed(() =>
  PATH_SLOTS.map((slot, idx) => ({ slot, idx })).filter((x) => x.slot.category === 'user'),
)

onMounted(async () => {
  bindIconUploadDocListener()
  try {
    await agents.load()
  } catch (e) {
    // 错误已经在 store.error 里;view 只显示
  }
})
</script>

<template>
  <div class="agents-view">
    <!-- 1. 页面头 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-emerald">
          <IconPark icon="mdi:agents" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('agents.title') }}</h1>
          <p>{{ t('agents.subtitle', { total, system: systemCount, user: userCount }) }}</p>
        </div>
      </div>
    </header>

    <!-- 2. Agent栏 -->
    <div class="toolbar">
      <div class="search-box">
        <IconPark icon="mdi:magnify" width="16" height="16" class="search-icon" />
        <input
          v-model="keywordDraft"
          type="text"
          :placeholder="t('agents.searchPlaceholder')"
          class="search-input"
          @keyup.enter="applyKeyword"
        />
      </div>

      <!-- 三选一过滤 -->
      <div class="filter-group">
        <button
          :class="['filter-btn', { active: agents.filter.source === 'all' }]"
          @click="selectSource('all')"
        >
          <IconPark icon="mdi:view-list" width="13" height="13" />
          {{ t('agents.filterAll') }}
          <span class="filter-count">{{ total }}</span>
        </button>
        <button
          :class="['filter-btn', { active: agents.filter.source === 'system' }]"
          @click="selectSource('system')"
        >
          <IconPark icon="mdi:shield-check-outline" width="13" height="13" />
          {{ t('agents.filterSystem') }}
          <span class="filter-count">{{ systemCount }}</span>
        </button>
        <button
          :class="['filter-btn', { active: agents.filter.source === 'user' }]"
          @click="selectSource('user')"
        >
          <IconPark icon="mdi:account-outline" width="13" height="13" />
          {{ t('agents.filterUser') }}
          <span class="filter-count">{{ userCount }}</span>
        </button>
      </div>

      <div class="toolbar-right">
        <button class="ghost with-icon" :disabled="agents.reloading" @click="onReload">
          <IconPark icon="mdi:refresh" width="14" height="14" />
          {{ t('agents.btnReload') }}
        </button>
        <button class="primary with-icon" @click="agents.openCreate()">
          <IconPark icon="mdi:plus" width="14" height="14" />
          {{ t('agents.btnNew') }}
        </button>
      </div>
    </div>

    <!-- 3. 错误提示 -->
    <p v-if="error" class="error-message">
      <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>

    <!-- 4. 卡片网格 -->
    <div v-if="loading && !items.length" class="loading-state">
      <span class="spinner"></span>
      <p>{{ t('agents.loading') }}</p>
    </div>

    <div v-else-if="items.length" class="agents-grid">
      <article
        v-for="t_item in items"
        :key="t_item.agent_id"
        :class="['agent-card', {
          'is-system': t_item.is_system,
          'is-disabled': !t_item.enabled,
        }]"
      >
        <!-- 顶部 -->
        <header class="agent-card-top">
          <div class="agent-card-icon">
            <AgentIcon :agent="t_item" :size="22" />
          </div>
          <div class="agent-card-titles">
            <h3 class="agent-card-name" :title="t_item.display_name">
              {{ t_item.display_name }}
            </h3>
            <code class="agent-card-id">{{ t_item.agent_id }}</code>
          </div>
          <span v-if="t_item.is_system" class="badge badge-system">
            <IconPark icon="mdi:shield-check-outline" width="10" height="10" />
            {{ t('agents.systemBadge') }}
          </span>
        </header>

        <!-- 中部 -->
        <div class="agent-card-meta">
          <span :class="['maturity-chip', `maturity-${t_item.maturity || 'stable'}`]">
            <IconPark :icon="maturityIcon(t_item.maturity)" width="10" height="10" />
            {{ t(`agents.maturity.${t_item.maturity || 'stable'}`) }}
          </span>
          <span class="meta-item">
            <IconPark icon="mdi:folder-multiple-outline" width="11" height="11" />
            {{ t('agents.pathCount', { n: (t_item.paths || []).length }) }}
          </span>
        </div>

        <p
          v-if="t_item.note"
          class="agent-card-note"
          :title="t_item.note"
        >
          {{ t_item.note }}
        </p>

        <!-- 底部 -->
        <footer class="agent-card-bottom">
          <label class="switch" @click.stop>
            <input
              type="checkbox"
              :checked="t_item.enabled"
              :disabled="agents.saving"
              @change="onToggleEnabled(t_item)"
            />
            <span class="switch-slider"></span>
          </label>
          <span class="agent-card-time">
            {{ formatRelative(t_item.updated_at) }}
          </span>
          <div class="agent-card-actions" @click.stop>
            <!-- 2026-07-06 增:打开Agent对应的 skills 目录(无任何 path 时置灰) -->
            <IconPark
              icon="mdi:folder-open-outline"
              class="action-icon action-icon-folder"
              :class="{ 'action-icon-disabled': !firstSkillsPath(t_item) }"
              :title="firstSkillsPath(t_item)
                ? t('agents.btnOpenSkillsDir')
                : t('agents.openNoPath')"
              width="14"
              height="14"
              @click="openSkillsDir(t_item)"
            />
            <IconPark
              icon="mdi:square-edit-outline"
              class="action-icon action-icon-edit"
              :title="t('agents.btnEdit')"
              width="14"
              height="14"
              @click="agents.openEdit(t_item)"
            />
            <IconPark
              v-if="!t_item.is_system"
              icon="mdi:trash-can"
              class="action-icon action-icon-danger"
              :title="t('common.delete')"
              width="14"
              height="14"
              @click="agents.askDelete(t_item)"
            />
            <IconPark
              v-else
              icon="mdi:lock-outline"
              class="action-icon action-icon-locked"
              :title="t('agents.systemLocked')"
              width="14"
              height="14"
            />
          </div>
        </footer>
      </article>
    </div>

    <div v-else class="empty-state">
      <IconPark icon="mdi:agents" width="48" height="48" />
      <p class="empty-title">{{ t('agents.empty') }}</p>
      <p class="empty-hint">{{ t('agents.emptyHint') }}</p>
    </div>

    <!-- 5. 新建 / 编辑 Modal -->
    <Modal
      v-model="agents.formOpen"
      size="lg"
      :title="agents.formMode === 'create'
        ? t('agents.formNewTitle')
        : t('agents.formEditTitle', { name: agents.form.display_name || agents.editingAgentId })"
      :close-on-mask="!agents.saving"
    >
      <template #title-icon>
        <IconPark
          :icon="agents.formMode === 'create' ? 'mdi:plus-box' : 'mdi:square-edit-outline'"
          width="18"
          height="18"
        />
      </template>
      <form class="form" @submit.prevent="onSubmitForm">
        <p class="form-hint">
          <IconPark icon="mdi:information-outline" width="14" height="14" />
          {{ t('agents.formHint') }}
        </p>

        <div class="form-grid">
          <!-- agent_id:新建可填,编辑锁死 -->
          <div class="form-field">
            <label>
              {{ t('agents.field.agentId') }}
              <span v-if="agents.formMode === 'create'" class="required">*</span>
            </label>
            <input
              v-model="agents.form.agent_id"
              :disabled="agents.formMode === 'edit'"
              :placeholder="t('agents.hint.agentId')"
              :readonly="agents.formMode === 'edit'"
            />
            <p v-if="agents.formMode === 'edit'" class="field-hint">
              {{ t('agents.hint.agentIdLocked') }}
            </p>
          </div>

          <div class="form-field">
            <label>
              {{ t('agents.field.displayName') }}
              <span class="required">*</span>
            </label>
            <input
              v-model="agents.form.display_name"
              :placeholder="t('agents.hint.displayName')"
              :disabled="agents.saving"
            />
          </div>

          <div class="form-field">
            <label>
              {{ t('agents.field.mdiIcon') }}
            </label>
            <input
              v-model="agents.form.mdi_icon"
              :placeholder="t('agents.hint.mdiIcon')"
              :disabled="agents.saving"
            />
            <p class="field-hint">{{ t('agents.hint.mdiIcon') }}</p>
          </div>

          <!-- 自定义图标上传区块 -->
          <div class="form-field form-field-full">
            <label>{{ t('agents.field.customIcon') }}</label>
            <div class="icon-upload-row">
              <div class="icon-preview">
                <img
                  v-if="agents.form.icon_file"
                  :src="iconPreviewURL(agents.form.icon_file)"
                  alt="icon"
                  class="icon-preview-img"
                />
                <IconPark
                  v-else
                  icon="mdi:image-off-outline"
                  width="22"
                  height="22"
                  class="icon-preview-empty"
                />
              </div>
              <div class="icon-upload-controls">
                <!--
                  2026-07-03 v4 修复(桌面端):走 wails3 原生 OpenFileDialog,完全绕开 webkit file picker。
                  v1 (label for + 屏幕外 input) → WKWebView+Modal Teleport 下 click 被静默吞
                  v2 (button + JS ref.click()) → Vue patch 阶段 onClick / ref 丢失
                  v3 (label 包裹 input)       → 在 chromium web 端 OK,桌面 WKWebView 仍不可靠
                  v4 (按平台分支)              → 桌面端 button → platform.fs.pickFile() 调 wails3
                                                原生 dialog,web 端保留 v3 label+input 走浏览器选择器
                -->
                <button
                  v-if="isDesktopRun"
                  type="button"
                  class="ghost with-icon upload-label"
                  :class="{ disabled: agents.saving || uploadingAgentFlag }"
                  :disabled="agents.saving || uploadingAgentFlag"
                  @click="pickIconFileByDesktop"
                >
                  <span v-if="uploadingAgentFlag" class="spinner spinner-sm"></span>
                  <IconPark v-else icon="mdi:upload" width="14" height="14" />
                  {{ uploadingAgentFlag ? t('common.processing') : t('agents.btnUploadIcon') }}
                </button>
                <label
                  v-else
                  class="ghost with-icon upload-label"
                  :class="{ disabled: agents.saving || uploadingAgentFlag }"
                  @click.prevent="pickIconFileByWeb"
                >
                  <input
                    type="file"
                    accept="image/png,image/svg+xml,image/jpeg,image/webp,image/x-icon,image/gif"
                    class="hidden-input icon-upload-input"
                    :disabled="agents.saving || uploadingAgentFlag"
                    @change="onIconFileChosen"
                  />
                  <span v-if="uploadingAgentFlag" class="spinner spinner-sm"></span>
                  <IconPark v-else icon="mdi:upload" width="14" height="14" />
                  {{ uploadingAgentFlag ? t('common.processing') : t('agents.btnUploadIcon') }}
                </label>
                <button
                  v-if="agents.form.icon_file"
                  type="button"
                  class="ghost icon-only-btn"
                  :disabled="agents.saving"
                  :title="t('agents.btnClearIcon')"
                  @click="clearIconFile"
                >
                  <IconPark icon="mdi:close" width="14" height="14" />
                </button>
                <code v-if="agents.form.icon_file" class="icon-file-name">
                  {{ agents.form.icon_file }}
                </code>
              </div>
            </div>
            <p class="field-hint">{{ t('agents.hint.customIcon') }}</p>
          </div>

          <div class="form-field">
            <label>{{ t('agents.field.maturity') }}</label>
            <select v-model="agents.form.maturity" :disabled="agents.saving">
              <option v-for="m in ALLOWED_MATURITY" :key="m" :value="m">
                {{ t(`agents.maturity.${m}`) }}
              </option>
            </select>
          </div>

          <div class="form-field">
            <label>{{ t('agents.field.sortOrder') }}</label>
            <input
              v-model.number="agents.form.sort_order"
              type="number"
              :disabled="agents.saving"
            />
          </div>

          <div class="form-field form-field-switch">
            <label>{{ t('agents.field.enabled') }}</label>
            <label class="switch">
              <input
                type="checkbox"
                v-model="agents.form.enabled"
                :disabled="agents.saving"
              />
              <span class="switch-slider"></span>
            </label>
          </div>

          <div class="form-field form-field-full">
            <label>{{ t('agents.field.note') }}</label>
            <input
              v-model="agents.form.note"
              :placeholder="t('agents.hint.note')"
              :disabled="agents.saving"
            />
          </div>
        </div>

        <!-- paths 子表 — 2026-07-04 改:4 个 (scope, category) 固定格子,
         每个格子 0 或 1 条 path。删了"添加路径"按钮 + 删除行图标。 -->
        <div class="paths-section">
          <div class="paths-section-header">
            <h4>{{ t('agents.paths.title') }}</h4>
          </div>

          <table class="paths-table">
            <thead>
              <tr>
                <th style="width: 90px">{{ t('agents.paths.scope') }}</th>
                <th style="width: 90px">{{ t('agents.paths.category') }}</th>
                <th>{{ t('agents.paths.path') }}</th>
                <th style="width: 40px"></th>
              </tr>
            </thead>
            <tbody>
              <!--
                2026-07-18 改:UI 上只展示 category=user 的格子(category=system 的格子
                仍然在 store.form.slots 里保留,只是不渲染)。这样:
                  - 用户编辑带 system 路径的Agent时,后端真实数据不丢
                  - 不影响 buildPayload 拍平和 clearSlotPath 逻辑(走原 store 索引)
                  - 用户新建Agent时,只能配置 user 路径,与"重点是用户路径"诉求一致
              -->
              <tr
                v-for="item in visiblePathSlots"
                :key="`${item.slot.scope}-${item.slot.category}-ui`"
                :class="{ 'paths-slot-empty': !agents.form.slots[item.idx].path }"
              >
                <td>
                  <span class="paths-slot-readonly">
                    {{ item.slot.scope === 'global' ? t('agents.paths.scopeGlobal') : t('agents.paths.scopeProject') }}
                  </span>
                </td>
                <td>
                  <span class="paths-slot-readonly">
                    {{ item.slot.category === 'user' ? t('agents.paths.categoryUser') : t('agents.paths.categorySystem') }}
                  </span>
                </td>
                <td>
                  <div class="input-with-action">
                    <input
                      v-model="agents.form.slots[item.idx].path"
                      :placeholder="item.slot.scope === 'global' ? t('agents.paths.pathHintGlobal') : t('agents.paths.pathHintProject')"
                      :disabled="agents.saving"
                    />
                    <button
                      type="button"
                      class="ghost icon-btn"
                      :disabled="agents.saving"
                      :title="t('agents.paths.pickFolder')"
                      @click="pickPath(agents.form.slots[item.idx])"
                    >
                      <IconPark icon="mdi:folder-open" width="14" height="14" />
                    </button>
                  </div>
                </td>
                <td class="paths-action-cell">
                  <IconPark
                    v-if="agents.form.slots[item.idx].path"
                    icon="mdi:close-circle-outline"
                    class="action-icon action-icon-danger"
                    :title="t('common.delete')"
                    width="14"
                    height="14"
                    @click="agents.clearSlotPath(item.idx)"
                  />
                </td>
              </tr>
            </tbody>
          </table>

          <p class="field-hint">{{ t('agents.paths.hint') }}</p>
        </div>
      </form>

      <template #footer>
        <button
          type="button"
          class="ghost with-icon"
          :disabled="agents.saving"
          @click="agents.closeForm()"
        >
          <IconPark icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="primary with-icon"
          :disabled="agents.saving"
          @click="onSubmitForm"
        >
          <span v-if="agents.saving" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:check" width="14" height="14" />
          {{ agents.saving ? t('common.processing') : t('common.save') }}
        </button>
      </template>
    </Modal>

    <!-- 6. 删除确认 Modal -->
    <Modal
      v-model="agents.confirmOpen"
      size="sm"
      :title="t('agents.confirmDeleteTitle')"
      :close-on-mask="!agents.removing"
    >
      <p class="confirm-message">
        {{
          t('agents.confirmDeleteMsg', {
            name: agents.confirmTarget?.display_name || agents.confirmTarget?.agent_id,
          })
        }}
        <span v-if="agents.confirmTarget?.note" class="confirm-hint">
          {{ agents.confirmTarget.note }}
        </span>
      </p>
      <template #footer>
        <button
          type="button"
          class="ghost"
          :disabled="agents.removing"
          @click="agents.cancelDelete()"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="danger"
          :disabled="agents.removing"
          @click="onConfirmDelete"
        >
          <span v-if="agents.removing" class="spinner spinner-sm"></span>
          <IconPark v-else icon="mdi:trash-can" width="14" height="14" />
          {{ agents.removing ? t('common.processing') : t('common.delete') }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.agents-view {
  /* 占满内容区宽度(与 MarketView 一致) */
  width: 100%;
  color: var(--text);
  transition: color 0.3s ease;
}

/* ===== Agent主题:Emerald Workshop(独立作用域变量) ===== */
.agents-view {
  /* 主色:teal-500(冷一点的青色,和翠绿 emerald 拉开层次) */
  --agent-primary: #14b8a6;
  --agent-primary-hover: #0d9488;
  /* 强调:emerald-500(绿,代表"Agent可用/成功") */
  --agent-accent: #10b981;
  /* 派生浅底/边/字 */
  --agent-bg: #f0fdfa;          /* teal-50 */
  --agent-bg-strong: #ccfbf1;   /* teal-100 */
  --agent-border: #99f6e4;      /* teal-200 */
  --agent-text: #0f766e;        /* teal-700 */
}
:global(html.dark) .agents-view {
  --agent-primary: #2dd4bf;     /* teal-400 提亮 */
  --agent-primary-hover: #5eead4; /* teal-300 */
  --agent-accent: #34d399;      /* emerald-400 */
  --agent-bg: #042f2e;          /* teal-950 */
  --agent-bg-strong: #134e4a;  /* teal-900 */
  --agent-border: #115e59;      /* teal-800 */
  --agent-text: #99f6e4;        /* teal-200 */
}

/* ===== 页面头 ===== */
.view-header {
  margin-bottom: 24px;
}

.view-title {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.view-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--text);
  color: var(--bg-card);
  flex-shrink: 0;
}

.view-icon-emerald {
  /* 主题色块:teal→emerald 渐变 + 发光阴影 */
  background: linear-gradient(135deg, var(--agent-primary) 0%, var(--agent-accent) 100%);
  color: #ffffff;
  box-shadow: 0 2px 8px -2px color-mix(in srgb, var(--agent-primary) 40%, transparent);
}

.view-title h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 4px;
}

.view-title p {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0;
}

/* ===== Agent栏 ===== */
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1 1 240px;
  min-width: 200px;
  max-width: 360px;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-faint);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding-left: 38px;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  /* 主题浅底 + 主题淡边(柔和不刺眼) */
  background: var(--agent-bg);
  border: 1px solid var(--agent-border);
  border-radius: var(--radius-sm);
}

.filter-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  background: transparent;
  border: none;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-btn:hover:not(.active) {
  /* hover 时上主题色,文字也跟随 */
  color: var(--agent-text);
  background: var(--agent-bg-strong);
}

.filter-btn.active {
  /* 激活态:teal→emerald 渐变(类似 Market 的 source-tab) */
  background: linear-gradient(135deg, var(--agent-primary) 0%, var(--agent-accent) 100%);
  color: #ffffff;
  box-shadow: 0 2px 6px -2px color-mix(in srgb, var(--agent-primary) 50%, transparent);
}

.filter-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 6px;
  font-size: 10px;
  font-weight: 600;
  background: var(--border);
  color: var(--text-dim);
  border-radius: 999px;
  min-width: 18px;
}

.filter-btn.active .filter-count {
  /* 激活态时:count 数字用主色突出 */
  background: color-mix(in srgb, #ffffff 25%, transparent);
  color: #ffffff;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

/* 新建按钮(CTA):主题 teal→emerald 渐变 + hover 上浮(视觉关键转化点) */
.toolbar-right button.primary {
  background: linear-gradient(135deg, var(--agent-primary) 0%, var(--agent-accent) 100%);
  color: #ffffff;
  border-color: transparent;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--agent-primary) 30%, transparent);
}
.toolbar-right button.primary:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--agent-primary-hover) 0%, var(--agent-accent) 100%);
  border-color: transparent;
  transform: translateY(-1px);
  box-shadow: 0 3px 8px -2px color-mix(in srgb, var(--agent-primary) 45%, transparent);
}

/* ===== 错误提示 ===== */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--danger-dim);
  color: var(--danger);
  border: 1px solid var(--danger);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin: 0 0 16px;
}

/* ===== 加载状态 ===== */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 64px 24px;
  color: var(--text-faint);
}

.loading-state .spinner {
  width: 24px;
  height: 24px;
  border-width: 3px;
}

/* ===== 卡片网格 ===== */
.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
}

.agent-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 14px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-left: 4px solid transparent;
  border-radius: var(--radius);
  position: relative;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

.agent-card:hover {
  /* hover 时显示主题色淡边 + 轻微抬起 */
  border-color: color-mix(in srgb, var(--agent-primary) 35%, var(--border));
  box-shadow: var(--shadow-card);
}

/* 系统Agent:左侧 emerald 渐变条 */
.agent-card.is-system {
  border-left: 4px solid;
  border-image: linear-gradient(180deg, var(--agent-primary) 0%, var(--agent-accent) 100%) 1;
}

/* 用户Agent:左侧浅主题色条(轻提示,与系统Agent区分但不刺眼) */
.agent-card:not(.is-system) {
  border-left-color: color-mix(in srgb, var(--agent-primary) 25%, transparent);
}

.agent-card.is-disabled {
  opacity: 0.55;
}

.agent-card-top {
  display: flex;
  align-items: center;
  gap: 10px;
}

.agent-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  /* 主题浅底 + 主题主色图标 */
  background: var(--agent-bg);
  color: var(--agent-primary);
  flex-shrink: 0;
  border: 1px solid var(--agent-border);
}

.agent-card-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.agent-card-name {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-card-id {
  font-size: 11px;
  font-family: 'JetBrains Mono', monospace;
  /* 主题色 chip */
  background: var(--agent-bg);
  color: var(--agent-text);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  align-self: flex-start;
  max-width: fit-content;
  border: 1px solid var(--agent-border);
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 6px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 999px;
  flex-shrink: 0;
}

.badge-system {
  /* 系统徽章:teal→emerald 渐变,和卡片左侧条带呼应 */
  background: linear-gradient(135deg, var(--agent-bg) 0%, var(--agent-bg-strong) 100%);
  color: var(--agent-text);
  border: 1px solid var(--agent-border);
}

.agent-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.maturity-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 999px;
}

.maturity-stable {
  background: var(--success-dim);
  color: var(--success);
}

.maturity-experimental {
  background: var(--warning-dim);
  color: var(--warning);
}

.maturity-deprecated {
  background: var(--danger-dim);
  color: var(--danger);
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--text-faint);
}

.agent-card-note {
  margin: 0;
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.agent-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.agent-card-time {
  font-size: 10px;
  color: var(--text-faint);
  flex: 1;
  text-align: left;
}

.agent-card-actions {
  display: flex;
  gap: 2px;
  /* 2026-07-06 改:编辑 / 锁定 图标常显,不再依赖 hover */
  opacity: 1;
  transition: opacity 0.15s ease;
}

/* hover 不再控制 actions 显隐(常显后保留 hover 过渡,避免视觉突变) */
.agent-card:hover .agent-card-actions {
  opacity: 1;
}

.action-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
  color: var(--text-dim);
}

.action-icon:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.action-icon-edit:hover {
  /* 编辑:hover 时用主题色(区别于危险) */
  background: var(--agent-bg);
  color: var(--agent-primary);
}

/* 2026-07-06 增:打开 skills 目录按钮:hover 走主题色,和编辑按钮一致 */
.action-icon-folder:hover {
  background: var(--agent-bg);
  color: var(--agent-primary);
}

/* 无任何 path 时禁用文件夹按钮 */
.action-icon-disabled {
  cursor: not-allowed;
  color: var(--text-faint);
  opacity: 0.5;
}
.action-icon-disabled:hover {
  background: transparent;
  color: var(--text-faint);
}

.action-icon-danger:hover {
  background: var(--danger-dim);
  color: var(--danger);
}

.action-icon-locked {
  cursor: not-allowed;
  color: var(--text-faint);
}

/* ===== 开关 ===== */
.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  flex-shrink: 0;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

/* 滑轨:未选用低饱和的深灰底 + 内阴影,有"关闭"的实感;
   选用主题渐变 + 外发光,一眼能区分。 */
.switch-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #cbd5e1;            /* 明显灰(slate-300),和深色页面拉开 */
  border: 1px solid #94a3b8;            /* slate-400 描边增加边界感 */
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
  border-radius: 24px;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.12);
}

/* 圆点:大一点、亮一点,带阴影在滑轨上"浮"起来 */
.switch-slider::before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 2px;
  top: 50%;
  transform: translateY(-50%);
  background-color: #ffffff;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
}

.switch input:checked + .switch-slider {
  /* 开启:teal→emerald 渐变 + 外发光,与卡片左侧条带呼应 */
  background: linear-gradient(135deg, var(--agent-primary) 0%, var(--agent-accent) 100%);
  border-color: transparent;
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--agent-primary) 25%, transparent),
              inset 0 1px 2px rgba(0, 0, 0, 0.1);
}

.switch input:checked + .switch-slider::before {
  /* 圆点位移 18 → 22,使开/关状态位移更明显 */
  transform: translate(22px, -50%);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

.switch input:disabled + .switch-slider {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ===== 表单(Modal 内) ===== */
.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--text-dim);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-field-full {
  grid-column: 1 / -1;
}

.form-field label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
}

/* 锁 select 行高与 input 一致:macOS Chrome native select 会比 input 多 2-3px,
   同时 min-height + height 双锁避免被全局 input 规则的 padding 撑高 */
.form-field select,
.form-field input {
  height: 36px;
  min-height: 36px;
  line-height: 1.4;
}

.form-field textarea {
  min-height: 60px;
}

.form-field-switch {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.required {
  color: var(--danger);
  font-weight: 700;
}

.field-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

.input-with-action {
  display: flex;
  align-items: stretch;
  gap: 6px;
}

.input-with-action input {
  flex: 1;
  min-width: 0;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  flex-shrink: 0;
}

/* ===== paths 子表 ===== */
.paths-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.paths-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 4px;
}

.paths-section-header h4 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

button.small {
  padding: 5px 10px;
  font-size: 12px;
}

/* 带图标的按钮:全局 button 默认 inline-block,svg 与文字按基线对齐会不齐。
   这里统一改成 flex 居中,顺带带 6px gap 让图标和文字有呼吸感。 */
button.with-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  line-height: 1;
}
button.with-icon :deep(svg) {
  display: block;
  flex-shrink: 0;
}

/* paths-section 的"添加路径"按钮:表单内的主动作,做成醒目的浅主题色按钮。
   选 button.add-path-btn(0,1,1)而不是 .add-path-btn(0,1,0),
   让特异性压过全局 button 规则 —— 否则全局 button 的 background/color
   会盖过这里,按钮看起来白底+深色文字。 */
button.add-path-btn {
  /* 浅主题色底 + 主题色文字:亮环境清晰,深色主题下也保留对比 */
  background: var(--agent-bg);
  border: 1px solid var(--agent-primary);
  color: var(--agent-text);
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 600;
  border-radius: var(--radius-sm);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--agent-primary) 20%, transparent);
}
button.add-path-btn:hover:not(:disabled) {
  /* hover:底色加深一档,文字颜色再深一档,border 不变 */
  background: var(--agent-bg-strong);
  border-color: var(--agent-primary-hover);
  color: var(--agent-primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 3px 8px -2px color-mix(in srgb, var(--agent-primary) 35%, transparent);
}

.paths-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.paths-table th {
  text-align: left;
  padding: 8px 10px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  background: var(--bg-subtle);
  border-bottom: 1px solid var(--border);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.paths-table td {
  padding: 6px 8px;
  border-top: 1px solid var(--border);
  vertical-align: middle;
}

.paths-table tr:first-child td {
  border-top: none;
}

.paths-table input,
.paths-table select {
  padding: 5px 8px;
  font-size: 12px;
  width: 100%;
}

.paths-action-cell {
  text-align: center;
}

/* 2026-07-04 改:4 格固定布局下,scope/category 列用 readonly 文字展示
   (不用 select),保持视觉简洁 */
.paths-slot-readonly {
  display: inline-block;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted, var(--text-faint));
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  min-width: 50px;
  text-align: center;
}

/* 空格子整行淡灰,提示用户这一格没配 */
.paths-slot-empty {
  opacity: 0.6;
}

.paths-empty {
  margin: 0;
  padding: 16px;
  text-align: center;
  font-size: 12px;
  color: var(--text-faint);
  background: var(--bg-subtle);
  border: 1px dashed var(--border);
  border-radius: var(--radius-sm);
}

/* ===== 删除确认 ===== */
.confirm-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
}

.confirm-hint {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-faint);
}

/* ===== 空状态 ===== */
.empty-state {
  padding: 48px 24px;
  text-align: center;
  color: var(--text-faint);
  background: var(--bg-subtle);
  border: 1px dashed var(--border);
  border-radius: var(--radius);
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--text);
  margin: 12px 0 0;
}

.empty-hint {
  font-size: 13px;
  margin: 6px 0 0;
  color: var(--text-faint);
}

/* ===== 响应式 ===== */
@media (max-width: 768px) {
  .agents-grid {
    grid-template-columns: 1fr;
  }

  .toolbar-right {
    width: 100%;
  }
}

/* ===== 自定义图标上传 ===== */
.icon-upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-preview {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
}

.icon-preview-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.icon-preview-empty {
  color: var(--text-faint);
}

.icon-upload-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  min-width: 0;
}

.icon-only-btn {
  padding: 6px 8px;
  font-size: 12px;
}

.icon-file-name {
  font-size: 11px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-dim);
  background: var(--bg-subtle);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hidden-input {
  /*
   * 2026-07-03 v3 修复:label 包裹 input 模式。
   * input 留视口内 0×0 + opacity:0,不放屏幕外(屏幕外 WKWebView 会优化掉),
   * pointer-events 保持 auto(因为 label 转发 click 时 input 仍需接收),
   * 由父级 .upload-label.disabled 用 pointer-events:none 统一禁用。
   */
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  left: 0;
  top: 0;
  z-index: -1;
}

/*
 * 2026-07-03 v3:label 包裹 input。复用 button.with-icon 的视觉(label 改成 inline-flex
 * 才能跟 button.ghost 视觉一致),cursor:pointer。
 * disabled 时 pointer-events:none,既不响应点击也阻止内部 input 触发。
 */
.upload-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
}
.upload-label.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
