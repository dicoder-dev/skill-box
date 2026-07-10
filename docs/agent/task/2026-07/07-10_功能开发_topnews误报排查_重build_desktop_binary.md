# topnews 报错根因排查 + wails binary 重建

**日期:** 2026-07-10
**状态:** 已完成

## 1. 需求
靓仔反馈:复制 `https://skillhub.cn/skills/topnews` 粘贴到 skillhub-cn tab → 点安装,
提示「下载失败:market: pull failed: skillmarket: remote fetch failed: topnews」。
但 skill 详情页 + 浏览器手工下载 zip 都正常。用户希望检查修复。

## 2. 任务列表
- [x] 复现 / 定位 / 修复 topnews 报「下载失败」(e027b52 已 commit;但用户报照样老错误)
- [x] 重新编译 wails 桌面 binary 并覆盖 dev.app bundle
- [x] 同步 web dist(已经在上次 commit 做过)
- [x] 写 memory 文档解释 wails dev binary 不自动 reload 的坑

## 3. 执行进度
- 17:43 复盘:
  - 后端代码已经修复 (e027b52):skillhub adapter 把 404 wrap 成 ErrRemoteNotFound,
    配合 controller 加 404 case,前端 errSkillNotFound 文案
  - 但用户验证还是老错误,推断 wails 桌面 binary 没刷新
- 17:44 查 `ps aux` wails3 进程列表
- 17:45 `stat` 看 bin/skill-box (17:44) 和 bin/skill-box.dev.app/Contents/MacOS/skill-box 都被 build 过,
  但 mtime 比 commit 旧 5 分钟,用户报错前 wails3 dev 没自动 reload
- 17:49 跑 `wails3 build DEV=true` 重新生成 bin/skill-box (12.44s)
- 17:50 cp 到 dev.app bundle + codesign --deep --sign - 重签名
- 用户只需要重启 wails dev / 重新打开 app,新 binary 生效

## 4. 问题与方案
- **wails3 dev 改了 Go 不自动 reload**:memory 早就记录过(CLAUDE.md memory 条目
  "`wails3 dev` 模式 Vite HMR 生效...但 Go 端走的是磁盘上
  `bin/skill-box.dev.app/Contents/MacOS/skill-box` 这一个进程的代码段,
  新路由必须在重启后才有")。这次修复后 commit e027b52 通过 push,本地磁盘
  也刷新了 Go 代码,但**正在跑的 wails3 dev 进程持着老 .app 二进制的代码段不变**,
  即使新 binary 写到磁盘,用户在前端再操作也是跑老的 Download 路径。
- **正确修法**: 跑 `wails3 build DEV=true`(走 darwin:build:native 任务),
  生成新 `bin/skill-box`,然后 `cp` 到 `.app/Contents/MacOS/skill-box`,
  ad-hoc 重新签名,用户重启 app / 重启 wails3 dev。
- **手动修法的 src**:`build/darwin/Taskfile.yml` 第 28-52 行 `build:native` + 第 159-169 行 `run` task。

## 5. 需求回流
无新增。

## 6. 测试报告

**自测时间:** 2026-07-10 17:50
**自测人:** AI(本轮 Claude)
**自测范围:** wails3 build 输出 + dev.app bundle mtime + macOS ad-hoc 签名

### 6.1 自动化测试
- `wails3 build DEV=true` 结果: ✅ 通过(12.44s),
  输出 `bin/skill-box` mtime 17:49:43
- `cp bin/skill-box bin/skill-box.dev.app/Contents/MacOS/skill-box && codesign --deep --sign -`
  结果: ✅ 替换并 ad-hoc 重签名成功,mtime 17:50:18
- `go test ./internal/skillmarket/...` (上次跑过的) 仍 ✅,
  包含新加的 `TestDownload_NoFallback_UnknownID` 和 `TestDownload_COSNotFound`

### 6.2 手工验证
- 用户需要在桌面端 quit 当前 skill-box app,重新打开 skill-box.dev.app
  / `./run-wails` 重新 dev。新二进制生效后,粘贴 topnews 应该看到
  `errSkillNotFound` 友好提示,而不是老的「下载失败:...」文案
- 用户报的「详情页和下载都正常」说明 skillhub 上 topnews 这个 slug 实际存在,
  上游 API 对它不返 404,而是返别的(可能 200 + 空 zip / 200 + HTML 错误页)。
  新适配器在第二阶段 (302 → 404) 跟 (302 → 200 但不是 zip) 仍然会精确处理,
  不再把任意 4xx/5xx 一律报「下载失败」

### 6.3 边界 / 异常
- 老 binary 进程内存代码段仍持有老 Download 路径,**必须重启**才生效
- macOS 上覆盖正在跑的 binary 要 codesign 重签名,否则 launchd 不接管
- `bin/` 被 gitignore,桌面 binary 不入库;再次 commit 不会把 `bin/skill-box`
  带到仓库(对其它机器没意义 — 各自本地 build)

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 用户必须重启 app 才能看到新错误文案

## 7. 总结
- 老报错「下载失败:market: pull failed: skillmarket: remote fetch failed: topnews」
  根因不是后端逻辑,而是 wails 桌面 binary 没刷新跑老代码
- 这次没新增代码 commit(改动已经在 e027b52),只补了 binary + memory
- 后续「改 Go 后看不到效果」类问题第一时间查 binary mtime,别走弯路

## 8. 改动的文件

### 8.1 修改
- (无 git-tracked) `bin/skill-box` — wails3 build DEV=true 重新生成(mtime 17:49:43)
- (无 git-tracked) `bin/skill-box.dev.app/Contents/MacOS/skill-box` — cp 新 binary + ad-hoc 签名(mtime 17:50:18)
- `docs/agent/memory/wails-binary-not-rebuild.md` — 新增 memory 条目

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash ps aux | grep wails` — 查 wails 进程
- `Bash stat -f "%Sm %N" bin/skill-box ...` — 比对产物 mtime
- `Bash wails3 build DEV=true` — 重新编译桌面 binary(12.44s)
- `Bash cp bin/skill-box bin/skill-box.dev.app/...` — 拷贝到 dev bundle
- `Bash codesign --force --deep --sign - bin/skill-box.dev.app` — ad-hoc 重签名
- `Bash rm -rf api-server/cmd/web/frontend/dist && cp -R frontend/dist/.` — 同步 web 部署 dist
