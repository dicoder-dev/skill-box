<script setup>
// MarketInstallConfirm.vue - 三方市场"安装"弹窗(2026-06-30 增)。
//
// 复用 frontend/src/components/Modal.vue。三态:
//   - 未冲突:scope 选择 + tools 多选 + 确认/取消
//   - 冲突:三按钮(覆盖 / 另存为 / 取消)
//     "另存为" 展开一个 input,前端自动生成候选 name-2 → name-3 ...
//
// 用法:
//   <MarketInstallConfirm
//     :item="marketSkill"
//     :installed="market.installed"
//     :projects="market.projects"
//     @confirm="(payload) => market.install(payload)"
//     @cancel="() => dialog = false"
//   />

import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import Modal from '@/components/Modal.vue'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  item: { type: Object, default: null }, // MarketSkill(必传)
  installed: { type: Object, default: () => ({}) }, // name -> bool
  projects: { type: Array, default: () => [] }, // [{id, name, alias}]
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const { t } = useI18n()

// 状态
const scope = ref('global') // global / project
const projectId = ref(0)
const selectedTools = ref(['codex', 'claude', 'opencode', 'cursor', 'trae'])
const isDuplicate = computed(() => !!props.item && !!props.installed?.[props.item.name])

// 另存为
const strategy = ref('overwrite') // overwrite / saveAs / cancel
const newName = ref('')

// 内部状态
const submitting = ref(false)
const formError = ref('')

// 切换 item 时重置
watch(
  () => props.item,
  (it) => {
    if (!it) return
    strategy.value = isDuplicate.value ? 'overwrite' : 'overwrite'
    // 自动生成候选名
    let candidate = `${it.name}-2`
    let n = 2
    while (props.installed?.[candidate]) {
      n += 1
      candidate = `${it.name}-${n}`
    }
    newName.value = candidate
    // 重置 scope/tools
    scope.value = 'global'
    projectId.value = 0
    selectedTools.value = ['codex', 'claude', 'opencode', 'cursor', 'trae']
    formError.value = ''
  },
  { immediate: true }
)

const finalName = computed(() => {
  if (!isDuplicate.value) return props.item?.name || ''
  if (strategy.value === 'overwrite') return props.item?.name || ''
  if (strategy.value === 'saveAs') return newName.value.trim()
  return ''
})

const canConfirm = computed(() => {
  if (!props.item) return false
  if (isDuplicate.value && strategy.value === 'saveAs' && !newName.value.trim()) return false
  if (scope.value === 'project' && !projectId.value) return false
  if (selectedTools.value.length === 0) return false
  return true
})

function toggleTool(tool) {
  if (selectedTools.value.includes(tool)) {
    selectedTools.value = selectedTools.value.filter((x) => x !== tool)
  } else {
    selectedTools.value = [...selectedTools.value, tool]
  }
}

function selectAll() {
  selectedTools.value = ['codex', 'claude', 'opencode', 'cursor', 'trae']
}
function deselectAll() {
  selectedTools.value = []
}

function close() {
  emit('update:modelValue', false)
  emit('cancel')
}

function onConfirm() {
  if (!canConfirm.value) {
    formError.value = t('common.invalidInput')
    return
  }
  submitting.value = true
  formError.value = ''
  emit('confirm', {
    sourceId: props.item.source_id,
    remoteId: props.item.remote_id,
    scope: scope.value,
    projectId: scope.value === 'project' ? projectId.value : 0,
    tools: [...selectedTools.value],
    finalName: finalName.value,
  })
  // 关闭交给父组件处理(等 install 完成后关闭)
  setTimeout(() => {
    submitting.value = false
  }, 100)
}
</script>

