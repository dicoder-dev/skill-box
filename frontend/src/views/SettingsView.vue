<script setup>
import { ref, reactive, onMounted, onUnmounted, watch, inject } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import IconPark from '@/components/IconPark.vue'
import AISettingsPanel from '@/components/AISettingsPanel.vue'
import { platform } from '@/platform'
import { useAppStore } from '@/core/store/app.js'
import { setLocale, getLocale } from '@/core/i18n'
import { migrateApplyMode, listApplies } from '@/api/skillbox/skill_apply.js'

const { t, locale } = useI18n()

// 当前语言响应式镜像,组件外修改 i18n.locale 也能即时反映
const currentLang = ref(getLocale())
watch(locale, (v) => { currentLang.value = v })

const store = useAppStore()
const { isDesktop } = storeToRefs(store)
const prefsSupported = ref(isDesktop.value)
const saveHint = ref('')
const notifyTest = ref('')
const langHint = ref('')

function onLangChange(loc) {
  if (loc !== 'zh-CN' && loc !== 'en-US') return
  if (loc === currentLang.value) return
  setLocale(loc)
  langHint.value = t('settings.saved')
  setTimeout(() => (langHint.value = ''), 1500)
}

const desktopPrefs = reactive({
  start_minimized: 'false',
  notify_enabled: 'true',
  shortcut_enabled: 'true',
  global_hotkey: 'Cmd+Shift+S',
})

// 2026-07-02 增:apply 模式(copy / symlink)。值用 'copy' / 'symlink' 字符串,
// 与后端 settings.apply_mode 一致,直接通过 platform.prefs 读写。
// applyModeSupported: web 端 platform.prefs 在 web 实现里返空,允许 UI 仍展示
// 但切换后端不会落盘,降级为"仅本会话生效"。这里通过首次读取的 snap 是否
// 拿到 key 来判断;首屏读不到时仍允许用户点,后端会忽略非空 key 之外的值。
//
// 2026-07-08 改回:默认值改回 'copy'。上一版改空串想区分"没加载"和"加载完
// 真的是 copy",但实测用户反馈'什么都没选中'——空串 + 拉不到数据 =
// 完全空白,体验更糟。回到 'copy' 默认值保证首屏永远有选中态,即便
// loadPrefs 暂时失败,UI 仍合理。后端真存 symlink 时,loadPrefs 拉
// 到会立刻覆盖回 'symlink'(几 ms 内),用户能感知到闪烁但不会出现
// '什么都没选'的卡死状态。
const applyMode = ref('copy') // 'copy' | 'symlink'
const applyModeHint = ref('')
const applyModeBusy = ref(false)
const applyModeSupported = ref(false) // 能否真正持久化(通过 getAll 拿到 keys 判断)

// 2026-07-03 增:跨页通知 — migrate-mode 成功后 emit 'skills:refresh',
// 让 SkillsView 静默重拉当前选中 skill 的 scope-status,以反映新的磁盘形态。
// appBus 由 App.vue 行 22-39 provide;window event 兜底兼容 web 端(无 inject)。
const appBus = inject('appBus', null)
// 2026-07-08 增:直接 inject App.vue 的 activeTab ref,watch 它决定何时刷 prefs。
// 这条链路比事件总线更稳:不依赖 on/off 注册、不依赖 emit 时序、
// 不会因为事件名拼错静默失败。事件总线路径保留作为兜底(主要面向 web 端
// 或未来跨 webview 场景)。
const activeTab = inject('activeTab', null)
function emitSkillsRefresh(payload) {
  if (appBus?.emit) {
    appBus.emit('skills:refresh', payload)
  }
  // 兼容写法:与 MarketView 行 119 的 dispatchEvent 兜底一致
  window.dispatchEvent(new CustomEvent('skillbox:skills-refresh', { detail: payload }))
}

