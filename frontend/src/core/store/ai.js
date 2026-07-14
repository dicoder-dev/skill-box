// core/store/ai.js - AI 对话持久化 store(2026-07-14 v2)
//
// v2 改动要点(从 v1 升):
//   - key 用磁盘绝对路径(source_path),不是文件相对路径(filePath) —— 修 v1 双源 bug。
//   - 活跃会话只走 localStorage(草稿);"+ 新建对话" 才触发 archiveCurrent
//     → 后端 .skill-box/history/<conv-id>.json 单条 upsert(权威)。
//   - 列表用 ConvMeta(metadata-only);点开条目再 async getHistory 拉完整 messages 装填。
//   - 删除单条直接 deleteHistory(API v2)。
//
// 用法:
//   import { useAiStore } from '@/core/store/ai'
//   const ai = useAiStore()
//   ai.hydrate()
//   ai.setCurrentSource(props.sourcePath)        // 绝对路径!
//   const user = ai.pushUser(text)
//   const aiMsg = ai.pushAssistantPlaceholder()
//   ai.patchAssistant(aiMsg.id, { content, status: 'streaming' })
//
//   // 显式归档(用户点 "+ 新建对话" 时):
//   try { await ai.archiveCurrent() } catch (e) { toast.error(...) }
//   // 选历史条目:
//   await ai.pickHistoryItem(meta)
//   // 删单条:
//   await ai.deleteHistoryItem(convId)
import { defineStore } from 'pinia'
import * as api from '@/api/skillbox/ai-history'

const STORAGE_KEY = 'skillbox.ai.v2.sessions'

let _msgSeq = 0
let _convSeq = 0

function uid(prefix = 'm') {
  // 唯一 id:prefix_<ts>_<seq>;单纯 Date.now() 同毫秒下可能撞 id
  const seq = prefix === 'conv' ? _convSeq++ : _msgSeq++
  return `${prefix}_${Date.now()}_${seq}`
}

