# CLAUDE.md — skill-box 项目协作入口

> 给 Claude / Claude Code 看的协作入口。**只放最关键的"我是谁/要遵守什么/去哪里查"**,
> 详细说明全部按需 `@` 到 `docs/` 下。

---

## 项目一句话

`skill-box` 是基于 Wails v3 + Gin + Vue 3 + Pinia 的桌面/Web 双形态应用,
本仓库为 Go 1.25 工作区,业务代码全部在 `api-server/` 与 `frontend/`。

---

## 必读文档(按场景按需加载)

| 场景 | 读什么 |
| --- | --- |
| **任何会话开头(冷启动)** | `@docs/agent/ONBOARDING.md`(按清单读完即可工作) |
| 任何会话开头 | `@docs/agent/memory/MEMORY.md` |
| 接到新任务 | `@docs/agent/task/README.md` + 同主题最近一份 `task/*.md` |
| 改 Go / Vue 代码 | `@docs/agent/project/conventions.md` `@docs/agent/project/architecture.md` |
| 提 PR / commit | `@docs/agent/project/workflow.md` |
| 项目业务背景(给人看的) | `docs/project/README.md` + `docs/project/项目架构.md` |

完整 AI 协作约定见 `@docs/agent/README.md`。

---

## 协作铁律(违反即停下来问用户)

1. **删除文件 / 删表 / `rm -rf` / `git reset --hard` / 强推** —— 必须先跟用户确认。
2. **其它操作**(写文件、跑测试、跑构建、调 MCP)—— 直接做,无需确认。
3. **生成的所有代码注释统一使用简体中文**。
4. **图片理解用 `MiniMax - understand_image`**;**联网搜索用 `MiniMax - web_search`**。
5. **回答时称呼用户为"靓仔"**。
6. **不一次性把所有 `docs/` 塞进上下文**,按上面表格按需 `@` 加载。

---

## 常用命令速查

```bash
# Web 端开发(单进程)
wails3 task run:web

# 桌面端开发(热更新)
wails3 dev -config ./build/config.yml -port 9245

# 后端测试
cd api-server && go test ./...

# 前端依赖
cd frontend && npm install && npm run dev
```

完整构建 / 打包 / 跨平台方案见 `@docs/agent/project/workflow.md` 与
`@docs/project/项目架构.md`。