<script setup>
import { ref, onMounted, onUnmounted, provide, computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import ProjectsView from './views/ProjectsView.vue'
import SkillsView from './views/SkillsView.vue'
import MarketView from './views/MarketView.vue'
import SettingsView from './views/SettingsView.vue'
// 2026-07-01 增:工具元数据管理视图
import ToolsView from './views/ToolsView.vue'
import ToastContainer from './components/ToastContainer.vue'
import { listSkills } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { getOnboardingStatus } from '@/api/skillbox/onboarding'
import { useAppStore } from '@/core/store/app.js'

const { t } = useI18n()
const { runMode } = storeToRefs(useAppStore())

const tab = ref('skills')

// 2026-07-08 增:把 tab key 响应式 provide 出去,SettingsView 等子视图
// 可以直接 watch 它来决定何时刷新数据,比事件总线更直接(无需关心
// 注册时机、emit 时序)。事件总线那条路径保留,作为兼容兜底。
provide('activeTab', tab)

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

// 初始化时从 localStorage 读取主题偏好
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
// 2026-07-11 增:仅 macOS 桌面端需要为交通灯按钮预留侧栏顶部空间;
// Web 端与其他桌面系统贴顶即可。
const isMacOS = ref(false)

function checkViewport() {
  isMobile.value = window.innerWidth < 768
  if (isMobile.value) sidebarOpen.value = false
  else sidebarOpen.value = true
}
onMounted(() => {
  checkViewport()
  window.addEventListener('resize', checkViewport)
  // 只在桌面端判断平台;web 端直接 false 走"贴顶"布局。
  // 后端目前没把 os 注入 __APP_RUNTIME__,统一靠 navigator.userAgent 兜底。
  if (runMode === 'desktop') {
    const ua = (typeof navigator !== 'undefined' && navigator.userAgent) || ''
    isMacOS.value = /Mac|iPhone|iPad/i.test(ua)
  }
})
onUnmounted(() => window.removeEventListener('resize', checkViewport))

// 顶部统计
const stats = ref({
  skills: 0,
  projects: 0,
  toolsReady: 0,
  toolsTotal: 0,
})
const backendOK = ref(false)

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
  } catch (_) {
    backendOK.value = false
  }
}

onMounted(refreshStats)

// 侧栏/顶栏配置
// 顺序:技能 / 工具 / 项目 / 市场 / 设置 — 把"工具"提前到"项目"之前。
// 2026-07-11 改:侧栏改为纯图标条(hover tooltip),导航主体迁到顶栏,
// 此处 navItems 同时供侧栏图标按钮和顶栏 tab 按钮复用,key/icon/label 不变。
const navItems = computed(() => [
  { key: 'skills',    label: t('app.nav.skills.label'),    icon: 'mdi:book-open-variant' },
  // 2026-07-06 调:工具提到 projects 之前(原来是 projects → tools)
  { key: 'tools',     label: t('app.nav.tools.label'),     icon: 'mdi:tools' },
  { key: 'projects',  label: t('app.nav.projects.label'),  icon: 'mdi:folder-multiple-outline' },
  { key: 'market',    label: t('app.nav.market.label'),    icon: 'mdi:cart-outline' },
  { key: 'settings',  label: t('app.nav.settings.label'),  icon: 'mdi:cog-outline' },
])

