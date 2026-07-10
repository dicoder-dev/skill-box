<script setup>
// MarketView.vue - 三方市场 v5(2026-07-10)。
//
// 在 v4 基础上新增:
//   1. 默认 tab 按系统语言自动选(中文→ skillhub-cn,英文→ skills.sh)
//   2. skillhub → skillhub-cn 改名(source id / UI 名 / 后端 source_type / 分组名全链路)
//   3. skills.sh 输入示例精简:删 GitHub 路径示例,只保留一条 skills.sh URL
//   4. GitHub tab 增加「知名 skill 仓库」快捷按钮,点击在系统浏览器打开
//   5. 装到 skill-box 按钮左侧加「粘贴」按钮,读剪贴板填入输入框
//
// 配色:SkillHub-CN 保留青色 #0ea5e9;Skills.sh 由原紫色 #8b5cf6 改为绿色 #10b981
// (符合项目 memory:avoid-violet-as-primary-color.md,紫色 AI 感强,主色禁用)。

import { ref, computed, inject, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import { platform } from '@/platform'
import { installFromInput } from '@/api/skillbox/market'
import { useToastStore } from '@/core/store/toast'
import { getCurrentLocale } from '@/core/i18n'

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
// 在源结构里更稳。源是固定 3 个(skillhub-cn / skills.sh / github),维护成本可控。
//
// 2026-07-10 改:skillhub → skillhub-cn(source id / UI 名 / 后端 source_type / 分组名全链路)
// 后端 SourceSkillhub 常量同步改成 "skillhub-cn"。skillssh examples 精简:删 GitHub 路径
// 示例,只保留一条 skills.sh URL。github 增加 famousRepos 块。
//
// 2026-07-10 增:GitHub 知名仓库列表(联网搜索整理):
//   1. anthropics/skills         — Anthropic 官方 Agent Skills 仓库(权威,示例)
//   2. vercel-labs/agent-skills   — Vercel Labs 出品,React 最佳实践等
//   3. mattpocock/skills         — TypeScript 训练师 mattpocock 的 skills(榜单常驻)
//   4. JackyST0/awesome-agent-skills — 中文社区 awesome 列表(1000+ skills)
// 每个对象只暴露最小字段:display / owner / repo / url;
// UI 渲染 display,点击调用 platform.platform.openExternal(url) 跳转。
const sources = [
  {
    id: 'skillhub-cn',
    name: 'SkillHub-CN',
    descKey: 'market.cards.skillhubDesc',
    url: 'https://skillhub.cn/skills',
    accent: '#0ea5e9',
    // 2026-07-10 改:跟后端 SourceSkillhub("skillhub-cn")对齐,
    // POST /api/skillbox/market/install-from-input 的 source_hint 字段会直接透传这个值。
    sourceType: 'skillhub-cn',
    placeholderKey: 'market.input.placeholderSkillhub',
    guideKey: 'market.guide.skillhub',
    examples: [
      'https://skillhub.cn/skills/code-review',
      'https://skillhub.cn/skills/commit-msg',
    ],
  },
  {
    id: 'skillssh',
    name: 'Skills.sh',
    descKey: 'market.cards.skillsshDesc',
    url: 'https://www.skills.sh/hot', // 2026-07-10 改:用户要求跳到 /hot(默认排序页)而非站点根
    accent: '#10b981',
    sourceType: 'skillssh',
    placeholderKey: 'market.input.placeholderSkillssh',
    guideKey: 'market.guide.skillssh',
    // 2026-07-10 改:按"用户要求"精简,只保留一条 skills.sh 详情页 URL 示例,
    // 删掉 GitHub blob URL(那一类 URL 走 GitHub tab 更合理)
    examples: [
      'https://skills.sh/anthropics/skills/pdf',
    ],
  },
  // 2026-07-09 增:GitHub 独立来源(从 skills.sh 拆出来)
  // 2026-07-10 改(美化):accent 由 #6b7280 中性灰改成更深邃的色系,
  // 跟蓝/绿两个 tab 形成层次而不是简陋灰。GitHub prime 风的深炭黑 + 暖灰底纹,
  // 既保留「非蓝非绿」的辨识度,也避免 AI 感强的紫色(violet 禁用)。
  // 浅色背景:bg 是 #f6f8fa(github light),dark:`#0d1117`(github dark)。
  {
    id: 'github',
    name: 'GitHub',
    descKey: 'market.cards.githubDesc',
    url: 'https://github.com',
    accent: '#1f2328',
    accentSoft: '#656d76',
    accentBgLight: '#f6f8fa',
    accentBgDark: '#0d1117',
    sourceType: 'github',
    placeholderKey: 'market.input.placeholderGithub',
    guideKey: 'market.guide.github',
    // 2026-07-10 改(用户要求):GitHub 示例精简到 1 条,
    // 只保留 anthropics/skills/tree/main/skills/pdf 这条具体 skill 路径
    examples: [
      'https://github.com/anthropics/skills/tree/main/skills/pdf',
    ],
    // 2026-07-10 增:GitHub tab 「知名 skill 仓库」快捷浏览块,
    // UI 用 famousReposBlock 渲染,按钮调 platform.platform.openExternal 跳转。
    famousRepos: [
      {
        id: 'anthropics-skills',
        display: 'anthropics/skills',
        owner: 'anthropics',
        repo: 'skills',
        // 2026-07-10 改:用户要求跳到具体 skills/ 子目录,而不是 repo 根
        url: 'https://github.com/anthropics/skills/tree/main/skills',
      },
      {
        id: 'vercel-labs-agent-skills',
        display: 'vercel-labs/agent-skills',
        owner: 'vercel-labs',
        repo: 'agent-skills',
        url: 'https://github.com/vercel-labs/agent-skills/tree/main/skills',
      },
      {
        id: 'mattpocock-skills',
        display: 'mattpocock/skills',
        owner: 'mattpocock',
        repo: 'skills',
        url: 'https://github.com/mattpocock/skills/tree/main/skills',
      },
      {
        id: 'jacky-st0-awesome-agent-skills',
        // 2026-07-10 改:awesome 列表例外,根 README 就是「目录页」,直接给 README 入口
        display: 'JackyST0/awesome-agent-skills',
        owner: 'JackyST0',
        repo: 'awesome-agent-skills',
        url: 'https://github.com/JackyST0/awesome-agent-skills',
      },
    ],
  },
]

// 2026-07-10 增:按当前语言选默认 tab。
// 中文(zh-CN / zh)→ skillhub-cn(国内源,主入口);
// 其他(en-US 等)→ skills.sh(海外源)。
//
// 用 getCurrentLocale()(暴露在 core/i18n)直接读 i18n 状态,避免 useI18n() 在
// v-if 懒挂载时拿到 Proxy 不调 t 的 bug(memory 中 SkillFileInlinePanel i18n 坑)。
function pickDefaultSourceId() {
  const loc = String(getCurrentLocale() || '').toLowerCase()
  if (loc.startsWith('zh')) return 'skillhub-cn'
  return 'skillssh'
}

// 2026-07-10 改:activeSourceId 初始化从 sources[0] 改为按 locale 算的 default,
// 切回 MarketView 重新进入时,SettingsView 改了语言也会重新计算(setup 阶段执行)。
const activeSourceId = ref(pickDefaultSourceId())
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

// 2026-07-10 改:粘贴按钮 — 把系统剪贴板文本塞进输入框 + 自动调安装。
// 桌面端走 platform.platform.clipboardText()(后端 GetClipboardText 走 wails ClipboardGetText);
// Web 端 WebClipboardText 兜底返空串,失败时 toast 提示。
//
// 2026-07-10 改(用户要求):按钮文案「粘贴」→「粘贴并安装」,粘贴成功后自动调
// handleInstall()(走正常的解析 → 下载 → 写盘流程),不再让用户再点一次主按钮。
// 失败(剪贴板为空 / 读取异常)不触发安装,只 toast / 错误条提示。
//
// 2026-07-10 改(修 [object Object] bug):
// 调用方拿到 clipboardText 返回值时统一走 _safeStringify ——
// 历史上 wails 端某个版本 ClipboardGetText binding 偶尔返回
// { text: 'xxx' } 或 string[],如果直接 `String(...)` 会拿到 '[object Object]'。
// 这里兼容三种形态(string / string[] / { text: '...' }),最终强制成 string。
function _clipboardToText(raw) {
  if (raw == null) return ''
  if (typeof raw === 'string') return raw
  if (Array.isArray(raw)) {
    return raw.map((v) => (typeof v === 'string' ? v : String(v || ''))).join('')
  }
  if (typeof raw === 'object') {
    // wails 旧 binding 形如 { text: 'xxx' } / { content: 'xxx' } 都接住
    if (typeof raw.text === 'string') return raw.text
    if (typeof raw.content === 'string') return raw.content
    if (typeof raw.value === 'string') return raw.value
  }
  try {
    return JSON.stringify(raw)
  } catch (_) {
    return ''
  }
}

async function pasteAndInstall() {
  if (installing.value) return
  let raw = null
  try {
    raw = await platform.platform.clipboardText()
  } catch (e) {
    // 失败原因:跨平台走 plainT(纯函数),不让 t 在 Proxy 包装下
    // 返回 [object Object];另一个保险:err string 兜底走 String(e?.message || e || '')
    const errMsg = (e && (e.message || e.error)) || (typeof e === 'string' ? e : '')
    toast.error(t('market.btnPasteFailed', { msg: errMsg || 'unknown' }))
    return
  }
  const text = _clipboardToText(raw).trim()
  if (!text) {
    toast.error(t('market.btnPasteEmpty'))
    return
  }
  // 2026-07-10 改:成功分支自动调 handleInstall(),不再二次点击主按钮
  userInput.value = text
  installError.value = ''
  await handleInstall()
}

// 2026-07-09 增:输入框 + 安装流程

const userInput = ref('')
const installing = ref(false)
const installError = ref('')

// 2026-07-09 增:同名 skill 冲突确认
// 弹 Modal 之前,先把 409 响应里的 conflict_existing_version / conflict_existing_path
// 暂存到这里,Modal 用这些字段渲染。
const conflict = ref(null) // null = 无冲突;object = 有冲突,待用户决策

// 进度条 4 阶段;每阶段一个独立 ref,确保切换时旧 ref 残留不会乱跳
const progressStage = ref('')        // 当前阶段 key(resolve/download/extract/write/done/fail/'')
const progressPercent = ref(0)       // 0-100
const lastFailedStage = ref('')      // 2026-07-09 增:失败前最后阶段(resolve/download/extract/write),
                                    // 用于 fail 阶段 hint 精确定位"卡哪步"
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
  conflict.value = null
  lastFailedStage.value = ''
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

// 2026-07-09 增:报错保留进度条文字,只把 stage 改成 'fail' + 红色 hint。
// 用户反馈:报错时进度文字全不见,不知道卡哪步。
// 现在保留 stage + percent,加 fail 红色 hint 告诉用户"在 X 阶段出错"。
function markFailed() {
  if (progressStage.value && progressStage.value !== 'done' && progressStage.value !== 'fail') {
    lastFailedStage.value = progressStage.value
  }
  progressStage.value = 'fail'
}

// 2026-07-09 增:retryInstall — fail 状态下「重试」按钮,直接用上次 input 重发。
// 不重置 input(用户可能要改),只清 progress + 重新 doInstall。
async function retryInstall() {
  if (installing.value) return
  const input = userInput.value.trim()
  if (!input) return
  installError.value = ''
  conflict.value = null
  progressStage.value = ''
  progressPercent.value = 0
  lastFailedStage.value = ''
  await doInstall(input, '')
}

// 2026-07-09 增:判断是否疑似 GitHub 限流,用于错误条下方显示额外提示 + 重试按钮
// 条件:download 阶段失败 + 错误信息含 timeout / deadline / 60s(就是前端超时)
const isLikelyRateLimit = computed(() => {
  if (progressStage.value !== 'fail') return false
  if (lastFailedStage.value !== 'download') return false
  const msg = installError.value || ''
  return /timeout|deadline|60s|请求超时/i.test(msg)
})

// 「装到 skill-box」按钮 — 走 4 阶段模拟 → 后端一次性 HTTP → 收尾
//
// 2026-07-09 改:加 conflict_mode 参数,默认 prompt(后端遇同名返 409 弹 Modal);
// Modal 三选一(overwrite / rename / cancel)走 retryInstall 二次提交。
async function handleInstall() {
  if (installing.value) return
  const input = userInput.value.trim()
  if (!input) {
    installError.value = t('market.input.errInvalidInput')
    return
  }
  await doInstall(input, '')
}

async function doInstall(input, conflictMode) {
  installing.value = true
  resetProgress()
  advanceProgress('resolve')
  await new Promise((r) => setTimeout(r, 350))
  advanceProgress('download')
  try {
    const out = await installFromInput({
      source_hint: activeSource.value.sourceType,
      input,
      scope: 'global',
      conflict_mode: conflictMode,
    })
    advanceProgress('extract')
    await new Promise((r) => setTimeout(r, 350))
    advanceProgress('write')
    await new Promise((r) => setTimeout(r, 250))
    progressStage.value = 'done'
    installing.value = false
    lastInstalledName.value = out.skill_name
    toast.success(t('market.success.msg', { name: out.skill_name, version: out.skill_version || '0.1.0' }))
  } catch (e) {
    const status = e?.response?.status || e?.status
    const data = e?.response?.data || e?.data || {}
    if (status === 409) {
      // 2026-07-09 增:同名冲突,弹 Modal
      // 进度不重置,留给用户看到「装到一半发现冲突」体感
      // (之后选覆盖/另存为时 doInstall 会 resetProgress)
      installing.value = false
      conflict.value = {
        name: data.skill_name || userInput.value,
        existingVersion: data.conflict_existing_version || '?',
        existingPath: data.conflict_existing_path || '',
        input, // 保留 input 以便 retry
      }
      return
    }
    installing.value = false
    // 2026-07-09 改:报错保留进度条文字,只把 stage 改成 'fail' + 红色 hint。
    // 用户反馈:报错时进度文字全不见,不知道卡在哪步。
    // 现在保留 stage + percent,加 fail 红色 hint 告诉用户"在 X 阶段出错"。
    markFailed()
    const msg = e?.message || String(e)
    if (status === 400) {
      installError.value = t('market.input.errInvalidInput')
    } else if (status === 404) {
      // 2026-07-10 改:404 区分两种:
      //  - 服务器返 err 字符串含 "remote fetch failed: <slug>" 或 "skillmarket: remote fetch failed"
      //    → 走 errSkillNotFound(用户给的具体 slug 不存在 / 已下架)
      //  - 其它(老 errSource 走 source 未注册 / 解析后 source 没匹配等)
      const errTxt = String(data?.error || msg || '')
      if (/remote (fetch )?not found/i.test(errTxt)) {
        installError.value = t('market.input.errSkillNotFound', { msg: errTxt })
      } else {
        installError.value = t('market.input.errSource')
      }
    } else if (/timeout/i.test(msg)) {
      // 前端 15s/60s timeout 触发
      installError.value = t('market.input.errTimeout', { msg })
    } else if (/download|fetch|zipball|context deadline/i.test(msg)) {
      installError.value = t('market.input.errPull', { msg })
    } else {
      installError.value = t('market.input.errGeneric', { msg })
    }
  }
}

// 2026-07-09 增:Modal 三选一回调
async function resolveConflict(mode) {
  // mode: 'overwrite' | 'rename' | 'cancel'
  if (mode === 'cancel' || !conflict.value) {
    conflict.value = null
    return
  }
  const c = conflict.value
  conflict.value = null // 立刻关掉 Modal
  if (mode === 'rename') {
    // 让后端自动找空闲名(-2 / -3 后缀)
    await doInstall(c.input, 'rename')
  } else {
    // overwrite
    await doInstall(c.input, 'overwrite')
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
          :data-source-id="activeSource.id"
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

          <!-- 2026-07-10 增:GitHub tab 「知名 skill 仓库」快捷浏览块。
               v-if 限定 activeSource.id === 'github' 才出现,
               其他 tab 不渲染。按钮调 platform.platform.openExternal 跳转浏览器,
               走跨平台通道(桌面 wails BrowserOpenURL / Web window.open),不离开当前页。 -->
          <div v-if="activeSource.id === 'github' && activeSource.famousRepos?.length" class="famous-repos">
            <div class="famous-title">
              <IconPark icon="mdi:star-circle-outline" width="14" height="14" />
              <span>{{ t('market.githubFamous.title') }}</span>
            </div>
            <p class="famous-desc">{{ t('market.githubFamous.desc') }}</p>
            <div class="famous-list">
              <div
                v-for="r in activeSource.famousRepos"
                :key="r.id"
                class="famous-item"
              >
                <code class="famous-repo-name">{{ r.display }}</code>
                <button
                  type="button"
                  class="famous-open-btn"
                  :title="`${t('market.githubFamous.btnOpen')} ${r.display}`"
                  @click="openInExternal(r.url)"
                >
                  <IconPark icon="mdi:open-in-new" width="12" height="12" />
                  <span>{{ t('market.githubFamous.btnOpen') }}</span>
                </button>
              </div>
            </div>
          </div>

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
              <!-- 2026-07-10 改(用户要求):「安装」按钮放到 paste 按钮左侧
                   形成 [安装] [粘贴并安装] 的「主按钮 + 副按钮」视觉,
                   主按钮的「下载」意图更清晰,粘贴按钮作为「快速入口」紧邻其右 -->
              <button
                type="button"
                class="primary"
                :disabled="installing || !userInput.trim()"
                @click="handleInstall"
              >
                <IconPark :icon="installing ? 'mdi:loading' : 'mdi:download'" width="14" height="14" />
                {{ installing ? t('market.input.btnInstalling') : t('market.input.btnInstall') }}
              </button>
              <!-- 2026-07-10 增:粘贴按钮 — 主按钮右侧,粘贴成功自动调 handleInstall()
                   (走 4 阶段下载流程),用户无需再点主按钮。
                   失败(剪贴板空 / 读异常)只 toast 不触发安装。 -->
              <button
                type="button"
                class="paste-btn"
                :title="t('market.btnPasteTitle')"
                :disabled="installing"
                @click="pasteAndInstall"
              >
                <IconPark icon="mdi:content-paste" width="14" height="14" />
                <span>{{ t('market.btnPaste') }}</span>
              </button>
            </div>
            <!-- 错误条(只在失败时显示) -->
            <div v-if="installError" class="install-error">
              <IconPark icon="mdi:alert-circle-outline" width="14" height="14" />
              <div class="install-error-content">
                <div class="install-error-msg">{{ installError }}</div>
                <!-- 2026-07-09 增:失败时给「重试」按钮 + 限流场景特殊提示 -->
                <div v-if="isLikelyRateLimit" class="install-error-hint">
                  <IconPark icon="mdi:timer-sand" width="12" height="12" />
                  疑似 GitHub 限流(未鉴权 IP 每小时约 60 次)。等几分钟再点「重试」,或去浏览器手动下好后从「首页 → 本地导入」装入。
                </div>
                <button
                  v-if="progressStage === 'fail'"
                  type="button"
                  class="install-retry-btn"
                  @click="retryInstall"
                >
                  <IconPark icon="mdi:refresh" width="12" height="12" />
                  重试
                </button>
              </div>
            </div>
          </div>

          <!-- 2026-07-09 增:进度条(只在 installing 或 done 时显示) -->
          <div v-if="progressStage" class="install-progress">
            <div class="progress-row">
              <span class="progress-label">
                <IconPark
                  :icon="progressStage === 'done'
                    ? 'mdi:check-circle'
                    : (progressStage === 'fail' ? 'mdi:alert-circle' : 'mdi:loading')"
                  width="14"
                  height="14"
                  :spin="progressStage !== 'done' && progressStage !== 'fail'"
                />
                {{ t(`market.progress.${progressStage}`) }}
              </span>
              <span class="progress-percent">{{ progressPercent }}%</span>
            </div>
            <div class="progress-track">
              <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
            </div>
            <!-- 2026-07-09 增:具体子步骤文字(类似 npm install 那种「已下载 x MB / y MB」体感) -->
            <p v-if="progressStage !== 'done' && progressStage !== 'fail'" class="progress-hint">
              <IconPark icon="mdi:chevron-right" width="12" height="12" />
              {{ t(`market.progress.hint${progressStage.charAt(0).toUpperCase() + progressStage.slice(1)}`) }}
            </p>
            <!-- done 阶段显示最终结果 -->
            <p v-else-if="progressStage === 'done'" class="progress-hint progress-hint-done">
              <IconPark icon="mdi:check-circle" width="12" height="12" />
              {{ t(`market.progress.hintDone`) }}
            </p>
            <!-- fail 阶段:精确定位"卡哪步" + 看下方错误条 -->
            <p v-else class="progress-hint progress-hint-fail">
              <IconPark icon="mdi:alert-circle" width="12" height="12" />
              {{ t(`market.progress.hintFail${lastFailedStage
                ? lastFailedStage.charAt(0).toUpperCase() + lastFailedStage.slice(1)
                : 'Unknown'}`) }}
            </p>
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

      <!-- 2026-07-09 增:同名 skill 冲突确认 Modal -->
      <div v-if="conflict" class="conflict-overlay" @click.self="resolveConflict('cancel')">
        <div class="conflict-modal" :style="{ '--accent': activeSource.accent }">
          <div class="conflict-head">
            <IconPark icon="mdi:alert-circle-outline" width="20" height="20" />
            <span class="conflict-title">{{ t('market.conflict.title') }}</span>
          </div>
          <p class="conflict-desc">
            {{ t('market.conflict.desc', {
              name: conflict.name,
              existingVersion: conflict.existingVersion,
              existingPath: conflict.existingPath,
            }) }}
          </p>
          <div class="conflict-actions">
            <button type="button" class="conflict-btn conflict-overwrite" @click="resolveConflict('overwrite')">
              <IconPark icon="mdi:content-save-outline" width="14" height="14" />
              <div>
                <div class="conflict-btn-title">{{ t('market.conflict.overwrite') }}</div>
                <div class="conflict-btn-tip">{{ t('market.conflict.overwriteTip') }}</div>
              </div>
            </button>
            <button type="button" class="conflict-btn conflict-rename" @click="resolveConflict('rename')">
              <IconPark icon="mdi:content-copy" width="14" height="14" />
              <div>
                <div class="conflict-btn-title">{{ t('market.conflict.rename') }}</div>
                <div class="conflict-btn-tip">{{ t('market.conflict.renameTip') }}</div>
              </div>
            </button>
            <button type="button" class="conflict-btn conflict-cancel" @click="resolveConflict('cancel')">
              <IconPark icon="mdi:close" width="14" height="14" />
              <span>{{ t('market.conflict.cancel') }}</span>
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
   2026-07-10 增:GitHub tab 专属配色 + 视觉美化
   ============================================
   GitHub 走深炭黑 + 暖灰底纹的「prime dark 风」,跟 skillhub-cn 的青蓝、
   skillssh 的翠绿形成第三种视觉,而不是简陋的 light gray(#6b7280)。

   设计要点:
   1. 卡顶 4px accent 条由 --accent(深炭黑)承担
   2. icon 块:浅色模式用 github light bg + 深炭黑边框;深色模式用 github dark bg
   3. 「在浏览器中打开」按钮默认 outline 风,在 github 卡里复用 accent 配色
   4. famousRepos 块用 accentSoft 浅灰文字 + 深炭黑按钮底色,形成「GitHub files list」质感
*/

/* 用 [data-source-id="github"] 选择器,精准挑到 GitHub 卡 */
.source-card[data-source-id="github"] {
  background: linear-gradient(180deg,
    color-mix(in srgb, #1f2328 4%, var(--bg-card)) 0%,
    var(--bg-card) 30%);
}
.source-card[data-source-id="github"] .source-card-icon {
  background: color-mix(in srgb, #1f2328 12%, var(--bg-card));
  color: #1f2328;
  border-color: color-mix(in srgb, #1f2328 30%, var(--border));
}
:global(html.dark) .source-card[data-source-id="github"] .source-card-icon {
  background: color-mix(in srgb, #f6f8fa 12%, var(--bg-card));
  color: #f6f8fa;
  border-color: color-mix(in srgb, #f6f8fa 25%, var(--border));
}
.source-card[data-source-id="github"] .open-browser-btn {
  border-color: color-mix(in srgb, #1f2328 25%, var(--border));
  color: #1f2328;
}
:global(html.dark) .source-card[data-source-id="github"] .open-browser-btn {
  border-color: color-mix(in srgb, #f6f8fa 25%, var(--border));
  color: #f6f8fa;
}
.source-card[data-source-id="github"] .open-browser-btn:hover {
  background: color-mix(in srgb, #1f2328 8%, var(--bg-card));
  border-color: #1f2328;
  color: #1f2328;
}
:global(html.dark) .source-card[data-source-id="github"] .open-browser-btn:hover {
  background: color-mix(in srgb, #f6f8fa 10%, var(--bg-card));
  border-color: #f6f8fa;
  color: #f6f8fa;
}
:global(html.dark) .source-card[data-source-id="github"] {
  background: linear-gradient(180deg,
    color-mix(in srgb, #f6f8fa 4%, var(--bg-card)) 0%,
    var(--bg-card) 30%);
}

/* ============================================
   2026-07-10 增:GitHub tab 知名 skill 仓库快捷浏览块
   ============================================ */
/* GitHub 卡里 famous item 用 monospace file list 风格,
   按钮走深炭黑底 + 白字,模拟「GitHub Action Button」质感 */
.source-card[data-source-id="github"] .famous-repos {
  background: #f6f8fa; /* github light bg */
  border-color: #d0d7de;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-repos {
  background: #15191f; /* 比 github dark #0d1117 稍亮,卡内层级 */
  border-color: #30363d;
}
.source-card[data-source-id="github"] .famous-title {
  color: #1f2328;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-title {
  color: #f6f8fa;
}
.source-card[data-source-id="github"] .famous-item {
  background: var(--bg-card);
  border-color: #d0d7de;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-item {
  background: #0d1117;
  border-color: #30363d;
}
.source-card[data-source-id="github"] .famous-repo-name {
  color: #1f2328;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-repo-name {
  color: #f6f8fa;
}
.source-card[data-source-id="github"] .famous-open-btn {
  background: #1f2328;
  border-color: #1f2328;
  color: #ffffff;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-open-btn {
  background: #f6f8fa;
  border-color: #f6f8fa;
  color: #1f2328;
}
.source-card[data-source-id="github"] .famous-open-btn:hover {
  background: #2d3138;
  border-color: #2d3138;
  filter: none;
  transform: translateY(-1px);
  box-shadow: 0 2px 6px -2px rgba(31, 35, 40, 0.4);
}
:global(html.dark) .source-card[data-source-id="github"] .famous-open-btn:hover {
  background: #ffffff;
  border-color: #ffffff;
  box-shadow: 0 2px 6px -2px rgba(246, 248, 250, 0.4);
}
.source-card[data-source-id="github"] .famous-desc {
  color: #57606a;
}
:global(html.dark) .source-card[data-source-id="github"] .famous-desc {
  color: #8b949e;
}

.famous-repos {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--accent) 4%, var(--bg-card));
  border: 1px solid color-mix(in srgb, var(--accent) 18%, var(--border));
  border-radius: var(--radius-sm);
}
.famous-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--accent);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.famous-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.5;
}
.famous-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.famous-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 4px;
}
.famous-repo-name {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}
.famous-open-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: color-mix(in srgb, var(--accent) 12%, var(--bg-card));
  border: 1px solid color-mix(in srgb, var(--accent) 30%, var(--border));
  color: var(--accent);
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
  white-space: nowrap;
}
.famous-open-btn:hover {
  background: color-mix(in srgb, var(--accent) 20%, var(--bg-card));
  border-color: var(--accent);
  transform: translateY(-1px);
}
.famous-open-btn:active {
  transform: translateY(0);
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

/* 2026-07-10 增:粘贴按钮(装到 skill-box 按钮左侧)。
   outline 风格跟 primary 按钮区分,操作不是主流程,
   视觉上像「附属操作」不抢焦点。 */
.paste-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 14px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
  white-space: nowrap;
}
.paste-btn:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-card));
}
.paste-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
  align-items: flex-start;
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
.install-error-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 0;
}
.install-error-msg {
  word-break: break-word;
}
.install-error-hint {
  /* 2026-07-09 增:限流场景额外提示(灰底 + timer-sand icon) */
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #b45309; /* amber-700 跟红色错开 */
  background: #fef3c7;
  padding: 4px 8px;
  border-radius: 4px;
  line-height: 1.4;
}
:global(html.dark) .install-error-hint {
  color: #fbbf24;
  background: color-mix(in srgb, #b45309 20%, transparent);
}
.install-retry-btn {
  /* 2026-07-09 增:失败时「重试」按钮(小号,跟错误条一起) */
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid color-mix(in srgb, #ef4444 50%, var(--border));
  background: var(--bg-card);
  color: #b91c1c;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}
.install-retry-btn:hover {
  background: #fef2f2;
  border-color: #ef4444;
}
:global(html.dark) .install-retry-btn {
  color: #fca5a5;
  border-color: color-mix(in srgb, #ef4444 60%, var(--border));
}
:global(html.dark) .install-retry-btn:hover {
  background: color-mix(in srgb, #ef4444 15%, var(--bg-card));
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

/* 2026-07-09 增:子步骤文字(类似 npm install 那种"正在 xxx"体感) */
.progress-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  font-size: 11px;
  color: var(--text-dim);
  line-height: 1.4;
  font-family: 'JetBrains Mono', monospace;
}
.progress-hint-done {
  color: var(--accent);
  font-weight: 500;
}
.progress-hint-fail {
  /* 2026-07-09 增:fail 阶段红色,跟下方 install-error 红条区分(安装中 vs 安装失败) */
  color: #b91c1c;
  font-weight: 500;
}
:global(html.dark) .progress-hint-fail {
  color: #fca5a5;
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

/* ============================================
   2026-07-09 增:同名 skill 冲突确认 Modal
   ============================================ */
.conflict-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}
.conflict-modal {
  --accent: #0ea5e9;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-top: 4px solid var(--accent);
  border-radius: var(--radius);
  box-shadow: 0 20px 50px -10px rgba(0, 0, 0, 0.3);
  padding: 20px 22px;
  width: min(440px, calc(100vw - 32px));
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.conflict-head {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--accent);
  font-weight: 600;
  font-size: 14px;
}
.conflict-title {
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.conflict-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: var(--text);
}
.conflict-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}
.conflict-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: var(--radius-sm);
  cursor: pointer;
  text-align: left;
  font-size: 13px;
  color: var(--text);
  transition: all 0.15s ease;
}
.conflict-btn:hover {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-card));
}
.conflict-btn-title {
  font-weight: 600;
  font-size: 13px;
  line-height: 1.3;
}
.conflict-btn-tip {
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 2px;
  line-height: 1.4;
}
.conflict-cancel {
  justify-content: center;
  color: var(--text-dim);
  font-weight: 500;
}
.conflict-cancel:hover {
  background: color-mix(in srgb, #ef4444 6%, var(--bg-card));
  border-color: color-mix(in srgb, #ef4444 40%, var(--border));
  color: #b91c1c;
}
:global(html.dark) .conflict-cancel:hover {
  color: #fca5a5;
}
</style>