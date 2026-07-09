<script setup>
// MarketView.vue - 三方市场(2026-07-09 改:iframe 嵌入外部站点)。
//
// 历史:2026-06 ~ 2026-07 期间,走自建后端代理三方源(skillhub.cn / skills.sh),
// 含卡片网格 + 搜索 + 拉取弹窗 + 源设置 + 详情弹窗,后端有 cmarket / smarket /
// skillmarket 完整链路(详见 api-server/internal/skillmarket),前端有
// useMarketStore + MarketPullConfirm + MarketSourceSettings。
// 改 iframe 之后前端这层全成冗余,已删除:
//   - frontend/src/core/store/market.js
//   - frontend/src/api/skillbox/market.js
//   - frontend/src/components/MarketPullConfirm.vue
//   - frontend/src/components/MarketSourceSettings.vue
// 后端 cmarket / smarket / skillmarket 模块未删(独立模块,后续按需清理),
// 暂时成为 dead code,但不影响前端 build。
//
// 当前形态(2026-07-09):
//   - 顶部 tab:两个固定源(SkillHub / Skills.sh),点哪个切哪个 iframe
//   - 主体:<iframe> 走 /api/skillbox/market-iframe-proxy/<site>/* 反代(后端
//     cmarket.iframe_proxy 抹掉 X-Frame-Options / CSP,让 iframe 能加载三方站)
//   - 加载/失败占位:iframe onLoad 前显示加载中,onError / 15s 超时显示降级提示
//   - 「在浏览器中打开」按钮走 originUrl 直连三方站,绕开代理


import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { platform } from '@/platform'

const { t } = useI18n()

// 固定源列表(2026-07-09 改:不再走后端 sources,直接硬编码两个 iframe 目标 URL)
// 顺序就是 tab 顺序。
//
// url 字段:不再直接写三方站点 URL(浏览器会被 X-Frame-Options: SAMEORIGIN 拒),
// 而是走后端 /api/skillbox/market-iframe-proxy/<site>/* 反代(由 cmarket.iframe_proxy
// 提供,该 handler 会抹掉 X-Frame-Options / Content-Security-Policy)。
//
// 原始 URL 用 `originUrl` 字段存,在「在浏览器中打开」按钮里调 platform.openExternal
// 时直接用原始 URL,绕开代理。
const sources = [
  {
    id: 'skillhub',
    name: 'SkillHub',
    // iframe 走的代理 URL(同源 → 不被 X-Frame-Options 拒)
    url: '/api/skillbox/market-iframe-proxy/skillhub/skills?sortBy=curated_score',
    // 「在浏览器中打开」用原始 URL(直连 skillhub,不再走代理)
    originUrl: 'https://skillhub.cn/skills?sortBy=curated_score',
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    url: '/api/skillbox/market-iframe-proxy/skillssh/hot',
    originUrl: 'https://www.skills.sh/hot',
  },
]

const activeSourceId = ref(sources[0].id)
const activeSource = computed(
  () => sources.find((s) => s.id === activeSourceId.value) || sources[0]
)

// iframe 加载态:
//   - loading=true:切换 tab 或首次进入,iframe 还没 onLoad
//   - error=true:iframe onError 触发,或加载超时(15s)后还没 onLoad
// 这两个 flag 互斥,error 优先于 loading。
const loading = ref(true)
const error = ref(false)
let loadTimer = null

function resetLoadState() {
  loading.value = true
  error.value = false
  // 15s 兜底:onError 某些情况下不会触发(被同源代理拦截就静默失败),
  // 所以挂个定时器,15s 后还没 onLoad 就当失败处理。
  if (loadTimer) clearTimeout(loadTimer)
  loadTimer = setTimeout(() => {
    if (loading.value) {
      loading.value = false
      error.value = true
    }
  }, 15_000)
}

function onIframeLoad() {
  if (loadTimer) {
    clearTimeout(loadTimer)
    loadTimer = null
  }
  loading.value = false
  error.value = false
}

