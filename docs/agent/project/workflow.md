# Workflow(开发 / 提交 / 验证流程)

> 提 PR 前必读。

## 一、日常开发

### Web 端(单进程)

```bash
# 1) 开发模式(Vite 热更新 + 后端热重启)
wails3 task web

# 2) 生产模式(完整 build)
wails3 task run:web
```

### 桌面端

```bash
# 开发(热更新)
wails3 dev -config ./build/config.yml -port 9245

# 打包(默认本平台)
task build
```

### 后端独立调试

```bash
cd api-server && go test ./...         # 测试
cd api-server && go run ./cmd/web      # 仅后端,前端走 Vite dev 代理
```

### 前端独立调试

```bash
cd frontend && npm install
cd frontend && npm run dev             # :5173,Vite 代理 /api 到后端
```

## 二、改代码前 / 中 / 后

### 改之前

- 涉及 Go 业务代码 → `@docs/agent/project/conventions.md` + `architecture.md`
- 涉及 Vue 业务代码 → `@docs/agent/project/conventions.md` + `tech_stack.md`
- 涉及启动流程 / 配置 → `@docs/agent/project/architecture.md`(关键不变量小节)
- 涉及跨平台构建 → `@docs/agent/project/workflow.md`(本文件) + `Taskfile.yml`

### 改之中

- **每完成一个子任务** → 在 `docs/agent/task/<日期>_<主题>.md` 里勾掉一项
- **遇到非平凡问题(>5 分钟定位或设计取舍)** → 追加到同 task 文件的"问题"小节
- **用户临时加塞、计划外需求** → 追加到"需求回流"小节,后续并入 `docs/project/需求规划.md`

### 改之后

- 跑 `go test ./...`(对应模块下)
- 前端改动:`npm run build` 确认能产 dist
- 大改动:`wails3 task run:web` 端到端跑一遍,console 执行 `__APP_RUNTIME__` 验证

## 三、提交

### Commit 信息

- 简短中文祈使句(参考仓库历史):
  - `修复接口样式`
  - `迁移 ginp 改动`
  - `web: 同步 embed 目录`
- 不强制 conventional-commit 前缀,但建议在 commit 主题里体现改动区域(后端 / 前端 / build)

### 频率

- 每个"完成的功能子集"提交一次
- **不要把半天 / 一天的工作压成一个巨型 commit**

### 不提交

- `bin/`、`data.db`、`logs/`、`frontend/dist/`(已 gitignore)
- `api-server/cmd/web/frontend/dist/`(同步目录,会被覆盖)
- 真实密钥 / 线上配置

## 四、Pull Request

### PR 内容

- 改动说明 + 关联 Issue
- UI 变更必须附截图
- 前后端改动放在 **同一个 PR**(因为 dist 是嵌入的,必须一起发布)

### PR 描述模板

```markdown
## 改了什么
- 业务侧:
- 技术侧:

## 怎么验
- [ ] 后端:`go test ./...` 通过
- [ ] 前端:`npm run build` 成功
- [ ] 端到端:`wails3 task run:web` 跑通关键路径
- [ ] 截图(UI 改动)

## 影响
- 文档:`docs/project/项目架构.md` / `docs/project/需求规划.md`(如有)
- 记忆:`docs/agent/memory/feedback_*.md`(如有新规则)
```

## 五、Release / 跨平台

- 跨平台 Taskfile 在 `build/<os>/Taskfile.yml`
- Docker 交叉编译镜像:`task setup:docker`
- 各平台产物落 `bin/<app>-<platform>-<arch>/`
- 详细部署形态对比见 `docs/project/项目架构.md` §2