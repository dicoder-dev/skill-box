<script setup>
// AIHistoryDialog - 历史对话列表弹窗(2026-07-14 增)
//
// 复用 components/Modal.vue,列表项显示 title / preview / ts,点击 inject 当前会话。
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Modal from '@/components/Modal.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  items: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'pick'])

const { t } = useI18n()

function close() {
  emit('update:modelValue', false)
}
function pick(it) {
  emit('pick', it)
}
function fmtTs(ts) {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString()
  } catch (_) {
    return ''
  }
}
// 给列表项的预览一个简单的 markdown-strip,把反引号去掉
function cleanPreview(p) {
  if (!p) return ''
  return p.replace(/[`*_#>\[\]]/g, '').trim()
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
          @click="pick(it)"
        >
          <div class="ai-hist-title">{{ it.title || it.id }}</div>
          <div class="ai-hist-preview">{{ cleanPreview(it.preview) }}</div>
          <div class="ai-hist-meta">
            <span class="ai-hist-ts">{{ fmtTs(it.ts) }}</span>
            <span v-if="it.provider" class="ai-hist-prov">{{ it.provider }}{{ it.model ? '/' + it.model : '' }}</span>
          </div>
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
  padding: 10px 12px;
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
  justify-content: space-between;
  align-items: center;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-faint);
}
.ai-hist-prov {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
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