// 2026-07-08 改:在 setup 同步阶段直接调 loadPrefs(),不等 onMounted。
// 原因:Vue 的 v-if 切换 tab 时,组件实例保留,onMounted 不会再触发;
// 之前依赖 onMounted(loadPrefs) 意味着切回设置页根本不刷新。
// 把 loadPrefs 调用挪到 setup() 同步阶段后,首次进设置页时立即发起请求;
// watch(activeTab) 负责后续切回时的刷新。
// 注意:loadPrefs 是 async,这里只是启动,不 await,不影响 setup 同步返回。
async function loadPrefs() {
  try {
    const snap = await platform.prefs.getAll()
    // 调试日志:DevTools console 可看到此次拉到的 snap,排查"切回还是 copy"必备。
    // 2026-07-08 加:之前一直查不出根因就是因为没看 console 日志。
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.log('[SettingsView] loadPrefs snap=', JSON.stringify(snap), 'applyMode before=', applyMode.value)
    }
    applyModeSupported.value = snap && typeof snap === 'object'
    // 2026-07-08 改:用 != null 替代 truthy 判断,避免后端存 'copy'/'symlink' 这种
    // 非空字符串时被误判。同时 snap['skillbox.apply_mode'] 可能是字符串也可能
    // 是 undefined,统一用 Object.hasOwn 判断 key 是否存在。
    if (snap && Object.prototype.hasOwnProperty.call(snap, 'skillbox.apply_mode')) {
      const v = snap['skillbox.apply_mode']
      const next = v === 'symlink' ? 'symlink' : 'copy'
      if (import.meta.env.DEV) {
        // eslint-disable-next-line no-console
        console.log('[SettingsView] applyMode value from snap=', v, 'next=', next)
      }
      applyMode.value = next
    }
    for (const k of Object.keys(desktopPrefs)) {
      if (snap[k] != null) desktopPrefs[k] = snap[k]
    }
  } catch (e) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[SettingsView] loadPrefs error=', e)
    }
    prefsSupported.value = false
    applyModeSupported.value = false
  }
}
loadPrefs()

async function savePref(key, value) {
  if (!isDesktop.value) return
  try {
    await platform.prefs.set(key, String(value))
    saveHint.value = t('settings.saved')
    setTimeout(() => (saveHint.value = ''), 1500)
  } catch (e) {
    saveHint.value = t('settings.errSave', { msg: e?.message || e })
  }
}

// 2026-07-02 增:apply 模式切换。
// 流程:用户点 segmented 切到新模式 → 弹 confirm(展示受影响 skill 数)
//       → 用户确认 → 调 /api/skillbox/skills/apply/migrate-mode 迁移所有
//       已 apply 的行 → 写 settings.apply_mode → toast 结果。
// 注意:applyModeSupported=false(web 端 prefs 不持久化)时,本次会话仍能切,
// 但刷新后回到 copy — 提示文案对此做了说明。
async function countApplied() {
  // 简单做法:通过 listApplies 拉所有 applied,只取 total。
  // 失败时返 0,前端 confirm 会按"0 条"展示(其实 0 条时后端 migrate 也无副作用)。
  try {
    const r = await listApplies({ status: 'applied', page: 1, size: 1 })
    return r?.total || 0
  } catch (e) {
    return 0
  }
}

