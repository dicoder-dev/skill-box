// store/update.js - 应用自身升级 store。
//
// 数据源:
//   - 后端 /api/desktop/update/{check,state,download} 三端点(Web 端 download 返 501,
//     只用 check 的 manifest/notes 走"去下载页"外链)。
//   - 上下文通过 @/platform 拿 runMode,Web 端用 platform.platform.openExternal 跳外链。
//
// 触发:
//   - App.vue onMounted 调一次 check()(启动后查最新);
//   - settings 里 "检查更新" 按钮也调 check();
//
// 状态机:
//   idle | checking | upToDate | available | downloading |
//   pendingRestart | failed | incomparable
//
// 进度由 backend state 接口轮询(每 2s,phase=downloading 时),停条件 phase≠downloading。

import { defineStore } from 'pinia'
import { http } from '@/core/utils/requests'
import { platform } from '@/platform'
import { getRuntime } from '@/core/utils/runtime'

const checkUrl = '/api/desktop/update/check'
const stateUrl = '/api/desktop/update/state'
const downloadUrl = '/api/desktop/update/download'

let _pollTimer = null

function normalizeBaseState(s) {
  if (!s) return 'upToDate'
  // 兼容 controller 返 upToDate / available / mustUpdate / incomparable
  return s
}

function pickNotes(notes, locale) {
  if (!notes || typeof notes !== 'object') return ''
  if (notes[locale]) return notes[locale]
  if (notes['en-US']) return notes['en-US']
  if (notes['zh-CN']) return notes['zh-CN']
  // 随便取第一个
  for (const k of Object.keys(notes)) {
    return notes[k]
  }
  return ''
}

export const useUpdateStore = defineStore('update', {
  state: () => ({
    state: 'idle',
    localVersion: '',
    remoteVersion: '',
    channel: 'stable',
    progress: 0,
    error: '',
    source: '',
    notes: '',
    manifest: null,
    hasUpdate: false,
    mustUpdate: false,
    downloadedPath: '',
    lastCheckedAt: 0,
  }),
  getters: {
    isDesktop(state) {
      return getRuntime().runMode === 'desktop'
    },
    message(state) {
      switch (state.state) {
        case 'checking':
          return '正在检查更新...'
        case 'upToDate':
          return state.localVersion
            ? `已是最新版本 ${state.localVersion}`
            : '已是最新版本'
        case 'available':
          return `发现新版本 ${state.remoteVersion}`
        case 'mustUpdate':
          return `必须升级到 ${state.remoteVersion}`
        case 'downloading':
          return `正在下载 ${state.progress}%`
        case 'pendingRestart':
          return '下载完成,软件将在重启后完成更新'
        case 'failed':
          return state.error || '升级失败'
        case 'incomparable':
          return `本地版本(${state.localVersion || '未知'})无法与远端(${state.remoteVersion || '?'})比较`
        default:
          return state.localVersion
            ? `当前版本 ${state.localVersion}`
            : '未检查'
      }
    },
  },
  actions: {
    async check() {
      this.state = 'checking'
      this.error = ''
      try {
        const resp = await http.get(checkUrl)
        const s = normalizeBaseState(resp?.status)
        this.localVersion = resp?.local_version || this.localVersion
        this.remoteVersion = resp?.remote_version || ''
        this.channel = resp?.channel || 'stable'
        this.mustUpdate = !!resp?.must_update
        this.hasUpdate = !!resp?.has_update
        this.manifest = resp?.assets ? resp : null
        this.notes = pickNotes(resp?.notes, getRuntime().lang || (typeof document !== 'undefined' ? (document.documentElement.lang || '') : ''))
        this.source = resp?.source || ''
        if (s === 'incomparable') {
          this.state = 'incomparable'
        } else if (s === 'upToDate') {
          this.state = 'upToDate'
        } else {
          this.state = this.hasUpdate ? (this.mustUpdate ? 'mustUpdate' : 'available') : 'upToDate'
        }
        this.lastCheckedAt = Date.now()
      } catch (e) {
        this.state = 'failed'
        this.error = e?.message || String(e)
      }
      return this.state
    },

    // 进入下载流(桌面端调 helper + Quit;Web 端打开 manifest 外链)。
    async download() {
      if (this.isDesktop) {
        this.state = 'downloading'
        this.progress = 0
        this.error = ''
        try {
          await http.post(downloadUrl, { old_version: this.localVersion })
          // 父进程随后 Quit,这条分支通常走不到
          this.state = 'pendingRestart'
          this.startPoll()
        } catch (e) {
          this.state = 'failed'
          this.error = e?.message || String(e)
        }
        return
      }
      // Web 端
      try {
        const urls = this.manifest?.assets?.flatMap(a => a?.urls || []) || []
        const firstUrl = urls[0] || ''
        if (!firstUrl) {
          this.error = '暂未配置下载链接'
          this.state = 'failed'
          return
        }
        await platform.platform.openExternal(firstUrl)
        this.state = 'available'
      } catch (e) {
        this.state = 'failed'
        this.error = e?.message || String(e)
      }
    },

    // 桌面端轮询进度(MVP 阶段简单实现,2s 一次,phase!=downloading 停)。
    async startPoll() {
      if (!this.isDesktop) return
      if (_pollTimer) {
        clearInterval(_pollTimer)
      }
      _pollTimer = setInterval(async () => {
        try {
          const r = await http.get(stateUrl)
          this.progress = typeof r?.progress === 'number' ? r.progress : this.progress
          const phase = r?.phase || 'idle'
          if (phase === 'downloading' || phase === 'verifying') {
            this.state = phase
          } else if (phase === 'pendingRestart') {
            this.state = 'pendingRestart'
          } else if (phase === 'failed') {
            this.state = 'failed'
            this.error = r?.error || '升级失败'
            this.stopPoll()
          } else if (phase === 'idle') {
            // 升级完成 → 父进程大概率已 Quit,不再 poll
            this.stopPoll()
          }
          if (r?.downloaded_path) {
            this.downloadedPath = r.downloaded_path
          }
        } catch (e) {
          // 网络错也只是下次再试,不打断
        }
      }, 2000)
    },
    stopPoll() {
      if (_pollTimer) {
        clearInterval(_pollTimer)
        _pollTimer = null
      }
    },

    // 重置(用户手动清状态)
    reset() {
      this.state = 'idle'
      this.progress = 0
      this.error = ''
      this.downloadedPath = ''
    },
  },
})
