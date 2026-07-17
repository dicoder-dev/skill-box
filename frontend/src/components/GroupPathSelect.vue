<script setup>
// GroupPathSelect - 目标分组下拉选择器(2026-07-18 增)。
//
// 放 OnboardingImportDialog 顶部 tab 上方一行,4 tab 共享同一选中态;
// 默认根分组(空字符串 = 走后端默认派生 / store 根目录)。
//
// 数据源:useSkillTreeStore().tree(嵌套 TreeNode 数组),递归 walk
// is_group=true 的节点收集所有 group path。
//
// 注:如果 tree 还没 load(用户首次打开弹窗时),只显示「根分组」一项;
// tree 加载完后会自动展开(因为是 computed)。

import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSkillTreeStore } from '@/core/store/skill-tree'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const { t } = useI18n()
const tree = useSkillTreeStore()

// 递归收集所有 group path(按层级缩进显示 label)。
// 空 path 跳过(根节点 path 是 ''),不重复列出。
function collectGroups(nodes, out) {
  if (!Array.isArray(nodes)) return
  for (const n of nodes) {
    if (n && n.is_group && n.path) {
      out.push(n)
      if (Array.isArray(n.children) && n.children.length) {
        collectGroups(n.children, out)
      }
    } else if (n && Array.isArray(n.children) && n.children.length) {
      // 兼容:有些后端实现 group 节点的 is_group 可能不准,保险起见也递归
      collectGroups(n.children, out)
    }
  }
}

// 收集到的 group 列表(扁平)
const allGroups = computed(() => {
  const out = []
  collectGroups(tree.tree, out)
  return out
})

// 计算每个 group 的"显示层级深度"用于缩进
function depthOf(node) {
  if (!node || !node.path) return 0
  // path 用 '/' 分隔,层级 = 段数
  return String(node.path).split('/').length - 1
}

// 选项列表:根分组(空) + 所有 group path
const options = computed(() => {
  const root = { value: '', label: t('onboarding.targetGroup.root'), depth: 0 }
  const groups = allGroups.value.map((n) => ({
    value: n.path,
    label: n.path,
    depth: depthOf(n),
  }))
  return [root, ...groups]
})

function onChange(e) {
  emit('update:modelValue', e.target.value)
}

// 首次 mount 时,如果 store 里 tree 为空就触发一次 load,
// 保证 4 tab 切换前 GroupPathSelect 已有数据可枚举。
// (SkillList 主动调 reload 时也会顺带刷新 tree)
onMounted(async () => {
  if (!tree.tree || tree.tree.length === 0) {
    try {
      await tree.load({})
    } catch (e) {
      // 静默失败,只显示「根分组」一项即可
    }
  }
})
</script>

<template>
  <div class="gps">
    <label class="gps-label" for="gps-select">
      <IconPark icon="mdi:folder-arrow-right-outline" width="13" height="13" />
      {{ t('onboarding.targetGroup.label') }}
    </label>
    <div class="gps-select-wrap">
      <select
        id="gps-select"
        :value="modelValue"
        class="gps-select"
        :title="t('onboarding.targetGroup.hint')"
        @change="onChange"
      >
        <option v-for="opt in options" :key="opt.value || '__root__'" :value="opt.value">
          {{ opt.depth > 0 ? '└─ '.repeat(opt.depth) + opt.label : opt.label }}
        </option>
      </select>
    </div>
    <span class="gps-hint">{{ t('onboarding.targetGroup.hint') }}</span>
  </div>
</template>

<style scoped>
.gps {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  margin-bottom: 14px;
  background: var(--surface-2, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border, #2a2a2a);
  border-radius: 8px;
}

.gps-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text, #ddd);
  white-space: nowrap;
  cursor: pointer;
}

.gps-select-wrap {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
}

/* 2026-07-18 改:加深的下拉箭头 indicator,让用户一眼看出是可下拉控件 */
.gps-select-wrap::after {
  content: '';
  position: absolute;
  right: 12px;
  top: 50%;
  width: 8px;
  height: 8px;
  border-right: 2px solid var(--text, #ddd);
  border-bottom: 2px solid var(--text, #ddd);
  transform: translateY(-65%) rotate(45deg);
  pointer-events: none;
  opacity: 0.7;
}

.gps-select {
  /* 2026-07-18 改:加宽 padding + 加深背景 + 加粗字体 + 自绘下拉箭头空间 */
  width: 100%;
  padding: 9px 32px 9px 12px;
  background: var(--bg, #1a1a1a);
  border: 1px solid var(--border-strong, #404040);
  border-radius: 6px;
  color: var(--text, #eee);
  font: inherit;
  font-size: 13.5px;
  font-weight: 500;
  outline: none;
  cursor: pointer;
  /* 隐藏原生箭头(用自绘的右上角 chevron) */
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.gps-select:hover {
  border-color: var(--accent-blue, #3b82f6);
}
.gps-select:focus {
  border-color: var(--accent-blue, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.18);
}

.gps-hint {
  font-size: 11.5px;
  color: var(--text-dim, #999);
  white-space: nowrap;
}
</style>