// 2026-07-02 增:apply 模式切换(改:两阶段 confirm)。
//
// 流程:
//   1) 立即把 settings.apply_mode 切到 newMode(未来的 apply 立刻按新模式走)。
//   2) 若当前有 total 条 status=applied 的 skill,弹一个**独立的二次确认**
//      "是否同时把已应用的 N 条 skill 重新落盘?",让用户单独选择是否迁移
//      现有数据(用户可能只想改未来行为,不动现有)。
//   3) 用户同意 → 调 /migrate-mode;拒绝 / 失败 → 模式已切但不动旧数据,toast 说明。
async function onApplyModeChange(newMode) {
  if (applyModeBusy.value) return
  if (newMode === applyMode.value) return
  // 2026-07-08 改:web 端也走真实 prefs 持久化(走 /api/desktop/prefs,与
  // 桌面端同实现)。原先 web 降级只改本地 ref 不持久化,切回设置页时
  // applyMode 回到初始 'copy',给用户"切换失败"的错觉。
  // 1) 先把模式切到 settings(后续 apply 立刻按新模式)
  applyModeBusy.value = true
  try {
    await platform.prefs.set('skillbox.apply_mode', newMode)
    applyMode.value = newMode
    applyModeHint.value = t('settings.applyMode.modeChanged', { mode: t(
      newMode === 'symlink' ? 'settings.applyMode.symlink' : 'settings.applyMode.copy',
    ) })
  } catch (e) {
    applyModeHint.value = t('settings.errSave', { msg: e?.message || e })
    applyModeBusy.value = false
    setTimeout(() => (applyModeHint.value = ''), 3000)
    return
  }

  // 2) 拉已应用数量,弹二次确认(独立选择"是否应用到现有 skill")
  const total = await countApplied()
  if (total === 0) {
    // 没已应用 skill,直接收尾
    applyModeBusy.value = false
    setTimeout(() => (applyModeHint.value = ''), 3000)
    return
  }
  const migrateKey = newMode === 'symlink'
    ? 'settings.applyMode.applyExistingToSymlinkConfirm'
    : 'settings.applyMode.applyExistingToCopyConfirm'
  const migrate = window.confirm(t(migrateKey, { total }))
  if (!migrate) {
    applyModeHint.value = t('settings.applyMode.modeChangedNoMigrate', {
      mode: t(newMode === 'symlink' ? 'settings.applyMode.symlink' : 'settings.applyMode.copy'),
      total,
    })
    applyModeBusy.value = false
    setTimeout(() => (applyModeHint.value = ''), 4000)
    return
  }

  // 3) 用户同意迁移 → 调 /migrate-mode
  applyModeHint.value = t('settings.applyMode.switchMigrating', { total })
  try {
    const res = await migrateApplyMode({ mode: newMode })
    applyModeHint.value = t('settings.applyMode.switchSuccess', {
      ok: res?.ok ?? 0,
      skipped: res?.skipped ?? 0,
      failed: res?.failed ?? 0,
    })
    if (res && res.failed > 0) {
      const failedEntries = (res.entries || []).filter((e) => !e.ok && !e.skipped)
      const detail = failedEntries
        .map((e) => `  • ${e.name} (${e.tool}): ${e.error}`)
        .join('\n')
      if (detail) {
        applyModeHint.value += '\n' + t('settings.applyMode.switchFailedDetail', { detail })
      }
    }
    // 2026-07-03 增:通知 SkillsView 刷新 scope-status —— 磁盘形态从 copy
    // 切到 symlink(或反向)后,首页 chip 只反映"是否存在",由 scope-status
    // 后端扫描给出形态;不通知的话用户切回首页看到的是旧磁盘快照。
    // 即便 res.ok=0 也要通知,因为可能某些条目虽然失败但 target_dir 已变更。
    emitSkillsRefresh({
      from: res?.from_mode,
      to: res?.to_mode,
      ok: res?.ok ?? 0,
      failed: res?.failed ?? 0,
    })
  } catch (e) {
    applyModeHint.value = t('settings.errSave', { msg: e?.message || e })
  } finally {
    applyModeBusy.value = false
    setTimeout(() => (applyModeHint.value = ''), 6000)
  }
}

function onToggleStart(v) {
  desktopPrefs.start_minimized = v ? 'true' : 'false'
  savePref('desktop.start_minimized', desktopPrefs.start_minimized)
}
function onToggleNotify(v) {
  desktopPrefs.notify_enabled = v ? 'true' : 'false'
  savePref('desktop.notify_enabled', desktopPrefs.notify_enabled)
}
function onToggleShortcut(v) {
  desktopPrefs.shortcut_enabled = v ? 'true' : 'false'
  savePref('desktop.shortcut_enabled', desktopPrefs.shortcut_enabled)
}
function onHotkeyChange(e) {
  const v = (e.target.value || '').trim()
  desktopPrefs.global_hotkey = v
  savePref('desktop.global_hotkey', v)
}

