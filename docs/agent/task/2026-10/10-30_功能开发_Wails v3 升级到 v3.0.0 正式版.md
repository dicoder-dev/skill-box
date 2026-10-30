# 10-30_功能开发_Wails v3 升级到 v3.0.0 正式版

## 需求

用户原话:「https://wails.io/zh-cn/quick-start/installation/ 帮我看看 把 wails 更新到 v3 正式最新版」

把项目从 `wails/v3 v3.0.0-alpha.60` 升级到 **v3.0.0 正式版**(2026-10-26 发布)。

## 背景

- 项目基线:`go.mod` 里 `github.com/wailsapp/wails/v3 v3.0.0-alpha.60`
- PATH 上的 CLI:`wails3 version` → `v3.0.0-alpha.59`(比依赖还旧一版)
- 前端 `@wailsio/runtime` 锁在 `3.0.0-alpha.79`
- 本机 Go:`go1.26.3 darwin/arm64`

## 升级动作清单

### 1. CLI

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
# ⚠️ 默认装到 /Users/augmax/go/bin,PATH 里排后面,wails3 version 仍显示旧版
cp /Users/augmax/go/bin/wails3 /Volumes/AugmaxDrive/home/ddd/go/bin/wails3
```

### 2. Go 依赖

```bash
go get github.com/wailsapp/wails/v3@v3.0.0
```

连带升级:`golang.org/x/image` 0.40→0.45、`x/crypto` 0.50→0.57、`x/net` 0.53→0.60、
`x/sys` 0.43→0.48、`x/text` 0.37→0.42、`x/tools` 0.44→0.49、`go-git` 5.19.1→5.19.2、
`ProtonMail/go-crypto` 1.1.6→1.3.0 等。

`go.mod` / `go.work` 的 `go` 指令:`1.25.0` → `1.26.0`(v3.0.0 要求 Go 1.26+)。

### 3. 前端 runtime

```bash
cd frontend && npm install --save-exact @wailsio/runtime@3.0.0
```

用 `--save-exact` 精确 pin —— 官方 changelog:"Project templates pin `@wailsio/runtime`
to the version of the CLI that generated them"。不用 `^` / `latest`。

### 4. build/config.yml

给 `executes` 的 primary 任务加 v3.0.0 新字段 `exit_policy: shutdown`
(主进程退出时自动关掉 background 的 Vite dev server,避免孤儿进程占 9245 端口)。

### 5. cmd/updater_check 迁移(顺带修既有问题)

原文件在**根 module** `cmd/updater_check/`,却 import 了 `ginp-api/internal/...`,
违反 Go internal 规则,`go build ./...` 一直失败:

```
use of internal package ginp-api/internal/gapi/controller/skillbox/cdesktop not allowed
```

`git stash` 验证过是**升级前就存在**的问题。本次 `git mv cmd/updater_check
api-server/cmd/updater_check` 修掉。

### 6. 注释版本号同步

`desktop/tray.go`、`desktop/wails_app.go` 里描述 alpha.60 时代行为的注释更新版本号,
并在 `wails_app.go` 补一条 macOS 坐标归一化的说明(Size/Bounds 仍是 DIP,结论不变)。

## 破坏性变更核查(逐条查过官方 changelog,本项目零命中)

| BREAKING 项 | 本项目 |
|---|---|
| macOS 坐标归一化(alpha.91,Position/SetPosition 改逻辑点 Y-down) | 未命中 —— 只用 `Screen.Size`/`Bounds` 宽高,不用 Position |
| bindings 里 Map key 变可选(#4943) | 未命中 —— `frontend/src` 完全没 import bindings |
| `EnableDragAndDrop`→`EnableFileDrop` 等拖放改名 | 未命中 —— 项目不用文件拖放 |
| 移除 package 级 dialog 函数 | 未命中 —— 已用新 API `app.Dialog.*` |
| 生产构建变默认 | 已符合 —— Taskfile 里显式 `DEV=true` |
| `NativeWindowHandle()`→`NativeWindow()` | 未命中 |
| per-window Windows flags 移到 app 级 | 未命中 |

**结论:无需改任何 Go 业务代码。**

## 顺带确认的事

- **托盘双击死代码在 v3.0.0 仍未修**:重新 grep `systemtray_darwin.go` 的 `processClick`,
  仍然只 `case leftButtonDown` / `case rightButtonDown`,`doubleClickHandler` 全仓只有
  `systemtray.go`(定义)+ `systemtray_windows.go:474`(真正 dispatch)。macOS 上双击依然
  不触发,`desktop/tray.go` 不绑 `OnDoubleClick` 的决定继续有效。
- **memory 文件缺失**:`MEMORY.md` 引用的 `wails-v3-tray-darwin-doubleclick-deadcode.md`
  实际不存在,本次补齐。
- **v3.0.0 新能力未启用**:内建 updater(`pkg/updater`)、MCP service、server 构建任务、
  macOS 26 Liquid Glass、notch windows —— 都跟本项目现状不冲突,后续按需评估。

## 验证结果(全过)

```bash
wails3 version                          # v3.0.0 ✅
go build ./...                          # ✅(build/ios 报错是既有的,需 GOOS=ios)
go vet ./desktop/...                    # ✅
go test ./desktop/...                   # ✅ ok skill-box/desktop
wails3 generate bindings -clean=true    # ✅ 730 Packages, 6 Services, 20 Methods
wails3 task build                       # ✅ 桌面端产物构建成功
wails3 dev -config ./build/config.yml   # ✅ dev 启动成功,UI 渲染正常,API 全 200
```

截图确认:应用窗口正常渲染(技能列表 / SKILL 目录 / Markdown 大纲 / 作用域区全部正常),
托盘图标在 dock 正常显示。

**api-server 的 `internal/db/pgsql` 测试失败是环境问题**(本地没建 `db_pg` 库),
与本次升级无关,升级前同样失败。

## 对话轮次

### 1.0 (15:20)

- 用户原话:「https://wails.io/zh-cn/quick-start/installation/ 帮我看看 把 wails 更新到 v3 正式最新版」
- 本轮做了:查版本 → 装 CLI v3.0.0 → `go get` 升依赖 → 升前端 runtime → 改 config.yml
  → 移 updater_check → 跑构建/测试/dev 验证 → 截图确认 UI → 写 memory + task
- 本轮决定:不改任何 Go 业务代码(alpha.60 → v3.0.0 的 BREAKING 项本项目零命中)
- 本轮待办:无
- 状态更新:已完成

### 1.1 (15:35)

- 用户原话:「继续」
- 本轮做了:修 `cmd/updater_check` 的 internal 包越界问题(`git mv` 到 api-server/cmd/),
  补齐缺失的 memory 文件,新建本 task 文档
- 本轮决定:顺带修掉这个既有问题,让 `go build ./...` 全绿
- 本轮待办:提交 + push
- 状态更新:已完成
