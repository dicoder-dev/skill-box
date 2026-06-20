// store/app.js - 全局运行时 store。
//
// 集中管理:
//   - runMode     web / desktop(由后端注入)
//   - needAuth    是否启用 JWT 鉴权
//   - appName     应用名
//   - isDesktop   是否桌面端(由 platform 检测)
//   - baseURL     后端 baseURL(Web 为空 / 桌面为 http://127.0.0.1:port)
//
// main.js bootstrap 时一次性写入;之后业务组件用 useAppStore() 取用。
// 不持久化(运行时配置由后端在 index.html 里注入)。

import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', {
  state: () => ({
    runMode: 'web',
    needAuth: true,
    appName: '',
    isDesktop: false,
    baseURL: '',
  }),
  getters: {
    isWeb: (s) => s.runMode === 'web',
    isDesktopMode: (s) => s.runMode === 'desktop',
    authEnabled: (s) => s.needAuth,
    // 给 UI 用的展示名
    deployLabel: (s) => (s.isDesktop ? '桌面端' : 'Web'),
  },
  actions: {
    setRuntime(rt) {
      if (!rt) return
      if (rt.runMode) this.runMode = rt.runMode
      if (typeof rt.needAuth === 'boolean') this.needAuth = rt.needAuth
      if (typeof rt.appName === 'string') this.appName = rt.appName
    },
    setPlatform(p) {
      this.isDesktop = !!p?.isDesktop
    },
    setBaseURL(u) {
      this.baseURL = u || ''
    },
  },
})