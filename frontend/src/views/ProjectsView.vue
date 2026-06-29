<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { listProjects, createProject, deleteProject, scanProject } from '@/api/skillbox/projects'
import { platform } from '@/platform'
import { formatRelative } from '@/core/utils/time.js'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'

const { t } = useI18n()
const toast = useToastStore()

const items = ref([])
const total = ref(0)
const loading = ref(false)
const error = ref('')

// 表单可见性:showImport 控制"导入项目"弹窗
const showImport = ref(false)
// 导入中 / 解析中 状态
const importing = ref(false)
const inspecting = ref(false)

// 表单数据:导入项目时 name/alias/root_path 三个核心字段都是必填
const form = reactive({ name: '', alias: '', root_path: '', description: '' })

const filter = reactive({ keyword: '', page: 1, size: 12 })

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filter.size)))

// 扫描结果缓存:按 project_id 缓存,避免重复请求(2026-06-29 卡片化新增)
//   scans[id] = { scanned_at, tools: [{ tool_id, display_name, icon, count, skills: [...] }] }
//   scans[id].error = "xxx" 表示本次扫描失败
const scans = reactive({})
const scanLoading = reactive({})

// 工具 skill 列表 Modal(点击 chip 触发)
const skillsModal = reactive({
  open: false,
  title: '',
  project: '',
  tool: '',
  skills: [],
})

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const resp = await listProjects({
      page: filter.page,
      size: filter.size,
      keyword: filter.keyword || undefined,
    })
    items.value = resp?.items || []
    total.value = resp?.total || 0
    // 拿到 items 后,为每个项目触发懒加载扫描(每个项目只发一次)
    items.value.forEach((p) => ensureScanned(p))
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

// ensureScanned 对单个项目跑"工具 / skill 扫描",并把结果缓存到 scans[p.id]。
//
// 触发时机:卡片 mouseenter(避免一进页面就 N 个并发请求炸后端)
// + reload 后兜底(items.value 整体重拉一遍时一次性补齐)。
// 守门:已有结果或在加载中 → 不重发。
async function ensureScanned(p) {
  if (!p || !p.id) return
  if (scans[p.id] || scanLoading[p.id]) return
  scanLoading[p.id] = true
  try {
    const r = await scanProject(p.id)
    scans[p.id] = r
  } catch (e) {
    // 扫描失败时也写入占位,这样 hover 时不会一直转圈
    scans[p.id] = { scanned_at: null, tools: [], error: e?.message || String(e) }
    console.warn('[projects] scanProject failed:', p.id, e?.message || e)
  } finally {
    scanLoading[p.id] = false
  }
}

// openInFinder 在系统文件管理器中打开项目根(桌面端走 platform.fs.reveal,
// 在 Finder 中显示该路径;Web 端 fs.reveal 内部会兜底到 file:// 链接)。
async function openInFinder(p) {
  if (!p?.root_path) return
  try {
    await platform.fs.reveal(p.root_path)
  } catch (e) {
    toast.push({ type: 'error', message: t('projects.openFailed', { msg: e?.message || String(e) }) })
  }
}

// openToolSkills 弹出 Modal,展示该项目在该工具下的所有 skill。
function openToolSkills(p, tool) {
  skillsModal.open = true
  skillsModal.project = p.name || p.alias || `#${p.id}`
  skillsModal.tool = tool.display_name || tool.tool_id
  skillsModal.title = t('projects.toolSkillsTitle', {
    project: skillsModal.project,
    tool: skillsModal.tool,
    count: tool.count,
  })
  skillsModal.skills = tool.skills || []
}

// 点击"导入项目"按钮:先让用户选目录,再弹"导入"弹窗预填
async function startImport() {
  error.value = ''
  let path = ''
  try {
    path = await platform.fs.pickFolder()
  } catch (e) {
    error.value = e?.message || String(e)
    return
  }
  // 用户取消选择(空串)→ 不弹弹窗,保持原状
  if (!path) return
  // 重置表单
  Object.assign(form, { name: '', alias: '', root_path: path, description: '' })
  showImport.value = true
  // 后台异步从路径推断 name/alias,不阻塞弹窗打开
  await inspectFromPath(path)
}