async function testNotify() {
  notifyTest.value = ''
  try {
    if (desktopPrefs.notify_enabled !== 'true') {
      notifyTest.value = t('settings.notifyDisabled')
      return
    }
    await platform.notify.show('', t('settings.testTitle'), t('settings.testBody'))
    notifyTest.value = t('settings.notifySent')
  } catch (e) {
    notifyTest.value = t('settings.errNotify', { msg: e?.message || e })
  }
}

onMounted(() => {
  // 注意:不再在这里调 loadPrefs(),改成 setup 同步阶段调(见 loadPrefs()
  // 函数定义之后)。原因:Vue 的 v-if 切换 tab 时,组件实例保留,
  // onMounted 不会再触发;在 setup 同步调一次后,首次进设置页就有数据。
  //
  // 监听 tab 切换,切回 settings 时重新拉 prefs:
  //   1) watch(activeTab): 主链路,直接追响应式 ref
  //   2) appBus 'app:tab-change': 跨 webview 兼容
  //   3) window 'skillbox:tab-change': 兜底 web 端无 inject
  if (activeTab) {
    watch(activeTab, (v) => {
      if (v === 'settings') loadPrefs()
    }, { immediate: false })
  }
  appBus?.on?.('app:tab-change', onTabChange)
  window.addEventListener('skillbox:tab-change', onWindowTabChange)
})

onUnmounted(() => {
  appBus?.off?.('app:tab-change', onTabChange)
  window.removeEventListener('skillbox:tab-change', onWindowTabChange)
})

function onTabChange(target) {
  // 只在切回 settings 时刷新;切走不刷(避免无意义请求)。
  // payload 在 App.vue switchTab 中为 tab key 字符串。
  if (target === 'settings') loadPrefs()
}
function onWindowTabChange(e) {
  onTabChange(e?.detail)
}
</script>

