<script setup>
// CollapsiblePanel - 通用可折叠面板(2026-07-17 增)
//
// 用途:统一 skill 详情区的"作用域"和"Git 同步"两个面板的标题/折叠交互样式,
// 避免每个面板各自实现一遍 .ssp-*-header + chevron 切换。
//
// 2026-07-17 设计:
//   - 标题样式按作用域面板 .ssp-scope-header 来(浅背景 + 左侧 icon + 标题文字
//     + 右侧 meta slot + chevron)
//   - 默认折叠,点 header 展开,跟作用域一致
//   - 暴露 v-model:expanded 双向绑定,父级可控制
//   - 提供 #title-meta 插槽放 badge/count 等右侧附加信息
//   - 提供 #default 插槽放面板内容
//   - 自身不带业务逻辑(状态/数据由调用方管)

import { computed } from 'vue'
import IconPark from '@/components/IconPark.vue'

const props = defineProps({
  // 是否展开(v-model)
  expanded: { type: Boolean, default: false },
  // 标题(必填,直接 string)
  title: { type: String, required: true },
  // 标题左侧 iconpark icon 名(可选)
  icon: { type: String, default: '' },
  // 折叠时也强制渲染 body(给"必看"信息用;默认 false 折叠时不渲染,省 DOM)
  forceMount: { type: Boolean, default: false },
})
const emit = defineEmits(['update:expanded', 'toggle'])

const chevronIcon = computed(() => (props.expanded ? 'mdi:minus' : 'mdi:plus'))

function toggle() {
  emit('update:expanded', !props.expanded)
  emit('toggle', !props.expanded)
}
</script>

<template>
  <section class="cp" :class="{ 'is-expanded': expanded }">
    <button
      type="button"
      class="cp-header"
      :aria-expanded="expanded"
      @click="toggle"
    >
      <IconPark v-if="icon" :type="icon" :size="13" />
      <span class="cp-title">{{ title }}</span>
      <span class="cp-meta">
        <slot name="title-meta" />
      </span>
      <IconPark :type="chevronIcon" :size="13" class="cp-chevron" />
    </button>
    <div v-if="expanded || forceMount" class="cp-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
/* 2026-07-17 增:通用可折叠面板。
   标题样式沿用作用域 .ssp-scope-header(浅背景 + icon + 标题 + meta + chevron),
   业务组件用 <CollapsiblePanel> 时不再各自实现 header 样式。 */

.cp {
  border: 1px solid var(--border-color, rgba(127, 127, 127, 0.15));
  border-radius: var(--radius, 6px);
  background: var(--bg-elevated, rgba(255, 255, 255, 0.02));
  margin-top: 8px;
}

.cp-header {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 10px;
  background: transparent;
  border: 0;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary, currentColor);
  text-align: left;
}
.cp-header:hover { background: rgba(127, 127, 127, 0.04); }

.cp-title {
  flex: 0 0 auto;
}

.cp-meta {
  flex: 1 1 auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cp-chevron {
  flex: 0 0 auto;
  color: var(--text-muted, rgba(127, 127, 127, 0.7));
  display: inline-flex;
  align-items: center;
}

.cp-body {
  padding: 8px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
</style>