# dmg 双击启动修复总结(macOS 26 Tahoe,2026-07-16)

> 配套文档:[dmg-分发说明.md](./dmg-分发说明.md) 是 dmg 产物形态 + 用户文档,本文档是
> **dmg 装后双击不能启动**这个问题的排查 + 修复全记录,2026-07-16 完整跑通。

## TL;DR

dmg 装到 `/Applications` 后双击 `Skill-Box.app` 闪退,**3 个独立 bug 串在一起**才暴露出来,
每个 commit 修一个,最终 dmg 双击启动 + 8082 LISTEN + 前端识别为桌面端。

| commit | 真凶 | 修复点 |
|---|---|---|
| `c52ab1a` | `plist` 的 `ProgramArguments[0]` 指向历史 dev 期 build 的 binary 路径 | `maybeBootstrapLaunchAgent` 检测 plist 路径漂移,自动重写 plist 指向当前 binary |
| `a94ba5f` | launchd 派发 dmg 时 `wd=/`,默认 `-config=./configs.yaml` 在根目录 read-only 失败 | 默认 configPath 锚定 `~/skill-box/configs.yaml`;main.go 加 startup 日志 |
| (本文) | dmg 打包没传 `VITE_DEPLOY_MODE=desktop`,前端 `import.meta.env.VITE_RUN_MODE` 编译期硬编码为 `"web"` | `dmg-arm64` / `dmg-amd64` task 给 `common:build:frontend` 传 `VITE_DEPLOY_MODE: desktop` |

## 详细排查过程

### Phase 1:launchd 派发链是不是有问题?

**误区**:之前一直怀疑 macOS 26 Tahoe 的 amfi Code=-423 / xpcproxy SIGTERM / launchd 派发链拒签。
但 runningboard log 实际显示:

```
launchd: _LSLaunchThruRunningboard: com.dicoder.skillbox / ...
runningboardd: Calculated state for app<application.com.dicoder.skillbox.67532394.67532399(504)>: running-active (role: UserInteractive)
launchservicesd: LAUNCH: Successful launched 0x0-0xb7bb7b pid=46092 com.dicoder.skillbox
```

launchd **成功派发了 binary**(pid 46092),进程也"running-active"。问题不是派发链,是 binary 自己
启动后 50ms 退。runningboard 报 `termination reported by launchd (0, 0, 256)` 实际是
`LSExitStatus=1`(binary `os.Exit(1)`),不是 amfi 静默拒。

### Phase 2:binary 为什么 exit(1)?

手动跑 `nohup /Applications/Skill-Box.app/Contents/MacOS/Skill-Box &` 能起来,
8082 正常 LISTEN。说明 binary 本身没问题。手动跑 vs 双击的区别:

- 手动跑:继承 terminal 的 working directory + LaunchAgent 环境
- 双击 / `open`:launchd 派发,`wd=/`,`ppid=1`,`LaunchAgentLabel` 空

但 — 手动跑也能起,看起来跟 plist 没关系?**错**。

**手动跑的实际路径**:
1. binary 启动
2. `maybeBootstrapLaunchAgent` 检测到 plist 已装 + 8082 没占用
3. `launchctl kickstart -k` 拉 launchd child
4. launchd child 也走 `maybeBootstrap`,plist 路径不对 → 也 kickstart → **死循环 5s 超时**
5. 本进程 `os.Exit(0)`,但用户看到的是"binary 跑了一下就退" — 因为 LaunchAgent plist 里
   program 路径指向 `/Volumes/MyDrive/.../bin/Skill-Box.dev.app/...`(dev 期 build 留下的),
   launchd child 是 dev 期 binary

**为什么 dmg 装的 binary 双击特别明显**:dmg 装的 binary 跟 plist 里 dev 期 binary 完全不是一个
launchd child 实例,但都走同一个 plist,plist program 错 → launchd 拉错 binary → 死循环 → 退。

修复:加 `InstalledBinaryPath()` 解析 plist 路径,启动时检测不一致就重写 plist。
[commit `c52ab1a`](../../../../git log --oneline)

### Phase 3:plist 改对了之后,dmg binary 启动还有问题

修了 plist 之后,手动跑 dmg binary:`pid` 在,**8082 LISTEN**。

但用 `open /Applications/Skill-Box.app`(模拟 Finder 双击)→ binary **立刻 exit 1**,8082 没起。
manual `lsof 8082` 空。runningboard log 看,这次连 `termination reported by launchd` 都来得更快。

**第二真凶**:startup 日志文件一加,真相大白:

