// core/composables/useMdOutlineVisible.js
//
// md 文件大纲面板的全局显示状态(布尔):
//   - true  → 显示(默认)
//   - false → 隐藏
//
// 状态用 localStorage 持久化(`skillbox.mdOutlineVisible`),跨刷新 + 跨文件保留:
// 用户收起大纲后,打开任何 md 文件、刷新页面、再打开别的 md 文件都默认收起。
// 跟 CodeViewer 的 mdHeadings 互相独立(本 composable 只管"面板是否显示",
// 不管当前文件有没有标题)。
//
// 暴露:
//   - outlineVisible: Ref<boolean> 当前状态
//   - toggleOutline(): void 切换并持久化
//   - showOutline(): void   强制显示
//   - hideOutline(): void   强制隐藏
//
// 用法:
//   import { outlineVisible, toggleOutline } from '@/core/composables/useMdOutlineVisible'
//   // 模板里直接 v-show="outlineVisible" / @click="toggleOutline"

import { ref, watch } from 'vue'

const STORAGE_KEY = 'skillbox.mdOutlineVisible'

// 全局单例 ref(整个 app 共享一个状态)
const _visible = ref(readInitial())

function readInitial() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    // 没存过 → 默认 true(显示大纲);存过 → 用存的值
    if (v === null) return true
    return v === '1' || v === 'true'
  } catch (_) {
    return true
  }
}

function writeStorage(v) {
  try {
    localStorage.setItem(STORAGE_KEY, v ? '1' : '0')
  } catch (_) { /* localStorage 不可用(隐私模式/磁盘满)时静默 */ }
}

// 持久化 watch(只触发一次,后续状态变更都跟着写)
watch(_visible, (v) => writeStorage(v))

export function useMdOutlineVisible() {
  return {
    outlineVisible: _visible,
    toggleOutline: () => { _visible.value = !_visible.value },
    showOutline: () => { _visible.value = true },
    hideOutline: () => { _visible.value = false },
  }
}
