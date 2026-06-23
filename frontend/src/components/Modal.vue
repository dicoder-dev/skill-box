<script setup>
/**
 * 通用模态弹窗组件
 * - 遮罩层 + 内容区
 * - 支持 ESC 关闭、点击遮罩关闭、滚动锁定
 * - 支持 size(sm/md/lg/xl/full) 与自定义 maxWidth
 * - 暴露 header / body / footer 三个 slot
 */
import { watch, onBeforeUnmount } from 'vue'
import { Icon } from '@iconify/vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  size: { type: String, default: 'md' }, // sm | md | lg | xl | full
  maxWidth: { type: String, default: '' }, // 覆盖 size 的最大宽度,例如 '720px'
  closeOnMask: { type: Boolean, default: true },
  closeOnEsc: { type: Boolean, default: true },
  showClose: { type: Boolean, default: true },
  // 锁定 body 滚动(默认开)
  lockScroll: { type: Boolean, default: true },
})

const emit = defineEmits(['update:modelValue', 'close', 'open'])

const sizeMap = {
  sm: '420px',
  md: '560px',
  lg: '760px',
  xl: '960px',
  full: 'min(96vw, 1200px)',
}

function close() {
  emit('update:modelValue', false)
  emit('close')
}

function onMaskClick() {
  if (props.closeOnMask) close()
}

function onKey(e) {
  if (!props.modelValue) return
  if (e.key === 'Escape' && props.closeOnEsc) close()
}

// 锁定 / 解锁 body 滚动
let savedOverflow = ''
function setLock(lock) {
  if (!props.lockScroll) return
  if (typeof document === 'undefined') return
  if (lock) {
    savedOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = savedOverflow || ''
  }
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      setLock(true)
      if (typeof window !== 'undefined') window.addEventListener('keydown', onKey)
      emit('open')
    } else {
      setLock(false)
      if (typeof window !== 'undefined') window.removeEventListener('keydown', onKey)
    }
  },
)

onBeforeUnmount(() => {
  setLock(false)
  if (typeof window !== 'undefined') window.removeEventListener('keydown', onKey)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal-mask" @click.self="onMaskClick">
        <div
          class="modal-container"
          :style="{ maxWidth: maxWidth || sizeMap[size] }"
          role="dialog"
          aria-modal="true"
        >
          <header v-if="title || showClose || $slots.header" class="modal-header">
            <slot name="header">
              <h3 class="modal-title">
                <slot name="title-icon" />
                {{ title }}
              </h3>
            </slot>
            <button
              v-if="showClose"
              class="modal-close"
              type="button"
              :aria-label="$t ? $t('common.close') : 'Close'"
              @click="close"
            >
              <Icon icon="mdi:close" width="18" height="18" />
            </button>
          </header>

          <div class="modal-body">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="modal-footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 8vh 16px 16px;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(2px);
  overflow: auto;
}

.modal-container {
  width: 100%;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: 0 24px 48px -12px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  max-height: 84vh;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.modal-title {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.modal-close {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-dim);
  border-radius: var(--radius-sm);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.modal-close:hover {
  background: var(--bg-hover);
  color: var(--text);
  border-color: var(--border);
}

.modal-body {
  flex: 1;
  min-height: 0;
  padding: 20px;
  overflow: auto;
  color: var(--text);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  background: var(--bg-subtle);
  flex-shrink: 0;
}

/* 进出动画 */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.18s ease;
}

.modal-enter-active .modal-container,
.modal-leave-active .modal-container {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-container,
.modal-leave-to .modal-container {
  opacity: 0;
  transform: translateY(-8px) scale(0.98);
}

@media (max-width: 640px) {
  .modal-mask {
    padding: 4vh 12px 12px;
  }
  .modal-container {
    max-height: 92vh;
  }
}

/* 暗黑模式遮罩更暗 */
:global(.dark) .modal-mask {
  background: rgba(0, 0, 0, 0.6);
}
</style>