// 从 root_path 推断 name / alias 候选。失败时表单允许用户手工填。
async function inspectFromPath(path) {
  if (!path) return
  inspecting.value = true
  try {
    const hint = await platform.fs.inspectProject(path)
    if (hint?.name) form.name = hint.name
    if (hint?.alias) form.alias = hint.alias
  } catch (e) {
    // 解析失败不致命,用户可以手工填;只在控制台留个 warning
    console.warn('[projects] inspectProject failed:', e?.message || e)
  } finally {
    inspecting.value = false
  }
}

// 让用户手工改 root_path 时也再解析一次
async function onRootPathBlur() {
  const p = form.root_path.trim()
  if (!p) return
  // 只在 name / alias 仍为空(或跟提示同名)时才覆盖,避免覆盖用户已编辑的内容
  await inspectFromPath(p)
}

async function submitImport() {
  error.value = ''
  if (!form.name.trim() || !form.alias.trim() || !form.root_path.trim()) {
    error.value = t('projects.errRequired')
    return
  }
  importing.value = true
  try {
    await createProject({ ...form })
    showImport.value = false
    Object.assign(form, { name: '', alias: '', root_path: '', description: '' })
    filter.page = 1
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  } finally {
    importing.value = false
  }
}

function cancelImport() {
  if (importing.value) return
  showImport.value = false
}

async function remove(id) {
  const ok = await openConfirm({
    title: t('common.delete'),
    message: t('projects.confirmDelete', { id }),
    variant: 'danger',
    confirmText: t('common.delete'),
  })
  if (!ok) return
  try {
    await deleteProject(id)
    await reload()
  } catch (e) {
    error.value = e?.message || String(e)
  }
}

// 通用确认弹窗
const confirmOpen = ref(false)
const confirmOpts = reactive({
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  variant: 'default',
  resolve: null,
})
function openConfirm(opts) {
  confirmOpts.title = opts.title || t('common.confirm')
  confirmOpts.message = opts.message || ''
  confirmOpts.confirmText = opts.confirmText || t('common.confirm')
  confirmOpts.cancelText = opts.cancelText || t('common.cancel')
  confirmOpts.variant = opts.variant || 'default'
  confirmOpen.value = true
  return new Promise((resolve) => { confirmOpts.resolve = resolve })
}
function resolveConfirm(ok) {
  if (confirmOpts.resolve) confirmOpts.resolve(ok)
  confirmOpen.value = false
}

function gotoPage(p) {
  if (p < 1 || p > totalPages.value) return
  filter.page = p
  reload()
}

onMounted(reload)
</script>

