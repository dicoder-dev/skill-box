<script setup>
import { ref, reactive, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { platform } from '@/platform'
import { useAppStore } from '@/core/store/app.js'

const { t } = useI18n()

const desktopPrefs = reactive({
  start_minimized: 'false',
  notify_enabled: 'true',
  shortcut_enabled: 'true',
  global_hotkey: 'Cmd+Shift+S',
})

const store = useAppStore()
const { isDesktop } = storeToRefs(store)
const prefsSupported = ref(isDesktop.value)
const saveHint = ref('')
const notifyTest = ref('')

async function loadPrefs() {
  if (!isDesktop.value) return
  try {
    const snap = await platform.prefs.getAll()
    for (const k of Object.keys(desktopPrefs)) {
      if (snap[k] != null) desktopPrefs[k] = snap[k]
    }
  } catch (e) {
    prefsSupported.value = false
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
