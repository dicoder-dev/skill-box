// store/toast.js - 全局轻量 toast store。
//
// 设计目标:
//   - 任何组件能 push 一条 toast(成功/失败/信息)
//   - 容器(ToastContainer.vue)订阅这个 store,渲染右上角浮层
//   - 默认 3s 自动消失;错误类型默认 5s
//   - 同时存在的 toast 数量上限 5,超出最早的先被挤掉
//
// 用法:
//   import { useToastStore } from '@/core/store/toast'
//   const toast = useToastStore()
//   toast.push({ type: 'success', message: '已启用' })
//   toast.push({ type: 'error',   message: '启用失败:xxx' })
//   // 2026-07-18 增:action 按钮(toast.action)用于"跳到设置"这类引导;
//   // strong 变体(toast.strongSuccess / toast.strongInfo)用于"测试通过"这类
//   // 需要强调的成功,背景色用对应语义色,而不是浅卡底。

import { defineStore } from 'pinia'

let _seq = 0

export const useToastStore = defineStore('toast', {
  state: () => ({
    // 2026-07-18 改:items 加 strong + action 字段;action 是 { label, onClick }。
    // strong=true 表示背景填语义色(显眼变体),用于"测试通过"等需要强调的提示。
    items: [], // { id, type, message, duration, createdAt, strong, action }
  }),
  actions: {
    // push 一条 toast;type: success | error | info
    // options: { strong?: bool, action?: { label, onClick }, duration?: number }
    push({ type = 'info', message = '', duration, strong, action } = {}) {
      if (!message) return
      _seq += 1
      const item = {
        id: _seq,
        type,
        message: String(message),
        duration: duration ?? (type === 'error' ? 5000 : 3000),
        createdAt: Date.now(),
        strong: !!strong,
        action: action || null,
      }
      this.items.push(item)
      // 上限保护:超过 5 条就把最早的挤掉
      while (this.items.length > 5) this.items.shift()
      // 自动消失(strong 默认 5s — 给用户多 2s 看背景填色)
      const dur = item.strong ? Math.max(item.duration, 5000) : item.duration
      setTimeout(() => this.dismiss(item.id), dur)
      return item.id
    },
    success(message, duration) { return this.push({ type: 'success', message, duration }) },
    error(message, duration)   { return this.push({ type: 'error',   message, duration }) },
    info(message, duration)    { return this.push({ type: 'info',    message, duration }) },
    // 2026-07-18 增:strong 变体 — 背景填对应语义色(toast-strong-success 用绿色)
    strongSuccess(message, action, duration) {
      return this.push({ type: 'success', message, duration, strong: true, action })
    },
    // 2026-07-18 增:带 action 按钮的快捷方式(type 默认 info,蓝底)
    action(label, onClick, options = {}) {
      const { type = 'info', message = '', duration } = options
      return this.push({
        type,
        message: message || label,
        duration,
        action: { label, onClick },
      })
    },
    dismiss(id) {
      this.items = this.items.filter((x) => x.id !== id)
    },
    clear() { this.items = [] },
  },
})
