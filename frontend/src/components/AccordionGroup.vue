<script setup>
// AccordionGroup - 协调一组 CollapsiblePanel,实现"一次只能展开一个"
//
// 2026-07-17 增:用 provide/inject 模式,不直接绑数据,CollapsiblePanel
// 通过 inject('cpCoordinator') 拿到 toggle 入口。
//
// 用法:
//   <AccordionGroup v-model:active="activePanelId">
//     <CollapsiblePanel panel-id="scope" ... />
//     <CollapsiblePanel panel-id="git" ... />
//     <CollapsiblePanel panel-id="history" ... />
//   </AccordionGroup>
//
// 行为:点击任一面板 header → 展开它,其他自动折叠;
// activePanelId = null 时全部折叠;非 null 时该 ID 对应面板展开。
//
// 注意:GitSyncPanel / ScopePanel 自己管 expanded 状态,
// 在 AccordionGroup 里用 v-model:active 接管。

import { provide, reactive, watch } from 'vue'

const props = defineProps({
  // 当前展开的面板 ID,v-model 双向绑定;null = 全折叠
  active: { type: String, default: null },
})
const emit = defineEmits(['update:active'])

// 全局协调器对象(给 CollapsiblePanel 注入用)
const state = reactive({
  activeId: props.active,
  // toggle 由 CollapsiblePanel 调用;同 ID 重复点 = 折叠(再次变 null),
  // 切到其他 ID = 折叠当前 + 展开新 ID。
  toggle(panelId, open) {
    let next
    if (open) {
      next = panelId
    } else {
      // 关闭自己:如果当前 active 就是这个 ID,置 null
      next = state.activeId === panelId ? null : state.activeId
    }
    if (next !== state.activeId) {
      state.activeId = next
      emit('update:active', next)
    }
  },
})

provide('cpCoordinator', state)

// 外部 v-model:active 改变时同步到内部
watch(() => props.active, (v) => {
  if (v !== state.activeId) state.activeId = v
})
</script>

<template>
  <div class="accordion-group">
    <slot />
  </div>
</template>

<style scoped>
.accordion-group {
  display: contents;
}
</style>