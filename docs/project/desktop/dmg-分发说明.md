# Skill-Box macOS DMG 分发说明

> 产物位置:`bin/Skill-Box-arm64.dmg` / `bin/Skill-Box-amd64.dmg`
> 构建入口:`wails3 task darwin:dmg-arm64` / `darwin:dmg-amd64`

## 当前产物

| 文件 | 架构 | 内含 .app | 签名 |
|---|---|---|---|
| `bin/Skill-Box-arm64.dmg` | arm64 (Apple Silicon) | `Skill-Box.app` (无 -arm64 后缀) | ad-hoc + entitlements |
| `bin/Skill-Box-amd64.dmg` | x86_64 (Intel Mac) | `Skill-Box.app` (无 -amd64 后缀) | ad-hoc + entitlements |

**两个 dmg 里 .app 文件夹统一叫 `Skill-Box.app`**——用户拖到 /Applications 后,在 Finder / Dock 看到的就是干净名字,不带架构后缀。

## macOS 26 首次打开流程(必读)

**这是 macOS 26 (Tahoe) 的硬限制,不是 dmg 打包 bug**:

本 dmg 是 **ad-hoc 本地签名**,**不是 Apple 公证**(notarized)版本。macOS 26 默认拒绝 `open` / Finder 双击 ad-hoc 签名的 .app,弹"你不能打开应用程序,因为它可能已损坏或不完整"或静默不响应。

> 想"双击即开",必须 Apple Developer ID + Apple 公证服务($99/年),详见末尾「正式发版:Developer ID 升级路径」。

### 方法一:右键打开(最快,推荐)

1. 双击 `Skill-Box-arm64.dmg`,弹出 Finder 窗口
2. 把 `Skill-Box.app` 拖到右侧 `Applications` 软链
3. 弹出 dmg(把 dmg 拖到废纸篓,或 `hdiutil detach "/Volumes/Skill-Box"`)
4. 在 `/Applications` 找到 `Skill-Box.app`,**右键 → 打开**
5. 系统弹"无法验证开发者"对话框,点「打开」
6. 之后双击就能正常启动

### 方法二:系统设置放行

1. 拖完 app 后,**先双击一次**(会失败、无任何提示)
2. 打开「系统设置 → 隐私与安全性」
3. 滚到页面底部,看到"已阻止使用 Skill-Box,因为它来自身份不明的开发者"
4. 点右边的「仍要打开」按钮
5. 再双击 `Skill-Box.app`,正常启动

### 方法三:命令行一键(开发者/CLI 用户,最快)

dmg 内附带 `install.sh`,免去手动 xattr + open:

```bash
# 双击 dmg 后,在弹出的 Finder 窗口右键 install.sh → 在终端打开
# 或在终端执行:
bash /Volumes/Skill-Box/install.sh
# 自动:
#   1. 拷 Skill-Box.app 到 /Applications
#   2. xattr -cr 清除 quarantine(解决 "已损坏" 报错)
#   3. open 启动
```

如果不方便跑 install.sh,直接一行复制粘贴:
```bash
xattr -cr /Applications/Skill-Box.app && open /Applications/Skill-Box.app
```

`xattr -cr` 是 macOS 社区对付 ad-hoc/local build .app 的**最常用一招** —— GitHub issue **electron-builder/electron-builder#8191**、Tauri 文档、RustDesk issues 都用它绕过 Gatekeeper。

### 方法四:命令行绕过(开发期 / 脚本)

```bash
# 跳过 LaunchServices 和 Gatekeeper,直接 exec binary
nohup /Applications/Skill-Box.app/Contents/MacOS/Skill-Box > /tmp/sb.log 2>&1 &
# 后端起在 8082
curl http://localhost:8082/api/v1/system/health
```

直接 exec 不走 Gatekeeper,ad-hoc 签名也照常启动。

## 架构选择

- **Apple Silicon Mac (M1/M2/M3/M4)**:用 `Skill-Box-arm64.dmg`
- **Intel Mac**:用 `Skill-Box-amd64.dmg`(Intel Mac 没装 Rosetta 的话,arm64 dmg 启动会失败)
- 不需要 universal dmg(arm64 native 比 Rosetta 转译更稳)

## 实现关键点

### 1. 两段签名(macOS 26 必需)

```bash
# 1) 先签 binary —— LaunchServices 校验 binary 的 entitlements 段
codesign --force --sign - --entitlements build/darwin/entitlements.plist \
  Skill-Box.app/Contents/MacOS/Skill-Box

# 2) 再签 .app --deep 把上面 binary 的签名带过去
codesign --force --deep --sign - --entitlements build/darwin/entitlements.plist \
  Skill-Box.app
```