export const useAiStore = defineStore('ai', {
  state: () => ({
    // 按 source_path(磁盘绝对路径)分组的活跃会话。
    // 每条是 { items: Msg[], convId: string, updatedAt: number }。
    // convId 用于归档时复用 id(同一会话多次 archiveCurrent 会 upsert 同一文件)。
    sessions: {},
    // 当前 source_path('' = 没选 skill 或 AIRightPanel 还没挂上,所有写 no-op)
    currentSourcePath: '',
    // 历史 Modal
    historyDialogOpen: false,
    historyItems: [], // ConvMeta[] (metadata-only)
    loadingList: false,
    savingConv: false, // archiveCurrent 进行中
  }),
  getters: {
    currentMessages(s) {
      return s.sessions[s.currentSourcePath]?.items || []
    },
    hasSession(s) {
      return !!s.currentSourcePath
    },
    hasCurrentContent(s) {
      return (s.sessions[s.currentSourcePath]?.items || []).length > 0
    },
  },
  actions: {
    // ===================== 本地草稿(localStorage) =====================

    // hydrate 启动水合一次。
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

    // persistLocal 写回 localStorage;quota exceeded 时静默。
    persistLocal() {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(this.sessions))
      } catch (_) {
        /* 静默 */
      }
    },

    setCurrentSource(p) {
      this.currentSourcePath = p || ''
    },

    // ===================== 消息构造(本组件) =====================

    // pushUser 把用户消息塞入当前活跃会话(items),返回完整 msg。
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

    // pushAssistantPlaceholder 创建一个 streaming 占位的 AI 消息,返回 msg。
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

    // patchAssistant 把字段增量更新到指定 AI 消息上(自动 persistLocal)。
    patchAssistant(id, patch) {
      if (!this.currentSourcePath) return
      const sess = this.sessions[this.currentSourcePath]
      if (!sess) return
      const m = sess.items.find((x) => x.id === id)
      if (!m) return
      Object.assign(m, patch)
      sess.updatedAt = Date.now()
      this.persistLocal()
    },

    setMessageApplied(id, v) {
      this.patchAssistant(id, { applied: v, canApply: !v, rejected: false })
    },
    setMessageRejected(id) {
      this.patchAssistant(id, { rejected: true, canApply: false })
    },

    _append(msg) {
      const k = this.currentSourcePath
      if (!this.sessions[k]) {
        this.sessions[k] = { items: [], convId: uid('conv'), updatedAt: 0 }
      }
      this.sessions[k].items.push(msg)
      this.sessions[k].updatedAt = Date.now()
      this.persistLocal()
    },

    // ===================== v2 历史 API(archive / list / pick / delete) =====================

    // archiveCurrent ★v2 新增 — 把当前活跃会话序列化成一条 HistoryItem,
    // upsert 到后端 .skill-box/history/<conv-id>.json,然后清空本地活跃。
    //
    // 行为契约(用户已确认 2026-07-14):
    //   - 当前会话为空 → no-op,不创建空文件
    //   - 无 source_path → no-op
    //   - 先本地清空 sessions[k](UI 立即空白)→ 再请求后端;失败 → 抛错,调用点 toast
    //
    // async,失败抛错(caller 用 try/catch + toast)。
    async archiveCurrent() {
      const k = this.currentSourcePath
      if (!k) return
      const sess = this.sessions[k]
      const items = sess?.items || []
      if (items.length === 0) return

      const firstUser = items.find((m) => m.role === 'user')
      const convId = sess.convId || uid('conv')
      // 优先从最近一条 assistant 拿 provider/model(若有)。
      const lastAi = [...items].reverse().find((m) => m.role === 'assistant')
      const item = {
        id: convId,
        title: (firstUser?.content || '').slice(0, 30).trim() || 'untitled',
        preview: '', // 后端 previewFromMessages 自动算
        ts: Date.now(),
        provider: lastAi?.provider || '',
        model: lastAi?.model || '',
        messages: JSON.parse(JSON.stringify(items)), // 深拷贝完整字段
      }
      // ★ 先本地清(决策已确认)
      delete this.sessions[k]
      this.persistLocal()

      this.savingConv = true
      try {
        await api.saveHistory({ source_path: k, item })
      } catch (e) {
        // 失败:本地已清,云端未存,抛给调用点 toast。
        // 状态:无补回,用户后续手动重新建会话或重试。
        throw e
      } finally {
        this.savingConv = false
      }
    },

    // loadFromBackend 拉远端 metadata 列表 → historyItems。
    // 失败静默:把 loadingList 关闭,historyItems 留空(UI 不阻塞)。
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

    // pickHistoryItem ★v2 异步 — meta 是 metadata-only,
    // 先 GET 单条拉完整 messages,再装填到当前活跃 sessions。
    // 抛错给调用点 toast。
    async pickHistoryItem(meta) {
      if (!this.currentSourcePath || !meta?.id) return
      this.savingConv = true
      try {
        const r = await api.getHistory({
          source_path: this.currentSourcePath,
          conv_id: meta.id,
        })
        const item = r?.item
        if (!item || !Array.isArray(item.messages)) {
          throw new Error('empty or malformed conv')
        }
        this.sessions[this.currentSourcePath] = {
          items: JSON.parse(JSON.stringify(item.messages)),
          convId: item.id,
          updatedAt: Date.now(),
        }
        this.persistLocal()
        this.historyDialogOpen = false
      } catch (e) {
        throw e
      } finally {
        this.savingConv = false
      }
    },

    // deleteHistoryItem 按 conv_id 删单条;成功则 historyItems 移除该 conv。
    // 失败抛错给调用点 toast。
    async deleteHistoryItem(convId) {
      if (!this.currentSourcePath || !convId) return
      await api.deleteHistory({
        source_path: this.currentSourcePath,
        conv_id: convId,
      })
      this.historyItems = this.historyItems.filter((it) => it.id !== convId)
    },

    // ===================== Modal helpers =====================

    openHistory() {
      this.historyDialogOpen = true
    },
    closeHistory() {
      this.historyDialogOpen = false
    },
  },
})