<template>
  <div class="projects-view">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-purple">
          <Icon icon="mdi:folder-multiple-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('projects.title') }}</h1>
          <p>{{ t('projects.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="search-box">
        <Icon icon="mdi:magnify" width="16" height="16" class="search-icon" />
        <input
          v-model="filter.keyword"
          :placeholder="t('projects.searchPlaceholder')"
          class="search-input"
          @keyup.enter="() => { filter.page = 1; reload() }"
        />
      </div>
      <button class="primary" :title="t('projects.btnImportTitle')" @click="startImport">
        <Icon icon="mdi:folder-upload-outline" width="16" height="16" />
        <span>{{ t('projects.btnImport') }}</span>
      </button>
    </div>

    <p v-if="error" class="error-message">
      <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
      {{ error }}
    </p>

    <!-- 导入项目弹窗 -->
    <Modal
      v-model="showImport"
      size="md"
      :title="t('projects.formTitle')"
      :close-on-mask="!importing"
    >
      <template #title-icon>
        <Icon icon="mdi:folder-upload-outline" width="18" height="18" />
      </template>
      <form class="form" @submit.prevent="submitImport">
        <p class="form-hint">
          <Icon icon="mdi:information-outline" width="14" height="14" />
          {{ t('projects.formHint') }}
        </p>
        <div class="form-grid">
          <div class="form-field form-field-full">
            <label>{{ t('projects.rootPath') }}</label>
            <div class="input-with-action">
              <input
                v-model="form.root_path"
                :placeholder="t('projects.rootPathHint')"
                :disabled="importing"
                @blur="onRootPathBlur"
              />
              <button
                type="button"
                class="ghost icon-btn"
                :disabled="importing"
                :title="t('projects.btnPickAgain')"
                @click="async () => {
                  let p = ''
                  try { p = await platform.fs.pickFolder() } catch (e) { error.value = e?.message || String(e); return }
                  if (p) { form.root_path = p; await inspectFromPath(p) }
                }"
              >
                <Icon icon="mdi:folder-search-outline" width="14" height="14" />
              </button>
            </div>
          </div>
          <div class="form-field">
            <label>
              {{ t('projects.name') }}
              <span v-if="inspecting" class="inspecting-tag">
                <span class="spinner spinner-sm"></span>
                {{ t('projects.inspecting') }}
              </span>
            </label>
            <input
              v-model="form.name"
              :placeholder="t('projects.nameHint')"
              :disabled="importing"
            />
          </div>
          <div class="form-field">
            <label>{{ t('projects.alias') }}</label>
            <input
              v-model="form.alias"
              :placeholder="t('projects.aliasHint')"
              :disabled="importing"
            />
          </div>
          <div class="form-field form-field-full">
            <label>{{ t('projects.description') }}</label>
            <input
              v-model="form.description"
              :placeholder="t('projects.descriptionHint')"
              :disabled="importing"
            />
          </div>
        </div>
      </form>
      <template #footer>
        <button type="button" class="ghost" :disabled="importing" @click="cancelImport">
          <Icon icon="mdi:close" width="14" height="14" />
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="primary" :disabled="importing" @click="submitImport">
          <span v-if="importing" class="spinner spinner-sm btn-spinner"></span>
          <Icon v-else icon="mdi:check" width="14" height="14" />
          {{ importing ? t('common.processing') : t('projects.btnImport') }}
        </button>
      </template>
    </Modal>

    <!-- 列表卡片(2026-06-29 改:表格 → 卡片网格) -->
    <div class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:format-list-bulleted" width="16" height="16" />
          {{ t('projects.listTitle') }}
          <span class="card-sub">— {{ t('common.totalCount', { count: total }) }}</span>
        </h3>
        <span v-if="loading" class="spinner"></span>
      </header>

      <div v-if="items.length" class="projects-grid">
        <article
          v-for="p in items"
          :key="p.id"
          class="project-card"
          @mouseenter="ensureScanned(p)"
        >
          <!-- 顶部:项目名 + alias 徽章 + 文件夹图标 + 删除 -->
          <header class="project-card-top">
            <div class="project-card-titles">
              <h3 class="project-card-name">{{ p.name }}</h3>
              <code class="project-card-alias">{{ p.alias }}</code>
            </div>
            <div class="project-card-actions">
              <!-- 直接用 Icon 标签,@click 绑逻辑,不用 button 包裹 -->
              <Icon
                icon="mdi:folder-outline"
                class="action-icon action-icon-finder"
                :title="t('projects.openInFinder')"
                width="14"
                height="14"
                @click.stop="openInFinder(p)"
              />
              <Icon
                icon="mdi:delete-outline"
                class="action-icon action-icon-danger"
                :title="t('common.delete')"
                width="14"
                height="14"
                @click.stop="remove(p.id)"
              />
            </div>
          </header>

          <!-- 中间:描述 -->
          <p class="project-card-desc" :title="p.description || ''">
            {{ p.description || t('common.dash') }}
          </p>
          <p class="project-card-path" :title="p.root_path">{{ p.root_path }}</p>

          <!-- 底部:工具 chips(数量 + 弹 Modal 列 skill) -->
          <div class="project-card-tools">
            <span v-if="scanLoading[p.id]" class="tools-loading">
              <span class="spinner spinner-sm"></span>
            </span>
            <button
              v-for="tool in (scans[p.id]?.tools || [])"
              :key="tool.tool_id"
              class="tool-chip"
              :title="tool.display_name"
              @click.stop="openToolSkills(p, tool)"
            >
              <span class="chip-label">{{ tool.tool_id }}</span>
              <span class="chip-count">{{ tool.count }}</span>
            </button>
            <span
              v-if="!scanLoading[p.id] && (scans[p.id]?.tools || []).length === 0 && !scans[p.id]?.error"
              class="tools-empty"
            >
              {{ t('projects.noTools') }}
            </span>
            <span v-if="scans[p.id]?.error" class="tools-error">
              <Icon icon="mdi:alert-circle-outline" width="11" height="11" />
              {{ t('projects.scanFailed') }}
            </span>
          </div>

          <!-- hover 才显示的扫描时间 -->
          <footer v-if="scans[p.id]?.scanned_at" class="project-card-meta">
            {{ t('projects.scannedAt', { time: formatRelative(scans[p.id].scanned_at) }) }}
          </footer>
        </article>
      </div>

      <div v-else-if="!loading" class="empty-state">
        <Icon icon="mdi:folder-open-outline" width="48" height="48" />
        <p class="empty-title">{{ t('projects.empty') }}</p>
        <p class="empty-hint">{{ t('projects.emptyHint') }}</p>
      </div>

      <footer v-if="totalPages > 1" class="pager">
        <button :disabled="filter.page <= 1" @click="gotoPage(filter.page - 1)">
          <Icon icon="mdi:chevron-left" width="14" height="14" />
          {{ t('common.prev') }}
        </button>
        <span class="pager-info">{{ filter.page }} / {{ totalPages }} ({{ t('common.totalCount', { count: total }) }})</span>
        <button :disabled="filter.page >= totalPages" @click="gotoPage(filter.page + 1)">
          {{ t('common.next') }}
          <Icon icon="mdi:chevron-right" width="14" height="14" />
        </button>
      </footer>
    </div>

    <!-- 工具 skill 列表 Modal(点击 chip 触发) -->
    <Modal v-model="skillsModal.open" :title="skillsModal.title" size="md">
      <ul v-if="skillsModal.skills.length" class="skill-list">
        <li v-for="s in skillsModal.skills" :key="s.source_path || s.name" class="skill-list-item">
          <code class="skill-list-name">{{ s.name }}</code>
          <span class="skill-list-path" :title="s.source_path">{{ s.source_path }}</span>
        </li>
      </ul>
      <p v-else class="empty-title">{{ t('common.dash') }}</p>
    </Modal>

    <!-- 通用确认弹窗 -->
    <Modal
      v-model="confirmOpen"
      size="sm"
      :title="confirmOpts.title"
      :close-on-mask="false"
    >
      <p class="confirm-message">{{ confirmOpts.message }}</p>
      <template #footer>
        <button type="button" class="ghost" @click="resolveConfirm(false)">
          {{ confirmOpts.cancelText }}
        </button>
        <button
          type="button"
          :class="confirmOpts.variant === 'danger' ? 'danger' : 'primary'"
          @click="resolveConfirm(true)"
        >
          {{ confirmOpts.confirmText }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<style scoped>
.projects-view {
  max-width: 1100px;
  margin: 0 auto;
  color: var(--text);
  transition: color 0.3s ease;
}

/* 页面头部 */
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

.view-icon-purple {
  background: var(--text-dim);
}

.view-title h1 {
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
  margin: 0 0 4px;
  transition: color 0.3s ease;
}

.view-title p {
  font-size: 14px;
  color: var(--text-dim);
  margin: 0;
  transition: color 0.3s ease;
}

/* 工具栏 */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  max-width: 400px;
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
  height: 40px;
}

/* 错误消息 */
.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--danger-dim);
  color: var(--danger);
  border-radius: var(--radius-sm);
  font-size: 13px;
  margin-bottom: 16px;
}

