// core/composables/useRightPanelMode.js
//
// 文件工具栏右侧面板的三态全局显示状态:
//   'outline' → 显示 md 大纲(原 useMdOutlineVisible 的等价行为)
//   'ai'      → 显示 AI 对话面板(新)
//   'none'    → 两边都不显示(用户主动隐藏)
//
// 持久化:
//   - 新键 skillbox.rightPanelMode(写入 'outline' / 'ai' / 'none')
//   - 旧键 skillbox.mdOutlineVisible(一次性迁移:旧 false → none,旧 true → outline)
//     旧键读一次后写新键并废弃,旧键值后续不再读取。
//
// 暴露:
//   - mode:           Ref<'outline' | 'ai' | 'none'>
//   - setMode(m):     void  直接设值(由 emit('update:right-panel-mode', m) 触发)
//   - toggleOutline():void  outline ↔ none
//   - toggleAI():     void  ai ↔ none
//   - showOutline():  void  强制 outline
//   - showAI():       void  强制 ai
//   - hidePanel():    void  强制 none
//   - outlineActive:  computed, 派生只读 mode === 'outline'
//   - aiActive:       computed, 派生只读 mode === 'ai'
//
// 用法(CodeViewer / SkillFileInlinePanel / AIRightPanel 共享同一个 Ref):
//   import { useRightPanelMode } from '@/core/composables/useRightPanelMode'
//   const { mode, showAI, hidePanel, aiActive } = useRightPanelMode()

import { ref, computed, watch } from 'vue'

const STORAGE_KEY_NEW = 'skillbox.rightPanelMode'
const STORAGE_KEY_OLD = 'skillbox.mdOutlineVisible'
const VALID_MODES = new Set(['outline', 'ai', 'none'])

function readInitial() {
  try {
    // 1) 优先读新键
    const v = localStorage.getItem(STORAGE_KEY_NEW)
    if (v && VALID_MODES.has(v)) return v
    // 2) 迁移旧键:旧 false → none,旧 true / 未设 → outline
    const old = localStorage.getItem(STORAGE_KEY_OLD)
    const migrated = (old === '0' || old === 'false') ? 'none' : 'outline'
    try { localStorage.setItem(STORAGE_KEY_NEW, migrated) } catch (_) { /* 写失败静默 */ }
    return migrated
  } catch (_) {
    // localStorage 不可用(隐私模式 / 磁盘满)→ 内存态,默认 outline
    return 'outline'
  }
}

function writeStorage(v) {
  try {
    if (VALID_MODES.has(v)) localStorage.setItem(STORAGE_KEY_NEW, v)
  } catch (_) { /* localStorage 不可用时静默 */ }
}

// 全局单例 ref(整个 app 共享一个状态)
const _mode = ref(readInitial())

// 持久化 watch(后续状态变更都跟着写)
watch(_mode, (v) => writeStorage(v))

export function useRightPanelMode() {
  return {
    mode: _mode,
    setMode: (m) => { if (VALID_MODES.has(m)) _mode.value = m },
    // 兼容旧 useMdOutlineVisible 的语义:仅在 outline ↔ none 之间切换
    toggleOutline: () => { _mode.value = _mode.value === 'outline' ? 'none' : 'outline' },
    toggleAI:      () => { _mode.value = _mode.value === 'ai' ? 'none' : 'ai' },
    showOutline:   () => { _mode.value = 'outline' },
    showAI:        () => { _mode.value = 'ai' },
    hidePanel:     () => { _mode.value = 'none' },
    outlineActive: computed(() => _mode.value === 'outline'),
    aiActive:      computed(() => _mode.value === 'ai'),
  }
}