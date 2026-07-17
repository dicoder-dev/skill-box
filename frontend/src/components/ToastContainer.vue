<script setup>
// ToastContainer - 全局 toast 浮层。
//
// 挂载位置:App.vue 末尾(z-index 最高,fixed 右上角)。
// 数据源:useToastStore。
// 行为:监听 store.items 变化,渲染栈;每条带 200ms 淡入/淡出动画;
// 鼠标悬停时暂停自动消失(避免用户看不全),离开后继续倒计时(简化:
// 这里没做精确暂停,直接复用 store 的 setTimeout;若 hover 时快到点了
// 用户体验上可接受,后续按需扩展)。
//
// 视觉:简洁 — 圆角 + 浅色背景 + 左侧语义色条 + 右侧关闭按钮。
// 不引入第三方 ui 库,沿用项目 --success / --danger / --accent-blue 变量。

import { useToastStore } from '@/core/store/toast'
import IconPark from '@/components/IconPark.vue'

const toast = useToastStore()

const ICON_MAP = {
  success: 'mdi:check-circle-outline',
  error:   'mdi:alert-circle-outline',
  info:    'mdi:information-outline',
}
</script>

<template>
  <div class="toast-stack" aria-live="polite" aria-atomic="false">
    <transition-group name="toast">
      <div
        v-for="item in toast.items"
        :key="item.id"
        :class="['toast-item', `toast-${item.type}`, item.strong ? 'toast-strong' : '']"
        role="status"
      >
        <IconPark :icon="ICON_MAP[item.type] || ICON_MAP.info" width="16" height="16" class="toast-icon" />
        <span class="toast-message">{{ item.message }}</span>
        <!-- 2026-07-18 增:action 按钮 — 用户点 action 时执行 onClick 并立即
             dismiss 当前 toast。label 用主色,点击区域 ≥ 24px 适配触屏。 -->
        <button
          v-if="item.action"
          class="toast-action"
          @click="() => { try { item.action.onClick && item.action.onClick() } finally { toast.dismiss(item.id) } }"
        >
          {{ item.action.label }}
        </button>
        <button class="toast-close" :aria-label="'close'" @click="toast.dismiss(item.id)">
          <IconPark icon="mdi:close" width="12" height="12" />
        </button>
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 1000; /* 高于 topbar(z-20) 和 modal(默认 z-50) */
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 360px;
  pointer-events: none; /* 让空白处不挡下方点击 */
}

.toast-item {
  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  font-size: 13px;
  color: var(--text);
  min-width: 240px;
  /* 左侧 3px 语义色条 */
  border-left-width: 3px;
}

/* 语义色:成功 / 失败 / 信息 */
.toast-success { border-left-color: var(--success); }
.toast-success .toast-icon { color: var(--success); }
.toast-error   { border-left-color: var(--danger); }
.toast-error   .toast-icon { color: var(--danger); }
.toast-info    { border-left-color: var(--accent-blue); }
.toast-info    .toast-icon { color: var(--accent-blue); }

.toast-message {
  flex: 1;
  min-width: 0;
  word-break: break-word;
  white-space: pre-wrap;
}

.toast-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: transparent;
  border: none;
  color: var(--text-faint);
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
}
.toast-close:hover { background: var(--bg-hover); color: var(--text); }

/* 2026-07-18 增:strong 变体 — 背景填语义色 + 白色字 + 强调阴影,
   用于"测试通过"这类需要高优可见的成功提示。
   只在 toast-strong + toast-success 同时存在时生效,避免普通成功 toast 被误染。 */
.toast-item.toast-strong.toast-success {
  background: var(--success);
  border-color: var(--success);
  color: #fff;
  box-shadow: 0 6px 20px rgba(34, 197, 94, 0.25);
}
.toast-item.toast-strong.toast-success .toast-icon { color: #fff; }
.toast-item.toast-strong.toast-success .toast-close { color: rgba(255, 255, 255, 0.85); }
.toast-item.toast-strong.toast-success .toast-close:hover { background: rgba(255, 255, 255, 0.15); color: #fff; }

/* 2026-07-18 增:action 按钮 — 跟 toast 同色系,按下立刻 dismiss */
.toast-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 24px;
  padding: 0 10px;
  background: rgba(255, 255, 255, 0.18);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: inherit;
  font-size: 12px;
  font-weight: 600;
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.12s ease;
}
.toast-action:hover { background: rgba(255, 255, 255, 0.32); }
/* 非 strong 变体的 toast:action 用主色文本 + 主色浅底 */
.toast-item:not(.toast-strong) .toast-action {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.3);
  color: var(--primary);
}
.toast-item:not(.toast-strong) .toast-action:hover {
  background: rgba(59, 130, 246, 0.18);
}

/* transition-group 动画 */
.toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.toast-enter-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
.toast-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.toast-move {
  transition: transform 0.2s ease;
}
</style>