<template>
  <Modal
    :model-value="modelValue"
    size="md"
    :title="t('market.installDialog.title', { name: item?.name || '' })"
    :close-on-mask="!submitting"
    @update:model-value="(v) => emit('update:modelValue', v)"
  >
    <template #title-icon>
      <Icon icon="mdi:download-outline" width="18" height="18" />
    </template>

    <div v-if="item" class="install-form">
      <!-- 描述 -->
      <p v-if="item.description" class="install-desc">{{ item.description }}</p>

      <!-- 重复检测提示 -->
      <div v-if="isDuplicate" class="dup-warn">
        <div class="dup-title">
          <Icon icon="mdi:alert-outline" width="16" height="16" />
          {{ t('market.installDialog.duplicateTitle', { name: item.name }) }}
        </div>
        <div class="dup-hint">{{ t('market.installDialog.duplicateHint') }}</div>
        <div class="dup-actions">
          <button
            type="button"
            :class="['dup-btn', { active: strategy === 'overwrite' }]"
            @click="strategy = 'overwrite'"
          >
            <Icon icon="mdi:content-save-outline" width="14" height="14" />
            {{ t('market.installDialog.btnOverwrite') }}
          </button>
          <button
            type="button"
            :class="['dup-btn', { active: strategy === 'saveAs' }]"
            @click="strategy = 'saveAs'"
          >
            <Icon icon="mdi:content-copy" width="14" height="14" />
            {{ t('market.installDialog.btnSaveAs') }}
          </button>
        </div>
        <div v-if="strategy === 'saveAs'" class="dup-saveas">
          <label class="saveas-label">{{ t('market.installDialog.newNameLabel') }}</label>
          <input v-model="newName" type="text" class="saveas-input" :placeholder="item.name + '-2'" />
          <p class="saveas-hint">{{ t('market.installDialog.saveAsHint') }}</p>
        </div>
      </div>

      <!-- scope 选择 -->
      <div class="form-row">
        <label class="form-label">{{ t('market.installDialog.scopeLabel') }}</label>
        <div class="form-controls">
          <label class="radio">
            <input v-model="scope" type="radio" value="global" />
            <span>{{ t('market.scopeGlobal') }}</span>
          </label>
          <label class="radio">
            <input v-model="scope" type="radio" value="project" />
            <span>{{ t('market.scopeProject') }}</span>
          </label>
          <select v-if="scope === 'project'" v-model="projectId" class="form-select">
            <option :value="0" disabled>{{ t('market.projectPlaceholder') }}</option>
            <option v-for="p in projects" :key="p.id" :value="p.id">
              {{ p.name || p.alias || ('#' + p.id) }}
            </option>
          </select>
        </div>
      </div>

      <!-- tools 多选 -->
      <div class="form-row">
        <label class="form-label">{{ t('market.installDialog.toolsLabel') }}</label>
        <div class="form-controls-col">
          <div class="tools-list">
            <label v-for="tool in ['codex', 'claude', 'opencode', 'cursor', 'trae']" :key="tool" class="tool-chip">
              <input
                type="checkbox"
                :checked="selectedTools.includes(tool)"
                @change="toggleTool(tool)"
              />
              <span>{{ tool }}</span>
            </label>
          </div>
          <div class="tools-actions">
            <button type="button" class="ghost sm" @click="selectAll">
              {{ t('market.installDialog.selectAll') }}
            </button>
            <button type="button" class="ghost sm" @click="deselectAll">
              {{ t('market.installDialog.selectNone') }}
            </button>
          </div>
          <p class="form-hint">{{ t('market.installDialog.toolsHint') }}</p>
        </div>
      </div>

      <div v-if="formError" class="form-error">{{ formError }}</div>
    </div>

    <template #footer>
      <button type="button" class="ghost" :disabled="submitting" @click="close">
        {{ t('market.installDialog.btnCancel') }}
      </button>
      <button type="button" class="primary" :disabled="!canConfirm || submitting" @click="onConfirm">
        <span v-if="submitting" class="spinner"></span>
        <Icon v-else icon="mdi:download" width="14" height="14" />
        {{ submitting ? t('market.installDialog.installing') : t('market.installDialog.confirm') }}
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.install-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  font-size: 13px;
}

.install-desc {
  margin: 0;
  color: var(--text-dim);
  line-height: 1.5;
  padding: 10px 12px;
  background: var(--bg-subtle);
  border-radius: var(--radius-sm);
  font-size: 12px;
}

/* 重复检测提示 */
.dup-warn {
  border: 1px solid var(--warning);
  background: var(--warning-dim);
  border-radius: var(--radius);
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dup-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--warning);
  font-size: 13px;
}

.dup-hint {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.5;
}

.dup-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.dup-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.dup-btn:hover {
  background: var(--bg-hover);
  color: var(--text);
}

.dup-btn.active {
  background: var(--warning);
  color: var(--bg-card);
  border-color: var(--warning);
}

.dup-saveas {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 6px;
}

.saveas-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
}

.saveas-input {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-family: 'JetBrains Mono', monospace;
}

.saveas-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

/* 表单行 */
.form-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
}

.form-controls {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.form-controls-col {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-select {
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13px;
  min-width: 200px;
  background: var(--bg-card);
  color: var(--text);
}

.radio {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text);
  cursor: pointer;
}

.tools-list {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
  cursor: pointer;
  user-select: none;
}

.tool-chip input {
  margin: 0;
}

.tools-actions {
  display: flex;
  gap: 6px;
}

.form-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

.form-error {
  font-size: 12px;
  color: var(--danger);
  padding: 6px 10px;
  background: var(--danger-dim);
  border-radius: var(--radius-sm);
}

.spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--bg-card);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
