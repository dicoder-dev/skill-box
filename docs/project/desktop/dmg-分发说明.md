# Skill-Box macOS DMG 分发说明

> 产物位置:`bin/Skill-Box-arm64.dmg` / `bin/Skill-Box-amd64.dmg`
> 构建入口:`wails3 task darwin:dmg-arm64` / `darwin:dmg-amd64`

## 现状(2026-07-16)

本 dmg 是 **ad-hoc 本地签名**,**没有 Apple Developer ID 公证**。在 macOS 26 Tahoe 上,所有走 LaunchServices / amfi 派发链的入口都被 `amfid Code=-423` 静默拒,**包括**:
- Finder 双击 .app
- Finder 右键 → 打开
- `open /Applications/Skill-Box.app`
- `open -a /Applications/Skill-Box.app`

amfi 在 Gatekeeper 之前就把进程杀了,所以「系统设置 → 隐私与安全 → 仍要打开」GUI 兜底**永远不出现**。

## 用户安装流程(macOS 26 Tahoe)

```
1. 双击 dmg,把 Skill-Box.app 拖到 /Applications
2. 在 /Applications 双击 Skill-Box.app
   → 进程会被 amfi -423 静默杀掉(看不到任何提示)
3. 再双击一次
   → binary 检测到 ~/Library/LaunchAgents/com.dicoder.skillbox.plist 未安装
   → 自动写 plist + launchctl bootstrap(注册到 launchd)
   → 本进程继续跑 Serve,8082 LISTEN,程序起来
4. 之后启动方式:
   - 退出程序后,再双击 .app → 进程被杀,但 launchd 那份会接管
   - 或:终端跑 `launchctl kickstart -k gui/$UID/com.dicoder.skillbox` 强制拉起
```

**首次双击需要 2 次**(第一次被杀,第二次 binary 自注册到 launchd),之后 launchd 那份会持续接管。

### 架构选择

| 你的 Mac | 用哪个 dmg |
|---|---|
| Apple Silicon (M1/M2/M3/M4) | `Skill-Box-arm64.dmg` |
| Intel Mac | `Skill-Box-amd64.dmg`(没装 Rosetta 的话,arm64 dmg 启动会失败) |

不需要 universal dmg —— arm64 native 比 Rosetta 转译更稳。

### 当前产物

| 文件 | 架构 | 内含 .app | 签名 |
|---|---|---|---|
| `bin/Skill-Box-arm64.dmg` | arm64 (Apple Silicon) | `Skill-Box.app` | ad-hoc + entitlements(双段签) |
| `bin/Skill-Box-amd64.dmg` | x86_64 (Intel Mac) | `Skill-Box.app` | ad-hoc + entitlements(双段签) |

**两个 dmg 里 .app 文件夹统一叫 `Skill-Box.app`** —— 拖到 /Applications 后,Finder / Dock 看到的就是干净名字,不带架构后缀。

**dmg 内容只有这些**:
- `Skill-Box.app`
- `Applications → /Applications` 软链(让用户能拖到 /Applications)
- `.DS_Store`(Finder 布局)

**没有 README、没有 install.sh、没有 LaunchAgent plist**。LaunchAgent 是 binary 启动时**自动**写到 `~/Library/LaunchAgents/`,不在 dmg 里。

### dmg 内不放 LaunchAgent plist 的原因

考虑过在 dmg 里附 `~/Library/LaunchAgents/com.dicoder.skillbox.plist` 让用户手动复制,但这条路被否决:
- dmg 写磁盘时,用户没勾选「替换」就会失败
- dmg 卸载后 plist 还在,用户感觉不到
- binary 自注册是更优雅的方案

## 想「双击即开,无任何提示」

走 Apple Developer ID + notarize($99/年 Apple Developer Program)。本 dmg 是 ad-hoc 签名,**必须靠 binary 自注册到 launchd 才能起**。

### 一次性环境准备

1. 加入 Apple Developer Program:https://developer.apple.com/programs/
2. 申请 Developer ID Application 证书,下载安装到 Keychain
3. 创建 app-specific password:https://appleid.apple.com/account/manage → App-Specific Passwords
4. 存 notarytool credentials 到 keychain:
   ```bash
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

当前 ad-hoc 两段签会变成走官方 `wails3 tool sign`:

```bash
codesign --force --sign - --entitlements build/darwin/entitlements.plist \
  Skill-Box.app/Contents/MacOS/Skill-Box
wails3 tool sign \
  --input Skill-Box.app \
  --identity "Developer ID Application: Your Company (TEAMID)" \
  --entitlements build/darwin/entitlements.plist \
  --hardened-runtime \
  --notarize \
  --keychain-profile skillbox-notary
```

升级后 dmg 用户流程只剩「拖 → 双击 → 启动」,**没有 macOS 26 Tahoe amfi -423 问题**。

### 实测验证 ad-hoc + hardened runtime 不通

2026-07-16 实测:在 dmg build 里加 `--options runtime` 后,`codesign -dvvv` 显示 `flags=0x10002(adhoc,runtime)`,但 `open -a /Applications/Skill-Box.app` **依然被 amfi -423 拒**。

参考 mixxxdj/mixxx PR #12290(2025-07-22)同样试过 ad-hoc + hardened runtime,**已关闭**(amfi 仍拒绝 ad-hoc + runtime)。

**这条路在 macOS 26 Tahoe 上走不通,只能 Developer ID**。

## 排错速查

| 现象 | 根因 | 解决 |
|---|---|---|
| dmg 内找不到 `Applications` 软链 | `ln -sfn /Applications` 失败 | 检查 `/Applications` 在宿主是否存在 |
| dmg 双击后 app 图标没布局 | `.DS_Store` 没写入 dmg 根 | 检查 `scripts/build-dmg.sh` AppleScript 是否 `delay 1` + `close winOpts` |
| codesign: `bundle format is ambiguous` | binary 被 `cp` 后未 strip | `go build -ldflags="-w -s"` 已 strip |
| `Apple partition list` 而不是 `zlib compressed` | dmg 转 UDZO 失败 | 检查 `hdiutil convert` 输出 |
| 用户**双击一闪而过**(系统设置没记录) | **macOS 26 Tahoe 正常现象** | 再双击一次,触发 binary LaunchAgent 自注册 |
| 用户**双击一闪而过,第二次也一闪而过** | LaunchAgent 自注册失败 | 看 `~/.skill-box/logs/launchagent.log` |
| 用户**双击一闪而过,第二次 8082 不监听** | launchd kickstart 死循环(场景 C) | 终端跑 `launchctl kickstart -k gui/$UID/com.dicoder.skillbox` 强制拉一次 |

## 相关 commit

- `3b1d68c` — dmg 内 .app 统一叫 Skill-Box.app + binary entitlements
- `f8c059c` — dmg 分发说明文档 + 官方 wails v3 对照
- `4d6142d` — `wails3 task dmg` 默认同时产出 arm64 + amd64 两份 dmg
- `2801b0a` — dmg 内不写 install.sh 和 README,只留 .app(用户明确不要 install.sh)
- TBD — binary 加 LaunchAgent 自注册 + dmg 文档重写为 macOS 26 Tahoe 真实路径