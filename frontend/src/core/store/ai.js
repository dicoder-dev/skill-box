// core/store/ai.js - AI 对话持久化 store(2026-07-14 增)
//
// 跨组件共享当前活跃会话 + 跨刷新持久化到 localStorage +
// 跨设备/双写到后端 .skill-box/history.json。
//
// 设计要点:
//   - 按 sourcePath 维度切分多个会话(同一个 AI 面板会切不同文件)
//   - currentSourcePath 为空时,所有 push/patch no-op + 历史按钮禁用
//   - 写盘是双份:localStorage 立刻同步;后端 800ms debounce
//   - historyDialogOpen / historyItems 由 AIRightPanel 直接绑定,避免引入额外 dispatcher
//
// 用法:
//   import { useAiStore } from '@/core/store/ai'
//   const ai = useAiStore()
//   ai.hydrate()
//   ai.setCurrentSource(props.filePath)
//   const user = ai.pushUser(text)
//   const aiMsg = ai.pushAssistantPlaceholder()
//   ai.patchAssistant(aiMsg.id, { content, status: 'streaming' })
//   ai.openHistory(); ai.closeHistory()
import { defineStore } from 'pinia'
import * as api from '@/api/skillbox/ai-history'

const STORAGE_KEY = 'skillbox.ai.v2.sessions'

let _uid = 0
function uid(prefix = 'm') {
  _uid += 1
  return `${prefix}_${Date.now()}_${_uid}`
}

export const useAiStore = defineStore('ai', {
  state: () => ({
    // 按 sourcePath(磁盘绝对路径)分组的会话
    sessions: {},
    // 当前 sourcePath('' = 没有文件,所有写操作 no-op)
    currentSourcePath: '',
    // 防抖写盘 timer
    saving: false,
    // 历史 Modal 相关
    historyDialogOpen: false,
    historyItems: [],
    loadingList: false,
  }),
  getters: {
    currentMessages(s) {
      return s.sessions[s.currentSourcePath]?.items || []
    },
    hasSession(s) {
      return !!s.currentSourcePath
    },
  },
  actions: {
    // 从 localStorage 水合(模块加载时或 onMounted 时调用一次)。
    hydrate() {
      try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (raw) {
          const obj = JSON.parse(raw)
          if (obj && typeof obj === 'object') {
            this.sessions = obj
          }
        }
      } catch (_) {
        // localStorage 不可用时静默
      }
    },
    // 写回 localStorage;quota exceeded 时静默。
    persistLocal() {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(this.sessions))
      } catch (_) {
        // 静默 — 写盘失败不影响 UI
      }
    },

    setCurrentSource(p) {
      this.currentSourcePath = p || ''
    },

    openHistory() {
      this.historyDialogOpen = true
    },
    closeHistory() {
      this.historyDialogOpen = false
    },

    // pushUser 把用户消息塞入当前会话。
    pushUser(text) {
      if (!this.currentSourcePath) return null
      const msg = {
        id: uid('m'),
        role: 'user',
        content: text,
        ts: Date.now(),
      }
      this._append(msg)
      return msg
    },

    // pushAssistantPlaceholder 创建一个 streaming 占位的 AI 消息。
    // 返回的 msg.id 后续 patchAssistant 用。
    pushAssistantPlaceholder() {
      if (!this.currentSourcePath) return null
      const msg = {
        id: uid('a'),
        role: 'assistant',
        content: '',
        reason: '',
        status: 'sending',
        pending: true,
        ts: Date.now(),
        canApply: false,
        needsApply: false,
        retriesLeft: 0,
      }
      this._append(msg)
      return msg
    },

    // patchAssistant 把字段增量更新到指定 AI 消息上。
    patchAssistant(id, patch) {
      if (!this.currentSourcePath) return
      const list = this.sessions[this.currentSourcePath]?.items
      if (!list) return
      const m = list.find((x) => x.id === id)
      if (!m) return
      Object.assign(m, patch)
      this.persistLocal()
      this._scheduleBackendSave()
    },

    setMessageApplied(id, v) {
      this.patchAssistant(id, { applied: v, canApply: !v, rejected: false })
    },
    setMessageRejected(id) {
      this.patchAssistant(id, { rejected: true, canApply: false })
    },
    setMessageRetrying(id, v) {
      this.patchAssistant(id, { retrying: v })
    },

    // 追加一条消息(私有)。
    _append(msg) {
      const k = this.currentSourcePath
      if (!this.sessions[k]) this.sessions[k] = { items: [], updatedAt: 0 }
      this.sessions[k].items.push(msg)
      this.sessions[k].updatedAt = Date.now()
      this.persistLocal()
      this._scheduleBackendSave()
    },

    // 清空当前活跃会话(消息全部删,前端 + 后端)。
    clearCurrent() {
      if (!this.currentSourcePath) return
      delete this.sessions[this.currentSourcePath]
      this.persistLocal()
      this._scheduleBackendSave({ clear: true })
    },

    // 从后端拉历史列表,塞到 historyItems。
    // 失败返空(静默不弹 toast)。
    async loadFromBackend() {
      if (!this.currentSourcePath) {
        this.historyItems = []
        return
      }
      this.loadingList = true
      try {
        const r = await api.listHistory({ source_path: this.currentSourcePath })
        this.historyItems = Array.isArray(r?.items) ? r.items : []
      } catch (_) {
        this.historyItems = []
      } finally {
        this.loadingList = false
      }
    },

    // 把选中的历史条目注入当前会话(覆盖式)。
    pickHistoryItem(item) {
      if (!this.currentSourcePath || !item) return
      const k = this.currentSourcePath
      this.sessions[k] = {
        items: Array.isArray(item.messages) ? JSON.parse(JSON.stringify(item.messages)) : [],
        updatedAt: Date.now(),
      }
      this.persistLocal()
      this.historyDialogOpen = false
    },

    // ----------------- 后端防抖写入 -----------------

    _saveTimer: null,
    _scheduleBackendSave({ clear } = {}) {
      if (this._saveTimer) clearTimeout(this._saveTimer)
      this._saveTimer = setTimeout(() => this.flushBackend({ clear }), 800)
    },
    async flushBackend({ clear } = {}) {
      this._saveTimer = null
      const k = this.currentSourcePath
      if (!k) return
      this.saving = true
      try {
        const items = clear ? [] : this.sessions[k]?.items || []
        // 转成 HistoryItem 形态给后端:每条需要 id/title/preview/ts/provider/model/messages。
        const out = items.map((m) => ({
          id: m.id,
          title: m.title || defaultTitle(m),
          preview: m.preview || defaultPreview(m),
          ts: m.ts || Date.now(),
          provider: m.provider || '',
          model: m.model || '',
          messages: Array.isArray(messagesOf(m)) ? messagesOf(m) : [],
        }))
        await api.saveHistory({ source_path: k, items: out })
      } catch (_) {
        // 写盘失败不弹 toast,UI 不阻塞
      } finally {
        this.saving = false
      }
    },
  },
})

function defaultTitle(m) {
  const txt = (m.content || '').slice(0, 40).trim()
  return txt || m.id || 'untitled'
}
function defaultPreview(m) {
  return (m.content || '').slice(0, 120).trim()
}
function messagesOf(m) {
  // 简化:把单条 AI 消息也包成 messages:[];前端后续可读全。
  return [
    { id: m.id, role: m.role, content: m.content, ts: m.ts },
  ]
}
