---
name: wails-v3-upgrade-3-0-0
description: Wails v3 升级到 v3.0.0 正式版(2026-10-30)的完整变更清单、破坏性变更核查结论、Go 1.26 要求与 CLI 安装路径坑
metadata:
  type: project
---

# Wails v3 从 alpha.60 升级到 v3.0.0 正式版(2026-10-30)

## 版本信息

- **正式版发布时间**:2026-10-26(v3.0.0),最新 tag `v3.0.0`,之后是 `v3.1.0-nightly.*`
- **Go 要求**:**Go 1.26+**(v3.0.0 go.mod 写 `go 1.26.0` + `toolchain go1.26.10`)
  - 本机 `go version go1.26.3 darwin/arm64`,满足
  - `go.mod` / `go.work` 都要跟着升到 `go 1.26.0`
- **CLI 安装**:`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`
- **前端 runtime**:`@wailsio/runtime` 要**精确 pin** 到 `3.0.0`(不要用 `^` / `latest`)
  - 官方 changelog 原话:"Project templates pin `@wailsio/runtime` to the version of the CLI that generated them"
  - 用 `npm install --save-exact @wailsio/runtime@3.0.0`

## 本次实际改动清单

| 文件 | 改动 |
|---|---|
| `go.mod` | `wails/v3` alpha.60 → **v3.0.0**;`go 1.25.0` → `go 1.26.0`;连带升级 `golang.org/x/image` 0.40→0.45、`x/crypto` 0.50→0.57、`x/net` 0.53→0.60 等 |
| `go.work` | `go 1.25.0` → `go 1.26.0` |
| `frontend/package.json` | `@wailsio/runtime` → `3.0.0`(精确) |
| `build/config.yml` | `executes` 的 primary 任务加 `exit_policy: shutdown`(v3.0.0 新字段) |
| `api-server/cmd/updater_check/main.go` | **从根 module `cmd/` 移到 `api-server/cmd/`** |
| `desktop/tray.go` / `desktop/wails_app.go` | 注释里的 alpha.60 版本号更新为 v3.0.0 |
| `frontend/bindings/events/index.js` | 新增(v3.0.0 的 bindings 生成器自动产出事件注册) |

## 破坏性变更核查结论(全部逐条查过,**本项目均未命中**)

v3.0.0 changelog 里从 alpha.60 之后累积的 BREAKING 项:

1. **macOS 坐标系统归一化**(alpha.91)—— `GetScreens` / `Position` / `SetPosition` 统一为
   逻辑点、Y-down、原点在主屏左上;屏幕在主屏上方时 `Bounds.Y` 变负数。
   → **本项目未命中**:只用了 `Screen.Size` / `Screen.Bounds` 的宽高(仍是 DIP),
   没用 `Position` / `SetPosition`。
