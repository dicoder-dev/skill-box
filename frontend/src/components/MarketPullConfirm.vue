<script setup>
// MarketPullConfirm.vue - 三方市场"拉取"弹窗(2026-07-01 重构)。
//
// 复用 frontend/src/components/Modal.vue。两态:
//   - 未冲突:scope 选择 + 分组选择 + 「将自动启用到」只读信息 + 确认/取消
//   - 冲突:三按钮(覆盖 / 另存为 / 取消)
//     "另存为" 展开一个 input,前端自动生成候选 name-2 → name-3 ...
//
// 2026-07-01 重构(本版本):
//   - 移除 tools 多选 + selectAll/selectNone:拉取时统一自动启用到本机全部 5 个工具
//     (skilladapter.AllTools = codex/claude/opencode/cursor/trae),
//     仍可在「技能」页按需关闭;原本"勾选工具"对三方市场这个场景收益不大,徒增噪音。
//   - 弹窗整体卡片化、分区清晰:大号头部 + 描述卡 + 重复检测 + 作用域(segment control) +
//     分组(行内 select + 按钮) + 「将自动启用到」信息条 + footer 大按钮。
//   - 所有按钮 inline-flex + align-items: center + gap,图标文字水平对齐。
//
// 用法(未变):
//   <MarketPullConfirm
//     :item="marketSkill"
//     :installed="market.installed"
//     :projects="market.projects"
//     @confirm="(payload) => market.pull(payload)"
//     @cancel="() => dialog = false"
//   />

import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@iconify/vue'
import Modal from '@/components/Modal.vue'
import { useSkillTreeStore } from '@/core/store/skill-tree'
import { createGroup as apiCreateGroup } from '@/api/skillbox/skills'

// 拉取时默认应用到的全部工具(与后端 skilladapter.AllTools 对齐)。
// 顺序与后端定义保持一致,展示稳定。
const ALL_TOOLS = ['codex', 'claude', 'opencode', 'cursor', 'trae']

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  item: { type: Object, default: null }, // MarketSkill(必传)
  installed: { type: Object, default: () => ({}) }, // name -> bool
  projects: { type: Array, default: () => [] }, // [{id, name, alias}]
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel', 'tree-changed'])

const { t } = useI18n()
const skillTree = useSkillTreeStore()

// 状态
const scope = ref('global') // global / project
const projectId = ref(0)
// 分组路径(空 = 根)
const groupPath = ref('')
const isDuplicate = computed(() => !!props.item && !!props.installed?.[props.item.name])

// 另存为
const strategy = ref('overwrite') // overwrite / saveAs / cancel
const newName = ref('')

// 内部状态
const submitting = ref(false)
const formError = ref('')

// 分组新建 inline
const newGroupOpen = ref(false)
const newGroupInput = ref('')
const newGroupErr = ref('')

// 派生:把树压成 option 数组(带 depth 缩进,用于 <select> + 缩进字符)。
// 格式: [{ value, label, depth }]
const groupOptions = computed(() => {
  const out = []
  const walk = (nodes, depth) => {
    for (const n of nodes || []) {
      if (n.is_group) {
        // 缩进字符 + 名字(后端 path 是 'a/b' 形式)
        out.push({ value: n.path, label: '— '.repeat(depth) + n.name, depth })
        walk(n.children, depth + 1)
      }
    }
  }
  walk(skillTree.tree || [], 0)
  return out
})

// 切换 item 时重置
watch(
  () => props.item,
  (it) => {
    if (!it) return
    strategy.value = 'overwrite'
    // 自动生成候选名
    let candidate = `${it.name}-2`
    let n = 2
    while (props.installed?.[candidate]) {
      n += 1
      candidate = `${it.name}-${n}`
    }
    newName.value = candidate
    // 重置 scope/group
    scope.value = 'global'
    projectId.value = 0
    groupPath.value = ''
    formError.value = ''
    newGroupOpen.value = false
    newGroupInput.value = ''
    newGroupErr.value = ''
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
  return true
})

function setScope(v) {
  scope.value = v
  if (v === 'global') projectId.value = 0
}

function close() {
  emit('update:modelValue', false)
  emit('cancel')
}

async function ensureTreeLoaded() {
  if (!skillTree.tree || skillTree.tree.length === 0) {
    try {
      await skillTree.load()
    } catch (e) {
      // 加载失败不阻塞弹窗
    }
  }
}

async function createNewGroup() {
  const path = newGroupInput.value.trim()
  if (!path) {
    newGroupErr.value = t('market.pullDialog.groupEmpty')
    return
  }
  try {
    const res = await apiCreateGroup({ group_path: path })
    const created = res?.group_path || path
    // 重新拉树(轻量)
    await skillTree.load()
    groupPath.value = created
    newGroupOpen.value = false
    newGroupInput.value = ''
    newGroupErr.value = ''
    emit('tree-changed')
  } catch (e) {
    newGroupErr.value = e?.message || String(e)
  }
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
    // 2026-07-01 重构:统一传 5 个工具,移除 UI 上的勾选交互。
    // 后端 PullV2 在 tools=[] 时只写盘不 apply,这里必须显式传非空数组。
    tools: [...ALL_TOOLS],
    finalName: finalName.value,
    groupPath: groupPath.value || '',
  })
  setTimeout(() => {
    submitting.value = false
  }, 100)
}

