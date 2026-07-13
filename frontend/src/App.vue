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
  // 后端目前没把 os / titleBarHeight 注入 __APP_RUNTIME__,统一靠 navigator.userAgent 兜底。
  // wails3 webview 在 macOS 上 UA 通常带 "Mac OS X" 或 "Macintosh",与普通 Safari 一致。
  // 2026-07-12 fix:runMode 是 storeToRefs 解构出的 Ref,直接 == 'desktop' 永远 false
  // (ref 对象 vs 字符串),导致 isMacOS 永远是 false,mac-desktop class 永远挂不上,
  // .topbar 的 padding-top: 50px / padding-left: 80px 让位红绿灯逻辑全部失效,
  // logo 文字与红绿灯重叠。必须用 .value 访问原值。
  if (runMode.value === 'desktop') {
    const ua = (typeof navigator !== 'undefined' && navigator.userAgent) || ''
    isMacOS.value = /Mac|iPhone|iPad/i.test(ua)
    // 2026-07-11 v8 增:macOS 桌面端给 html 加 .is-mac-desktop class,
    // CSS 凭这个把 .topbar 顶部 padding 抬 50px,让 logo 与红绿灯垂直错开
    // (侧栏 spacer 已经让了 50px,但主内容区 topbar 没让位,导致 logo
    // 视觉上被压扁在红绿灯旁边)。
    if (isMacOS.value && typeof document !== 'undefined') {
      document.documentElement.classList.add('is-mac-desktop')
    }
    // 2026-07-11 v4 增:调试 log,方便桌面端用户在 DevTools 确认 isMacOS 是否正确命中
    // 与 sidebar-top-spacer 实际渲染情况(打开 webview DevTools 看 layout)。
    if (typeof console !== 'undefined') {
      console.log('[App] macOS detection:', {
        runMode: runMode.value,
        ua: ua.substring(0, 200),
        isMacOS: isMacOS.value,
      })
    }
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

    <!-- 2026-07-12 改:顶栏提到 app-container 第一行,与 sidebar/main-content 平级,
         横穿整个屏幕宽度(包括覆盖左侧导航栏所在区域),由 .topbar-overlay-left
         把 logo / 后端状态 / 主题等子元素定位到侧栏之上的视觉层;
         原 64px 宽侧栏在 .body-row 内仍然占位,只是被顶栏视觉压住。 -->
    <header :class="['topbar', isMacOS ? 'mac-desktop' : '']">
      <div class="topbar-left">
        <button
          v-if="isMobile"
          class="menu-toggle"
          @click="sidebarOpen = true"
          :aria-label="t('app.openSidebar')"
        >
          <IconPark icon="mdi:menu" width="22" height="22" />
        </button>

        <!-- 2026-07-12 改:logo 固定在顶栏最左侧(0 起),盖住侧栏 64px 区域。
             这里 topbar-left 仍占位,但视觉上由 absolute 子层把 logo 推出来,
             不再受 sidebar 64px 宽度影响。
             2026-07-13 增:logo 文字前加 app icon 图片(对齐文字基线,作为视觉锚点)。
             图源 frontend/public/skill-box-logo.png(1024×1024 透明背景),
             跟 build/appicon.png / desktop/appicon.png 保持一致。 -->
        <div class="topbar-logo">
          <img class="topbar-logo-img" src="/skill-box-logo.png" alt="" />
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

    <!-- 2026-07-12 改:body-row 是 sidebar + main-content 的 flex 行容器,
         顶栏在上面一行。桌面端 sidebar 是 fixed(脱离文档流,这里只放
         main-content 占位;z-40 的 sidebar 在 main-content 左侧浮在上方,
         main-content 自身 padding-left: 64px 让位)。
         移动端 sidebar 抽屉化(main-content 仍占满 row)。 -->
    <div class="body-row flex-1 flex min-h-0">
    <!-- 侧边栏 - 2026-07-12 改:从 flex 子项改为 fixed 定位(顶部 48px 起,
         顶栏贴顶,侧栏紧接在顶栏下方),让顶栏视觉上覆盖侧栏。
         桌面端 z-35 浮在顶栏(z-45)之下,顶栏 logo 横穿压住侧栏背景。
         移动端抽屉化照旧,top 统一用 topbar-h 让位避免抽屉盖住顶栏。 -->
    <aside
      :class="[
        'sidebar flex flex-col',
        'transition-transform duration-300 ease-out',
        isMobile
          ? (sidebarOpen ? 'mobile-sidebar fixed left-0 translate-x-0' : 'mobile-sidebar fixed left-0 -translate-x-full')
          : 'fixed left-0',
      ]"
      :style="{ top: 'var(--topbar-h, 48px)', height: 'calc(100vh - var(--topbar-h, 48px))' }"
    >

      <!-- 导航菜单(图标 + label 垂直布局)
           2026-07-11 v6 改:激活态去掉蓝色竖条,改成图标填充蓝色 + label 蓝色,
           无背景色;label 始终可见,删 hover tooltip。
           2026-07-12 改:删掉 isMacOS 的 sidebar-top-spacer(50px 红绿灯让位),
           顶栏现在统一在 z=25 横穿全屏,侧栏从 topbar 下方开始,红绿灯只
           落在顶栏区域,不会延伸到侧栏,spacer 不再需要。 -->
      <nav class="sidebar-nav">
        <button
          v-for="n in navItems"
          :key="n.key"
          :class="['nav-item', tab === n.key ? 'nav-item-active' : '']"
          :aria-label="n.label"
          @click="switchTab(n.key)"
        >
          <span class="nav-icon">
            <!-- 激活态 theme="filled" 让图标变成实心蓝色;非激活保持 outline -->
            <IconPark
              :icon="n.icon"
              :theme="tab === n.key ? 'filled' : 'outline'"
              width="22"
              height="22"
            />
          </span>
          <span class="nav-label">{{ n.label }}</span>
        </button>
      </nav>

      <!-- 底部状态组 - 与导航同一栏,横向排列;
           2026-07-11 v8 改:删除刷新按钮,只留后端状态(左)+ 主题(右)。 -->
      <div class="sidebar-footer">
        <!-- 后端连接状态(左侧) -->
        <div
          :class="['footer-status', backendOK ? 'status-ok' : 'status-error']"
          :data-tooltip="backendOK ? t('app.backendOk') : t('app.backendDown')"
          role="tooltip"
          :aria-label="backendOK ? t('app.backendOk') : t('app.backendDown')"
        >
          <span :class="['footer-status-dot', backendOK ? 'dot-ok' : 'dot-error']"></span>
        </div>

        <!-- 主题切换(右侧) -->
        <button
          class="footer-icon-btn"
          @click="toggleTheme"
          :data-tooltip="isDark ? t('app.themeToggle.toLight') : t('app.themeToggle.toDark')"
          role="tooltip"
          :aria-label="isDark ? t('app.themeToggle.toLight') : t('app.themeToggle.toDark')"
        >
          <IconPark :icon="isDark ? 'mdi:weather-sunny' : 'mdi:weather-night'" width="14" height="14" />
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

    <!-- 主内容区 - 2026-07-12 改:被 body-row 包住,
         桌面端 main-content 自身有 64px 左侧 padding 避开 fixed 侧栏;
         移动端 sidebar 抽屉化,padding-left 不需要。 -->
    <main
      :class="[
        'main-content flex flex-col min-w-0',
        isMobile ? 'w-full' : 'main-content-desktop',
      ]"
    >

      <!-- 内容区域 -->
      <div class="content-area">
        <ProjectsView v-if="tab === 'projects'" />
        <ToolsView v-else-if="tab === 'tools'" />
        <SkillsView v-else-if="tab === 'skills'" />
        <MarketView v-else-if="tab === 'market'" />
        <SettingsView v-else-if="tab === 'settings'" />
      </div>
    </main>
    </div>

    <!-- 全局 toast 浮层(右上角) -->
    <ToastContainer />
  </div>
