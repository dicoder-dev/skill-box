<script setup>
import { ref, onMounted, onUnmounted, provide, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import ProjectsView from './views/ProjectsView.vue'
import SkillsView from './views/SkillsView.vue'
import MarketView from './views/MarketView.vue'
import OnboardingView from './views/OnboardingView.vue'
import AuditView from './views/AuditView.vue'
import SettingsView from './views/SettingsView.vue'
import { listSkills } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { getOnboardingStatus } from '@/api/skillbox/onboarding'

const { t } = useI18n()

const tab = ref('skills')

// 轻量事件总线
const eventBus = (() => {
  const listeners = new Map()
  return {
    on(name, fn) {
      if (!listeners.has(name)) listeners.set(name, new Set())
      listeners.get(name).add(fn)
    },
    off(name, fn) {
      listeners.get(name)?.delete(fn)
    },
    emit(name, payload) {
      listeners.get(name)?.forEach((fn) => {
        try { fn(payload) } catch (e) { console.error(`[eventBus] ${name} listener error:`, e) }
      })
    },
  }
})()
provide('appBus', eventBus)

// 暗黑模式控制
const isDark = ref(false)

// 侧边栏宽度（可拖拽调节，持久化）
const sidebarWidth = ref(260)
const MIN_SIDEBAR_WIDTH = 200
const MAX_SIDEBAR_WIDTH = 420

// 初始化时从 localStorage 读取主题偏好和侧边栏宽度
onMounted(() => {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark') {
    isDark.value = true
    document.documentElement.classList.add('dark')
  } else if (savedTheme === 'light') {
    isDark.value = false
    document.documentElement.classList.remove('dark')
  } else {
    // 检测系统偏好
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    isDark.value = prefersDark
    if (prefersDark) {
      document.documentElement.classList.add('dark')
    }
  }

  // 恢复侧边栏宽度
  const savedWidth = localStorage.getItem('sidebarWidth')
  if (savedWidth) {
    const w = parseInt(savedWidth, 10)
    if (w >= MIN_SIDEBAR_WIDTH && w <= MAX_SIDEBAR_WIDTH) {
      sidebarWidth.value = w
    }
  }
})

// 切换主题
function toggleTheme() {
  isDark.value = !isDark.value
  if (isDark.value) {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  } else {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  }
}

// 响应式
const sidebarOpen = ref(true)
const isMobile = ref(false)

function checkViewport() {
  isMobile.value = window.innerWidth < 768
  if (isMobile.value) sidebarOpen.value = false
  else sidebarOpen.value = true
}
onMounted(() => {
  checkViewport()
  window.addEventListener('resize', checkViewport)
})
onUnmounted(() => window.removeEventListener('resize', checkViewport))

// 侧边栏拖拽调节宽度
const isResizing = ref(false)

function startResize(e) {
  if (isMobile.value) return
  isResizing.value = true
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  e.preventDefault()
}

function onResizeMove(e) {
  if (!isResizing.value) return
  const newWidth = Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, e.clientX))
  sidebarWidth.value = newWidth
}

function stopResize() {
  if (!isResizing.value) return
  isResizing.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  localStorage.setItem('sidebarWidth', String(sidebarWidth.value))
}

onMounted(() => {
  window.addEventListener('mousemove', onResizeMove)
  window.addEventListener('mouseup', stopResize)
})
onUnmounted(() => {
  window.removeEventListener('mousemove', onResizeMove)
  window.removeEventListener('mouseup', stopResize)
})

// 顶部统计
const stats = ref({
  skills: 0,
  projects: 0,
  toolsReady: 0,
  toolsTotal: 0,
})
const backendOK = ref(false)
const lastHealth = ref('')

