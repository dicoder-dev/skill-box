<script setup>
// CollapsiblePanel - 通用可折叠面板(2026-07-17 增)
//
// 2026-07-17 设计:
//   - 标题样式按作用域面板 .ssp-scope-header 来(浅背景 + 左侧 icon + 标题文字
//     + 右侧 meta slot + chevron)
//   - 默认折叠,点 header 展开,跟作用域一致
//   - 暴露 v-model:expanded 双向绑定,父级可控制
//   - 提供 #title-meta 插槽放 badge/count 等右侧附加信息
//   - 提供 #default 插槽放面板内容
//   - 自身不带业务逻辑(状态/数据由调用方管)
//
// 2026-07-17 改:加 panelId + inject 协调器,实现"一次只能展开一个"
// 的 accordion 行为。同一 parent 树内,任一面板被展开时其他已展开面板自动折叠。
// 不传 panelId 则走普通独立模式(向后兼容)。

import { computed, inject } from 'vue'
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
  // 2026-07-17 增:面板 ID,用于 accordion 协调。不传 = 独立面板,跟其他面板互不干扰。
  panelId: { type: String, default: '' },
})
const emit = defineEmits(['update:expanded', 'toggle'])

// 注入 accordion 协调器;父组件用 <AccordionGroup> 包一层即可启用。
// 没包时 coordinator = null,行为跟旧版一致。
const coordinator = inject('cpCoordinator', null)

const chevronIcon = computed(() => (props.expanded ? 'mdi:minus' : 'mdi:plus'))

function toggle() {
  const next = !props.expanded
  if (coordinator && props.panelId) {
    // 走协调器 — 通知父级更新 activeId
    coordinator.toggle(props.panelId, next)
  } else {
    emit('update:expanded', next)
  }
  emit('toggle', next)
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
      <IconPark v-if="icon" :icon="icon" :size="13" />
      <span class="cp-title">{{ title }}</span>
      <span class="cp-meta">
        <slot name="title-meta" />
      </span>
      <IconPark :icon="chevronIcon" :size="13" class="cp-chevron" />
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
  margin-top: 2px;
  overflow: hidden;
}

.cp-header {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 10px;
  background: var(--bg-elevated-strong, rgba(127, 127, 127, 0.08));
  border: 0;
  border-bottom: 1px solid var(--border-color, rgba(127, 127, 127, 0.2));
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary, currentColor);
  text-align: left;
}
.cp-header:hover { background: var(--bg-elevated-stronger, rgba(127, 127, 127, 0.14)); }

.cp-title { flex: 0 0 auto; }

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