function onIframeError() {
  if (loadTimer) {
    clearTimeout(loadTimer)
    loadTimer = null
  }
  loading.value = false
  error.value = true
}

// 切源:重置加载态,iframe src 由 v-if 卸载 → 重建触发重新加载
function selectSource(id) {
  if (id === activeSourceId.value) return
  activeSourceId.value = id
}

// 「在新窗口打开」 — 走 platform.openExternal 跨平台:
//   - Web:window.open
//   - Desktop:wails BrowserOpenURL(系统默认浏览器)
async function openInExternal(url) {
  try {
    await platform.platform.openExternal(url)
  } catch (e) {
    // web 端 window.open 被拦截也算异常,这里静默吞掉,UI 已有提示
  }
}

function reload() {
  // 切源走 v-if 重建,同源刷新就强制重建一次:用 src 引用做 key,先切到一个空 src 再切回
  const cur = activeSourceId.value
  activeSourceId.value = '__none__'
  // 下一 tick 切回去,触发 v-if 重建
  setTimeout(() => {
    activeSourceId.value = cur
    resetLoadState()
  }, 30)
}

// 切源时重置加载态
watch(activeSourceId, () => {
  resetLoadState()
})

onMounted(() => {
  resetLoadState()
})
</script>

<template>
  <div class="market">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-market">
          <IconPark icon="mdi:cart-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('market.title') }}</h1>
          <p>{{ t('market.subtitle') }}</p>
        </div>
      </div>
    </header>

    <div class="card">
      <!-- 顶部源 tab(2026-07-09 改:从动态 sources 简化为两个固定 tab) -->
      <nav class="source-tabs">
        <button
          v-for="s in sources"
          :key="s.id"
          :class="['source-tab', { active: s.id === activeSourceId }]"
          @click="selectSource(s.id)"
        >
          <IconPark icon="mdi:radio-tower" width="14" height="14" />
          {{ s.name }}
        </button>
        <span class="source-tabs-spacer" />
        <button
          v-if="!error"
          class="ghost source-reload"
          :title="t('market.btnReload')"
          @click="reload"
        >
          <IconPark icon="mdi:refresh" width="14" height="14" />
        </button>
        <button
          class="ghost source-open"
          :title="t('market.btnOpenInBrowser')"
          @click="openInExternal(activeSource.originUrl)"
        >
          <IconPark icon="mdi:open-in-new" width="14" height="14" />
          {{ t('market.btnOpenInBrowser') }}
        </button>
      </nav>

      <!-- iframe 主体 — key 用 activeSourceId 强制重建,切换源时 src 重新加载 -->
      <div class="market-frame-wrap">
        <iframe
          v-if="activeSourceId !== '__none__'"
          :key="activeSourceId"
          :src="activeSource.url"
          class="market-frame"
          :title="activeSource.name"
          referrerpolicy="no-referrer"
          allow="clipboard-read; clipboard-write"
          @load="onIframeLoad"
          @error="onIframeError"
        />

        <!-- 加载占位 -->
        <div v-if="loading && !error" class="frame-state frame-state-loading">
          <IconPark icon="mdi:loading" width="36" height="36" class="loading-icon" />
          <p class="frame-state-text">{{ t('market.iframe.loading', { source: activeSource.name }) }}</p>
        </div>

        <!-- 失败占位(2026-07-09 增:第三方站点 X-Frame-Options 限制时降级) -->
        <div v-if="error" class="frame-state frame-state-error">
          <IconPark icon="mdi:cloud-off-outline" width="48" height="48" class="error-icon" />
          <p class="frame-state-title">{{ t('market.iframe.blockedTitle') }}</p>
          <p class="frame-state-hint">{{ t('market.iframe.blockedHint', { source: activeSource.name }) }}</p>
          <div class="frame-state-actions">
            <button type="button" class="primary" @click="openInExternal(activeSource.originUrl)">
              <IconPark icon="mdi:open-in-new" width="14" height="14" />
              {{ t('market.btnOpenInBrowser') }}
            </button>
            <button type="button" class="ghost" @click="reload">
              <IconPark icon="mdi:refresh" width="14" height="14" />
              {{ t('market.btnReload') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ============================================
   市场主题色 - Ocean Teal Market(2026-07-09 沿用)
   亮色: sky-500 / cyan-500
   暗色: sky-400 / cyan-400
   ============================================ */
.market {
  --mkt-primary: #0ea5e9;
  --mkt-primary-hover: #0284c7;
  --mkt-accent: #06b6d4;
  --mkt-bg: #f0f9ff;
  --mkt-bg-strong: #e0f2fe;
  --mkt-border: #bae6fd;
  --mkt-text: #0369a1;

  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  color: var(--text);
  transition: color 0.3s ease;
}

:global(html.dark) .market {
  --mkt-primary: #38bdf8;
  --mkt-primary-hover: #7dd3fc;
  --mkt-accent: #22d3ee;
  --mkt-bg: #082f49;
  --mkt-bg-strong: #0c4a6e;
  --mkt-border: #0369a1;
  --mkt-text: #bae6fd;
}

/* 页面头部 */
.view-header {
  margin-bottom: 24px;
  flex-shrink: 0;
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
.view-icon-market {
  background: linear-gradient(135deg, var(--mkt-primary) 0%, var(--mkt-accent) 100%);
  color: #ffffff;
  box-shadow: 0 2px 8px -2px color-mix(in srgb, var(--mkt-primary) 40%, transparent);
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

/* 卡片容器 */
.card {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  padding: 20px;
  transition: all 0.3s ease;
}

/* 顶部源 tab(2026-07-09 改:右侧新增 reload + open 按钮) */
.source-tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
  flex-shrink: 0;
}
.source-tabs-spacer {
  flex: 1;
}
.source-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.15s ease;
}
.source-tab:hover:not(.active) {
  background: var(--mkt-bg);
  border-color: var(--mkt-border);
  color: var(--mkt-text);
}
.source-tab.active {
  background: linear-gradient(135deg, var(--mkt-primary) 0%, var(--mkt-accent) 100%);
  border-color: transparent;
  color: #ffffff;
  box-shadow: 0 2px 6px -2px color-mix(in srgb, var(--mkt-primary) 50%, transparent);
}

/* 右侧 reload / open 按钮(覆盖全局 ghost 透明底,常驻可见) */
.source-reload,
.source-open {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-card);
  border-color: var(--border);
}
.source-reload:hover:not(:disabled),
.source-open:hover:not(:disabled) {
  background: var(--bg-hover);
  border-color: var(--text-faint);
}

/* iframe 主体区:flex:1 接管 .card 剩余高度,内部 iframe 100% 撑满;
   position:relative 让加载/失败占位可以 absolute 居中覆盖 */
.market-frame-wrap {
  position: relative;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--bg-subtle);
}
.market-frame {
  width: 100%;
  height: 100%;
  border: 0;
  display: block;
  background: var(--bg-card);
}

/* 加载 / 失败占位 — 覆盖在 iframe 上方,absolute 居中 */
.frame-state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  text-align: center;
  padding: 24px;
  background: var(--bg-subtle);
  pointer-events: auto;
  z-index: 1;
}
.frame-state-text,
.frame-state-title,
.frame-state-hint {
  margin: 0;
}
.frame-state-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  margin-top: 6px;
}
.frame-state-hint {
  font-size: 13px;
  color: var(--text-dim);
  max-width: 420px;
  line-height: 1.6;
}
.frame-state-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}
.frame-state-loading .loading-icon {
  color: var(--mkt-primary);
  animation: spin 1s linear infinite;
}
.frame-state-error .error-icon {
  color: var(--text-faint);
}
.frame-state-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  flex-wrap: wrap;
  justify-content: center;
}
.frame-state-actions .primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.frame-state-actions .ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* 响应式 */
@media (max-width: 768px) {
  .source-tabs {
    flex-wrap: wrap;
  }
  .source-tabs-spacer {
    display: none;
  }
  .source-reload,
  .source-open {
    flex: 1;
    justify-content: center;
  }
}
</style>