2. **Map key 在生成的 JS/TS bindings 里变可选**(#4943)—— TS 里取值变成 `T | undefined`。
   → 本项目 bindings **完全没在 `frontend/src` 里 import**(前端只用 `@wailsio/runtime` 的
   `Events`/`Call` 之外的能力,且走 HTTP API 到 gin),无影响。
3. **`EnableDragAndDrop` → `EnableFileDrop`**、`DropZoneDetails` → `DropTargetDetails`、
   `WindowDropZoneFilesDropped` → `WindowFilesDropped`、`data-wails-dropzone` →
   `data-file-drop-target`、CSS `wails-dropzone-hover` → `file-drop-target-active`。
   → 本项目**没用文件拖放**,无影响。
4. **移除 package 级 dialog 函数**(`application.InfoDialog()` 等)→ 改用 `app.Dialog.*`。
   → 本项目用的是 `app.Dialog`(新 API),已经符合。
5. **生产构建变成默认** —— dev 构建要显式 `DEV=true`。项目 Taskfile 里已经是 `DEV=true`。
6. **`window.NativeWindowHandle()` → `window.NativeWindow()`** —— 项目未用。
7. **per-window `EnabledFeatures`/`DisabledFeatures`/`AdditionalLaunchArgs` 移到 app 级
   `Options.Windows`** —— 项目未用。

**结论:alpha.60 → v3.0.0 无需改任何 Go 业务代码,`go build ./...` 与 `go vet ./desktop/...` 直接通过。**

## v3.0.0 新增能力(本项目尚未用,后续可评估)

- **内建 updater**(`pkg/updater`):可插拔 provider(含 GitHub Releases)、加密校验、
  原子替换 + 回滚、可换主题的默认 UI。本项目目前是**自研** updater
  (`desktop/services/updater_helper.go` + `api-server/.../cdesktop/update.a.go`)。
- **MCP service**:让 AI 工具直接驱动/检视运行中的 Wails 应用。
- **Server 构建**:`build:server` / `run:server` 任务,同一份 app 跑成纯 HTTP server。
  本项目已有自己的 `web:*` 任务链(gin 单进程),功能重叠,不需要迁。
- **macOS 26 原生 Liquid Glass** + 早期版本回退。
- **notch windows**(macOS 刘海屏窗口)。
- **iOS / Android** 正式进模板(仍 experimental,不在稳定性承诺内)。

## CLI 安装路径的坑(本机实测)

本机有两套 GOPATH,`PATH` 里**优先的是** `/Volumes/AugmaxDrive/home/ddd/go/bin`:

```bash
# go install 默认装到 /Users/augmax/go/bin —— PATH 里排后面,`wails3 version` 仍显示旧版!
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0

# 解决:直接 cp 到 PATH 优先的位置
cp /Users/augmax/go/bin/wails3 /Volumes/AugmaxDrive/home/ddd/go/bin/wails3
wails3 version   # → v3.0.0
```

**排查口诀**:升级 wails CLI 后 `wails3 version` 还显示旧版 → 一定是装到了 PATH 靠后的
GOPATH,先 `which -a wails3` 看实际命中哪个二进制。

## 验证清单(本次全过)

```bash
wails3 version                          # v3.0.0
go build ./...                          # 通过(build/ios 报错是既有的,需 GOOS=ios)
go vet ./desktop/...                    # 通过
go test ./desktop/...                   # ok skill-box/desktop
wails3 task build                       # 桌面端产物构建成功
wails3 dev -config ./build/config.yml   # dev 启动成功,UI 渲染正常,API 全 200
wails3 generate bindings -clean=true    # 生成成功(730 Packages, 6 Services, 20 Methods)
```

**api-server 的 `internal/db/pgsql` 测试失败是环境问题**(本地没建 `db_pg` 库),
与 wails 升级无关,升级前同样失败。

## `cmd/updater_check` 移动的原因

它 import 了 `ginp-api/internal/gapi/controller/skillbox/cdesktop`,但原文件在**根 module
`skill-box`** 里 —— Go 的 internal 规则规定 `ginp-api/internal/...` 只能被 `ginp-api` 模块内的
包导入,所以 `go build ./...` 一直报:

```
use of internal package ginp-api/internal/gapi/controller/skillbox/cdesktop not allowed
```

这是**升级前就存在的**问题(用 `git stash` 验证过),本次顺带修掉:
`git mv cmd/updater_check api-server/cmd/updater_check`。

## How to apply

- 以后升级 wails 版本时,**先查官方 changelog 的 BREAKING 段落**,再对本项目用到的 API
  逐条比对,不要盲改。本项目的 wails API 使用面很窄(`app.Dialog` / `app.Browser` /
  `app.Clipboard` / `app.Menu` / `app.Window` / `app.RegisterService`),基本不会踩坑。
- 升级后**必须重新 grep 一遍 `desktop/` 里对旧行为的假设**(本项目有一堆注释记录了
  alpha.60 时代的 bug,如托盘双击死代码、`SetSize` 不可靠),确认上游是否已修。
- `go build ./...` 里 `build/ios` 报的 `function main is undeclared` 是**正常的**
  —— iOS 包靠 `//go:build ios` tag,只有 `GOOS=ios` 交叉编译时才有 main,别去"修"它。

相关:[[wails-v3-tray-darwin-doubleclick-deadcode]]、[[wails-v3-getscreen-primary-resize]]、[[wails-binary-not-rebuild]]