</template>

<style scoped>
/* 应用容器 - 2026-07-12 改:flex 方向改为 column,
   第一行是 .topbar(横穿全屏),第二行是 .body-row(包 sidebar + main-content)。
   锁定视口高度,内部各自滚动,避免内容多时撑高外层导致 fixed 侧栏跟着滚。 */
.app-container {
  @apply flex flex-col h-screen overflow-hidden;
  background: var(--bg);
  color: var(--text);
  transition: background-color 0.3s ease, color 0.3s ease;
}

/* ============================================
   body-row 容器 - 包住 sidebar + main-content。
   桌面端 sidebar 是 fixed,只占位,真正撑开高度的是 main-content。
   移动端 sidebar 是 fixed 抽屉,只 main-content 占位。
   ============================================ */
.body-row {
  position: relative;
  width: 100%;
}

/* ============================================
   侧边栏样式 - 2026-07-12 改:改为 fixed 定位,顶部 48px(顶栏高度)起。
   顶栏在 z-20 浮在上方,侧栏 z-40 浮在顶栏之上(确保按钮可点),
   桌面端宽度仍 64px。
   ============================================ */
.sidebar {
  width: 64px;
  flex-shrink: 0;
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border-sidebar);
  box-shadow: var(--shadow-sidebar);
  /* 2026-07-12 改:z-index 从 40 降到 35,让 topbar(z=45)横穿
     屏幕时 logo 文字不会被 sidebar 遮住。sidebar 仍浮在
     main-content(z=auto)之上,内容正常被压住。 */
  z-index: 35;
  transition: background-color 0.3s ease, border-color 0.3s ease;
}