function switchTab(k) {
  tab.value = k
  if (k === 'skills') refreshStats()
  if (isMobile.value) sidebarOpen.value = false
  // 2026-07-08 增:广播 tab 切换事件,让 SettingsView 等视图在重新进入
  // 时能重新拉 prefs(避免 v-if 复用组件导致 onMounted 不再触发,
  // 应用方式等设置状态停留在旧值)。window event 兜底兼容 web 端。
  try { eventBus.emit('app:tab-change', k) } catch (_) { /* no-op */ }
  window.dispatchEvent(new CustomEvent('skillbox:tab-change', { detail: k }))
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

    <!-- 侧边栏 - 2026-07-11 改:64px 纯图标条。
         顶部 macOS 交通灯预留区只在桌面 macOS 渲染(web / Windows / Linux 贴顶)。
         底部状态组:后端状态 / 主题 / 刷新。 -->
    <aside
      :class="[
        'sidebar flex flex-col z-40',
        'transition-transform duration-300 ease-out',
        isMobile
          ? (sidebarOpen ? 'fixed inset-y-0 left-0 translate-x-0' : 'fixed inset-y-0 left-0 -translate-x-full')
          : 'sticky top-0 h-screen',
      ]"
    >
      <!-- 仅 macOS 桌面端需要为交通灯预留空间;
           web / Windows / Linux 全部贴顶,不留白 -->
      <div v-if="isMacOS" class="sidebar-top-spacer" aria-hidden="true"></div>

      <!-- 导航菜单(纯图标 + hover tooltip) -->
      <nav class="sidebar-nav">
        <button
          v-for="n in navItems"
          :key="n.key"
          :class="['nav-item', tab === n.key ? 'nav-item-active' : '']"
          :data-tooltip="n.label"
          :aria-label="n.label"
          role="tooltip"
          @click="switchTab(n.key)"
        >
          <span class="nav-icon">
            <IconPark :icon="n.icon" width="22" height="22" />
          </span>
          <span v-if="tab === n.key" class="nav-indicator"></span>
        </button>
      </nav>

      <!-- 底部状态组 - 与导航同一栏,纵向居中;2026-07-11 从顶栏下移 -->
      <div class="sidebar-footer">
        <!-- 后端连接状态(小圆点) -->
        <div
          :class="['footer-status', backendOK ? 'status-ok' : 'status-error']"
          :data-tooltip="backendOK ? t('app.backendOk') : t('app.backendDown')"
          role="tooltip"
          :aria-label="backendOK ? t('app.backendOk') : t('app.backendDown')"
        >
          <span :class="['footer-status-dot', backendOK ? 'dot-ok' : 'dot-error']"></span>
        </div>

        <!-- 主题切换 -->
        <button
          class="footer-icon-btn"
          @click="toggleTheme"
          :data-tooltip="isDark ? t('app.themeToggle.toLight') : t('app.themeToggle.toDark')"
          role="tooltip"
          :aria-label="isDark ? t('app.themeToggle.toLight') : t('app.themeToggle.toDark')"
        >
          <IconPark :icon="isDark ? 'mdi:weather-sunny' : 'mdi:weather-night'" width="18" height="18" />
        </button>

        <!-- 刷新统计 -->
        <button
          class="footer-icon-btn"
          @click="refreshStats"
          :data-tooltip="t('app.refreshStats')"
          role="tooltip"
          :aria-label="t('app.refreshStats')"
        >
          <IconPark icon="mdi:refresh" width="16" height="16" />
        </button>
      </div>

      <!-- 移动端关闭按钮(锚在状态组下方,占位用) -->
      <button
        v-if="isMobile"
        class="mobile-close-btn"
        @click="sidebarOpen = false"
        :aria-label="t('app.closeSidebar')"
      >
        <IconPark icon="mdi:close" width="18" height="18" />
      </button>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content flex flex-col min-w-0">
      <!-- 顶部栏 - 2026-07-11 改:左侧 = Logo,右侧 = 3 个 stat-badge;
           顶部 tabs 已删除(与侧栏图标重复),后端状态 / 主题 / 刷新下移到侧栏底部。 -->
      <header class="topbar">
        <div class="topbar-left">
          <button
            v-if="isMobile"
            class="menu-toggle"
            @click="sidebarOpen = true"
            :aria-label="t('app.openSidebar')"
          >
            <IconPark icon="mdi:menu" width="22" height="22" />
          </button>

          <div class="topbar-logo">
            <span class="topbar-logo-icon">
              <IconPark icon="mdi:package-variant-closed" width="18" height="18" />
            </span>
            <span class="topbar-logo-text">{{ t('app.brand') }}</span>
          </div>
        </div>

        <div class="topbar-right">
          <div class="stat-badge stat-badge-blue">
            <IconPark icon="mdi:book-open-variant" width="12" height="12" />
            <span>{{ t('app.nav.skills.label') }}</span>
            <strong>{{ stats.skills }}</strong>
          </div>
          <div class="stat-badge stat-badge-violet">
            <IconPark icon="mdi:folder-multiple-outline" width="12" height="12" />
            <span>{{ t('app.nav.projects.label') }}</span>
            <strong>{{ stats.projects }}</strong>
          </div>
          <div class="stat-badge stat-badge-emerald">
            <IconPark icon="mdi:tools" width="12" height="12" />
            <span>{{ t('app.toolsLabel') }}</span>
            <strong>{{ stats.toolsReady }}/{{ stats.toolsTotal }}</strong>
          </div>
        </div>
      </header>

      <!-- 内容区域 -->
      <div class="content-area">
        <ProjectsView v-if="tab === 'projects'" />
        <ToolsView v-else-if="tab === 'tools'" />
        <SkillsView v-else-if="tab === 'skills'" />
        <MarketView v-else-if="tab === 'market'" />
        <SettingsView v-else-if="tab === 'settings'" />
      </div>
    </main>

    <!-- 全局 toast 浮层(右上角) -->
    <ToastContainer />
  </div>