**不**用 `--options runtime` —— ad-hoc + hardened runtime 在 macOS 26 比单 ad-hoc 更严,反而触发"损坏"误报。

**对照官方 wails v3 alpha.60 `internal/commands/sign.go`**:官方默认走单段 `codesign --deep --sign --entitlements`,**不带 `--options runtime`**(只有 `--hardened-runtime` 或 `--notarize` 才加)。我们两段签是 macOS 26 LaunchServices 的特判,在官方默认之上多签一次 binary。

### 2. dmg 两段法(可拖拽布局必需)

```bash
# 1) UDRW staging(macOS 不允许 UDRW/UDRO 用 -mountpoint /Volumes/...,
#    用 /tmp/dmg-mount.$$ 兜底)
hdiutil create -srcfolder Skill-Box.app -format UDRW -fs HFS+ -volname Skill-Box -size 200m Skill-Box.dmg.staging.dmg
hdiutil attach -readwrite -nobrowse -mountpoint /tmp/dmg-mount.$$ Skill-Box.dmg.staging.dmg

# 2) 写 Finder 布局:icon view、bounds、arrangement not arranged、
#    关闭触发 .DS_Store 落盘
ln -sfn /Applications /tmp/dmg-mount.$$/Applications
osascript <<'OSA'
tell application "Finder"
  set winOpts to container window of (POSIX file "/tmp/dmg-mount.$$" as alias)
  set current view of winOpts to icon view
  set arrangement of icon view options of winOpts to not arranged
  set position of item "Skill-Box.app" of winOpts to {170, 190}
  set position of item "Applications" of winOpts to {410, 190}
  close winOpts
end tell
OSA
hdiutil detach /tmp/dmg-mount.$$

# 3) UDZO 压缩
hdiutil convert Skill-Box.dmg.staging.dmg -format UDZO -o Skill-Box-arm64.dmg
```

**对照官方 wails v3 alpha.60 `internal/commands/dmg/dmg.go`**:官方走单段 `hdiutil create -format UDZO -srcfolder`,**不写 Finder 布局**。我们两段法是为了让 dmg 里有可拖拽布局,这是用户体验上的关键差异。

### 3. Trap 自动清理

`scripts/build-dmg.sh` 用 `trap cleanup EXIT INT TERM`,任何退出路径(成功 / 失败 / Ctrl-C)都会自动 detach 挂载点 + 删 staging dmg,不留半成品。

### 4. 关键防御点

- `UDRW` staging 而不是 `UDRO` —— UDRO 不能 `-readwrite` 挂(macOS 报"操作不被允许")
- `HFS+` 而不是 APFS —— Finder 拖拽布局在 HFS+ 唯一稳定
- AppleScript 先 `set current view to icon view` 再 `set bounds`,先 `set arrangement to not arranged` 再 `set position`(默认网格会把图标 snap 到错坐标)
- `delay 1` 让 Finder flush `.DS_Store` 到 dmg 根
- 不走 `tell disk "Skill Box"`(会报 -1728),改用 `POSIX file` 直接定位挂载点
- dmg 内放 `首次打开说明.txt`,用户看到指引不用瞎猜

## 正式发版:Developer ID 升级路径

要从 ad-hoc 升级到 "双击即开" 的正式发版,需要 Apple Developer ID + 公证服务:

### 一次性环境准备

```bash
# 1. 加入 Apple Developer Program($99/年)
#    https://developer.apple.com/programs/

# 2. 申请 Developer ID Application 证书 + 下载安装到 Keychain

# 3. 创建 app-specific password 用于 notarytool
#    https://appleid.apple.com/account/manage → App-Specific Passwords

# 4. 存 notarytool credentials 到 keychain
xcrun notarytool store-credentials \
  --apple-id "you@email.com" \
  --team-id "TEAMID" \
  --password "abcd-efgh-ijkl-mnop" \
  --profile "skillbox-notary"
```

### 配置 build/darwin/Taskfile.yml

```yaml
vars:
  SIGN_IDENTITY: "Developer ID Application: Your Company (TEAMID)"
  KEYCHAIN_PROFILE: "skillbox-notary"
  ENTITLEMENTS: "build/darwin/entitlements.plist"
```

### 改 dmg-arm64 / dmg-amd64 签名段

把当前两段签改成官方 `wails3 tool sign` + `--hardened-runtime --notarize`:

