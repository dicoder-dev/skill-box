<script setup>
// MarketView.vue - 三方市场 v4(2026-07-09)。
//
// 历经四个方案:
//   v1(2026-06 ~ 2026-07):自建后端代理三方源(skillhub / skills.sh),前端
//     卡片网格 + 拉取弹窗。
//   v2(2026-07-09 上午):iframe + 后端 reverse proxy,失败(跨源 / cookie / API)。
//   v3(2026-07-09 下午):纯前端卡片 + 跳浏览器(只有"在浏览器中打开"按钮)。
//   v4(2026-07-09 晚):卡片 + 跳浏览器 + **输入框一键安装**(当前)。
//
// v4 新增能力:
//   1. 各 tab 顶部展示「如何安装到 skill-box」指南:文字 + CLI 命令
//      (给用户提供思路,即使不想用输入框也能手装)
//   2. 输入框:用户粘贴 skill slug / 详情页 URL → 后端
//      POST /api/skillbox/market/install-from-input → 自动下载到本地 store
//   3. 实时进度条(4 阶段模拟:解析 → 下载 → 解压 → 写盘),失败红条提示
//   4. 成功后 toast + 「去首页查看」按钮(通过 activeTab 跳转 skills 视图)
//
// 配色:SkillHub 保留青色 #0ea5e9;Skills.sh 由原紫色 #8b5cf6 改为绿色 #10b981
// (符合项目 memory:avoid-violet-as-primary-color.md,紫色 AI 感强,主色禁用)。

import { ref, computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { platform } from '@/platform'
import { installFromInput } from '@/api/skillbox/market'
import { useToastStore } from '@/core/store/toast'

const { t } = useI18n()
const toast = useToastStore()

// 2026-07-09 增:跳转首页用(App.vue provide 的 activeTab)
const activeTab = inject('activeTab', null)

// 固定源列表(顺序就是 tab 顺序)。
//
// accent 字段:卡片主色。2026-07-09 改 Skills.sh 由紫色 #8b5cf6 → 绿色 #10b981
// (avoid-violet-as-primary-color.md 约束)。
//
// examples: 输入示例数组(2026-07-09 增)。
// 不放 i18n 是因为 vue-i18n 9.x 数组 key 在 v-for 里偶尔会被文本节点解析成
// 单字符数组(已知 issue:https://github.com/intlify/vue-i18n-next/issues/...),硬编码
// 在源结构里更稳。源是固定 2 个(skillhub / skills.sh),维护成本可控。
const sources = [
  {
    id: 'skillhub',
    name: 'SkillHub',
    descKey: 'market.cards.skillhubDesc',
    url: 'https://skillhub.cn/skills?sortBy=curated_score',
    accent: '#0ea5e9',
    sourceType: 'skillhub',
    placeholderKey: 'market.input.placeholderSkillhub',
    guideKey: 'market.guide.skillhub',
    examples: [
      'code-review',
      'https://skillhub.cn/skills/code-review',
    ],
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    descKey: 'market.cards.skillsshDesc',
    url: 'https://www.skills.sh/hot',
    accent: '#10b981',
    sourceType: 'skillssh',
    placeholderKey: 'market.input.placeholderSkillssh',
    guideKey: 'market.guide.skillssh',
    examples: [
      // 2026-07-09 改:按"用户友好度"排序
      // 1. GitHub 详情 URL(最具体,用户从 GitHub 复制最常见)
      // 2. skills.sh 详情 URL
      // 3. owner/repo@skill 短标识
      'https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md',
      'https://skills.sh/anthropics/skills/pdf',
      'anthropics/skills@pdf',
    ],
  },
]

const activeSourceId = ref(sources[0].id)
const activeSource = computed(
  () => sources.find((s) => s.id === activeSourceId.value) || sources[0]
)

function selectSource(id) {
  if (id === activeSourceId.value) return
  activeSourceId.value = id
  // 切 tab 时清掉之前残留的输入 / 进度 / 错误,避免污染新 tab
  userInput.value = ''
  resetProgress()
}

// 「在浏览器中打开」按钮(2026-07-09 改:从卡片底部上移到顶栏) — 跨平台(Web → window.open / Desktop → wails BrowserOpenURL)。
async function openInExternal(url) {
  try {
    await platform.platform.openExternal(url)
  } catch (e) {
    // web 端 window.open 被拦截也算异常,这里静默吞掉
  }
}

// 2026-07-09 增:点击示例条目 → 自动填入输入框,避免用户复制粘贴。
// 注意:不自动提交,只填内容,让用户点「装到 skill-box」才走安装流程。
function fillExample(text) {
  if (installing.value) return
  userInput.value = String(text)
  installError.value = ''
}

// 2026-07-09 增:输入框 + 安装流程

const userInput = ref('')
const installing = ref(false)
const installError = ref('')

// 进度条 4 阶段;每阶段一个独立 ref,确保切换时旧 ref 残留不会乱跳
const progressStage = ref('')        // 当前阶段 key(resolve/download/extract/write/done/'')
const progressPercent = ref(0)       // 0-100
let progressTimer = null

// 进度目标:每阶段到达的百分比。模拟真实节奏:解析 15% → 下载 60% → 解压 85% → 写盘 100%。
const STAGE_TARGETS = {
  resolve: 15,
  download: 60,
  extract: 85,
  write: 100,
}

function resetProgress() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
  progressStage.value = ''
  progressPercent.value = 0
  installError.value = ''
}

