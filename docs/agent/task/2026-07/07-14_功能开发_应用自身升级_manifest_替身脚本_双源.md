# 应用自身升级(manifest + 替身脚本 + 双源)

**日期:** 2026-07-14
**状态:** 已批准方案,开工执行中

## 1. 需求

### 1.1 原始诉求(用户原话)
> 现在计划做升级功能:
> 1. 升级包存放:GitHub 或 Git 当数据来源,国内走 Git(其实是 Gitea/Gitee mirror),国外走 GitHub;
> 2. 检测升级在两处:设置界面 + 角标,都指向同一后端来源,主要给桌面端;
> 3. 不同系统升级方式:
>    - Mac:自动重启后完成更新;
>    - Windows:关闭软件,重启即可完成更新(Windows 走 NSIS 安装器);
>    - Linux:类似 Mac 重启流程(暂未要求,顺带覆盖);
> 4. wails3 没有现成 self-update,要先规划方案,同意了再执行。

### 1.2 细化目标(已与用户对齐)
- 远端统一 JSON manifest,字段含 `urls: [string]` 数组(GitHub + Gitea mirror 同时挂)。
- 桌面端:check → 用户确认 → 下载 → Quit → 替身脚本 detach → relaunch。
- Web 端:check 同样接,download 端 501,前端弹"去下载页"外链。
- release 时 `scripts/release.sh` 一处填 5 处版本号,避免散落手改。
- 签名机制砍(MVP),只做 SHA256。

## 2. 任务列表
- [x] 调研升级方案现状
- [x] 设计升级方案 + Plan 文档(写入 `/Users/brody/.claude/plans/fluttering-finding-grove.md`)
- [ ] Phase A1:写 `docs/agent/project/updater-manifest.md` + `build/updater/manifest.example.json`
- [ ] Phase A2:写 `api-server/internal/gapi/service/updater/ssupdater/`(manifest/compare/download/verify/helper 5 文件)
- [ ] Phase A3:建 `api-server/internal/gapi/controller/skillbox/cdesktop/update.a.go`(3 路由)
- [ ] Phase A4:扩展 `runtimePayload`(`server.go`)注入 version/channel/platform/arch
- [ ] Phase A5:扩展 `BootstrapHooks`(`wails_app.go`)新增 `UpdaterSpawnHelper` + `UpdaterDownloadPath`
- [ ] Phase B6:在 `desktop/assets/updater/` 下放 3 个替身脚本 + `//go:embed`
- [ ] Phase B7:B8:B9:B10:写入 helper 注释 + 控制器调用顺序硬规则
- [ ] Phase C11:App.vue 顶栏红点角标
- [ ] Phase C12:SettingsView "软件更新" 卡片 + `UpdatePanel.vue` 子组件
- [ ] Phase C13:`frontend/src/core/store/update.js` Pinia store
- [ ] Phase C14:i18n 双语同步
- [ ] Phase C15:`desktop/prefs_keys.go` 新增 `PrefKeyUpdateChannel`
- [ ] Phase D15:`scripts/release.sh` 双源上传 + 5 处版本号统一替换
- [ ] Phase D16:`build/{darwin,windows,linux}/Taskfile.yml` 加 `-X skill-box/desktop/services.Version=...`
- [ ] Phase D17:根 `Taskfile.yml` 加 release 任务
- [ ] Phase E:本机自测(dev mode)

## 3. 执行进度

- 2026-07-14 (HH:MM 待填) 用户提升级功能需求,wails3 alpha.60 无官方 Updater,需自规划方案。
- 2026-07-14 完成调研与 Plan,9 个 Phase 文件,获用户批准。

## 4. 问题与方案
> 待 Phase 开工后按"现象 → 定位 → 方案 → 教训"格式记录。MVP 阶段不预判。

## 5. 需求回流
> 待开工后追加。

## 6. 测试报告
> 待 Phase E 后填写。

## 7. 总结
> 待任务结束填。

## 8. 改动的文件
> 待开工后逐项追加。

## 9. 工具与用途
### 9.1 MCP 工具
- `MCP MiniMax::web_search` — (本任务内尚未调用,留待 Phase A 调研 GitHub release 限流或 sha256 校验时使用)
### 9.2 Skill
- `Skill ui-ux-pro-max` — (本任务内尚未调用,留待 Phase C12 "软件更新"卡片视觉设计时)
### 9.3 CLI
- `Bash TaskUpdate` — 维护任务列表
- `Bash Read / Glob / Grep` — 调研现有 skills/settings/cdesktop 路由

## 1.1 对话轮次(已发生)

### 1.1 对话轮次 (14:30)
> 用户提"计划做升级功能,先帮我规划方案,同意了再开始执行"。

- **本轮做了:**
  - 读了 `desktop/prefs_keys.go`、`build/config.yml`、`build/{darwin,windows}/Taskfile.yml`、`frontend/src/App.vue`、`frontend/src/views/SettingsView.vue`、`api-server/internal/gapi/router/*`、`api-server/cmd/bootstrap/server.go`、`desktop/wails_app.go`、`desktop/services/app_svc.go`、`go.mod`、`api-server/internal/gapi/controller/skillbox/cdesktop/prefs.a.go`,把"版本号来源"、"runtime 注入"、"桌面端包路径"三件事核清楚。
  - AskUserQuestion 收齐 4 个关键决策(manifest 形态 / 升级机制 / 版本同步 / Web 端行为)。
  - 派 Plan agent 设计 5 个 Phase 的实施清单,采纳 "urls:[] 数组、签名砍掉、ldflags 路径=skill-box/desktop/services" 三处 X→Y 修正。
  - 把最终方案写入 `/Users/brody/.claude/plans/fluttering-finding-grove.md`。
  - ExitPlanMode 拿到用户批准,退出 plan mode。
- **本轮决定:**
  - 关键不变量 I-1~I-8 全部敲定,已写入计划文件头部。
  - helper 触发顺序硬规则:`SpawnHelper.Start() 成功 → 再 AppQuit()`,违反则 child 跟父进程一起死。
  - ldflags 路径 = `skill-box/desktop/services.Version`,**不是** `ginp-api/...`(走 go.mod:1 的 `module skill-box`)。
  - dev 模式 `Version` 兜底 `"0.0.0-dev"`,ldflags 仅在 `eq .DEV "false"` 注入,避免 wails3 dev 卡死。
- **本轮待办:**
  - 进 Phase A1:写文档与样板 manifest。
- **本轮工具:**
  - `Bash TaskUpdate` ×3 — 维护任务状态
  - `Bash Read / Glob / Grep` — 调研现有 5 处版本号、`cdesktop` 路由风格、`runtimePayload` 注入
  - `Bash AskUserQuestion` — 4 选决策
  - `Agent Explore(subagent)` — 调研 GitHub release API 限流、wails v3 Updater API、三平台重启做法
  - `Agent Plan(subagent)` — 出 5 Phase 实施清单 + X→Y 修正
  - `Bash EnterPlanMode` / `Bash ExitPlanMode` — 进入与退出 plan 模式
- **状态更新:**
  - Task #1 / #2 已 completed;Task #3 进行中。
