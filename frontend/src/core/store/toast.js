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

import { defineStore } from 'pinia'

let _seq = 0

export const useToastStore = defineStore('toast', {
  state: () => ({
    items: [], // { id, type, message, duration, createdAt }
  }),
  actions: {
    // push 一条 toast;type: success | error | info
    push({ type = 'info', message = '', duration } = {}) {
      if (!message) return
      _seq += 1
      const item = {
        id: _seq,
        type,
        message: String(message),
        duration: duration ?? (type === 'error' ? 5000 : 3000),
        createdAt: Date.now(),
      }
      this.items.push(item)
      // 上限保护:超过 5 条就把最早的挤掉
      while (this.items.length > 5) this.items.shift()
      // 自动消失
      setTimeout(() => this.dismiss(item.id), item.duration)
      return item.id
    },
    success(message, duration) { return this.push({ type: 'success', message, duration }) },
    error(message, duration)   { return this.push({ type: 'error',   message, duration }) },
    info(message, duration)    { return this.push({ type: 'info',    message, duration }) },
    dismiss(id) {
      this.items = this.items.filter((x) => x.id !== id)
    },
    clear() { this.items = [] },
  },
})
