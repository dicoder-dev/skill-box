# 仓库贡献指南

## 项目结构与模块划分

`skill-box` 是一个基于 Wails v3 的桌面应用（Go 1.25），内嵌 Gin HTTP 后端与 Vue 3 SPA。

- `main.go` — 桌面端入口。启动进程内后端，再拉起 Wails UI；支持 `-config <path>` 指定配置文件。
- `api-server/cmd/` — 可执行入口：`web`（仅 HTTP）、`gapi`（管理 CLI）、`gencode`（实体/API/CRUD 生成器）、`bootstrap`（共享启动流程）。
- `api-server/internal/` — 服务端内部包（`db`、`gapi`、`gen`）。
- `api-server/pkg/` — 可复用库：`cfg`、`dbops`、`ginp`、`logger`、`task`、`email`、`cos`、`httpclient` 等。
- `api-server/configs/` — 从 YAML 加载的强类型配置结构体。
- `desktop/` — Wails UI 组装（窗口、菜单、托盘、快捷键、服务）。
- `frontend/` — Vue 3 + Vite + Pinia SPA。`frontend/dist/` 通过 `//go:embed` 嵌入到 Go 二进制中。
- `build/` — 各平台 Taskfile 与 Wails 配置（`build/{windows,darwin,linux,ios,android}/`）。
- `configs.yaml` — 运行时配置（默认 sqlite，支持 mysql/pgsql）。
- `bin/` — 构建产物（已 gitignore）。

## 构建、测试与开发命令

任务由 [Task](https://taskfile.dev) 驱动，详见根目录 `Taskfile.yml`。

- `task dev` — Wails 开发模式，Go + Vite 热更新（默认端口 9245）。
- `task build` / `task package` / `task run` — 编译、打包或运行原生二进制。
- `task setup:docker` — 构建移动端打包所用的 Docker 交叉编译镜像。
- `cd frontend && npm install` — 安装前端依赖。
- `cd frontend && npm run dev | build` — 启动 Vite 开发服务器或生产构建。
- `cd api-server && go test ./...` — 运行 Go 测试（目前桌面端根模块没有测试）。
- `go mod tidy` — 在根目录与 `api-server/` 下各执行一次，二者通过 `go.work` 共用工作区。

## 编码风格与命名约定

- Go：使用标准 `gofmt`（Tab 缩进，goimports 顺序）；鼓励写包级文档注释。仓库内现有文件混用中英文注释，编辑时与目标文件保持一致即可。
- Vue 3：`<script setup>` SFC，2 空格缩放。Pinia store 放 `frontend/src/store/`，接口封装放 `frontend/src/api/`。
- Wails 生成的 bindings 位于 `frontend/bindings/` 与 `frontend/src/api/`，需重新生成，不要手改。
- 一个 API 接口一个文件（如 `get_user_info.go`）；包名小写、不带分隔符。

## 测试规范

- 框架：标准 `testing` 包，测试文件以 `*_test.go` 命名，与代码同目录。
- 在所属模块下运行（如 `cd api-server && go test ./...`）。
- 命名遵循 `TestXxx`；配置/解析类代码优先使用表驱动测试。

## 提交与 Pull Request 规范

- 提交历史使用简短的中文祈使句（如 `修复接口样式`、`迁移 ginp 改动`），单行主题为主，不强制 conventional-commit 前缀。
- PR 需说明改动内容、关联相关 Issue，UI 变更请附截图。前后端改动建议放在同一个 PR 内——`frontend/dist/` 会在构建时嵌入 Go 二进制，必须一起发布。

## 安全与配置提示

- `configs.yaml` 中含开发用凭据与默认 sqlite 路径。生产环境通过 `-config` 指向脱敏后的配置文件，切勿提交真实密钥。
- `data.db` 与 `bin/` 已 gitignore，请勿手工修改。
- 若 9245 端口被占用，可通过环境变量 `WAILS_VITE_PORT` 覆盖 Wails 开发端口。