/* 顶部 macOS 红绿灯避让区 - 2026-07-11 v4 改:
   桌面端 wails3 在 wails_app.go:512 设了 InvisibleTitleBarHeight: 50
   + MacTitleBarHiddenInset,意味着 webview 区域从屏幕顶 0px 开始,
   红绿灯浮在 webview 上方 0-50px、左 0-80px 区域。
   侧栏宽度 64px 正好覆盖红绿灯水平位置,所以侧栏顶部必须预留
   50px(与 InvisibleTitleBarHeight 一致)才能让红绿灯显示完整。
   web / Windows / Linux 由 v-if 直接不渲染,贴顶。 */
.sidebar-top-spacer {
  height: 50px;
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

/* 导航菜单 - 图标 + label 垂直布局,label 始终可见,无需 hover tooltip。
   2026-07-11 v6 改:每个 nav-item 宽 64px(贴满侧栏宽度),高 56px,
   内部 flex-col 让图标和 label 垂直堆叠、上下间距 4px;gap 改成 4px,
   5 个 item 总高度 5*56 + 4*4 = 296px,侧栏高度足够时仍由 justify-content:
   center 居中。 */
.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  justify-content: center;
  min-height: 0;
}

.nav-item {
  @apply flex flex-col items-center justify-center rounded-md;
  width: 64px;            /* 贴满侧栏宽 */
  height: 56px;           /* 图标 + label 垂直占位 */
  background: transparent;
  border: none;
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: color 0.15s ease;
  padding: 6px 2px;
  gap: 4px;
}
.nav-item:hover {
  color: var(--text);
}
.nav-item:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: -2px;
}

/* 2026-07-11 v6 改:激活态去背景色,只让图标(theme="filled")和
   label 文字变蓝,保持视觉简洁。 */
.nav-item-active {
  background: transparent;
  color: var(--accent-blue);       /* label 文字变蓝 */
}
.nav-item-active:hover {
  background: transparent;
  color: var(--accent-blue);
}
.nav-item-active .nav-icon {
  color: var(--accent-blue);       /* 图标变蓝(IconPark theme=filled 自动实心) */
}

.nav-icon {
  @apply flex items-center justify-center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
}

/* 2026-07-11 v6 增:nav-label 始终显示,不再依赖 hover tooltip。
   字号小一点,与图标形成视觉层级。 */
.nav-label {
  font-size: 10px;
  line-height: 1.2;
  font-weight: 500;
  letter-spacing: 0.2px;
  white-space: nowrap;
  text-align: center;
  color: inherit;          /* 跟随 .nav-item 的 color,激活态变蓝 */
  transition: color 0.15s ease;
}