async function refreshStats() {
  try {
    const [skillRes, projRes, obRes] = await Promise.all([
      listSkills({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
      listProjects({ page: 1, size: 1 }).catch(() => ({ total: 0 })),
      getOnboardingStatus().catch(() => ({ adapters: [] })),
    ])
    stats.value.skills = skillRes?.total || 0
    stats.value.projects = projRes?.total || 0
    const adapters = obRes?.adapters || []
    stats.value.toolsTotal = adapters.length
    stats.value.toolsReady = adapters.filter((a) => a.global_ok).length
    backendOK.value = true
    lastHealth.value = new Date().toLocaleTimeString()
  } catch (_) {
    backendOK.value = false
  }
}

onMounted(refreshStats)

// 侧栏配置
const navItems = computed(() => [
  { key: 'skills',      label: t('app.nav.skills.label'),      desc: t('app.nav.skills.desc'),      icon: 'mdi:book-open-variant' },
  { key: 'projects',   label: t('app.nav.projects.label'),    desc: t('app.nav.projects.desc'),    icon: 'mdi:folder-multiple-outline' },
  { key: 'market',     label: t('app.nav.market.label'),      desc: t('app.nav.market.desc'),      icon: 'mdi:cart-outline' },
  { key: 'onboarding',  label: t('app.nav.onboarding.label'),  desc: t('app.nav.onboarding.desc'),  icon: 'mdi:compass-outline' },
  { key: 'audit',      label: t('app.nav.audit.label'),       desc: t('app.nav.audit.desc'),       icon: 'mdi:script-text-outline' },
  { key: 'settings',    label: t('app.nav.settings.label'),   desc: t('app.nav.settings.desc'),    icon: 'mdi:cog-outline' },
])

function switchTab(k) {
  tab.value = k
  if (k === 'onboarding' || k === 'audit' || k === 'skills') refreshStats()
  if (isMobile.value) sidebarOpen.value = false
}

// 跨组件跳转
function onBusEvent(name, payload) {
  if (name === 'switch-tab') {
    switchTab(payload)
  }
}
function onWindowEvent(e) {
  if (e?.type === 'skillbox:switch-tab') onBusEvent('switch-tab', e.detail)
}
onMounted(() => {
  eventBus.on('switch-tab', onBusEvent)
  window.addEventListener('skillbox:switch-tab', onWindowEvent)
})
onUnmounted(() => {
  eventBus.off('switch-tab', onBusEvent)
  window.removeEventListener('skillbox:switch-tab', onWindowEvent)
})
</script>

<template>
  <div :class="['app-container', isDark ? 'dark' : '']">
    <!-- 移动端遮罩 -->
    <div
      v-if="isMobile && sidebarOpen"
      class="fixed inset-0 bg-black/50 z-30 backdrop-blur-sm transition-opacity duration-200"
      @click="sidebarOpen = false"
    ></div>

    <!-- 侧边栏 - 重设计的现代风格 -->
    <aside
      :class="[
        'sidebar flex flex-col z-40',
        'transition-transform duration-300 ease-out',
        isMobile
          ? (sidebarOpen ? 'fixed inset-y-0 left-0 translate-x-0' : 'fixed inset-y-0 left-0 -translate-x-full')
          : 'sticky top-0 h-screen',
      ]"
      :style="!isMobile ? { width: sidebarWidth + 'px' } : {}"
    >
      <!-- 品牌区域 - 留出 macOS 交通灯按钮的空间 -->
      <div class="sidebar-brand">
        <div class="brand-icon">
          <Icon icon="mdi:package-variant-closed" width="24" height="24" />
        </div>
        <div class="brand-text">
          <span class="brand-name">{{ t('app.brand') }}</span>
          <span class="brand-tagline">{{ t('app.tagline') }}</span>
        </div>
        <button
          v-if="isMobile"
          class="mobile-close-btn"
          @click="sidebarOpen = false"
          :aria-label="t('app.closeSidebar')"
        >
          <Icon icon="mdi:close" width="18" height="18" />
        </button>
      </div>

      <!-- 导航菜单 -->
      <nav class="sidebar-nav flex-1">
        <button
          v-for="n in navItems"
          :key="n.key"
          :class="[
            'nav-item',
            tab === n.key ? 'nav-item-active' : ''
          ]"
          @click="switchTab(n.key)"
        >
          <span class="nav-icon">
            <Icon :icon="n.icon" width="20" height="20" />
          </span>
          <span class="nav-content">
            <span class="nav-label">{{ n.label }}</span>
            <span :class="['nav-desc', tab === n.key ? 'nav-desc-active' : '']">
              {{ n.desc }}
            </span>
          </span>
          <span v-if="tab === n.key" class="nav-indicator"></span>
        </button>
      </nav>

      <!-- 底部区域 -->
      <div class="sidebar-footer">
        <!-- 健康状态 -->
        <div :class="['status-indicator', backendOK ? 'status-ok' : 'status-error']">
          <span :class="['status-dot', backendOK ? 'dot-ok' : 'dot-error']"></span>
          <span class="status-text">
            {{ backendOK ? t('app.backendOk') : t('app.backendDown') }}
            <span v-if="lastHealth" class="status-time">{{ lastHealth }}</span>
          </span>
        </div>

        <!-- 主题切换 -->
        <button class="theme-toggle" @click="toggleTheme" :title="isDark ? '切换到亮色模式' : '切换到暗黑模式'">
          <Icon :icon="isDark ? 'mdi:weather-sunny' : 'mdi:weather-night'" width="18" height="18" />
        </button>

        <!-- 刷新按钮 -->
        <button class="refresh-btn" @click="refreshStats" :title="t('app.refreshStats')">
          <Icon icon="mdi:refresh" width="16" height="16" />
        </button>
      </div>

      <!-- 拖拽手柄 -->
      <div
        v-if="!isMobile"
        class="resize-handle"
        :class="{ active: isResizing }"
        @mousedown="startResize"
        :title="`侧边栏宽度: ${sidebarWidth}px（拖拽调节）`"
      >
        <div class="resize-grip">
          <span></span>
          <span></span>
        </div>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content flex flex-col min-w-0">
      <!-- 顶部栏 -->
      <header class="topbar">
        <div class="topbar-left">
          <button
            v-if="isMobile"
            class="menu-toggle"
            @click="sidebarOpen = true"
            :aria-label="t('app.openSidebar')"
          >
            <Icon icon="mdi:menu" width="22" height="22" />
          </button>
          <div class="breadcrumb">
            <span class="breadcrumb-brand">{{ t('app.brand') }}</span>
            <Icon icon="mdi:chevron-right" width="14" height="14" class="breadcrumb-sep" />
            <span class="breadcrumb-current">{{ navItems.find((x) => x.key === tab)?.label }}</span>
          </div>
        </div>

        <div class="topbar-right">
          <div class="stat-badge">
            <Icon icon="mdi:book-open-variant" width="12" height="12" />
            <span>{{ t('app.nav.skills.label') }}</span>
            <strong>{{ stats.skills }}</strong>
          </div>
          <div class="stat-badge stat-badge-purple">
            <Icon icon="mdi:folder-multiple-outline" width="12" height="12" />
            <span>{{ t('app.nav.projects.label') }}</span>
            <strong>{{ stats.projects }}</strong>
          </div>
          <div class="stat-badge stat-badge-green">
            <Icon icon="mdi:tools" width="12" height="12" />
            <span>{{ t('app.toolsLabel') }}</span>
            <strong>{{ stats.toolsReady }}/{{ stats.toolsTotal }}</strong>
          </div>
        </div>
      </header>

      <!-- 内容区域 -->
      <div class="content-area">
        <ProjectsView v-if="tab === 'projects'" />
        <SkillsView v-else-if="tab === 'skills'" />
        <MarketView v-else-if="tab === 'market'" />
        <OnboardingView v-else-if="tab === 'onboarding'" />
        <AuditView v-else-if="tab === 'audit'" />
        <SettingsView v-else-if="tab === 'settings'" />
      </div>
    </main>
  </div>
