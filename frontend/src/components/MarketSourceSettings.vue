<script setup>
// MarketSourceSettings.vue - "源设置" 弹窗(2026-06-30 增,P1)。
//
// 复用 frontend/src/components/Modal.vue。
// 每个源卡片显示:名称、类型、enabled 开关、base_url 输入框、保存按钮。
// 修改后调 market.updateSource 同步到后端 + store。
//
// 用法:
//   <MarketSourceSettings v-model="settingsOpen" />

import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import { useMarketStore } from '@/core/store/market'
import { useToastStore } from '@/core/store/toast'
import Modal from '@/components/Modal.vue'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
})
const emit = defineEmits(['update:modelValue'])

const { t } = useI18n()
const market = useMarketStore()
const toast = useToastStore()

// 编辑状态:{ sourceId: { enabled, baseUrl, dirty } }
const edits = ref({})
const saving = ref({})

watch(
  () => market.sources,
  (sources) => {
    // 初始化编辑状态(只在初次打开或新增源时填默认)
    for (const s of sources) {
      if (edits.value[s.id]) continue
      edits.value[s.id] = {
        enabled: !!s.enabled,
        baseUrl: extractBaseUrl(s.config_json),
        dirty: false,
      }
    }
  },
  { immediate: true, deep: true }
)

function extractBaseUrl(configJSON) {
  if (!configJSON) return ''
  try {
    const obj = JSON.parse(configJSON)
    return obj.base_url || ''
  } catch {
    return ''
  }
}

function markDirty(id) {
  if (edits.value[id]) {
    edits.value[id].dirty = true
  }
}

async function saveOne(id) {
  const edit = edits.value[id]
  if (!edit || !edit.dirty) return
  saving.value[id] = true
  try {
    await market.updateSource(id, {
      enabled: edit.enabled,
      config_json: edit.baseUrl ? JSON.stringify({ base_url: edit.baseUrl }) : '',
    })
    edit.dirty = false
    toast.push({ type: 'success', message: t('market.sourcesSettings.saved') })
  } catch (e) {
    toast.push({ type: 'error', message: t('market.sourcesSettings.saveFailed', { msg: e?.message || e }) })
  } finally {
    saving.value[id] = false
  }
}

async function saveAll() {
  const dirtyIds = Object.entries(edits.value)
    .filter(([, e]) => e.dirty)
    .map(([id]) => id)
  for (const id of dirtyIds) {
    await saveOne(Number(id))
  }
}

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    size="md"
    :title="t('market.sourcesSettings.title')"
    @update:model-value="(v) => emit('update:modelValue', v)"
  >
    <template #title-icon>
      <Icon icon="mdi:cog-outline" width="18" height="18" />
    </template>

    <div class="source-settings">
      <div v-for="s in market.sources" :key="s.id" class="source-card">
        <div class="source-head">
          <div class="source-info">
            <Icon icon="mdi:radio-tower" width="14" height="14" class="source-icon" />
            <span class="source-name">{{ s.name }}</span>
            <span class="source-type-badge">{{ s.type }}</span>
          </div>
          <label class="toggle">
            <input
              type="checkbox"
              :checked="edits[s.id]?.enabled"
              @change="(e) => { edits[s.id].enabled = e.target.checked; markDirty(s.id) }"
            />
            <span>{{ edits[s.id]?.enabled ? t('market.sourcesSettings.enabled') : t('market.sourcesSettings.disabled') }}</span>
          </label>
        </div>

        <div class="source-row">
          <label class="field-label">{{ t('market.sourcesSettings.baseUrl') }}</label>
          <input
            v-model="edits[s.id].baseUrl"
            type="text"
            class="field-input"
            :placeholder="'https://example.com'"
            @input="markDirty(s.id)"
          />
        </div>

        <div class="source-actions">
          <button
            type="button"
            class="primary sm"
            :disabled="!edits[s.id]?.dirty || saving[s.id]"
            @click="saveOne(s.id)"
          >
            <span v-if="saving[s.id]" class="spinner"></span>
            <Icon v-else icon="mdi:content-save-outline" width="12" height="12" />
            {{ t('market.sourcesSettings.btnSave') }}
          </button>
        </div>
      </div>

      <div v-if="!market.sources.length" class="empty-hint">
        {{ t('market.noSources') }}
      </div>
    </div>

    <template #footer>
      <button type="button" class="ghost" @click="close">
        {{ t('common.close') }}
      </button>
      <button type="button" class="primary" @click="saveAll">
        <Icon icon="mdi:content-save-all-outline" width="14" height="14" />
        {{ t('market.sourcesSettings.btnSave') }}
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.source-settings {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 60vh;
  overflow-y: auto;
}

.source-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 12px 14px;
  background: var(--bg-subtle);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.source-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.source-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.source-icon {
  color: var(--text-dim);
}

.source-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.source-type-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 10px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text-faint);
  text-transform: uppercase;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-dim);
  cursor: pointer;
}

.source-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.field-input {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  background: var(--bg-card);
  color: var(--text);
}

.source-actions {
  display: flex;
  justify-content: flex-end;
}

button.sm {
  padding: 4px 10px;
  font-size: 12px;
}

.empty-hint {
  text-align: center;
  padding: 24px;
  color: var(--text-faint);
  font-size: 13px;
}

.spinner {
  display: inline-block;
  width: 10px;
  height: 10px;
  border: 2px solid var(--bg-card);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