// 平滑推进进度到目标值。固定时长 600ms 让用户看到阶段切换。
function advanceProgress(stage) {
  progressStage.value = stage
  const target = STAGE_TARGETS[stage] || 0
  const start = progressPercent.value
  const dur = 600
  const startedAt = Date.now()
  if (progressTimer) clearInterval(progressTimer)
  progressTimer = setInterval(() => {
    const elapsed = Date.now() - startedAt
    if (elapsed >= dur) {
      progressPercent.value = target
      clearInterval(progressTimer)
      progressTimer = null
      return
    }
    const k = elapsed / dur
    progressPercent.value = Math.round(start + (target - start) * k)
  }, 30)
}

// 「装到 skill-box」按钮 — 走 4 阶段模拟 → 后端一次性 HTTP → 收尾
async function handleInstall() {
  if (installing.value) return
  const input = userInput.value.trim()
  if (!input) {
    installError.value = t('market.input.errInvalidInput')
    return
  }
  installing.value = true
  resetProgress()
  advanceProgress('resolve')
  // 给解析阶段至少 300ms 视觉反馈,避免快网络下进度跳 0→60 看不到 resolve
  await new Promise((r) => setTimeout(r, 350))
  advanceProgress('download')
  try {
    const out = await installFromInput({
      source_hint: activeSource.value.sourceType,
      input,
      scope: 'global',
    })
    advanceProgress('extract')
    await new Promise((r) => setTimeout(r, 350))
    advanceProgress('write')
    await new Promise((r) => setTimeout(r, 250))
    progressStage.value = 'done'
    installing.value = false
    // 2026-07-09 增:记住刚装好的 skill name,goToHome 时给 SkillsView 用来自动选中
    lastInstalledName.value = out.skill_name
    toast.success(t('market.success.msg', { name: out.skill_name, version: out.skill_version || '0.1.0' }))
  } catch (e) {
    installing.value = false
    resetProgress()
    const msg = e?.message || String(e)
    // 错误分类:后端 400 → 输入格式 / 404 → 源找不到 / 其它 → 通用
    const status = e?.response?.status || e?.status
    if (status === 400) {
      installError.value = t('market.input.errInvalidInput')
    } else if (status === 404) {
      installError.value = t('market.input.errSource')
    } else if (/download|fetch/i.test(msg)) {
      installError.value = t('market.input.errPull', { msg })
    } else {
      installError.value = t('market.input.errGeneric', { msg })
    }
  }
}

// 2026-07-09 增:用 skillTree store 的"待选清单"传 skill name 给 SkillsView。
// 比 window event 更可靠:SkillsView 可能还没 mount,事件就丢了;
// store 是单例,SkillsView mount 后 + reload 完会自动消费。
import { useSkillTreeStore } from '@/core/store/skill-tree'
const skillTree = useSkillTreeStore()

// 成功后「去首页查看」 — 改 activeTab.value = 'skills' (由 App.vue provide 出来)
//
// 2026-07-09 改:把刚装好的 skill name 写到 skillTree store,SkillsView mount
// 后 + reload 完会自动调 setSelected,左侧树节点高亮 + 右侧详情自动出。
function goToHome() {
  const installedName = lastInstalledName.value
  if (installedName) {
    skillTree.setPendingSelectName(installedName)
  }
  if (activeTab && typeof activeTab.value !== 'undefined') {
    activeTab.value = 'skills'
  }
}