</template>

<style scoped>
/* 应用容器 */
.app-container {
  @apply flex min-h-screen;
  background: var(--bg);
  color: var(--text);
  transition: background-color 0.3s ease, color 0.3s ease;
}

/* ============================================
   侧边栏样式
   ============================================ */
.sidebar {
  width: 260px;
  position: relative;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-sidebar);
  box-shadow: var(--shadow-sidebar);
  transition: background-color 0.3s ease, border-color 0.3s ease;
}

/* 品牌区域 - 顶部留出空间避开 macOS 交通灯按钮 */
.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 56px 16px 20px;  /* 顶部 56px 留给 macOS 红绿灯 */
  border-bottom: 1px solid var(--border-sidebar);
  position: relative;
}

/* 非 macOS 系统 / web 端:品牌区域顶部空间缩小 */
@media (max-width: 768px), (hover: none) {
  .sidebar-brand {
    padding-top: 16px;
  }
}

/* 拖拽手柄 - 位于侧边栏右边缘 */
.resize-handle {
  position: absolute;
  top: 0;
  right: -3px;
  width: 6px;
  height: 100%;
  cursor: col-resize;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s ease;
}

.resize-handle:hover,
.resize-handle.active {
  background: linear-gradient(
    to right,
    transparent,
    rgba(37, 99, 235, 0.08),
    rgba(37, 99, 235, 0.2)
  );
}

.resize-grip {
  display: flex;
  flex-direction: column;
  gap: 3px;
  opacity: 0;
  transition: opacity 0.15s ease;
  padding: 8px 0;
}

.resize-handle:hover .resize-grip,
.resize-handle.active .resize-grip {
  opacity: 1;
}

.resize-grip span {
  display: block;
  width: 2px;
  height: 16px;
  background: var(--primary);
  border-radius: 1px;
}

.brand-icon {
  @apply flex items-center justify-center rounded-lg;
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #0d9488 0%, #f59e0b 100%);
  color: white;
  flex-shrink: 0;
}

