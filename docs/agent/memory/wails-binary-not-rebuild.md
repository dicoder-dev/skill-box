- name: 改名 / 新增功能必跑 wails3 build
  description: Go 后端改名 (常量化、groupPath) 后必须跑 `wails3 build DEV=true` 重新生成 `bin/skill-box` 和 `bin/skill-box.dev.app/Contents/MacOS/skill-box`,老的 wails3 dev 进程不会自动 reload 新代码,内存里的代码段还是上次的 build。表现:用户报"修复没生效"但 `go test ./internal/...` 全绿。
metadata:
  type: project

# 改名 / 重构后必须重新 build 桌面二进制

`wails3 dev` 启动后,Go 端代码段会 map 进进程内存,后续 *.go 改动理论上触发
blocking rebuild (看 build/config.yml 的 `executes: wails3 build DEV=true`),但**实际
 跑经常 build 失败 / 卡住 / rebuild 产物没替换 dev.app bundle**,用户以为生效了
 但老 binary 还在跑。

## 排错路径(2026-07-10 案例:skillhub → skillhub-cn 改名 + topnews 404 区分)

1. **查进程**: `ps aux | grep -E "skill-box|wails|web"` 看 wails3 dev 主进程 PID
2. **查产物时间**: `stat -f "%Sm %N" bin/skill-box bin/skill-box.dev.app/Contents/MacOS/skill-box`
   - 两个 mtime 应该都跟最近 commit 时间接近
   - 如果 `.app/MacOS/skill-box` 比 `bin/skill-box` 旧 → dev.app bundle 没刷新
3. **手动刷新**:
   ```bash
   wails3 build DEV=true
   cp bin/skill-box bin/skill-box.dev.app/Contents/MacOS/skill-box
   codesign --force --deep --sign - bin/skill-box.dev.app
   ```
   1) build 生成新 `bin/skill-box` (wails3 task darwin:build:native 的 OUTPUT)
   2) 拷到 `.app` bundle (build/darwin/Taskfile.yml `run` task 第 165 行 cp 步骤)
   3) ad-hoc 重签名(macOS 不签名没法覆盖运行的 binary)
4. **用户重启 app**:`./run-wails` 选 1 重新 wails3 dev;或者直接关掉重启新 binary
5. **验证**: `stat -f "%Sm" bin/skill-box.dev.app/Contents/MacOS/skill-box` 时间戳应新于最近 commit

## 易混淆点

- `bin/` 在 `.gitignore` 里,桌面 binary **不入库**,commit 里看不到 Go 改动对 binary 的影响
- `frontend/dist` 和 `api-server/cmd/web/frontend/dist` **不入库**(gitignore),本地 build 完同步
- Web 单进程 binary `bin/web` 也不入库
- 真正入库的只有 Go 源码 + 前端源码 + 任务文档

## 教训

- 用户反馈「修改没生效」时,先 `stat` 看 binary 时间,不要盲目贴磁盘文件路径
- `bin/skill-box.dev.app/Contents/MacOS/skill-box` 是 wails3 dev 实际跑的二进制,跟 `bin/skill-box` 不是自动同步的
- macOS 上覆盖正在跑的 binary 必须先 quit app,或者用 `codesign --deep --sign -` ad-hoc 重签名后 launchctl 接管
