# 补齐桌面端 HTTP 能力 + wails3 dev 桌面形态 + 删 RunMode 配置歧义

**日期:** 2026-06-23
**状态:** 已完成

---

## 1. 需求

> 用户原话(分段):
> 1. "桌面端偏好 一需要重启桌面应用生效 偏好服务不可用(可能后端未启动或prefs存储未就绪)"
> 2. "测试通知失败 / 404 POST /api/desktop/notify/show"
> 3. "wails3 dev 这么启动会被认为是 web 端,桌面端偏好仅在桌面应用里可见,请用桌面端/系统托盘来打开设置" → 期望 wails3 dev 是桌面端
> 4. "现在 runMode 是怎么界定的?其实很简单,我希望可以直接根据 wails3 task 命令来判断,而不是通过配置文件或者其他"
> 5. "task web 是否可以自动在默认浏览器中打开对应的前端界面"
> 6. "configs/system.go RunMode 可以删掉了,否则可能会产生歧义"

澄清后目标:
- 桌面端所有 OS 能力(通知 / 剪贴板 / 窗口 / 快捷键 / open-external / 偏好)走 Gin HTTP 端点
- Wails v3 alpha.60 不再像 v2 注入 `window.go.*`,所有 webview 调用走 HTTP
- wails3 dev 启动的 Vite 端能识别为 desktop 形态(不再兑底 web)
- runMode 单一权威:启动命令 → env → vite.config.js → 前端 import.meta.env / __APP_RUNTIME__
- 配置文件 `system.run_mode` 字段删除,避免双源歧义
- `task web` 默认自动打开浏览器

不做什么:
- 不实现 Wails v3 binding 自动生成(走 HTTP 抽象,不再依赖 `window.go`)
- 不改 SettingsView / SettingsView.vue 的 UI(只动后端能力 + 前端平台抽象)
- 不改 prompt 实施,只跑通桌面 + web 双形态

---

## 2. 任务列表

- [x] 分析 Wails v3 alpha.60 bindings 为什么 404(`$Call.ByID` 走 `/wails/runtime`,本项目 webview 是 gin 提供,无此路由)
- [x] 设计 hooks 桥接方案,避开 `bootstrap → router → cdesktop → bootstrap` 导入环
- [x] 新增 cdesktop 控制器 11 个端点(noise/clipboard/window/shortcut/open-external)
- [x] desktop.NewApp 注入 BootstrapHooks 到 backend
- [x] bootstrap.Serve 桥接 backend hooks 到 cdesktop
- [x] 修 vite.config.js,dev 模式根据 `VITE_DEPLOY_MODE` 注入 `__APP_RUNTIME__`
- [x] 改 `frontend/src/platform/index.js` 与 `runtime.js` 优先级:env > window.__APP_RUNTIME__ > 兑底 web
- [x] 改根 Taskfile.yml:`dev` 任务默认 `VITE_DEPLOY_MODE=desktop`;`web:dev:frontend` 显式 `web`
- [x] `task web` 默认 `WEB_DEV_OPEN_BROWSER=1`,启动后 2 秒用 `open` 命令拉起默认浏览器
- [x] 修 hooks 时序问题(Serve 早于 NewApp 注入导致 hooks 永远 Set 成空值)
- [x] 删 `configs.System.RunMode` 字段 + `configs.yaml` run_mode 行
- [x] `ServerOptions.RunMode` 字段 + `ServerOptionsBuilder` 签名改为 `func(runMode string) ServerOptions`
- [x] `dbs.SetRunMode` / `dbs.IsDesktop` 取代 `configs.System.RunMode == "desktop"` 判定
- [x] `init_db.go` / `prefs.a.go` 全部改走 `dbs.IsDesktop()`
- [x] 验证:api-server / desktop module 编译通过,gin 路由注册齐全,curl 端点 200/501 符合预期
- [x] 留任务过程文件(本次任务)

---

## 3. 执行进度

> 时间倒序,最新的在最上面。