<template>
  <div class="settings-view">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-gray">
          <IconPark icon="mdi:cog-outline" width="24" height="24" />
        </div>
        <div>
          <h1>{{ t('settings.title') }}</h1>
          <p>{{ t('settings.subtitle') }}</p>
        </div>
      </div>
    </header>

    <!-- 通用偏好(语言切换,Web / 桌面端均可见) -->
    <section class="card">
      <header class="card-header">
        <h3>
          <IconPark icon="mdi:tune-variant" width="18" height="18" />
          {{ t('settings.general.title') }}
          <span class="card-sub">— {{ t('settings.general.subtitle') }}</span>
        </h3>
      </header>

      <div class="pref-list">
        <!-- 界面语言 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.general.language') }}</div>
            <div class="pref-hint">{{ t('settings.general.languageHint') }}</div>
          </div>
          <div class="lang-segmented">
            <button
              type="button"
              :class="['lang-btn', currentLang === 'zh-CN' ? 'lang-active' : '']"
              @click="onLangChange('zh-CN')"
            >
              <IconPark icon="mdi:check" width="14" height="14" v-if="currentLang === 'zh-CN'" />
              {{ t('settings.general.langZhCN') }}
            </button>
            <button
              type="button"
              :class="['lang-btn', currentLang === 'en-US' ? 'lang-active' : '']"
              @click="onLangChange('en-US')"
            >
              <IconPark icon="mdi:check" width="14" height="14" v-if="currentLang === 'en-US'" />
              {{ t('settings.general.langEnUS') }}
            </button>
          </div>
        </div>

        <!-- 2026-07-02 增:Skill 应用方式(copy / symlink)。Web / 桌面端均可见。 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.applyMode.title') }}</div>
            <div class="pref-hint">
              {{ applyMode === 'symlink'
                ? t('settings.applyMode.symlinkHint')
                : t('settings.applyMode.copyHint') }}
            </div>
          </div>
          <div class="mode-segmented">
            <button
              type="button"
              :class="['mode-btn', applyMode === 'copy' ? 'mode-btn-active' : '']"
              :disabled="applyModeBusy"
              @click="onApplyModeChange('copy')"
            >
              <IconPark icon="mdi:check-circle" width="16" height="16" class="mode-btn-icon" v-if="applyMode === 'copy'" />
              <IconPark icon="mdi:content-copy-outline" width="16" height="16" class="mode-btn-icon" v-else />
              <span class="mode-btn-label">{{ t('settings.applyMode.copy') }}</span>
            </button>
            <button
              type="button"
              :class="['mode-btn', applyMode === 'symlink' ? 'mode-btn-active' : '']"
              :disabled="applyModeBusy"
              @click="onApplyModeChange('symlink')"
            >
              <IconPark icon="mdi:check-circle" width="16" height="16" class="mode-btn-icon" v-if="applyMode === 'symlink'" />
              <IconPark icon="mdi:link-variant" width="16" height="16" class="mode-btn-icon" v-else />
              <span class="mode-btn-label">{{ t('settings.applyMode.symlink') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 切换提示 -->
      <div v-if="langHint" class="hint-box lang-hint">
        <IconPark icon="mdi:check-circle" width="14" height="14" class="hint-icon hint-success" />
        <span>{{ langHint }}</span>
      </div>
      <div v-if="applyModeHint" class="hint-box lang-hint apply-mode-hint">
        <IconPark icon="mdi:information" width="14" height="14" class="hint-icon" />
        <span style="white-space: pre-line">{{ applyModeHint }}</span>
      </div>
    </section>

    <!-- 2026-07-12 增:AI 模型配置(独立 card,可见即可用,不分 Web / 桌面端) -->
    <AISettingsPanel />

    <!-- 桌面端设置 -->
    <section v-if="isDesktop" class="card">
      <header class="card-header">
        <h3>
          <IconPark icon="mdi:desktop-classic" width="18" height="18" />
          {{ t('settings.desktop.title') }}
          <span class="card-sub">— {{ t('settings.desktop.subtitle') }}</span>
        </h3>
      </header>

      <div v-if="!prefsSupported" class="error-box">
        <IconPark icon="mdi:alert-circle-outline" width="16" height="16" />
        {{ t('settings.prefsUnavailable') }}
      </div>

      <div v-else class="pref-list">
        <!-- 启动最小化 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.desktop.startMinimized') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.startMinimizedHint') }}</div>
          </div>
          <label class="toggle">
            <input
              type="checkbox"
              :checked="desktopPrefs.start_minimized === 'true'"
              @change="(e) => onToggleStart(e.target.checked)"
            />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <!-- 通知设置 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.desktop.notifyEnabled') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.notifyEnabledHint') }}</div>
          </div>
          <label class="toggle">
            <input
              type="checkbox"
              :checked="desktopPrefs.notify_enabled === 'true'"
              @change="(e) => onToggleNotify(e.target.checked)"
            />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <!-- 快捷键设置 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.desktop.shortcutEnabled') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.shortcutEnabledHint') }}</div>
          </div>
          <label class="toggle">
            <input
              type="checkbox"
              :checked="desktopPrefs.shortcut_enabled === 'true'"
              @change="(e) => onToggleShortcut(e.target.checked)"
            />
            <span class="toggle-slider"></span>
          </label>
        </div>

        <!-- 全局快捷键 -->
        <div class="pref-item">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.desktop.globalHotkey') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.globalHotkeyHint') }}</div>
          </div>
          <input
            class="hotkey-input"
            type="text"
            :value="desktopPrefs.global_hotkey"
            @change="onHotkeyChange"
            :placeholder="t('settings.desktop.globalHotkeyPh')"
          />
        </div>

        <!-- 测试通知 -->
        <div class="pref-item pref-item-action">
          <div class="pref-info">
            <div class="pref-label">{{ t('settings.testNotify') }}</div>
            <div class="pref-hint">{{ t('settings.testNotifyHint') }}</div>
          </div>
          <button class="primary" @click="testNotify">
            <IconPark icon="mdi:bell-ring-outline" width="14" height="14" />
            {{ t('settings.btnTestNotify') }}
          </button>
        </div>

        <!-- 保存提示 -->
        <div v-if="saveHint || notifyTest" class="hint-box">
          <IconPark v-if="saveHint" icon="mdi:check-circle" width="14" height="14" class="hint-icon hint-success" />
          <IconPark v-if="notifyTest" icon="mdi:information" width="14" height="14" class="hint-icon" />
          <span>{{ saveHint || notifyTest }}</span>
        </div>
      </div>
    </section>

    <!-- Web 端提示 -->
    <section v-else class="card">
      <div class="empty-state">
        <IconPark icon="mdi:monitor-dashboard" width="48" height="48" />
        <p class="empty-title">{{ t('settings.webOnlyHint') }}</p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-view {
  max-width: 900px;
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

.view-icon-gray {
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

/* 卡片 */
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
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.card-header h3 {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.card-sub {
  font-size: 12px;
  color: var(--text-dim);
  font-weight: normal;
}

/* 错误提示 */
.error-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: var(--bg-subtle);
  color: var(--danger);
  border: 1px solid var(--border);
  border-left: 3px solid var(--danger);
  border-radius: var(--radius);
  font-size: 13px;
}

/* 设置列表 */
.pref-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.pref-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border);
  transition: background 0.15s ease;
}

