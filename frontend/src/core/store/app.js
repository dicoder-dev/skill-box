// store/app.js - 全局运行时 store。
//
// 集中管理:
//   - runMode     web / desktop(由后端在 index.html 注入到 window.__APP_RUNTIME__)
//   - needAuth    是否启用 JWT 鉴权
//   - appName     应用名
//   - baseURL     后端 baseURL(Web 为空 / 桌面为 http://127.0.0.1:port)
//
// 平台形态(runMode)是后端按启动命令注入的:
//   - wails3 dev / 桌面二进制 → runMode = "desktop"
//   - go run ./cmd/web / Web 单进程二进制 → runMode = "web"
//
// 所有"是否桌面端"的判断都走 runMode,不要再单独探测 window.go,避免和后端注入不一致。
//
// main.js bootstrap 时一次性写入;之后业务组件用 useAppStore() 取用。
// 不持久化(运行时配置由后端在 index.html 里注入)。

import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', {
  state: () => ({
    runMode: 'web',
    needAuth: true,
    appName: '',
    baseURL: '',
  }),
  getters: {
    isWeb: (s) => s.runMode === 'web',
    isDesktop: (s) => s.runMode === 'desktop',
    authEnabled: (s) => s.needAuth,
    // 给 UI 用的展示名
    deployLabel: (s) => (s.runMode === 'desktop' ? '桌面端' : 'Web'),
  },
  actions: {
    setRuntime(rt) {
      if (!rt) return
      if (rt.runMode) this.runMode = rt.runMode
      if (typeof rt.needAuth === 'boolean') this.needAuth = rt.needAuth
      if (typeof rt.appName === 'string') this.appName = rt.appName
    },
    setBaseURL(u) {
      this.baseURL = u || ''
    },
  },
})
