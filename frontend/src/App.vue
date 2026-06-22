<script setup>
import { ref, onMounted, onUnmounted, provide, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import ProjectsView from './views/ProjectsView.vue'
import SkillsView from './views/SkillsView.vue'
import MarketView from './views/MarketView.vue'
import OnboardingView from './views/OnboardingView.vue'
import AuditView from './views/AuditView.vue'
import { listSkills } from '@/api/skillbox/skills'
import { listProjects } from '@/api/skillbox/projects'
import { getOnboardingStatus } from '@/api/skillbox/onboarding'

const { t } = useI18n()

const tab = ref('skills')

// 轻量事件总线:Onboarding 完成页"去 Skills 页查看"等跨组件跳转走这里。
// 走 provide/inject 避免引第三方库,API 形似 mitt:`bus.on(name, fn)` / `bus.off` / `bus.emit`。
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

// 响应式:md 以下(768px)侧栏变抽屉
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

// 侧栏配置(label/desc 由 i18n 在模板里现取,这里只放图标与 key)
const navItems = computed(() => [
  { key: 'skills',      label: t('app.nav.skills.label'),      desc: t('app.nav.skills.desc'),      icon: 'mdi:book-open-variant' },
  { key: 'projects',    label: t('app.nav.projects.label'),    desc: t('app.nav.projects.desc'),    icon: 'mdi:folder-multiple-outline' },
  { key: 'market',      label: t('app.nav.market.label'),      desc: t('app.nav.market.desc'),      icon: 'mdi:cart-outline' },
  { key: 'onboarding',  label: t('app.nav.onboarding.label'),  desc: t('app.nav.onboarding.desc'),  icon: 'mdi:compass-outline' },
  { key: 'audit',       label: t('app.nav.audit.label'),       desc: t('app.nav.audit.desc'),       icon: 'mdi:script-text-outline' },
])

function switchTab(k) {
  tab.value = k
  if (k === 'onboarding' || k === 'audit' || k === 'skills') refreshStats()
  // 移动端切 tab 后自动收起侧栏
  if (isMobile.value) sidebarOpen.value = false
}

// 跨组件跳转入口:子组件 emit('switch-tab', 'skills') → 这里接管走 switchTab
function onBusEvent(name, payload) {
  if (name === 'switch-tab') {
    switchTab(payload)
  }
}
// 兜底:也监听 window CustomEvent,eventBus.inject 失败时仍能跳转。
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
  <div class="flex min-h-screen bg-sb-bg text-sb-text">
    <!-- 移动端遮罩 -->
    <div
      v-if="isMobile && sidebarOpen"
      class="fixed inset-0 bg-black/40 z-30 md:hidden"
      @click="sidebarOpen = false"
    ></div>

    <!-- 侧栏 -->
    <aside
      :class="[
        'flex flex-col bg-sb-sidebar text-gray-300 z-40',
        'transition-transform duration-200 ease-out',
        isMobile
          ? (sidebarOpen ? 'fixed inset-y-0 left-0 translate-x-0 w-64' : 'fixed inset-y-0 left-0 -translate-x-full w-64')
          : 'sticky top-0 h-screen w-60',
      ]"
    >
      <!-- Brand -->
      <div class="flex items-center gap-3 px-4 py-5 border-b border-sb-sidebar-border">
        <div class="w-9 h-9 rounded-lg flex items-center justify-center text-xl bg-gradient-to-br from-sb-primary to-purple-600 text-white">
          <Icon icon="mdi:package-variant-closed" width="22" height="22" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-white font-semibold text-[15px] leading-tight">{{ t('app.brand') }}</div>
          <div class="text-sb-sidebar-muted text-[11px] truncate">{{ t('app.tagline') }}</div>
        </div>
        <button
          v-if="isMobile"
          class="text-sb-sidebar-muted hover:text-white text-lg p-1 flex items-center justify-center"
          @click="sidebarOpen = false"
          :aria-label="t('app.closeSidebar')"
        >
          <Icon icon="mdi:close" width="18" height="18" />
        </button>
      </div>

      <!-- Nav -->
      <nav class="flex-1 px-2 py-2.5 space-y-0.5 overflow-y-auto">
        <button
          v-for="n in navItems"
          :key="n.key"
          :class="[
            'w-full flex items-center gap-3 px-3 py-2.5 rounded text-left',
            'transition-colors duration-150',
            tab === n.key
              ? 'bg-sb-sidebar-active text-white'
              : 'text-gray-300 hover:bg-sb-sidebar-hover',
          ]"
          @click="switchTab(n.key)"
        >
          <span class="text-lg shrink-0 leading-none flex items-center justify-center">
            <Icon :icon="n.icon" width="18" height="18" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block text-[13px] font-medium leading-tight">{{ n.label }}</span>
            <span :class="['block text-[11px] leading-tight mt-0.5 truncate', tab === n.key ? 'text-indigo-200' : 'text-sb-sidebar-muted']">
              {{ n.desc }}
            </span>
          </span>
        </button>
      </nav>

      <!-- Foot:health + refresh -->
      <div class="px-4 py-3 border-t border-sb-sidebar-border flex items-center gap-2">
        <div :class="['flex items-center gap-1.5 flex-1 min-w-0', backendOK ? 'text-emerald-400' : 'text-red-400']">
          <span :class="['w-2 h-2 rounded-full shrink-0', backendOK ? 'bg-emerald-400 shadow-[0_0_0_2px_rgba(52,211,153,0.2)]' : 'bg-red-400 shadow-[0_0_0_2px_rgba(248,113,113,0.2)]']"></span>
          <span class="text-[11px] truncate">
            {{ backendOK ? '后端已连接' : '后端断开' }}
            <span v-if="lastHealth" class="text-sb-sidebar-muted">· {{ lastHealth }}</span>
          </span>
        </div>
        <button
          class="w-7 h-7 rounded border border-sb-sidebar-border bg-transparent text-gray-300 hover:bg-sb-sidebar-hover hover:border-gray-500 flex items-center justify-center"
          @click="refreshStats"
          :title="t('app.refreshStats')"
        >
          <Icon icon="mdi:refresh" width="14" height="14" />
        </button>
      </div>
    </aside>

    <!-- 主区 -->
    <main class="flex-1 flex flex-col min-w-0">
      <!-- Topbar -->
      <header class="bg-sb-card border-b border-sb-border px-4 md:px-5 py-2.5 flex items-center justify-between gap-4 sticky top-0 z-20">
        <div class="flex items-center gap-2 text-[13px] min-w-0">
          <button
            v-if="isMobile"
            class="p-1.5 -ml-1.5 rounded text-sb-dim hover:bg-gray-100"
            @click="sidebarOpen = true"
            aria-label="打开侧栏"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <line x1="3" y1="6" x2="21" y2="6"/>
              <line x1="3" y1="12" x2="21" y2="12"/>
              <line x1="3" y1="18" x2="21" y2="18"/>
            </svg>
          </button>
          <span class="bg-gradient-to-br from-sb-primary to-purple-600 text-white px-2 py-0.5 rounded-full text-[11px] font-medium">Skill Box</span>
          <span class="text-sb-faint">/</span>
          <span class="text-sb-text font-medium truncate">{{ navItems.find((x) => x.key === tab)?.label }}</span>
        </div>
        <div class="flex gap-2 flex-wrap justify-end">
          <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 bg-gray-100 rounded-full text-[12px] text-sb-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-sb-primary"></span>Skills <b class="text-sb-text font-semibold ml-0.5">{{ stats.skills }}</b>
          </span>
          <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 bg-gray-100 rounded-full text-[12px] text-sb-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-purple-600"></span>Projects <b class="text-sb-text font-semibold ml-0.5">{{ stats.projects }}</b>
          </span>
          <span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 bg-gray-100 rounded-full text-[12px] text-sb-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-sb-success"></span>Tools <b class="text-sb-text font-semibold ml-0.5">{{ stats.toolsReady }} / {{ stats.toolsTotal }}</b>
          </span>
        </div>
      </header>

      <div class="flex-1 p-4 md:p-5 overflow-auto">
        <ProjectsView v-if="tab === 'projects'" />
        <SkillsView v-else-if="tab === 'skills'" />
        <MarketView v-else-if="tab === 'market'" />
        <OnboardingView v-else-if="tab === 'onboarding'" />
        <AuditView v-else />
      </div>
    </main>
  </div>
</template>
