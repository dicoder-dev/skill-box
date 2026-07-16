// core/composables/useResizablePanel.js
//
// 可复用「面板拖拽改宽」composable —— 把「拖拽把手改宽度 + localStorage 持久化 +
// min/max 限幅」封装成统一 API，供多个位置复用：
//   1. SkillsView   左侧技能列表（grid 第一列列宽，target='grid-col'）
//   2. SkillFileInlinePanel 目录树（自身 flex 宽度，target='flex-width'）
//   3. CodeViewer   右侧大纲（写 CSS 变量，target='css-var'，沿用 --ai-panel-w 范式）
//
// 三种写入模式统一为「往某个 DOM 写样式」：
//   - 'grid-col'   → applyTo.value.style.setProperty(cssVar||'--panel-w', w+'px')
//                    （父容器 grid-template-columns 引用该变量）
//   - 'flex-width' → applyTo.value.style.width = w+'px'
//   - 'css-var'    → (scopeEl.value||documentElement).style.setProperty(cssVar, w+'px')
//
// 拖拽方向 direction：
//   - 'right' 把手在面板右边界，向右拖 = 增宽（左侧面板用）
//   - 'left'  把手在面板左边界，向左拖 = 增宽（右侧面板用，如大纲）
//
// 持久化键：skillbox:panel:<storageKey>，存像素数字字符串；读时 clamp 到 [min,max]。
//
// 暴露：
//   - width:     Ref<number>   当前宽度（响应式，用于 aria-valuenow 展示）
//   - dragging:  Ref<boolean>  是否正在拖拽（把手视觉态）
//   - startDrag: (e) => void   把手 @mousedown 绑定
//   - reset:     () => void    恢复默认宽度（可绑 @dblclick）
//   - sync:      () => void    按当前 width 写一次样式（onMounted 调用，setup 时 DOM 未挂载）
//
// 用法见 SkillsView / SkillFileInlinePanel / CodeViewer 三处调用点。

import { ref, onBeforeUnmount } from 'vue'

const PREFIX = 'skillbox:panel:'

function clamp(v, lo, hi) {
  return Math.min(hi, Math.max(lo, v))
}

function readStorage(key, fallback, lo, hi) {
  try {
    const raw = localStorage.getItem(PREFIX + key)
    if (raw === null) return fallback
    const n = Number(raw)
    return Number.isFinite(n) ? clamp(n, lo, hi) : fallback
  } catch (_) {
    return fallback
  }
}

function writeStorage(key, v) {
  try {
    localStorage.setItem(PREFIX + key, String(v))
  } catch (_) {
    /* localStorage 不可用（隐私模式 / 磁盘满）时静默 */
  }
}

export function useResizablePanel(options) {
  const {
    target = 'flex-width',
    direction = 'right',
    storageKey,
    defaultWidth,
    min,
    max,
    applyTo,                    // grid-col / flex-width 时必填（面板/容器 ref）
    cssVar = '--panel-w',       // grid-col / css-var 时使用的 CSS 变量名
    scopeEl,                    // css-var 时可选，默认写到 document.documentElement
  } = options

  if (!storageKey || typeof defaultWidth !== 'number') {
    throw new Error('[useResizablePanel] storageKey 与 defaultWidth 必填')
  }

  const width = ref(readStorage(storageKey, defaultWidth, min, max))
  const dragging = ref(false)
  let lastX = 0

  // 按当前 width 值写入目标 DOM
  function apply(v) {
    const clamped = clamp(v, min, max)
    width.value = clamped
    writeStorage(storageKey, clamped)
    if (target === 'flex-width') {
      if (applyTo?.value) applyTo.value.style.width = clamped + 'px'
    } else if (target === 'grid-col') {
      // 父容器 grid-template-columns 引用 cssVar，这里只改变量值
      if (applyTo?.value) applyTo.value.style.setProperty(cssVar, clamped + 'px')
    } else if (target === 'css-var') {
      const root = scopeEl?.value || document.documentElement
      root.style.setProperty(cssVar, clamped + 'px')
    }
  }

  // 初始同步：setup 阶段 DOM 可能未挂载，交给调用方在 onMounted 里调一次
  function sync() {
    apply(width.value)
  }

  function onMove(e) {
    if (!dragging.value) return
    const dx = e.clientX - lastX
    lastX = e.clientX
    // right：向右拖增宽；left：向左拖增宽
    apply(width.value + (direction === 'left' ? -dx : dx))
  }

  function onUp() {
    if (!dragging.value) return
    dragging.value = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.removeProperty('user-select')
    document.body.style.removeProperty('cursor')
  }

  function startDrag(e) {
    e.preventDefault()
    dragging.value = true
    lastX = e.clientX
    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
    // 拖拽期间禁止选中文本，并全程保持列宽调整光标
    document.body.style.userSelect = 'none'
    document.body.style.cursor = 'col-resize'
  }

  function reset() {
    apply(defaultWidth)
  }

  onBeforeUnmount(() => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.removeProperty('user-select')
    document.body.style.removeProperty('cursor')
  })

  return { width, dragging, startDrag, reset, sync }
}
