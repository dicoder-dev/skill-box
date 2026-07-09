<script setup>
// MarketView.vue - 三方市场(2026-07-09 改:卡片 + 跳浏览器方案)。
//
// 历经三个方案:
//   v1(2026-06 ~ 2026-07):自建后端代理三方源(skillhub / skills.sh),前端
//     卡片网格 + 拉取弹窗。已被 iframe 方案替代,前端 store/api/弹窗
//     全部删除(后端 cmarket 模块保留为 dead code,后续按需清理)。
//
//   v2(2026-07-09 上午):iframe + 后端 reverse proxy,抹掉 X-Frame-Options
//     / CSP,让 iframe 能加载三方站。问题:
//       - skillhub 站点本身不允许跨源嵌入(CORS 拒 iframe 内部 API 调用)
//       - 桌面 webview 里 Vercel dpl cookie 验证不通过(已用 <base href> 注入修复)
//       - 即使 HTML 渲染,API 调不通,内容始终空白
//
//   v3(2026-07-09 当前):纯前端卡片 + 跳浏览器。
//     - 顶 tab 切换 SkillHub / Skills.sh
//     - 主体显示当前 tab 对应的站点介绍卡片(名称 + 描述 + 「在浏览器中打开」)
//     - 点按钮调 platform.platform.openExternal 跨平台打开外部 URL:
//         Web:window.open / Desktop:wails BrowserOpenURL(系统默认浏览器)
//     - 不依赖任何 iframe / 代理,稳如老狗

import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { platform } from '@/platform'

const { t } = useI18n()

// 固定源列表(顺序就是 tab 顺序)
// 字段:
//   id      - tab 唯一 key
//   name    - tab 显示名 + 卡片标题
//   desc    - 卡片描述(i18n key,运行时 t() 解析)
//   url     - 「在浏览器中打开」按钮的目标 URL
//   accent  - 卡片主色(亮色 hex),与市场海蓝主题色区分,让两个卡片有视觉差异
const sources = [
  {
    id: 'skillhub',
    name: 'SkillHub',
    descKey: 'market.cards.skillhubDesc',
    url: 'https://skillhub.cn/skills?sortBy=curated_score',
    accent: '#0ea5e9',
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    descKey: 'market.cards.skillsshDesc',
    url: 'https://www.skills.sh/hot',
    accent: '#8b5cf6',
  },
]

const activeSourceId = ref(sources[0].id)
const activeSource = computed(
  () => sources.find((s) => s.id === activeSourceId.value) || sources[0]
)

function selectSource(id) {
  if (id === activeSourceId.value) return
  activeSourceId.value = id
}

// 「在浏览器中打开」 — 跨平台
//   Web:window.open
//   Desktop:wails BrowserOpenURL(系统默认浏览器)
async function openInExternal(url) {
  try {
    await platform.platform.openExternal(url)
  } catch (e) {
    // web 端 window.open 被拦截也算异常,这里静默吞掉
  }
}
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
      <!-- 顶部源 tab(顺序:SkillHub / Skills.sh) -->
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
      </nav>

      <!-- 主体:当前 tab 对应的站点介绍卡片 -->
      <div class="market-body">
        <div
          :key="activeSource.id"
          class="source-card"
          :style="{ '--accent': activeSource.accent }"
        >
          <div class="source-card-head">
            <div class="source-card-icon">
              <IconPark icon="mdi:open-in-new" width="28" height="28" />
            </div>
            <div class="source-card-titles">
              <h2 class="source-card-name">{{ activeSource.name }}</h2>
              <p class="source-card-url">{{ activeSource.url }}</p>
            </div>
          </div>

          <p class="source-card-desc">
            {{ t(activeSource.descKey) }}
          </p>

          <div class="source-card-actions">
            <button type="button" class="primary" @click="openInExternal(activeSource.url)">
              <IconPark icon="mdi:open-in-new" width="14" height="14" />
              {{ t('market.btnOpenInBrowser') }}
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

/* 顶部源 tab */
.source-tabs {
  display: flex;
  gap: 6px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
  flex-shrink: 0;
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

/* 主体:flex:1 接管 .card 剩余高度,内部居中放站点卡片 */
.market-body {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 0;
}

/* 站点介绍卡片(2026-07-09 改:替代 iframe) */
.source-card {
  --accent: #0ea5e9;
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: min(520px, 100%);
  padding: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
  /* 顶部 4px accent 边条,跟原卡片 left bar 风格统一 */
  border-top: 4px solid var(--accent);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.source-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px -2px color-mix(in srgb, var(--accent) 30%, transparent);
}

.source-card-head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.source-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--accent) 15%, var(--bg-card));
  color: var(--accent);
  border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border));
  flex-shrink: 0;
}
.source-card-titles {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.source-card-name {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
}
.source-card-url {
  margin: 0;
  font-size: 12px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.source-card-desc {
  margin: 0;
  font-size: 14px;
  color: var(--text-dim);
  line-height: 1.6;
}
.source-card-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}
.source-card-actions .primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  /* 用 inline style 注入的 --accent 作为按钮主色,跟卡片边条一致 */
  background: var(--accent);
  border-color: var(--accent);
  color: #ffffff;
}
.source-card-actions .primary:hover:not(:disabled) {
  filter: brightness(0.92);
  transform: translateY(-1px);
}

/* 响应式 */
@media (max-width: 768px) {
  .source-card {
    padding: 18px;
  }
  .source-card-head {
    gap: 10px;
  }
  .source-card-icon {
    width: 40px;
    height: 40px;
  }
  .source-card-name {
    font-size: 16px;
  }
}
</style>