/* tooltip - 纯 CSS,200ms 延迟避免划过闪烁;主题色自动反转。
   2026-07-11 v6 改:tooltip 仅作用于底部状态组(.footer-icon-btn / .footer-status),
   .nav-item 不再需要(已有永久 label)。 */
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
/* 侧栏底部 tooltip - 浮在图标右侧 */
.footer-icon-btn::after,
.footer-status::after {
  left: calc(100% + 12px);
  top: 50%;
  transform: translateY(-50%);
}
.footer-icon-btn:hover::after,
.footer-icon-btn:focus-visible::after,
.footer-status:hover::after,
.footer-status:focus-visible::after {
  opacity: 1;
  transition-delay: 0.2s;
}

/* ============================================
   侧栏底部状态组 - 2026-07-11 增:与导航同一栏,纵向排列;
   2026-07-11 v7 改:三个图标改成同一行水平排列,尺寸缩小,
   去掉外框,只在 hover 时显示浅背景。
   ============================================ */
.sidebar-footer {
  margin-top: auto;          /* 关键:把状态组推到侧栏底部 */
  padding: 6px 4px;
  display: flex;
  flex-direction: row;        /* v7:水平排列 */
  align-items: center;
  justify-content: center;    /* v7:三个图标整体居中 */
  gap: 2px;                   /* v7:收紧间距 */
  border-top: 1px solid var(--border-sidebar);
}

.footer-status {
  position: relative;       /* tooltip 锚点 */
  @apply rounded-md flex items-center justify-center;
  width: 24px;
  height: 24px;
  /* v7:去除外框,只在 hover 时显示背景 */
  border: none;
  background: transparent;
  cursor: default;
}
.footer-status:hover {
  background: var(--bg-sidebar-hover);
}
.footer-status-dot {
  @apply w-2 h-2 rounded-full;   /* v7:圆点从 2.5 缩到 2 */
}
.footer-status .dot-ok {
  background: var(--success);
  box-shadow: 0 0 0 2px rgba(21, 128, 61, 0.18);
}
.footer-status .dot-error {
  background: var(--danger);
  box-shadow: 0 0 0 2px rgba(185, 28, 28, 0.18);
}

.footer-icon-btn {
  position: relative;       /* tooltip 锚点 */
  @apply rounded-md flex items-center justify-center;
  width: 24px;
  height: 24px;
  /* v7:去除外框,缩小到 14px 图标 */
  border: none;
  background: transparent;
  color: var(--text-sidebar-muted);
  cursor: pointer;
  transition: color 0.15s ease, background-color 0.15s ease;
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
  @apply flex flex-col min-w-0;
  flex: 1;                   /* body-row 内的 flex 子项,撑开剩余空间 */
}
/* 2026-07-12 改:桌面端 sidebar 是 fixed 不占位,
   这里主动加 64px 左侧 padding 让 main-content 不被侧栏盖住。 */
.main-content-desktop {
  padding-left: 64px;
}

/* 顶部栏 - 2026-07-12 改:提到 app-container 第一行,横穿整个屏幕宽度。
   原 .main-content 内部 sticky 改 app-container 顶部 sticky 即可。
   高度由内容撑起,在 .topbar 上声明 --topbar-h 给 sidebar fixed top 引用,
   macOS 桌面端用更具体的 .topbar.mac-desktop 选择器把 --topbar-h 改成 98px
   (48 + 50 让位红绿灯),sidebar 会自动跟着 mac 模式对齐,无需 JS 改值。
   2026-07-12 v2 改:macOS 桌面端顶栏内容不需要 padding-top 50px 让位
   (用户反馈会换行/视觉错位),只保留水平 padding-left: 80px 让位红绿灯。
   macOS 模式下 --topbar-h 仍保持 48px(不撑高顶栏),红绿灯浮在 webview
   顶部 0~50px / 0~80px 区域,只是 logo 文字水平方向避开它。
   2026-07-12 改:z-index 从 20 提到 45,确保与 sidebar(z-35)协调 —
   顶栏在侧栏之上,横穿屏幕时 logo 文字浮在 sidebar 背景之上。 */
