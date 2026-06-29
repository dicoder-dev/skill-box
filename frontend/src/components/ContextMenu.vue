<script setup>
/**
 * ContextMenu 轻量自研右键菜单组件
 *
 * 2026-06-29 增:为支持首页 skill 树形 UI 的右键操作。零依赖,数据驱动。
 *
 * 特性:
 *   - Teleport 到 body,fixed 定位
 *   - 点击外部 / ESC 关闭(emit close)
 *   - 屏幕边界自动翻转(右/下越界时反向)
 *   - items: [{ key, label, icon, danger, disabled, divided, onClick }]
 *   - 支持键盘 Enter 触发当前 hover 项
 *
 * 用法:
 *   <ContextMenu
 *     v-if="ctx.open"
 *     :x="ctx.x" :y="ctx.y" :items="ctx.items"
 *     @close="ctx.open = false"
 *   />
 */
import { ref, watch, onBeforeUnmount, nextTick } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  x: { type: Number, required: true },
  y: { type: Number, required: true },
  items: { type: Array, default: () => [] },
  // 触发元素的 ref,用于让 clickoutside 区分"右键触发元素"和"外部"
  anchor: { type: Object, default: null },
})

const emit = defineEmits(['close'])

// 内部状态:计算后的 left/top(翻转后)+ 当前 hover 项索引
const style = ref({ left: '0px', top: '0px' })
const menuRef = ref(null)
const hoverIndex = ref(-1)

// 计算位置 + 翻转
function calcPos() {
  const MENU_W = 200 // 估算菜单宽度(实际由 min-width 控制)
  const MENU_H = props.items.length * 32 + 16 // 估算高度
  const PAD = 8 // 离屏幕边缘的最小间距
  let x = props.x
  let y = props.y
  if (typeof window !== 'undefined') {
    const vw = window.innerWidth
    const vh = window.innerHeight
    if (x + MENU_W + PAD > vw) {
      x = Math.max(PAD, vw - MENU_W - PAD)
    }
    if (y + MENU_H + PAD > vh) {
      y = Math.max(PAD, vh - MENU_H - PAD)
    }
  }
  style.value = { left: `${x}px`, top: `${y}px` }
}

// 首次挂载后定位;items 变化时重新定位(高度可能变)
watch(
  () => [props.x, props.y, props.items],
  () => { nextTick(calcPos) },
  { immediate: true },
)

function onItemClick(item) {
  if (item.disabled) return
  try {
    item.onClick?.()
  } finally {
    emit('close')
  }
}

function onMouseEnter(idx) {
  hoverIndex.value = idx
}

function onKeydown(e) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    emit('close')
    return
  }
  if (e.key === 'Enter' && hoverIndex.value >= 0) {
    const item = props.items[hoverIndex.value]
    if (item) onItemClick(item)
  }
}

// 全局 clickoutside:点击菜单外(包括 anchor 自身)关闭
function onDocClick(e) {
  const menu = menuRef.value?.$el || menuRef.value
  if (!menu) return
  if (menu.contains(e.target)) return
  // anchor(右键触发的元素):点击它不算"外部"(因为有些交互是"右键 → 弹菜单 → 再点同一项取消")
  if (props.anchor && props.anchor.contains?.(e.target)) return
  emit('close')
}

function onDocContextMenu(e) {
  // 在菜单显示期间如果用户再次右键,关闭当前菜单(避免多层菜单叠加)
  emit('close')
}

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('contextmenu', onDocContextMenu, true)
  document.removeEventListener('keydown', onKeydown, true)
})

// 挂载后绑事件
nextTick(() => {
  document.addEventListener('mousedown', onDocClick, true)
  document.addEventListener('contextmenu', onDocContextMenu, true)
  document.addEventListener('keydown', onKeydown, true)
})
</script>

<template>
  <Teleport to="body">
    <div ref="menuRef" class="ctx-menu" :style="style" role="menu" @contextmenu.prevent>
      <template v-for="(it, idx) in items" :key="it.key || idx">
        <div v-if="it.divided" class="ctx-divider"></div>
        <button
          type="button"
          role="menuitem"
          :class="[
            'ctx-item',
            it.danger ? 'ctx-item-danger' : '',
            it.disabled ? 'ctx-item-disabled' : '',
            hoverIndex === idx ? 'ctx-item-hover' : '',
          ]"
          :disabled="it.disabled"
          @click="onItemClick(it)"
          @mouseenter="onMouseEnter(idx)"
          @mouseleave="hoverIndex = -1"
        >
          <Icon v-if="it.icon" :icon="it.icon" width="14" height="14" class="ctx-item-icon" />
          <span class="ctx-item-label">{{ it.label }}</span>
          <span v-if="it.shortcut" class="ctx-item-shortcut">{{ it.shortcut }}</span>
        </button>
      </template>
    </div>
  </Teleport>
</template>

<style scoped>
.ctx-menu {
  position: fixed;
  z-index: 1000;
  min-width: 180px;
  max-width: 260px;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12), 0 2px 6px rgba(0, 0, 0, 0.06);
  display: flex;
  flex-direction: column;
  gap: 2px;
  user-select: none;
}

.ctx-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color 0.1s ease, color 0.1s ease;
  text-align: left;
  width: 100%;
  outline: none;
}
.ctx-item-hover {
  background: var(--bg-hover);
}
.ctx-item-icon {
  color: var(--text-dim);
  flex-shrink: 0;
}
.ctx-item-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ctx-item-shortcut {
  font-size: 11px;
  color: var(--text-faint);
  font-family: 'JetBrains Mono', monospace;
  flex-shrink: 0;
}

.ctx-item-danger .ctx-item-icon,
.ctx-item-danger .ctx-item-label { color: var(--danger); }
.ctx-item-danger.ctx-item-hover { background: var(--danger-dim); }

.ctx-item-disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ctx-divider {
  height: 1px;
  margin: 4px 6px;
  background: var(--border);
}
</style>
