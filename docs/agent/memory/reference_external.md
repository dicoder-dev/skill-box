---
name: 外部资源指针
description: 项目相关的外部系统 / 文档 / 看板 / 平台地址
type: reference
---

## 文档

- 项目业务文档根:`docs/project/`(`README.md` 是索引)
- 仓库贡献指南:根目录 `AGENTS.md`
- Task 任务清单:根目录 `Taskfile.yml`
- 运行时配置:根目录 `configs.yaml` / `configs.e2e.yaml`

## 框架文档

- Wails v3:`https://v3.wails.io/`
- Gin:`https://gin-gonic.com/zh-cn/docs/`
- GORM v2:`https://gorm.io/zh_CN/docs/`
- Vite:`https://cn.vitejs.dev/`
- Pinia:`https://pinia.vuejs.org/zh/`
- Task:`https://taskfile.dev/`

## 内部子项目文档

- 后端框架封装:`api-server/pkg/ginp/README.md` / `README_zh.md`
- 通用查询条件:`api-server/pkg/where/`
- 通用 DAO:`api-server/pkg/dbops/`

## 配置 / 鉴权依赖

- 用户中心 JWKS:`<system.user_center_url>/.well-known/jwks.json`
  (具体 URL 在 `configs.yaml` 的 `system.user_center_url`)

## 构建产物

- 跨平台 Taskfile:`build/<windows|darwin|linux|ios|android>/Taskfile.yml`
- 编译产物目录:`bin/`(gitignored)

---

**新增原则:** Claude 在项目里发现新的外部资源(看板 / 文档站点 / API),
主动追加到这里。