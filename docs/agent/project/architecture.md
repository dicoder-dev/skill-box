# Architecture(Claude 视角)

> 只装 Claude 改代码时必须知道的关键节点,详细原理看 `docs/project/项目架构.md`。

## 顶层分模块

```
main.go                     桌面端入口
api-server/                 后端(模块名 ginp-api,go.mod 在这里)
├── cmd/web/                Web 单进程入口
├── cmd/bootstrap/          cfg→DB→Task→Logger→Server 启动流程
├── internal/gapi/          业务 API(controllers / services / entities / router)
├── internal/db/            数据库初始化(GORM 多驱动)
├── pkg/                    可复用公共库(ginp / cfg / dbops / where ...)
└── configs/                配置结构体定义

desktop/                    Wails 层(windows / menu / tray / services)
frontend/                   Vue 3 SPA(dist 通过 //go:embed 进 Go 二进制)
build/                      跨平台 Taskfile(windows/darwin/linux/ios/android)
```

## 改前必知的几个不变量

1. **业务路由必须在 SPA fallback 之前注册**(`api-server/cmd/bootstrap/server.go:New`)。
   改 `router.Register` 顺序 → 业务路由被前端 catch-all 抢走。
2. **配置文件 `init()` 比 `cfg.InitCfg()` 跑得早**。`api-server/cmd/bootstrap/bootstrap.go:Boot()`
   里有 5 个 struct 的 reparse,**别删**;删了会导致 `configs.yaml` 不生效。
3. **`pkg/ginp` 的 `ContextPlus` 不要换成 `*gin.Context`**。所有 controller 都基于它,
   改它会全量冲击。
4. **Wails bindings 不要手改**(`frontend/bindings/` 与 `frontend/src/api/`)。
   改完 `wails3 generate bindings` 会覆盖。
5. **frontend/dist 是嵌入路径**。改前端后:
   - 桌面端:`wails3 dev` 自动监听
   - Web 端:`npm run build` 后 `wails3 task web:sync:embed` 同步到 `api-server/cmd/web/frontend/dist`

## 关键调用链

```
HTTP 请求
  → gin.Logger / Recovery
  → mountStatic(/assets, /static)
  → CORSMiddleware
  → router.Register(/api/* 业务路由)
  → AuthUserCenterMiddleware(NeedLogin=true 才查 JWKS)
  → 权限校验(NeedPermission=true)
  → controller(解析参数 → service → c.SuccessData / c.Fail)
```

## 前端启动链

```
main.js
  → 读 window.__APP_RUNTIME__(同步,后端注入)
  → 探测 window.go(Wails 绑定)
  → resolveBaseURL()   ← Web:空 / Desktop:127.0.0.1:<port>
  → 写 useAppStore
  → enableDebug()?(Vite dev / ?debug=req)
  → 探测 /api/health
  → app.mount('#app')
```

业务侧只 `import { http } from '@/core/utils/requests'`,**不要直接 fetch**。