```
[2026-07-16 17:48:01.799] START: pid=59227 ppid=1 uid=504
[2026-07-16 17:48:01.803] START: exe=/Applications/Skill-Box.app/Contents/MacOS/Skill-Box
[2026-07-16 17:48:01.807] START: wd=/
[2026-07-16 17:48:01.826] ERROR: bootstrap.Boot failed: open configs.yaml: read-only file system
```

`open` 派发时 launchd 把 working directory 设为 `/`,默认 `-config=./configs.yaml` 解析成
`/configs.yaml` → `cfg.InitCfg` 试图在 `/` 下创建文件 → macOS 根目录 read-only → Boot 失败
fatal。

手动跑时 cwd 是 terminal 的项目根,`./configs.yaml` 写得进去,所以没暴露。

修复:默认 configPath 锚定 `~/skill-box/configs.yaml`,完全不依赖 cwd。

[startup 日志系统](main.go 加的 `setupStartupLog` / `writeStartupLine` / `logStartupContext`)
从此成为 dmg 调试的标配:launchd 派发链 + amfi 静默拒 + 双击闪退,所有 binary 内部 trace
都写到 `~/.skill-box/logs/startup-YYYYMMDD-HHMMSS-<pid>.log`,log show 看不到的细节也能
从用户家目录抓出来。[commit `a94ba5f`](../../../../git log --oneline)

### Phase 4:进程跑起来了,但前端识别成 web

dmg binary 启动 + 8082 LISTEN + 进程持续跑,**视觉上能开窗口**,但:

- 左上角红绿灯和 logo 重叠
- Settings 页"桌面端偏好"section 不可见,提示"桌面端偏好仅在桌面应用里可见"
- 设置入口提示"请用桌面端 / 系统托盘来打开设置"

第三真凶:前端 `runtime.js` 的 `readRunMode()` 三段优先级:

```js
function readRunMode() {
  // 1) import.meta.env.VITE_RUN_MODE —— Vite dev 注入(编译期常量)
  if (typeof import.meta !== 'undefined' && import.meta.env?.VITE_RUN_MODE) {
    return import.meta.env.VITE_RUN_MODE
  }
  // 2) window.__APP_RUNTIME__.runMode —— 后端 gin 运行时注入
  if (typeof window !== 'undefined' && window.__APP_RUNTIME__?.runMode) {
    return window.__APP_RUNTIME__.runMode
  }
  return 'web'  // 兜底
}
```

Vite build 时,**`import.meta.env.VITE_RUN_MODE` 被编译期硬编码到产物 JS 里**(不是 runtime 注入)。
`frontend/vite.config.js`:

```js
const deployMode = (process.env.VITE_DEPLOY_MODE || "web").toLowerCase();
// ...
"import.meta.env.VITE_RUN_MODE": JSON.stringify(deployMode === "desktop" ? "desktop" : "web"),
```

`process.env.VITE_DEPLOY_MODE` 在 dmg build 时**没传**,默认 `"web"`,产物 JS 里写死
`runMode="web"`,**优先级 1 直接命中**,后端 `__APP_RUNTIME__.runMode="desktop"` 永远拿不到。

所以 dmg 双击后前端按 web 形态渲染:不隐藏红绿灯 + chrome、桌面端 section 不显示,
托盘菜单的"偏好设置"虽然存在但前端没识别 desktop 形态没正确响应。

修复:在 `dmg-arm64` / `dmg-amd64` 任务里给 `common:build:frontend` 传 `VITE_DEPLOY_MODE: desktop`,
`build/Taskfile.yml` 里的 `common:build:frontend` 加 `env: VITE_DEPLOY_MODE: ...` 透传。

## 复盘:为什么 3 个 bug 串在一起

**每个 bug 都在另一种启动方式下"恰好能跑"**:
- plist 路径错:dev 期 build 写下的,dmg build 不会自动清理,**只在 dmg 装后第一次启动时**暴露
- wd=/ read-only:terminal 启动 cwd 是项目根,能写 configs.yaml;**只有 launchd 派发**才把 cwd 设成 /
- VITE_DEPLOY_MODE 缺:`wails3 task dev` 显式传 desktop,build 命令链不传,**只有 dmg 任务链**才暴露

调试顺序很关键 — 任何一次少看一个变量,都会让修复停在半路:

1. 不修 plist → 手动跑 dmg binary 也退(被 launchd child 死循环搞挂)
2. 不修 wd → open 派发永远退
3. 不修 VITE_DEPLOY_MODE → dmg 能跑但前端是 web 形态,用户说"打开后奇怪"

## 配套改动