.brand-text {
  @apply flex flex-col min-w-0 flex-1;
}

.brand-name {
  @apply font-semibold text-base;
  color: var(--text);
  transition: color 0.3s ease;
}

.brand-tagline {
  @apply text-xs truncate;
  color: var(--text-sidebar-muted);
  transition: color 0.3s ease;
}

.mobile-close-btn {
  @apply p-1.5 rounded-lg;
  color: var(--text-sidebar-muted);
  background: transparent;
  border: none;
  padding: 8px;
}

.mobile-close-btn:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}

/* 导航菜单 */
.sidebar-nav {
  @apply px-3 py-4;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-item {
  @apply relative flex items-center gap-3 px-3 py-3 rounded-lg;
  background: transparent;
  border: none;
  color: var(--text-sidebar-muted);
  text-align: left;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}

.nav-item-active {
  background: var(--bg-sidebar-active);
  color: var(--bg-sidebar-active-text);
}

.nav-item-active:hover {
  background: var(--bg-sidebar-active);
  color: var(--bg-sidebar-active-text);
}

.nav-icon {
  @apply flex items-center justify-center flex-shrink-0;
  width: 24px;
  height: 24px;
}

.nav-content {
  @apply flex flex-col min-w-0 flex-1;
}

.nav-label {
  @apply text-sm font-medium;
  color: inherit;
  transition: color 0.2s ease;
}

.nav-desc {
  @apply text-xs truncate mt-0.5;
  color: var(--text-sidebar-muted);
  transition: color 0.2s ease;
}

.nav-desc-active {
  color: var(--bg-sidebar-active-text);
  opacity: 0.85;
}

/* 导航激活指示器 */
.nav-indicator {
  @apply absolute right-2 w-1.5 h-1.5 rounded-full;
  background: var(--bg-sidebar-active-text);
}

/* 侧边栏底部 */
.sidebar-footer {
  @apply flex items-center gap-2 px-4;
  padding: 16px;
  border-top: 1px solid var(--border-sidebar);
}

.status-indicator {
  @apply flex items-center gap-2 flex-1 min-w-0;
}

.status-dot {
  @apply w-2 h-2 rounded-full flex-shrink-0;
}

.dot-ok {
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
}

.dot-error {
  background: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
}

.status-text {
  @apply text-xs truncate;
  color: var(--text-sidebar-muted);
}

.status-time {
  @apply opacity-70;
}

.theme-toggle {
  @apply p-2 rounded-lg flex items-center justify-center;
  background: transparent;
  border: none;
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.theme-toggle:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}

.refresh-btn {
  @apply p-2 rounded-lg flex items-center justify-center;
  background: transparent;
  border: 1px solid var(--border-sidebar);
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.refresh-btn:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}

/* ============================================
   主内容区样式
   ============================================ */
.main-content {
  @apply flex-1 flex flex-col min-w-0;
}

/* 顶部栏 */
.topbar {
  @apply flex items-center justify-between px-5 py-3;
  background: var(--bg-header);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 20;
  transition: all 0.3s ease;
}

.topbar-left {
  @apply flex items-center gap-3;
}

.menu-toggle {
  @apply p-2 -ml-2 rounded-lg flex items-center justify-center;
  color: var(--text-dim);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
}

.menu-toggle:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.breadcrumb {
  @apply flex items-center gap-2 text-sm;
}

.breadcrumb-brand {
  @apply px-2.5 py-1 rounded-full text-xs font-medium;
  background: linear-gradient(135deg, #0d9488 0%, #f59e0b 100%);
  color: white;
}

.breadcrumb-sep {
  @apply opacity-40;
  color: var(--text);
}

.breadcrumb-current {
  @apply font-medium;
  color: var(--text);
}

.topbar-right {
  @apply flex items-center gap-2 flex-wrap;
}

.stat-badge {
  @apply inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text-dim);
  transition: all 0.15s ease;
}

.stat-badge strong {
  color: var(--text);
  font-weight: 600;
}

.stat-badge-purple {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.1) 0%, rgba(245, 158, 11, 0.05) 100%);
  border-color: rgba(245, 158, 11, 0.2);
}

.stat-badge-purple strong {
  color: #d97706;
}

.stat-badge-green {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(16, 185, 129, 0.05) 100%);
  border-color: rgba(16, 185, 129, 0.2);
}

.stat-badge-green strong {
  color: #059669;
}

/* 内容区域 */
.content-area {
  @apply flex-1 p-5 overflow-auto;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .sidebar {
    width: 280px;
  }

  .topbar-right {
    display: none;
  }

  .content-area {
    padding: 16px;
  }
}
</style>
