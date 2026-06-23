import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath, URL } from "node:url";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

// 注意:Wails v3 的 wails 插件需要 bindings 目录(由 `wails generate` 生成)。
// 我们走"双部署 + 走 HTTP"的方案后,业务完全脱离 Wails 绑定,只保留
// 桌面能力(window.go.app/window.go.desktop/window.go.platform)的小部分手写绑定。
// 后续如需重新启用类型生成,可跑 `wails generate module` 重新生成 ./bindings。

// 从仓库根 configs.yaml 读取后端端口,保证 vite 代理与后端监听端口一致。
// 解析失败 / 字段缺失时兑底为 8082;也可通过 WEB_API_PORT 环境变量显式覆盖。
const FALLBACK_API_PORT = 8082;
const CONFIG_FILE = resolve(dirname(fileURLToPath(import.meta.url)), "..", "configs.yaml");

function readServerPortFromYaml(text) {
  // 只匹配 server 段下的 port 字段,避免误命中 db.mysql.port 等同名字段。
  const m = text.match(/^server:\s*$([\s\S]*?)(?=^\S|\Z)/m);
  if (!m) return null;
  const inner = m[1];
  const port = inner.match(/^\s*port:\s*["'\u2018\u2019]?(\d+)["'\u2018\u2019]?\s*$/m);
  return port ? parseInt(port[1], 10) : null;
}

function resolveBackendPort() {
  const envPort = process.env.WEB_API_PORT;
  if (envPort && /^\d+$/.test(envPort)) return parseInt(envPort, 10);
  try {
    const text = readFileSync(CONFIG_FILE, "utf-8");
    const p = readServerPortFromYaml(text);
    if (p && p > 0) return p;
  } catch (_) {
    // 读不到就兑底
  }
  return FALLBACK_API_PORT;
}

const backendPort = resolveBackendPort();
const backendTarget = `http://127.0.0.1:${backendPort}`;

// 桌面端 dev 模式识别:由启动命令注入 VITE_DEPLOY_MODE=desktop。
// wails3 dev 启动 Vite 时通常不会自动设这个变量,所以 wails 的 dev 任务
// 应当显式 `VITE_DEPLOY_MODE=desktop vite` 来开启桌面形态。
// 未设置时兑底为 web(默认行为,不影响 Web 单进程开发)。
const deployMode = (process.env.VITE_DEPLOY_MODE || "web").toLowerCase();
const runtimeScript = `<script>window.__APP_RUNTIME__=${JSON.stringify({
  runMode: deployMode === "desktop" ? "desktop" : "web",
  // 桌面 dev 模式下后端已经 SetDesktopHooks 注入了真能力,前端可以直接走。
  needAuth: true,
  appName: "skill-box",
})};</script>`;

export default defineConfig({
  // 把 deployMode 暴露给前端代码:platform/index.js 在拿不到 __APP_RUNTIME__
  // 时(SSR / 早期报错)也能读到正确的 runMode。
  define: {
    "import.meta.env.VITE_RUN_MODE": JSON.stringify(deployMode === "desktop" ? "desktop" : "web"),
  },
  plugins: [
    vue(),
    {
      // dev 模式下,直接把 __APP_RUNTIME__ 注入到 index.html。
      // 之所以不靠后端 gin 注入:wails3 dev 的 webview 加载 Vite dev server,
      // 不经过 gin,所以后端 injectRuntimeScript 永远不会被调用。
      name: "inject-app-runtime",
      apply: "serve",
      transformIndexHtml() {
        return [
          {
            tag: "script",
            injectTo: "head-prepend",
            children: runtimeScript.replace(/^<script>|<\/script>$/g, ""),
          },
        ];
      },
    },
  ],
  resolve: {
    alias: {
      // @ → frontend/src,业务侧 import { http } from '@/core/utils/requests'
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    // Web 模式下,resolveBaseURL() 返回 "" 走同源,这里把 /api/* 转发到后端,
    // 避免浏览器请求打到 vite dev server 自己(否则要么 404,要么被 SPA 兑底返回 index.html)。
    proxy: {
      "/api": {
        target: backendTarget,
        changeOrigin: true,
      },
    },
  },
});
