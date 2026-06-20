// platform/index.js - 平台能力抽象。
//
// Web 端:返回 web 实现(全部 no-op 或抛"不支持")。
// 桌面端:返回 desktop 实现,通过 window.go.* Wails 绑定调桌面能力。
//
// 业务代码统一 import { platform } from '@/platform' 使用,
// 永远不要直接读 window.go.* —— 这样以后从桌面切换到 Web、或反过来,不用改业务。

const isDesktop = typeof window !== 'undefined' && !!window?.go?.app?.AppService

function createWebPlatform() {
  return {
    isDesktop: false,
    app: {
      async getVersion() { return 'web' },
      async getServerPort() { return 0 },
      async health() { return 'web' },
      async quit() { /* no-op */ },
    },
    window: {
      async toggleAlwaysOnTop() { return false },
      async show() { /* no-op */ },
      async toggleMaximise() { /* no-op */ },
    },
    platform: {
      os: () => 'web',
      arch: () => 'web',
      async clipboardText() { return '' },
      async setClipboardText() { return false },
      async openExternal(url) {
        // Web 端打开外链直接用 window.open
        window.open(url, '_blank', 'noopener')
      },
    },
  }
}

function createDesktopPlatform() {
  return {
    isDesktop: true,
    app: {
      async getVersion() {
        return window.go.app.AppService.GetVersion()
      },
      async getServerPort() {
        return window.go.app.AppService.GetServerPort()
      },
      async health() {
        return window.go.app.AppService.Health()
      },
      async quit() {
        return window.go.app.AppService.Quit()
      },
    },
    window: {
      async toggleAlwaysOnTop() {
        return window.go.desktop.WindowService.ToggleAlwaysOnTop()
      },
      async show() {
        return window.go.desktop.WindowService.Show()
      },
      async toggleMaximise() {
        return window.go.desktop.WindowService.ToggleMaximise()
      },
    },
    platform: {
      os: () => window.go.platform.PlatformService.OS(),
      arch: () => window.go.platform.PlatformService.Arch(),
      async clipboardText() {
        return window.go.platform.PlatformService.ClipboardText()
      },
      async setClipboardText(text) {
        return window.go.platform.PlatformService.SetClipboardText(text)
      },
      async openExternal(url) {
        return window.go.platform.PlatformService.OpenExternal(url)
      },
    },
  }
}

export const platform = isDesktop ? createDesktopPlatform() : createWebPlatform()