/* 卡片样式 */
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  padding: 20px;
  margin-bottom: 16px;
  transition: all 0.3s ease;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.card-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

/* 弹窗内表单 */
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
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

.inspecting-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--primary);
  font-weight: 400;
}

/* 路径输入 + 重新选择按钮 */
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

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 项目卡片网格(2026-06-29 改:表格 → 卡片) */
.projects-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
  margin: 0 -4px;
  padding: 0 4px;
}

.project-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  position: relative;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

.project-card:hover {
  border-color: var(--text-faint);
  box-shadow: var(--shadow-card);
}

.project-card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.project-card-titles {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.project-card-name {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-card-alias {
  font-size: 11px;
  background: var(--primary-dim);
  color: var(--primary);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  font-family: 'JetBrains Mono', monospace;
}

.project-card-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

/* 卡片右侧操作图标:直接用 Icon 标签,@click 绑逻辑 */
.action-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
  color: var(--text-dim);
}
.action-icon:hover {
  background: var(--bg-hover);
  color: var(--text);
}
/* Finder 图标:hover 显主题色 */
.action-icon-finder {
  color: var(--primary);
}
.action-icon-finder:hover {
  background: var(--primary-dim);
  color: var(--primary);
}
/* 删除图标:hover 显警示色 */
.action-icon-danger:hover {
  background: var(--danger-dim);
  color: var(--danger);
}

/* 按钮内 spinner 跟文字留点间距 */
.btn-spinner {
  margin-right: 4px;
  vertical-align: -2px;
}

/* 小号 spinner(给 inspecting 标签、扫描中、按钮内用) */
.spinner-sm {
  width: 12px;
  height: 12px;
  border-width: 2px;
}

.project-card-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.project-card-path {
  margin: 0;
  font-size: 11px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-card-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 26px;
  align-items: center;
}

.tools-loading {
  display: inline-flex;
  align-items: center;
  color: var(--text-faint);
  font-size: 11px;
}

.tools-empty {
  font-size: 11px;
  color: var(--text-faint);
}

.tools-error {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--danger);
}

