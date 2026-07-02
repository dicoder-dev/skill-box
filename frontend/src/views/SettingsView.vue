<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
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
const applyMode = ref('copy') // 'copy' | 'symlink'
const applyModeHint = ref('')
const applyModeBusy = ref(false)
const applyModeSupported = ref(false) // 能否真正持久化(通过 getAll 拿到 keys 判断)

async function loadPrefs() {
  if (!isDesktop.value) return
  try {
    const snap = await platform.prefs.getAll()
    applyModeSupported.value = snap && typeof snap === 'object'
    if (snap && snap['skillbox.apply_mode']) {
      applyMode.value = snap['skillbox.apply_mode'] === 'symlink' ? 'symlink' : 'copy'
    }
    for (const k of Object.keys(desktopPrefs)) {
      if (snap[k] != null) desktopPrefs[k] = snap[k]
    }
  } catch (e) {
    prefsSupported.value = false
    applyModeSupported.value = false
  }
}

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
  if (!isDesktop.value) {
    // Web 端:平台层 prefs 不持久化,直接改本地 ref + 提示。
    applyMode.value = newMode
    applyModeHint.value = t('settings.saved')
    setTimeout(() => (applyModeHint.value = ''), 1500)
    return
  }
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

onMounted(loadPrefs)
</script>

<template>
  <div class="settings-view">
    <!-- 页面头部 -->
    <header class="view-header">
      <div class="view-title">
        <div class="view-icon view-icon-gray">
          <Icon icon="mdi:cog-outline" width="24" height="24" />
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
          <Icon icon="mdi:tune-variant" width="18" height="18" />
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
              <Icon icon="mdi:check" width="14" height="14" v-if="currentLang === 'zh-CN'" />
              {{ t('settings.general.langZhCN') }}
            </button>
            <button
              type="button"
              :class="['lang-btn', currentLang === 'en-US' ? 'lang-active' : '']"
              @click="onLangChange('en-US')"
            >
              <Icon icon="mdi:check" width="14" height="14" v-if="currentLang === 'en-US'" />
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
              <Icon icon="mdi:check-circle" width="16" height="16" class="mode-btn-icon" v-if="applyMode === 'copy'" />
              <Icon icon="mdi:content-copy-outline" width="16" height="16" class="mode-btn-icon" v-else />
              <span class="mode-btn-label">{{ t('settings.applyMode.copy') }}</span>
            </button>
            <button
              type="button"
              :class="['mode-btn', applyMode === 'symlink' ? 'mode-btn-active' : '']"
              :disabled="applyModeBusy"
              @click="onApplyModeChange('symlink')"
            >
              <Icon icon="mdi:check-circle" width="16" height="16" class="mode-btn-icon" v-if="applyMode === 'symlink'" />
              <Icon icon="mdi:link-variant" width="16" height="16" class="mode-btn-icon" v-else />
              <span class="mode-btn-label">{{ t('settings.applyMode.symlink') }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 切换提示 -->
      <div v-if="langHint" class="hint-box lang-hint">
        <Icon icon="mdi:check-circle" width="14" height="14" class="hint-icon hint-success" />
        <span>{{ langHint }}</span>
      </div>
      <div v-if="applyModeHint" class="hint-box lang-hint apply-mode-hint">
        <Icon icon="mdi:information" width="14" height="14" class="hint-icon" />
        <span style="white-space: pre-line">{{ applyModeHint }}</span>
      </div>
    </section>

    <!-- 桌面端设置 -->
    <section v-if="isDesktop" class="card">
      <header class="card-header">
        <h3>
          <Icon icon="mdi:desktop-classic" width="18" height="18" />
          {{ t('settings.desktop.title') }}
          <span class="card-sub">— {{ t('settings.desktop.subtitle') }}</span>
        </h3>
      </header>

      <div v-if="!prefsSupported" class="error-box">
        <Icon icon="mdi:alert-circle-outline" width="16" height="16" />
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
            <Icon icon="mdi:bell-ring-outline" width="14" height="14" />
            {{ t('settings.btnTestNotify') }}
          </button>
        </div>

        <!-- 保存提示 -->
        <div v-if="saveHint || notifyTest" class="hint-box">
          <Icon v-if="saveHint" icon="mdi:check-circle" width="14" height="14" class="hint-icon hint-success" />
          <Icon v-if="notifyTest" icon="mdi:information" width="14" height="14" class="hint-icon" />
          <span>{{ saveHint || notifyTest }}</span>
        </div>
      </div>
    </section>

    <!-- Web 端提示 -->
    <section v-else class="card">
      <div class="empty-state">
        <Icon icon="mdi:monitor-dashboard" width="48" height="48" />
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