- 17:50 ~ 18:30 删 `configs.System.RunMode` 字段,改 `ServerOptions.RunMode` 透传,改 `dbs.SetRunMode`/`IsDesktop`,改 `applyDataDir` 不再回写配置文件
- 17:50 wails3 dev 启动后,发现 `bootstrap: WARNING desktop hooks EMPTY` 与 `desktop: SetDesktopHooks installed (Notify=true ...)` 矛盾 → 时序问题
- 17:30 改 hooks 子包:`Set/Get` 改为 `Bind(provider) + Get` 实时读 backend 指针,解决 NewApp 晚于 Serve 启动的时序
- 17:00 加 `web:dev:frontend` 自动开浏览器(WEB_DEV_OPEN_BROWSER=1)
- 16:50 改根 Taskfile.yml:`dev` 任务 env 默认 `desktop`,`web` 任务 env 显式 `web`
- 16:40 改 `frontend/src/core/utils/runtime.js` 与 `platform/index.js`,三段优先级
- 16:30 改 `frontend/vite.config.js`,dev 模式读 `VITE_DEPLOY_MODE` 注入 `import.meta.env.VITE_RUN_MODE` 与 index.html 的 `__APP_RUNTIME__`
- 16:10 实现 hooks 桥接 + desktop.NewApp 注入 BootstrapHooks,curl 11 端点 200/501
- 16:00 新增 cdesktop 控制器 11 个端点(notify/clipboard/window/shortcut/open-external)
- 15:30 分析根因:Wails v3 alpha.60 bindings 走 `/wails/runtime` 但本项目 webview 是 gin 提供,无此路由

---

## 4. 问题与方案

### 问题 1:桌面端 cdesktop 端点全部 501,通知测试失败

- **现象**:`POST /api/desktop/notify/show` 返回 501
- **定位**:`bootstrap.Serve` 在 goroutine 里立刻跑,调 `hooks.Set(b.GetDesktopHooks())`,此时 `desktop.NewApp` 还没执行 `backend.SetDesktopHooks()`,`desktopHooks` 还是零值,`Notify` 是 nil,Set 之后所有请求都拿到 nil
- **方案**:hooks 子包改为持有 `Provider` 指针(b *Backend 实现 `GetDesktopHooks` 接口),`Get()` 时实时从 backend 读,而不是缓存到 current 变量
- **教训**:启动时序(并发 vs 同步)导致的"已 Set 但值是旧/空"问题,通用解法是**指针 + 实时读**而非**值 + 一次性拷贝**

### 问题 2:wails3 dev 启动后前端被识别为 web

- **现象**:`wails3 dev` 跑起来后,SettingsView 仍显示"web 端,请用桌面端/系统托盘来打开设置"
- **定位**:wails3 dev 的 webview 加载 Vite dev server(端口 9245),**不走**后端 gin,所以后端 `injectRuntimeScript` 永远不被调用,`__APP_RUNTIME__` 拿不到,兑底为 web
- **方案**:让 `vite.config.js` 在 dev 模式读 `VITE_DEPLOY_MODE` 环境变量,通过 `transformIndexHtml` 钩子把 `__APP_RUNTIME__` 注入到 index.html;同时 `define` 暴露 `import.meta.env.VITE_RUN_MODE` 给前端代码
- **教训**:dev 形态(走 Vite)与 release 形态(走 gin embed)注入运行时配置的路径不同,前者只能靠 build 工具插件,后者靠后端 HTTP middleware

### 问题 3:runMode 双源歧义

- **现象**:`configs.System.RunMode` 既能从 yaml 读,又能在 `applyDataDir` 里被 override 后回写,启动命令传的 `BootOptions.RunMode` 与配置文件值可能不一致
- **定位**:`system.run_mode: web` 是历史遗留,新方案下完全用不到(部署形态由启动命令决定)
- **方案**:删 `SystemConfig.RunMode` 字段 + `configs.yaml` run_mode 行;新增 `dbs.SetRunMode/IsDesktop` 让 dbs 包接收运行形态;`ServerOptions.RunMode` 透传启动命令的 RunMode
- **教训**:配置 vs 命令的"双源"是经典歧义坑,删除冗余源、保留单一权威才能彻底避免

### 问题 4:internal package 跨 module 不能 import

- **现象**:`skill-box/desktop` 想用 `ginp-api/internal/...` 下的 hooks 类型,被 Go internal 规则拒绝
- **定位**:Go 的 internal/ 目录只能被同 module 树 import
- **方案**:把 `BootstrapHooks` 类型定义放 `cdesktop/hooks` 子包(self-contained,不 import bootstrap),bootstrap 用类型别名 `type BootstrapHooks = hooks.BootstrapHooks` 透出给跨 module 的 desktop 端使用
- **教训**:internal/ 是 namespace 而非 module 边界,跨 module 共享类型只能用非 internal 路径或别名