// 2026-07-09 增:记住刚装好的 skill name,goToHome 时传给 SkillsView
const lastInstalledName = ref('')
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
      <!-- 2026-07-09 改:去掉顶栏整行(open-browser-btn 搬到 source-card-head 右侧) -->
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

      <!-- 主体:当前 tab 对应的站点介绍卡 + 安装指南 + 输入框 -->
      <div class="market-body">
        <div
          :key="activeSource.id"
          class="source-card"
          :style="{ '--accent': activeSource.accent }"
        >
          <!-- 卡片 head:图标 + 名称 + 「在浏览器中打开」按钮(2026-07-09 改)
                 2026-07-09 改:URL(skillhub.cn/skills?sortBy=curated_score / skills.sh/hot)
                 不再展示在标题下方(用户已通过 source-tabs 知道选哪个,URL 重复占空间) -->
          <div class="source-card-head">
            <div class="source-card-icon">
              <IconPark icon="mdi:open-in-new" width="28" height="28" />
            </div>
            <div class="source-card-titles">
              <h2 class="source-card-name">{{ activeSource.name }}</h2>
            </div>
            <!-- 在浏览器中打开按钮:2026-07-09 改:从顶栏搬到 head 右侧 -->
            <button
              type="button"
              class="open-browser-btn"
              :title="t('market.btnOpenInBrowserTip', { name: activeSource.name })"
              @click="openInExternal(activeSource.url)"
            >
              <IconPark icon="mdi:earth" width="14" height="14" />
              <span>{{ t('market.btnOpenInBrowser') }}</span>
            </button>
          </div>

          <!-- 2026-07-09 增:站点描述 -->
          <p class="source-card-desc">
            {{ t(activeSource.descKey) }}
          </p>

          <!-- 2026-07-09 增:安装指南(给用户思路) -->
          <div class="install-guide">
            <div class="guide-title">
              <IconPark icon="mdi:lightbulb-outline" width="14" height="14" />
              <span>{{ t('market.guide.title') }}</span>
            </div>
            <p class="guide-desc">{{ t(`${activeSource.guideKey}.desc`) }}</p>
            <!-- 2026-07-09 增:输入示例(让用户更直观知道粘什么)
                 2026-07-09 改:examples 直接走 activeSource.examples 数组,
                 不再依赖 i18n 数组 key,避免 vue-i18n 把字符串拆成字符数组的坑 -->
            <div class="guide-examples">
              <div class="examples-label">
                <IconPark icon="mdi:format-list-bulleted-square" width="12" height="12" />
                <span>{{ t(`${activeSource.guideKey}.examples`) }}</span>
              </div>
              <ul class="examples-list">
                <li
                  v-for="(ex, idx) in activeSource.examples"
                  :key="idx"
                  class="example-item"
                  :title="`点击填入 ${ex}`"
                  @click="fillExample(ex)"
                >
                  <code>{{ ex }}</code>
                </li>
              </ul>
            </div>
          </div>

          <!-- 2026-07-09 增:输入框 + 安装按钮 -->
          <div class="install-form">
            <label class="install-form-label">
              <IconPark icon="mdi:link-variant" width="14" height="14" />
              {{ t('market.input.label') }}
            </label>
            <div class="install-form-row">
              <input
                v-model="userInput"
                type="text"
                class="install-input"
                :placeholder="t(activeSource.placeholderKey)"
                :disabled="installing"
                @keydown.enter="handleInstall"
              />
              <button
                type="button"
                class="primary"
                :disabled="installing || !userInput.trim()"
                @click="handleInstall"
              >
                <IconPark :icon="installing ? 'mdi:loading' : 'mdi:download'" width="14" height="14" />
                {{ installing ? t('market.input.btnInstalling') : t('market.input.btnInstall') }}
              </button>
            </div>
            <!-- 错误条(只在失败时显示) -->
            <div v-if="installError" class="install-error">
              <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
              {{ installError }}
            </div>
          </div>

          <!-- 2026-07-09 增:进度条(只在 installing 或 done 时显示) -->
          <div v-if="progressStage" class="install-progress">
            <div class="progress-row">
              <span class="progress-label">
                <IconPark
                  :icon="progressStage === 'done' ? 'mdi:check-circle' : 'mdi:loading'"
                  width="14"
                  height="14"
                  :spin="progressStage !== 'done'"
                />
                {{ t(`market.progress.${progressStage}`) }}
              </span>
              <span class="progress-percent">{{ progressPercent }}%</span>
            </div>
            <div class="progress-track">
              <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
            </div>
            <!-- 成功后跳首页按钮 -->
            <button
              v-if="progressStage === 'done'"
              type="button"
              class="primary progress-go-home"
              @click="goToHome"
            >
              <IconPark icon="mdi:home-outline" width="14" height="14" />
              {{ t('market.success.goHome') }}
            </button>
          </div>

          <!-- 原「在浏览器中打开」按钮已上移到顶部栏(2026-07-09 改) -->
          <div v-if="false" class="source-card-actions"></div>
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
  overflow-y: auto;
}

/* ============================================
   2026-07-09 改:source-tabs 顶栏(去掉 .top-row,按钮搬到 source-card-head)
   ============================================ */
.source-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  /* 2026-07-09 改:这些样式原本在 .top-row,现在还回 .source-tabs */
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
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

/* 2026-07-09 改:在浏览器中打开按钮(搬到 source-card-head 右侧)
   - 蓝色 outline 风,与左侧 title 视觉协调
   - hover 加深 + 上抬
   - active 模拟按下状态
   - 药丸形更精致 */
