<script setup>
// AIHistoryDialog - 历史对话列表弹窗(2026-07-14 v2 增)
//
// v2 改动:
//   - 列表项加 "删除" 按钮(右上方),mdi:trash-can-outline;调用 ai.deleteHistoryItem(convId)。
//   - 整个列表项的 click 触发 pickHistoryItem(异步,失败 toast)。
//   - pick 进行中按 savingConv 渲染 loading。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Modal from '@/components/Modal.vue'
import { useAiStore } from '@/core/store/ai'
import { useToastStore } from '@/core/store/toast'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  items: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'pick'])

const { t } = useI18n()
const ai = useAiStore()
const toast = useToastStore()

function close() {
  emit('update:modelValue', false)
}

async function pick(it) {
  try {
    await ai.pickHistoryItem(it)
  } catch (e) {
    toast.error(t('skills.aiPanel.loadFailed', '加载失败'))
  }
}

async function remove(it, ev) {
  ev.stopPropagation() // 不触发 pick
  try {
    await ai.deleteHistoryItem(it.id)
  } catch (e) {
    toast.error(t('skills.aiPanel.deleteFailed', '删除对话失败'))
  }
}

function fmtTs(ts) {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString()
  } catch (_) {
    return ''
  }
}
function cleanPreview(p) {
  if (!p) return ''
  return p.replace(/[`*_#>\[\]]/g, '').trim()
}
function fmtSize(b) {
  if (!b || b <= 0) return ''
  if (b < 1024) return `${b} B`
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`
  return `${(b / (1024 * 1024)).toFixed(2)} MB`
}

const cItems = computed(() => props.items || [])
</script>

<template>
  <Modal
    :model-value="modelValue"
    @update:model-value="(v) => emit('update:modelValue', v)"
    size="md"
    :title="t('skills.aiPanel.historyDialog.title', '历史对话')"
    @close="close"
  >
    <div class="ai-history-dialog">
      <p v-if="loading" class="ai-hist-desc">
        {{ t('skills.aiPanel.historyDialog.loading', '加载中…') }}
      </p>
      <p v-else-if="!cItems.length" class="ai-hist-desc">
        {{ t('skills.aiPanel.historyDialog.empty', '暂无历史对话') }}
      </p>
      <ul v-else class="ai-hist-list">
        <li
          v-for="it in cItems"
          :key="it.id"
          class="ai-hist-item"
          :class="{ 'ai-hist-item-saving': ai.savingConv }"
          @click="pick(it)"
        >
          <div class="ai-hist-title">{{ it.title || it.id }}</div>
          <div class="ai-hist-preview">{{ cleanPreview(it.preview) }}</div>
          <div class="ai-hist-meta">
            <span class="ai-hist-ts">{{ fmtTs(it.ts) }}</span>
            <span class="ai-hist-size" v-if="it.size">{{ fmtSize(it.size) }}</span>
            <span v-if="it.provider" class="ai-hist-prov">{{ it.provider }}{{ it.model ? '/' + it.model : '' }}</span>
          </div>
          <button
            type="button"
            class="ai-hist-del"
            :data-tip="t('skills.aiPanel.deleteConv', '删除对话')"
            :aria-label="t('skills.aiPanel.deleteConv', '删除对话')"
            @click="remove(it, $event)"
          >
            <span>✕</span>
          </button>
        </li>
      </ul>
    </div>
    <template #footer>
      <button type="button" class="ai-hist-btn" @click="close">
        {{ t('common.close', '关闭') }}
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.ai-history-dialog { font-size: 13px; color: var(--text); }
.ai-hist-desc {
  margin: 0; padding: 28px 0; text-align: center;
  color: var(--text-dim); font-size: 12.5px;
}
.ai-hist-list {
  list-style: none; padding: 0; margin: 0;
  max-height: 60vh; overflow-y: auto;
}
.ai-hist-item {
  position: relative;
  padding: 10px 36px 10px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: pointer;
  background: var(--bg-card);
  transition: border-color 120ms, background 120ms;
}
.ai-hist-item:hover {
  border-color: var(--primary);
  background: var(--bg-subtle);
}
.ai-hist-item-saving {
  opacity: 0.7;
  pointer-events: none;
}
.ai-hist-title {
  font-weight: 600; font-size: 13px;
  margin-bottom: 4px;
  color: var(--text);
}
.ai-hist-preview {
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.ai-hist-meta {
  display: flex;
  gap: 12px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-faint);
}
.ai-hist-size {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.ai-hist-prov {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.ai-hist-del {
  position: absolute;
  top: 8px; right: 8px;
  width: 22px; height: 22px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--text-dim);
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.ai-hist-del:hover {
  border-color: var(--border);
  color: var(--text);
  background: var(--bg-subtle);
}
.ai-hist-btn {
  padding: 6px 14px;
  font-size: 12.5px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: 4px;
  cursor: pointer;
}
.ai-hist-btn:hover { border-color: var(--primary); }
</style>