---

## 5. 需求回流

> 用户临时加塞、计划外需求。

- **task web 自动开浏览器** → 在 web:dev:frontend 任务内 sleep 2 然后 `open http://localhost:9245/`(macOS)→ 简单 shell,无需引入新依赖
- **删 RunMode 字段** → 触发 ServerOptions / dbs / applyDataDir / prefs.a.go 一连串改动,2 次编译错误,最终清掉所有引用

---

## 6. 总结

### 完成了什么

1. Wails v3 alpha.60 桌面端能力全部走 Gin HTTP 端点(11 个新端点)
2. hooks 桥接机制:`backend.SetDesktopHooks(BootstrapHooks{...})` → bootstrap 桥接到 hooks 子包 → cdesktop 各 handler 通过 `hooks.Get()` 实时读
3. wails3 dev 桌面形态识别:根 Taskfile 任务 env 注入 → vite.config.js 读 env 注入 `__APP_RUNTIME__` 与 `import.meta.env`
4. 删 `configs.System.RunMode` 字段,runMode 单一权威 = 启动命令
5. `task web` 默认自动开浏览器

### 留下了什么

**代码**:
- `api-server/internal/gapi/controller/skillbox/cdesktop/{notify,clipboard,window,shortcut,external,prefs}.a.go`
- `api-server/internal/gapi/controller/skillbox/cdesktop/hooks/hooks.go`(独立子包)
- `api-server/cmd/bootstrap/bootstrap.go` + `server.go`(ServerOptions.RunMode, hooks.Bind, buildRuntimeScript)
- `api-server/internal/db/dbs/{get_db,init_db}.go`(SetRunMode/IsDesktop)
- `desktop/wails_app.go`(NewApp 末尾 SetDesktopHooks)
- `desktop/{window,shortcut}.go`(补 ToggleMaximise、Unregister)
- `frontend/vite.config.js`(dev 模式注入 runtime)
- `frontend/src/{core/utils/runtime,platform/index}.js`(三段优先级)
- `Taskfile.yml` 根任务 + `build/Taskfile.yml` 任务(env 注入)

**配置**:
- `configs.yaml` 删 `run_mode` 行
- `configs/system.go` 删 `RunMode` 字段

**决策**:
- runMode 单一权威 = 启动命令(单源真相,避免双源歧义)
- 桌面端 OS 能力统一走 HTTP 端点(放弃 Wails bindings 自动生成)
- dev 形态由 env 注入,release 形态由后端 gin 注入(两种路径都支持,前端代码不感知差异)

### 留给下次的事

- [ ] wails3 dev 启动后端时如果 Wails webview 连接 Vite 慢,可能首屏空白;观察一段时间,如果频繁需要手动刷
- [ ] macOS 通知授权被拒后再触发,前端要友好提示(目前只 throw)
- [ ] BootstrapHooks 字段持续增加时考虑改成 map[string]any 减少样板
- [ ] task dev 任务的 wails3 dev 进程如果前端 dist 改了就 rebuild → restart(已观察到这个循环),优化 debounce 或忽略 dist 变化

### 复盘

**做得好**:
- 一开始就把"导入环"想清楚,直接用 hooks 子包打破
- 时序问题用"指针 + 实时读"而不是"同步等待",优雅很多
- runMode 双源问题一次性彻底解决,不留尾巴

**能改进**:
- 第一轮我就该 ONBOARDING 第 3 节"分支 A"立刻建任务过程文件,而不是会话说完了才补
- 修 wails3 dev 形态时,应该一开始就查 wails3 dev 的实际行为(用 `wails3 dev --help` 查参数)而不是猜测

### 提炼清单

- [x] 是否要新增 / 更新 `docs/agent/memory/feedback_*.md` → 暂不需要(没有新的稳定反馈)
- [x] 是否要新增 / 更新 `docs/agent/memory/project_*.md` → 暂不需要(技术决策已写在本文件)
- [x] 是否要更新 `docs/project/开发报告.md` → 待定,本次改动可能可以写一笔
- [x] 是否要更新 `docs/project/需求规划.md` / `进度.md` → 不需要,本次是 bug 修复 + 重构,非新需求
