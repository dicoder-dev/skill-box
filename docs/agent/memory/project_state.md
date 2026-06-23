---
name: skill-box 项目当前状态
description: 项目目标 / 阶段 / 关键决策 — 用于理解当下在做什么
type: project
---

## 一句话

把 skill-box 从 "Wails 桌面 + Gin 后端 + Vue 3 SPA" 升级为 **AI 编程工具
skill 统一管理桌面应用**,12 步落地(详见 `docs/project/需求规划.md`)。

## 当前阶段

- 状态:**已落地 web 单进程形态**,桌面端能力补齐中
- 进度:见 `docs/project/进度.md`(权威)
- 关键技术决策:
  - Web 端走单进程(`api-server/cmd/web`)+ `//go:embed frontend/dist`
  - 桌面端走 Wails v3,HTTP server 进程内起,Webview 加载 `http://127.0.0.1:<port>`
  - 配置单点入口 `configs.yaml`(`cfg` 包统一加载)

## 关键约束

- 桌面端默认禁用鉴权(`system.need_auth: false`)
- Web 端默认启用鉴权,依赖外部 user_center JWKS
- 前端 dist 通过 embed 嵌入,前后端必须同 PR 发布

## 近期重点(下次会话回来时该关心什么)

1. **Wails bindings 重新生成** — 改完 `window.go / services/...` 后必须跑 `wails3 generate bindings`
2. **configs.yaml 不生效 bug** — 不要删 `api-server/cmd/bootstrap/bootstrap.go:Boot()` 里的 reparse(详见 `docs/project/开发报告.md` §8.4.1)
3. **GetBool bug** — 不要回退 `cfg.GetBool` 到旧版白名单实现(详见 `docs/project/开发报告.md` §8.4.2)

**Why:** 这些是已经踩过坑、付出过定位成本的关键节点,新会话不需要再踩一次。

**How to apply:** 接手这个项目时,先读 `docs/project/进度.md` 看最新状态,
再读 `docs/project/开发报告.md` 看最近的踩坑记录。