onMounted(() => {
  ensureTreeLoaded()
})
</script>

<template>
  <Modal
    :model-value="modelValue"
    size="md"
    :title="t('market.pullDialog.title', { name: item?.name || '' })"
    :close-on-mask="!submitting"
    @update:model-value="(v) => emit('update:modelValue', v)"
  >
    <template #title-icon>
      <Icon icon="mdi:download-outline" width="18" height="18" />
    </template>

    <div v-if="item" class="pull-form">
      <!-- 描述卡 -->
      <p v-if="item.description" class="pull-desc">
        <Icon icon="mdi:text-box-outline" width="14" height="14" class="pull-desc-icon" />
        <span>{{ item.description }}</span>
      </p>

      <!-- 重复检测卡片 -->
      <div v-if="isDuplicate" class="dup-card">
        <div class="dup-head">
          <div class="dup-icon">
            <Icon icon="mdi:alert-octagon-outline" width="16" height="16" />
          </div>
          <div class="dup-text">
            <div class="dup-title">
              {{ t('market.pullDialog.duplicateTitle', { name: item.name }) }}
            </div>
            <div class="dup-hint">{{ t('market.pullDialog.duplicateHint') }}</div>
          </div>
        </div>
        <div class="dup-actions">
          <button
            type="button"
            :class="['seg-btn', { active: strategy === 'overwrite' }]"
            @click="strategy = 'overwrite'"
          >
            <Icon icon="mdi:content-save-outline" width="14" height="14" />
            {{ t('market.pullDialog.btnOverwrite') }}
          </button>
          <button
            type="button"
            :class="['seg-btn', { active: strategy === 'saveAs' }]"
            @click="strategy = 'saveAs'"
          >
            <Icon icon="mdi:content-copy" width="14" height="14" />
            {{ t('market.pullDialog.btnSaveAs') }}
          </button>
        </div>
        <div v-if="strategy === 'saveAs'" class="dup-saveas">
          <label class="saveas-label">{{ t('market.pullDialog.newNameLabel') }}</label>
          <input
            v-model="newName"
            type="text"
            class="saveas-input"
            :placeholder="item.name + '-2'"
          />
          <p class="saveas-hint">{{ t('market.pullDialog.saveAsHint') }}</p>
        </div>
      </div>

      <!-- 作用域(segment control 风格) -->
      <div class="field">
        <label class="field-label">{{ t('market.pullDialog.scopeLabel') }}</label>
        <div class="scope-seg">
          <button
            type="button"
            :class="['seg-btn flex', { active: scope === 'global' }]"
            @click="setScope('global')"
          >
            <Icon icon="mdi:earth" width="14" height="14" />
            {{ t('market.scopeGlobal') }}
          </button>
          <button
            type="button"
            :class="['seg-btn flex', { active: scope === 'project' }]"
            @click="setScope('project')"
          >
            <Icon icon="mdi:folder-account-outline" width="14" height="14" />
            {{ t('market.scopeProject') }}
          </button>
        </div>
        <select
          v-if="scope === 'project'"
          v-model="projectId"
          class="form-select mt"
        >
          <option :value="0" disabled>{{ t('market.projectPlaceholder') }}</option>
          <option v-for="p in projects" :key="p.id" :value="p.id">
            {{ p.name || p.alias || ('#' + p.id) }}
          </option>
        </select>
      </div>

      <!-- 分组 -->
      <div class="field">
        <label class="field-label">{{ t('market.pullDialog.groupLabel') }}</label>
        <div class="group-row">
          <select v-model="groupPath" class="form-select">
            <option value="">{{ t('market.pullDialog.groupNone') }}</option>
            <option
              v-for="opt in groupOptions"
              :key="opt.value"
              :value="opt.value"
            >{{ opt.label }}</option>
          </select>
          <button
            type="button"
            class="ghost sm inline-flex"
            :title="t('market.pullDialog.btnNewGroup')"
            @click="newGroupOpen = !newGroupOpen"
          >
            <Icon icon="mdi:folder-plus-outline" width="12" height="12" />
            {{ t('market.pullDialog.btnNewGroup') }}
          </button>
        </div>
        <div v-if="newGroupOpen" class="group-create">
          <input
            v-model="newGroupInput"
            type="text"
            class="form-input"
            :placeholder="t('market.pullDialog.groupPlaceholder')"
            @keyup.enter="createNewGroup"
          />
          <button
            type="button"
            class="primary sm inline-flex"
            :disabled="!newGroupInput.trim()"
            @click="createNewGroup"
          >
            <Icon icon="mdi:check" width="12" height="12" />
          </button>
          <button
            type="button"
            class="ghost sm inline-flex"
            @click="newGroupOpen = false; newGroupInput = ''"
          >
            <Icon icon="mdi:close" width="12" height="12" />
          </button>
        </div>
        <p v-if="newGroupErr" class="form-hint form-hint-error">{{ newGroupErr }}</p>
        <p v-else class="form-hint">{{ t('market.pullDialog.groupHint') }}</p>
      </div>

      <!-- 「将自动启用到」信息条(2026-07-01 重构:替代原多选) -->
      <div class="apply-info">
        <div class="apply-info-icon">
          <Icon icon="mdi:rocket-launch-outline" width="14" height="14" />
        </div>
        <div class="apply-info-text">
          <div class="apply-info-title">{{ t('market.pullDialog.applyToTitle') }}</div>
          <div class="apply-info-tools">
            <span v-for="tool in ALL_TOOLS" :key="tool" class="tool-badge">{{ tool }}</span>
          </div>
          <div class="apply-info-hint">{{ t('market.pullDialog.applyToHint') }}</div>
        </div>
      </div>

      <div v-if="formError" class="form-error">{{ formError }}</div>
    </div>

    <template #footer>
      <button type="button" class="ghost inline-flex" :disabled="submitting" @click="close">
        <Icon icon="mdi:close" width="14" height="14" />
        {{ t('market.pullDialog.btnCancel') }}
      </button>
      <button
        type="button"
        class="primary inline-flex"
        :disabled="!canConfirm || submitting"
        @click="onConfirm"
      >
        <span v-if="submitting" class="spinner"></span>
        <Icon v-else icon="mdi:download" width="14" height="14" />
        {{ submitting ? t('market.pullDialog.pulling') : t('market.pullDialog.confirm') }}
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.pull-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
  font-size: 13px;
}

