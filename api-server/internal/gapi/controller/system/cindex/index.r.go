package cindex

// 注意:此路由原为 Wails demo 占位,渲染 view/index.html 模板。
// 双部署架构下,首页统一由 pkg/server.NoRoute 接管为前端 SPA 入口,
// 因此这里不再注册 GET /,避免与前端根路径冲突。
// 原 IndexView 函数仍保留在 index.c.go,以便需要 ViewGlob 模板时回滚。