```yaml
- codesign --force --sign - --entitlements build/darwin/entitlements.plist "{{.BIN_DIR}}/{{.APP_NAME}}.app/Contents/MacOS/{{.APP_NAME}}"
- wails3 tool sign \
    --input "{{.BIN_DIR}}/{{.APP_NAME}}.app" \
    --identity "{{.SIGN_IDENTITY}}" \
    --entitlements {{.ENTITLEMENTS}} \
    --hardened-runtime \
    --notarize \
    --keychain-profile {{.KEYCHAIN_PROFILE}}
```

### 不需要改的地方

- dmg 打包脚本 `scripts/build-dmg.sh`(布局逻辑跟签名无关)
- `首次打开说明.txt`(Developer ID 版可改成"如果没有自动启动,请联系开发者")

## 排错速查

| 现象 | 根因 | 解决 |
|---|---|---|
| `you cannot open ... damaged or incomplete` | binary 没带 entitlements,macOS 26 Gatekeeper 拒绝 | 重跑 dmg 任务,两段签会重新嵌入 |
| `open` 静默无反应 | macOS 26 对 ad-hoc 拒绝 LaunchServices 启动 | 右键 → 打开 / 系统设置放行 |
| `spctl --assess` rejected | ad-hoc 永远 reject(预期) | 升级到 Developer ID 才不 reject |
| dmg 双击后 app 图标没布局 | `.DS_Store` 没写入 dmg 根 | 检查 AppleScript 是否 `delay 1` + `close winOpts` |
| dmg 内找不到 `Applications` 软链 | `ln -sfn /Applications` 失败 | 检查 `/Applications` 在宿主是否存在 |
| codesign: `bundle format is ambiguous` | binary 被 `cp` 后未 strip | `go build -ldflags="-w -s"` 已 strip |
| `Apple partition list` 而不是 `zlib compressed` | dmg 转 UDZO 失败 | 检查 `hdiutil convert` 输出 |

## 开源项目常见做法对比

调研 GitHub Issues / Reddit / CSDN 后的总结(不申请 Apple Developer ID 也能让用户跑起来的方案):

| 方案 | 适用场景 | 局限 | 我们采用 |
|---|---|---|---|
| **右键 → 打开 / 系统设置放行** | 普通用户单次放行 | macOS 15+ 唯一官方 GUI 路径 | ✅ 写在 dmg README 方法一 |
| **`xattr -cr <app>` 清 quarantine** | CLI 用户 / 装一次到本机 | 不是普适,用户要知道这命令 | ✅ 写在 dmg README 方法二 + `install.sh` 自动执行 |
| **`sudo spctl --master-disable`(任何来源)** | 全局放行所有 ad-hoc | macOS 15+ Apple 隐藏此开关,系统设置仍需二次确认 | ❌ 不推荐 |
| **`hdiutil convert -format UDRO`(只读 staging)再 detach** | dmg 离线分发 | 不能 install.sh 这种交互 | ❌ 我们用 UDRW 才能 mount 后写 |
| **Homebrew Cask `brew install --cask skill-box`** | 用户已有 Homebrew | Cask 自身走 Apple 公证链路(2024+),不能装未公证 | 后续 PR 单独做 |
| **Apple Developer ID + notarize** | 正式发版 "双击即开" | 需要 $99/年 Apple Developer 账号 | 文档已写升级路径 |
| **Tauri/RustDesk/GitHub Desktop 都走 Developer ID** | 大厂有预算 | 个人/小团队不现实 | 已知,我们走 ad-hoc + 提示方案 |

**关键证据**(2026-07-15 验证过):
- 我们 dmg 内 `.app` 和 binary **本身没有任何 xattr**(我们 build 阶段不写 quarantine)。
- dmg 本身的 quarantine 是 **下载渠道**加的(浏览器/QQ 拖过来),build 链解不了。
- 用户从 dmg 拖出 .app 后,LaunchServices 才标 quarantine → 这时候 `xattr -cr` 能清掉 → 真正"无法打开"提示就是因为 quarantine 在。
- GitHub issue [electron-builder/electron-builder#8191](https://github.com/electron-userland/electron-builder/issues/8191) —— 跟我们现象完全一致,fix 主流就是 `xattr -cr`。

## 相关 commit

- `3b1d68c` — dmg 内 .app 统一叫 Skill-Box.app + binary entitlements
- `f8c059c` — dmg 分发说明文档
- `TBD` — dmg 内新增 `install.sh`(一键 xattr -cr + open)+ README 加方法二 + docs 加开源做法对比