.topbar {
  @apply flex items-center justify-between py-0;
  padding-left: 20px;        /* 默认左侧留白(原 px-5) */
  padding-right: 20px;       /* 默认右侧留白(原 px-5) */
  --topbar-h: 48px;          /* 给 sidebar 的 top 引用 */
  height: var(--topbar-h);
  min-height: 48px;
  background: var(--bg-header);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 45;
  flex-shrink: 0;
  transition: all 0.3s ease;
}
.topbar.mac-desktop {
  padding-left: 80px;        /* macOS 左侧让位红绿灯水平位置(0~80px) */
}
.topbar.mac-desktop .topbar-logo-img {
  /* 2026-07-13 改:.topbar.mac-desktop 的 padding-left: 80px 在某些布局下
     会被 .topbar-left 的 flex 布局吃掉,这里给 logo img 单独加 margin-left
     兜底,确保 logo 离红绿灯足够远(80px 红绿灯 + 16px 安全间距 = 96px)。 */
  margin-left: 16px;
}
.topbar.mac-desktop .topbar-logo-text {
  margin-top: 6px;          /* 把 logo 视觉中心下移,对齐红绿灯圆心(~25px) */
  margin-left: 12px;        /* logo 离红绿灯远一点,避免视觉粘连 */
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

/* Logo 区 - 2026-07-11 增:从侧栏挪到顶栏左侧;
   2026-07-11 v9 改:删掉图标,纯文字;去掉 gap / padding-right / margin-right /
   border-right(图标分隔用的样式不再需要)。 */
.topbar-logo {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;                  /* logo 图与文字的间距(2026-07-13 改:8→4,贴紧) */
}
.topbar-logo-img {
  /* 顶栏高 48px,文字 22px 行高;logo 图取 24px,视觉权重跟文字相当
     (略偏小,避免抢文字风头),透明 PNG 直接放,顶栏背景透出。 */
  width: 24px;
  height: 24px;
  display: block;
  user-select: none;
  -webkit-user-drag: none;
}
.topbar-logo-text {
  /* 2026-07-11 v10:字号 15→20,加粗到 700,作为主要视觉锚点
     2026-07-11 v11:艺术感强化 — Inter weight 800 + letter-spacing 2px 拉开
     (i18n 已配 'SKILL-BOX' 大写,text-transform:uppercase 双保险)+ 主色渐变
     + -webkit-background-clip:text + 微 text-shadow 形成「刻印」质感。
     2026-07-12 v12:回归 'SkillBox' 首字母大写,去掉横杠;删 text-transform,
     letter-spacing 收紧回 0.5px;其他艺术感(weight 800 / 渐变 / 微高光)保留。
     2026-07-12 v13 改:macOS 桌面端让 logo 文字与红绿灯水平对齐 —
     红绿灯圆心在 0~50px 区域垂直中心(~25px),logo 文字 22px 行高,
     加 margin-top: 6px 让文字视觉中心下移对齐红绿灯圆心;
     margin-left: 12px 让 logo 离红绿灯远一点(图里挨太近)。 */
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.5px;
  line-height: 1;              /* 锁住行高,让 margin-top 精准控制位置 */
  color: var(--text);
  background: linear-gradient(135deg, var(--text) 0%, var(--text-dim) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.04);  /* 暗色模式柔高光 */
}
/* v11 增:暗黑模式下渐变两端都用亮色,确保可读性;
   text-shadow 反向为柔光,营造「发光字体」质感。 */
:global(html.dark) .topbar-logo-text {
  background: linear-gradient(135deg, #fafafa 0%, #a3a3a3 100%);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 0 12px rgba(250, 250, 250, 0.18);
}

.topbar-right {
  @apply flex items-center gap-2;
  flex-shrink: 0;
  flex-wrap: nowrap;          /* 顶栏高度 48px,3 个 badge 一行,不允许换行 */
  white-space: nowrap;
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

/* 响应式调整 - 2026-07-12 改:移动端 sidebar 是抽屉化 fixed,
   顶栏在 z-25 之上 0~48px 始终可见,移动端 main-content 不需要
   64px 左侧 padding(抽屉盖住 main-content 但不挡顶栏)。 */
@media (max-width: 768px) {
  .sidebar {
    width: 64px;
  }
  .topbar {
    padding: 10px 12px;
  }
  /* v9 改:纯文字 logo 移动端保留文字显示(就一个 "Skill-Box" 字串,
     不需要像之前带图标时那样隐藏) */
  .content-area {
    padding: 16px;
  }
  .main-content-desktop {
    padding-left: 0;
  }
}
</style>