/* 工具 chip:claude5 风格(小写工具名 + 角标数字) */
.tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 4px 3px 8px;
  font-size: 11px;
  font-weight: 500;
  background: var(--primary-dim);
  color: var(--primary);
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: 'JetBrains Mono', monospace;
}

.tool-chip:hover {
  border-color: var(--primary);
}

.chip-label {
  letter-spacing: 0.2px;
}

.chip-count {
  background: var(--primary);
  color: var(--bg-card);
  border-radius: 999px;
  padding: 0 6px;
  font-size: 10px;
  font-weight: 600;
  min-width: 18px;
  text-align: center;
}

.project-card-meta {
  position: absolute;
  right: 10px;
  bottom: 4px;
  font-size: 10px;
  color: var(--text-faint);
  opacity: 0;
  transition: opacity 0.15s ease;
  pointer-events: none;
}

.project-card:hover .project-card-meta {
  opacity: 1;
}

/* 工具 skill 列表 Modal 内部样式 */
.skill-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 60vh;
  overflow-y: auto;
}

.skill-list-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.skill-list-name {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
}

.skill-list-path {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 分页器 */
.pager {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
}

.pager button {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
}

.pager-info {
  font-size: 13px;
  color: var(--text-dim);
}

/* 确认弹窗 */
.confirm-message {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre-line;
}

/* 空状态 */
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
  color: var(--text-dim);
  margin: 6px 0 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .search-box {
    max-width: none;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  /* 移动端:卡片网格降到 1 列 */
  .projects-grid {
    grid-template-columns: 1fr;
    margin: 0;
    padding: 0;
  }
}
</style>
