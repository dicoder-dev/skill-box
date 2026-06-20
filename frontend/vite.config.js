import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// 注意:Wails v3 的 wails 插件需要 bindings 目录(由 `wails generate` 生成)。
// 我们走"双部署 + 走 HTTP"的方案后,业务完全脱离 Wails 绑定,只保留
// 桌面能力(window.go.app/window.go.desktop/window.go.platform)的小部分手写绑定。
// 后续如需重新启用类型生成,可跑 `wails generate module` 重新生成 ./bindings。
export default defineConfig({
  plugins: [vue()],
});