</template>

<style scoped>
/* 应用容器 - 锁定视口高度,内部各自滚动,
   避免内容多时撑高外层导致 sticky 侧栏跟着滚 */
.app-container {
  @apply flex h-screen overflow-hidden;
  background: var(--bg);
  color: var(--text);
  transition: background-color 0.3s ease, color 0.3s ease;
}

/* ============================================
   侧边栏样式 - 2026-07-11 改:纯图标条 64px
   ============================================ */
.sidebar {
  width: 64px;
  flex-shrink: 0;            /* flex 容器里不被压窄,固定宽度 */
  align-self: stretch;        /* 占满父容器(app-container)高度 */
  position: sticky;           /* 相对 app-container 锁定,不再跟随内容滚 */
  top: 0;
  height: 100vh;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-sidebar);
  box-shadow: var(--shadow-sidebar);
  transition: background-color 0.3s ease, border-color 0.3s ease;
}

/* 顶部 macOS 交通灯预留区 - 2026-07-11 改:只在 macOS 桌面端渲染,
   高度调小到 32px(原 56px 太多空白);web / Windows / Linux 由 v-if
   直接不渲染,贴顶。 */
.sidebar-top-spacer {
  height: 32px;
  flex-shrink: 0;
}

.mobile-close-btn {
  @apply mx-auto mb-3 p-2 rounded-lg flex items-center justify-center;
  color: var(--text-sidebar-muted);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
}
.mobile-close-btn:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}

/* 导航菜单 - 纯图标,hover 显示 tooltip。
   2026-07-11 改:导航区不再 flex:1,改成自然高度,把空间让给底部状态组;
   状态组用 mt-auto 沉底。 */
.sidebar-nav {
  @apply px-2 py-4;
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
}

.nav-item {
  position: relative;         /* tooltip 锚点 */
  @apply flex items-center justify-center rounded-lg;
  width: 44px;
  height: 44px;
  background: transparent;
  border: none;
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}
.nav-item:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}
.nav-item:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 2px;
}

.nav-item-active {
  background: var(--bg-sidebar-active);
  color: var(--bg-sidebar-active-text);
}
.nav-item-active:hover {
  background: var(--bg-sidebar-active);
  color: var(--bg-sidebar-active-text);
}
.nav-item-active .nav-icon {
  color: var(--accent-blue);
}

.nav-icon {
  @apply flex items-center justify-center;
  width: 22px;
  height: 22px;
}

/* tooltip - 纯 CSS,200ms 延迟避免划过闪烁;主题色自动反转。
   2026-07-11 改:tooltip 元素改为 .nav-item / .footer-icon-btn / .footer-status,
   删除 .topbar-icon-btn / .topbar-status 引用(状态组已下移到侧栏底部)。 */
.nav-item::after,
.footer-icon-btn::after,
.footer-status::after {
  content: attr(data-tooltip);
  position: absolute;
  background: var(--text);
  color: var(--bg-card);
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  z-index: 60;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
  transition: opacity 0.15s ease;
}
/* 侧栏 tooltip - 浮在图标右侧 */
.nav-item::after,
.footer-icon-btn::after,
.footer-status::after {
  left: calc(100% + 12px);
  top: 50%;
  transform: translateY(-50%);
}
.nav-item:hover::after,
.nav-item:focus-visible::after,
.footer-icon-btn:hover::after,
.footer-icon-btn:focus-visible::after,
.footer-status:hover::after,
.footer-status:focus-visible::after {
  opacity: 1;
  transition-delay: 0.2s;
}

/* 导航激活指示器 - 左侧彩色竖条(克制不刺眼) */
.nav-indicator {
  position: absolute;
  left: -2px;
  top: 25%;
  height: 50%;
  width: 3px;
  border-radius: 0 2px 2px 0;
  background: var(--accent-blue);
}

/* ============================================
   侧栏底部状态组 - 2026-07-11 增:与导航同一栏,纵向排列
   ============================================ */