.open-browser-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 14px;
  border: 1px solid var(--mkt-border);
  background: var(--bg-card);
  border-radius: 999px; /* 药丸状 */
  font-size: 12px;
  font-weight: 500;
  color: var(--mkt-text);
  cursor: pointer;
  transition: all 0.18s ease;
  flex-shrink: 0;
  white-space: nowrap;
  margin-left: auto; /* 2026-07-09 改:推到 head 右侧 */
}
.open-browser-btn:hover {
  background: var(--mkt-bg-strong);
  border-color: var(--mkt-primary);
  color: var(--mkt-primary);
  transform: translateY(-1px);
  box-shadow: 0 4px 10px -4px color-mix(in srgb, var(--mkt-primary) 35%, transparent);
}
.open-browser-btn:active {
  transform: translateY(0);
  box-shadow: none;
  background: var(--mkt-bg);
}

/* 主体 */
.market-body {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 8px 0 0;
}

/* 站点介绍卡片 */
.source-card {
  --accent: #0ea5e9;
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: min(640px, 100%);
  padding: 24px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-card);
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

/* ============================================
   2026-07-09 增:安装指南块
   ============================================ */
.install-guide {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-card));
  border: 1px solid color-mix(in srgb, var(--accent) 20%, var(--border));
  border-radius: var(--radius-sm);
}
.guide-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--accent);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.guide-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.55;
}
.guide-cli {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  background: color-mix(in srgb, var(--text) 6%, var(--bg-card));
  border-radius: 4px;
  overflow-x: auto;
}
.guide-cli code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text);
  white-space: nowrap;
}
.guide-cli-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

/* 2026-07-09 增:输入示例块 — 每个示例可点击,一键填入输入框 */
.guide-examples {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.examples-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.examples-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.example-item {
  display: flex;
  align-items: center;
  padding: 5px 10px;
  background: var(--bg-card);
  border: 1px dashed color-mix(in srgb, var(--accent) 30%, var(--border));
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
  overflow-x: auto;
}
.example-item code {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-dim);
  white-space: nowrap;
}
.example-item:hover {
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-card));
  border-style: solid;
  border-color: color-mix(in srgb, var(--accent) 50%, var(--border));
}
.example-item:hover code {
  color: var(--accent);
}

/* ============================================
   2026-07-09 增:输入框 + 安装按钮
   ============================================ */
.install-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.install-form-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}
.install-form-row {
  display: flex;
  gap: 8px;
  align-items: stretch;
}
.install-input {
  flex: 1;
  min-width: 0;
  padding: 9px 12px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text);
  font-family: inherit;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.install-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 20%, transparent);
}
.install-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 按钮:primary(主操作)+ ghost(次操作) */
.source-card-actions .primary,
.install-form-row .primary,
.progress-go-home.primary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--accent);
  border: 1px solid var(--accent);
  color: #ffffff;
  padding: 9px 16px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: filter 0.15s ease, transform 0.15s ease;
}
.source-card-actions .primary:hover:not(:disabled),
.install-form-row .primary:hover:not(:disabled),
.progress-go-home.primary:hover:not(:disabled) {
  filter: brightness(0.92);
  transform: translateY(-1px);
}
.source-card-actions .primary:disabled,
.install-form-row .primary:disabled,
.progress-go-home.primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.source-card-actions .ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
  padding: 9px 16px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}
.source-card-actions .ghost:hover {
  border-color: var(--accent);
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-card));
}

/* 错误条 */
.install-error {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: color-mix(in srgb, #ef4444 10%, var(--bg-card));
  border: 1px solid color-mix(in srgb, #ef4444 40%, var(--border));
  border-radius: var(--radius-sm);
  color: #b91c1c;
  font-size: 12px;
  line-height: 1.5;
}
:global(html.dark) .install-error {
  color: #fca5a5;
}

/* ============================================
   2026-07-09 增:进度条
   ============================================ */
.install-progress {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--accent) 5%, var(--bg-card));
  border: 1px solid color-mix(in srgb, var(--accent) 15%, var(--border));
  border-radius: var(--radius-sm);
}
.progress-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.progress-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text);
}
.progress-percent {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--accent);
  font-weight: 600;
}
.progress-track {
  width: 100%;
  height: 6px;
  background: color-mix(in srgb, var(--accent) 15%, var(--bg-card));
  border-radius: 3px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), color-mix(in srgb, var(--accent) 60%, var(--mkt-accent)));
  transition: width 0.15s linear;
  border-radius: 3px;
}
.progress-go-home {
  align-self: flex-start;
  margin-top: 4px;
}

/* 卡片底部 action 区 */
.source-card-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
  border-top: 1px solid var(--border);
  padding-top: 12px;
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
  .install-form-row {
    flex-direction: column;
  }
}
</style>