.pref-item:first-child {
  padding-top: 0;
}

.pref-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.pref-item-action {
  padding-top: 20px;
  margin-top: 8px;
  border-top: 1px solid var(--border);
}

.pref-info {
  flex: 1;
  min-width: 0;
}

.pref-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
  margin-bottom: 4px;
}

.pref-hint {
  font-size: 12px;
  color: var(--text-dim);
  max-width: 480px;
}

/* 开关 */
.toggle {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
  flex-shrink: 0;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--border);
  transition: 0.3s;
  border-radius: 26px;
}

.toggle-slider::before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: var(--bg-card);
  transition: 0.3s;
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.toggle input:checked + .toggle-slider {
  background-color: var(--primary);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(22px);
}

/* 快捷键输入框 */
.hotkey-input {
  width: 180px;
  padding: 8px 12px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  text-align: center;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  transition: all 0.15s ease;
}

.hotkey-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}

/* 语言切换器(分段式按钮组) */
.lang-segmented {
  display: inline-flex;
  align-items: stretch;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
  flex-shrink: 0;
}

.lang-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 32px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
  background: transparent;
  border: 1px solid transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  cursor: pointer;
  transition: all 0.12s ease;
  white-space: nowrap;
}

.lang-btn:hover { color: var(--text); }

.lang-btn.lang-active {
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.lang-hint {
  margin-top: 16px;
}

/* 2026-07-02 增:apply mode segmented(独立样式,与 lang 共用一套太低调)。
 * 选中态用主色填充 + 阴影 + check 图标,跟未选中态形成强对比,避免
 * 用户看不清当前模式。颜色用 --primary 蓝(主色),不踩紫色禁条。
 */
.mode-segmented {
  display: inline-flex;
  align-items: stretch;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 3px;
  gap: 3px;
  flex-shrink: 0;
}

.mode-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-dim);
  background: transparent;
  border: 1px solid transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  cursor: pointer;
  transition: all 0.12s ease;
  white-space: nowrap;
}

.mode-btn:hover:not(:disabled):not(.mode-btn-active) {
  color: var(--text);
  background: var(--bg-card);
}

.mode-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mode-btn-icon {
  flex-shrink: 0;
}

.mode-btn-label {
  line-height: 1;
}

/* 选中态:主色背景 + 白字 + 阴影 + 描边,跟未选中态明显区分。 */
.mode-btn.mode-btn-active {
  background: var(--primary);
  color: var(--primary-contrast, #fff);
  border-color: var(--primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.18);
  font-weight: 600;
}

.mode-btn.mode-btn-active .mode-btn-icon {
  color: var(--primary-contrast, #fff);
}

/* 提示框 */
.hint-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 10px 14px;
  background: var(--success-dim);
  color: var(--success);
  border-radius: var(--radius-sm);
  font-size: 13px;
}

.hint-icon {
  flex-shrink: 0;
}

.hint-success {
  color: var(--success);
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

/* 响应式 */
@media (max-width: 768px) {
  .pref-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .hotkey-input {
    width: 100%;
  }
}
</style>