.sidebar-footer {
  margin-top: auto;          /* 关键:把状态组推到侧栏底部 */
  padding: 8px 0 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border-top: 1px solid var(--border-sidebar);
}

.footer-status {
  position: relative;       /* tooltip 锚点 */
  @apply p-2 rounded-lg flex items-center justify-center;
  border: 1px solid var(--border-sidebar);
  background: transparent;
  cursor: default;
}
.footer-status:hover {
  background: var(--bg-sidebar-hover);
}
.footer-status-dot {
  @apply w-2.5 h-2.5 rounded-full;
}
.footer-status .dot-ok {
  background: var(--success);
  box-shadow: 0 0 0 3px rgba(21, 128, 61, 0.18);
}
.footer-status .dot-error {
  background: var(--danger);
  box-shadow: 0 0 0 3px rgba(185, 28, 28, 0.18);
}

.footer-icon-btn {
  position: relative;       /* tooltip 锚点 */
  @apply p-2 rounded-lg flex items-center justify-center;
  background: transparent;
  border: 1px solid var(--border-sidebar);
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: all 0.15s ease;
}
.footer-icon-btn:hover {
  background: var(--bg-sidebar-hover);
  color: var(--text);
}
.footer-icon-btn:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 1px;
}

/* ============================================
   主内容区样式
   ============================================ */
.main-content {
  @apply flex-1 flex flex-col min-w-0;
}

/* 顶部栏 - 2026-07-11 改:左侧只剩 Logo,右侧只剩 3 个 stat-badge;
   顶部 tabs 与后端状态/主题/刷新全部迁出。 */
.topbar {
  @apply flex items-center justify-between px-5 py-2.5;
  background: var(--bg-header);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 20;
  transition: all 0.3s ease;
}

.topbar-left {
  @apply flex items-center gap-3 min-w-0 flex-1;
}

.menu-toggle {
  @apply p-2 -ml-2 rounded-lg flex items-center justify-center;
  color: var(--text-dim);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.menu-toggle:hover {
  background: var(--bg-hover);
  color: var(--text);
}

/* Logo 区 - 2026-07-11 增:从侧栏挪到顶栏左侧 */
.topbar-logo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-right: 12px;
  margin-right: 4px;
  border-right: 1px solid var(--border);
  flex-shrink: 0;
}
.topbar-logo-icon {
  @apply flex items-center justify-center rounded-md;
  width: 28px;
  height: 28px;
  background: var(--text);
  color: var(--bg-card);
}
.topbar-logo-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: 0.3px;
}

.topbar-right {
  @apply flex items-center gap-2 flex-wrap;
  flex-shrink: 0;
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

/* 三种语义色彩 - 技能(蓝) / 项目(紫) / 工具(翠) */
.stat-badge-blue {
  background: var(--accent-blue-bg);
  border-color: var(--accent-blue-border);
  color: var(--accent-blue);
}
.stat-badge-blue :deep(.iconify),
.stat-badge-blue strong { color: var(--accent-blue); }

.stat-badge-violet {
  background: var(--accent-violet-bg);
  border-color: var(--accent-violet-border);
  color: var(--accent-violet);
}
.stat-badge-violet :deep(.iconify),
.stat-badge-violet strong { color: var(--accent-violet); }

.stat-badge-emerald {
  background: var(--accent-emerald-bg);
  border-color: var(--accent-emerald-border);
  color: var(--accent-emerald);
}
.stat-badge-emerald :deep(.iconify),
.stat-badge-emerald strong { color: var(--accent-emerald); }

/* 内容区域 - 内部滚动,让 sticky 侧栏相对 app-container 锁定
   而不跟随 body/html 一起滚 */
.content-area {
  @apply flex-1 p-5 overflow-y-auto;
  min-height: 0;
}

/* 响应式调整 - 移动端:侧栏抽屉 64px,顶栏 logo 文字隐藏,
   状态组继续在侧栏底部(移动端 viewport 内 nav-item 不一定可见,
   底部状态组始终可见是合理的)。 */
@media (max-width: 768px) {
  .sidebar {
    width: 64px;
  }
  .topbar {
    padding: 10px 12px;
  }
  .topbar-logo-text {
    display: none;          /* 移动端空间紧,只留 logo 图标 */
  }
  .topbar-logo {
    padding-right: 8px;
    margin-right: 2px;
  }
  .content-area {
    padding: 16px;
  }
}
</style>