/* 描述卡 */
.pull-desc {
  margin: 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: var(--text-dim);
  line-height: 1.55;
  padding: 10px 12px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
}
.pull-desc-icon {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--text-faint);
}

/* 重复检测卡片(更醒目) */
.dup-card {
  border: 1px solid var(--warning);
  background: var(--warning-dim);
  border-radius: var(--radius);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dup-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}
.dup-icon {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--warning);
  color: var(--bg-card);
  border-radius: 50%;
}
.dup-text {
  flex: 1;
  min-width: 0;
}
.dup-title {
  font-weight: 600;
  color: var(--warning);
  font-size: 13px;
  line-height: 1.4;
}
.dup-hint {
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1.5;
  margin-top: 2px;
}

.dup-actions {
  display: flex;
  gap: 6px;
  padding: 3px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.dup-saveas {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.saveas-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-dim);
}
.saveas-input {
  padding: 11px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-family: 'JetBrains Mono', monospace;
  background: var(--bg-card);
  color: var(--text);
  line-height: 1.4;
}
.saveas-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.saveas-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

/* 通用 field */
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
}

/* segment control(作用域) */
.scope-seg {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  gap: 2px;
}
.seg-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px 16px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
}
.seg-btn:hover:not(.active) {
  background: var(--bg-card);
  color: var(--text);
}
.seg-btn.active {
  background: var(--bg-card);
  color: var(--text);
  border-color: var(--border);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}
.seg-btn.flex {
  flex: 1;
}

/* 表单控件(2026-07-01:加大高度 + chevron,显得大气不局促) */
.form-select,
.form-input {
  padding: 11px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 14px;
  min-width: 200px;
  background: var(--bg-card);
  color: var(--text);
  font-family: inherit;
  line-height: 1.4;
}
.form-select {
  flex: 1;
  min-width: 0;
  /* 自绘 chevron(2026-07-01),更精致,不再用浏览器默认丑下拉箭头 */
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  padding-right: 36px;
  background-image: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%2364748b' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><polyline points='6 9 12 15 18 9'/></svg>");
  background-repeat: no-repeat;
  background-position: right 12px center;
  background-size: 14px 14px;
  cursor: pointer;
}
.form-select:focus,
.form-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-dim);
}
.form-select:hover:not(:disabled) {
  border-color: var(--text-faint);
}
.mt {
  margin-top: 4px;
}

.group-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-create {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}
.group-create .form-input {
  flex: 1;
  min-width: 0;
}

/* 「将自动启用到」信息卡 */
.apply-info {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  background: var(--primary-dim, rgba(99, 102, 241, 0.08));
  border: 1px solid var(--primary-border, rgba(99, 102, 241, 0.25));
  border-radius: var(--radius);
}
.apply-info-icon {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--primary);
  color: var(--bg-card);
  border-radius: 50%;
}
.apply-info-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.apply-info-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
}
.apply-info-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.tool-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  background: var(--bg-card);
  border: 1px solid var(--primary-border, rgba(99, 102, 241, 0.25));
  border-radius: 999px;
  font-size: 11px;
  font-weight: 500;
  color: var(--primary);
  font-family: 'JetBrains Mono', monospace;
}
.apply-info-hint {
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}

/* 通用按钮尺寸 */
button.sm {
  padding: 8px 12px;
  font-size: 13px;
}

/* footer 按钮覆盖全局 ghost / primary 透明 + 居中 */
button.inline-flex {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.form-hint {
  margin: 0;
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.4;
}
.form-hint-error {
  color: var(--danger);
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
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  opacity: 0.7;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