### 新增的 startup 日志系统(`main.go`)

```go
// ~/.skill-box/logs/startup-YYYYMMDD-HHMMSS-<pid>.log
// 同步落盘 + flush,panic / os.Exit 来不及也能保留 trace
func setupStartupLog() error { ... }
func writeStartupLine(line string) { ... }
func logStartupContext() {
    // 打印 pid/ppid/uid/exe/wd/LaunchAgentLabel/argv
    // 便于排查 macOS 26 launchd 派发链细节
}
```

启动时打:
- `START: pid=... ppid=...` — ppid=1 就是 launchd 派发
- `START: wd=...` — wd=/ 就是 dmg 派发典型现象
- `START: LaunchAgentLabel=...` — 空 = 非 launchd child(双击 / open / 终端)

所有错误 / panic 都同步 flush 到这个文件,不再依赖 Console.app / `log show`。

### `InstalledBinaryPath()` 新接口(`pkg/launchagent/launchagent.go`)

```go
// 从 plist XML 解析 ProgramArguments[0],用来检测 plist 路径漂移。
// 不引第三方 plist 解析包,用极简字符串匹配,plist 结构稳定够用。
func InstalledBinaryPath() (string, error)
```

### `maybeBootstrapLaunchAgent` 三分支(commit `c52ab1a`)

```
plist 未装        → 写 plist + bootstrap + 本进程 Serve
plist 已装+路径错 → 用当前 binary 重写 plist(bootout + write + bootstrap) + 本进程 Serve
plist 已装+路径对 → 8082 被占?退 / 8082 没占?直接 Serve
```

不再 kickstart -k(实测 launchd child 启动 + Serve 通常 8-12s,5s 超时不够)。

## 自测脚本

### 1. plist 重写路径

```bash
# plist 故意改回 dev 期路径(模拟历史遗留)
sed -i.bak 's|/Applications/Skill-Box.app/Contents/MacOS/Skill-Box|/Volumes/MyDrive/.../Skill-Box.dev.app/.../Skill-Box|g' \
  ~/Library/LaunchAgents/com.dicoder.skillbox.plist
open /Applications/Skill-Box.app
# 期望:plist 自动改写回 dmg 路径,binary 持续跑,8082 LISTEN
plutil -p ~/Library/LaunchAgents/com.dicoder.skillbox.plist | grep ProgramArguments
```

### 2. wd=/ read-only 修复

```bash
# 模拟 launchd 派发(wd=/)
cd /
open /Applications/Skill-Box.app
# 期望:8082 LISTEN,startup log 不再有 "open configs.yaml: read-only file system"
```

### 3. 前端识别 desktop

```bash
# 启动后 curl gin 注入的 HTML
curl -s http://127.0.0.1:8082/ | grep APP_RUNTIME
# 期望:window.__APP_RUNTIME__={"runMode":"desktop",...}

# 看 embed 进 binary 的 index.js 是否 VITE_RUN_MODE="desktop"
curl -s http://127.0.0.1:8082/assets/index-*.js | grep -oE 'Ix\)return"[^"]+"'
# 期望:Ix)return"desktop"
```

### 4. 看 startup 日志

```bash
ls -lat ~/.skill-box/logs/startup-*.log | head -3
cat ~/.skill-box/logs/startup-$(date +%Y%m%d)-*.log
```

## 教训

1. **macOS 26 Tahoe 派发链特殊行为**:dmg binary 双击一定走 launchd 派发,`wd=/`,`ppid=1`,
   `LaunchAgentLabel` 空。terminal / dev 期 build 永远复现不出来,必须 `open` 测。
2. **launchd 派发链上 binary stdout 关掉**:`log show` 看不到 binary 内部输出,必须自己写日志
   文件,这是 `setupStartupLog` 永久保留的根因。
3. **Vite build 期常量必须显式传 env**:`import.meta.env.VITE_*` 系列被编译期硬编码,
   调用链不传就会编译错的值到产物里。dmg task 跟 dev task 必须显式区分。
4. **plist 路径漂移是 dmg 长期隐患**:每次 build 完不清理 plist,plist program 就指向老 binary。
   修复后 plist 自带路径漂移检测,后续 dmg 升级自动修正。

## 相关 commit

- `c52ab1a` — fix(launchagent): dmg 装后 plist program 路径漂移自动修复
- `a94ba5f` — fix(desktop): main.go 加启动期异常日志 + 默认 config 锚定 ~/skill-box/
- (本文) — fix(darwin): dmg 任务链传 VITE_DEPLOY_MODE=desktop,前端识别桌面端形态