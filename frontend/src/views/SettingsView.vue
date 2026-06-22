<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { platform } from '@/platform'

const { t } = useI18n()

// 桌面端偏好(在 desktop/{prefs_keys.go} 与 internal/settings 持久化)。
// 字段为空时取 platform.prefs.getAll() 拿到的快照。
const desktopPrefs = reactive({
  start_minimized: 'false',
  notify_enabled: 'true',
  shortcut_enabled: 'true',
  global_hotkey: 'Cmd+Shift+S',
})
const isDesktop = ref(platform.isDesktop)
const prefsSupported = ref(isDesktop.value) // web 端 prefs.getAll() 返回 {},不阻塞 UI
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
    <header class="head">
      <h2 class="flex items-center gap-2">
        <Icon icon="mdi:cog-outline" width="20" height="20" class="text-sb-primary" />
        {{ t('settings.title') }}
      </h2>
      <p class="muted">{{ t('settings.subtitle') }}</p>
    </header>

    <!-- 桌面端 section(仅桌面端可见;web 端显示提示) -->
    <section v-if="isDesktop" class="card">
      <h3>{{ t('settings.desktop.title') }}
        <span class="card-sub">— {{ t('settings.desktop.subtitle') }}</span>
      </h3>

      <div v-if="!prefsSupported" class="err inline-flex items-center gap-1.5">
        <Icon icon="mdi:alert-circle-outline" width="14" height="14" />
        {{ t('settings.prefsUnavailable') }}
      </div>

      <div v-else class="pref-list">
        <label class="pref-row">
          <div>
            <div class="pref-label">{{ t('settings.desktop.startMinimized') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.startMinimizedHint') }}</div>
          </div>
          <input
            type="checkbox"
            :checked="desktopPrefs.start_minimized === 'true'"
            @change="(e) => onToggleStart(e.target.checked)"
          />
        </label>

        <label class="pref-row">
          <div>
            <div class="pref-label">{{ t('settings.desktop.notifyEnabled') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.notifyEnabledHint') }}</div>
          </div>
          <input
            type="checkbox"
            :checked="desktopPrefs.notify_enabled === 'true'"
            @change="(e) => onToggleNotify(e.target.checked)"
          />
        </label>

        <label class="pref-row">
          <div>
            <div class="pref-label">{{ t('settings.desktop.shortcutEnabled') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.shortcutEnabledHint') }}</div>
          </div>
          <input
            type="checkbox"
            :checked="desktopPrefs.shortcut_enabled === 'true'"
            @change="(e) => onToggleShortcut(e.target.checked)"
          />
        </label>

        <label class="pref-row">
          <div>
            <div class="pref-label">{{ t('settings.desktop.globalHotkey') }}</div>
            <div class="pref-hint">{{ t('settings.desktop.globalHotkeyHint') }}</div>
          </div>
          <input
            class="hotkey"
            type="text"
            :value="desktopPrefs.global_hotkey"
            @change="onHotkeyChange"
            :placeholder="t('settings.desktop.globalHotkeyPh')"
          />
        </label>

        <div class="pref-row actions">
          <div>
            <div class="pref-label">{{ t('settings.testNotify') }}</div>
            <div class="pref-hint">{{ t('settings.testNotifyHint') }}</div>
          </div>
          <button class="primary" @click="testNotify">
            <Icon icon="mdi:bell-ring-outline" width="14" height="14" />
            {{ t('settings.btnTestNotify') }}
          </button>
        </div>

        <p v-if="saveHint" class="hint ok">{{ saveHint }}</p>
        <p v-if="notifyTest" class="hint">{{ notifyTest }}</p>
      </div>
    </section>

    <section v-else class="card">
      <div class="empty-state">
        <span class="empty-icon">
          <Icon icon="mdi:monitor-dashboard" width="36" height="36" />
        </span>
        {{ t('settings.webOnlyHint') }}
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-view { max-width: 900px; margin: 0 auto; }
.head h2 { margin: 0 0 4px; font-size: 18px; }
.head p { margin: 0 0 16px; font-size: 13px; }

.pref-list { display: flex; flex-direction: column; gap: 4px; }
.pref-row {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 10px 12px; border-bottom: 1px solid #f3f4f6;
}
.pref-row:last-child { border-bottom: none; }
.pref-row.actions { padding-top: 14px; border-top: 1px solid var(--border); margin-top: 6px; }
.pref-label { font-size: 13px; font-weight: 500; color: var(--text); }
.pref-hint  { font-size: 12px; color: var(--text-dim); margin-top: 2px; max-width: 480px; }

.pref-row input[type="checkbox"] { width: 18px; height: 18px; cursor: pointer; }
.pref-row input.hotkey { width: 180px; font-family: ui-monospace, monospace; font-size: 12px; }

.hint { margin-top: 8px; font-size: 12px; }
.hint.ok { color: var(--success); }
.err  { background: var(--danger-dim); color: var(--danger); border: 1px solid #fecaca; padding: 8px 12px; border-radius: var(--radius-sm); font-size: 13px; }
</style>
