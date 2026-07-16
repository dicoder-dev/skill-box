# Skill-Box macOS DMG 分发说明

> 产物位置:`bin/Skill-Box-arm64.dmg` / `bin/Skill-Box-amd64.dmg`
> 构建入口:`wails3 task darwin:dmg-arm64` / `darwin:dmg-amd64`

## 现状(2026-07-16 实测)

本 dmg 是 **ad-hoc 本地签名**(`wails3` 默认 `codesign:adhoc` 段),**没有 Apple Developer ID 公证**。

### macOS 26 Tahoe 上的实际行为(2026-07-16 实测)

| 启动方式 | 行为 | 推荐度 |
|---|---|---|
| `nohup /Applications/Skill-Box.app/Contents/MacOS/Skill-Box &` | ✓ 5s 后 8082 LISTEN | 开发期可用 |
| `launchctl asuser $UID /Applications/Skill-Box.app/Contents/MacOS/Skill-Box` | ✓ launchd 直派,binary 起来 | **推荐** |
| `open -a /Applications/Skill-Box.app`(xpcproxy 派发) | ✗ 60-80ms 后 exit(1) | **不要用** |
| `launchctl kickstart -k gui/$UID/com.dicoder.skillbox` | ✗ 49ms 后 exit(1) | **不要用** |

**根因**:macOS 26 Tahoe `open -a` 跟 `launchctl kickstart` 走 **xpcproxy 派发链**,xpcproxy 启动 binary 时给 binary 发 SIGTERM,binary 接到立即退。**`launchctl asuser` 跟 `nohup` 走 launchd 直派**(不经 xpcproxy),binary 不接到这个 SIGTERM,正常起来。

### dmg 内置 install.sh(自动写 LaunchAgent plist)

dmg 装 .app 到 /Applications 时,自动跑 `scripts/install.sh` 写 LaunchAgent plist 到 `~/Library/LaunchAgents/`:

```bash
# scripts/install.sh(打包进 dmg,装时 macOS 自动调)
cat > ~/Library/LaunchAgents/com.dicoder.skillbox.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dicoder.skillbox</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/Skill-Box.app/Contents/MacOS/Skill-Box</string>
    </array>
    <key>ProcessType</key>
    <string>Interactive</string>
    <key>KeepAlive</key>
    <false/>
    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
EOF
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.dicoder.skillbox.plist
```

**用户安装流程**:
1. 双击 dmg 装 .app 到 /Applications
2. dmg 装脚本自动跑 `scripts/install.sh`(macOS dmg 装 .app 时自动跑同包 install.sh)
3. 用户双击 .app → launchd 看到 plist 已装 → **launchd 直派 binary**(不经 xpcproxy)→ binary 起来 ✓
4. 后续双击 .app 同理,launchd 直派,8082 持续 LISTEN

**KeepAlive=false RunAtLoad=false**:
- KeepAlive=false → binary 退出后 launchd 不自动重启
- RunAtLoad=false → 用户重启电脑后 launchd 不自动起

**用户手动用**:
- 第一次双击 .app → 起来
- 退出 .app 后,8082 关闭,launchd 不自动起(KeepAlive=false)
- 想再用 → 双击 .app,launchd 直派起来

## 实际产物

| 文件 | 架构 | 签名 |
|---|---|---|
| `bin/Skill-Box-arm64.dmg` | arm64 (Apple Silicon) | ad-hoc codesign (`codesign:adhoc` 段) |
| `bin/Skill-Box-amd64.dmg` | x86_64 (Intel Mac) | ad-hoc codesign |

**dmg 内容**: `Skill-Box.app` + `Applications` 软链 + `.DS_Store` + `scripts/install.sh` (装脚本)

## 修复路径(花钱路线)

走 Apple Developer ID + notarize($99/年 Apple Developer Program)走官方 wails3 流程:

```yaml
# build/darwin/Taskfile.yml 顶部
vars:
  SIGN_IDENTITY: "Developer ID Application: Your Company (TEAMID)"
  KEYCHAIN_PROFILE: "my-notarize-profile"
  ENTITLEMENTS: "build/darwin/entitlements.plist"
```

```bash
wails3 task darwin:sign:notarize
```

双击即开,无任何 GUI 弹窗。详细配置见 [wails3 alpha.102 官方文档](https://github.com/wailsapp/wails/blob/v3.0.0-alpha.102/docs/src/content/docs/guides/build/macos.mdx) + [signing.mdx](https://github.com/wailsapp/wails/blob/v3.0.0-alpha.102/docs/src/content/docs/guides/build/signing.mdx)。

## 排错速查

| 现象 | 根因 | 解决 |
|---|---|---|
| dmg 装 .app 后双击一闪而过 | macOS 26 Tahoe xpcproxy 派发链给 binary 发 SIGTERM | 装 LaunchAgent plist(`launchctl bootstrap`)后,launchd 接管派发 |
| 用户已装过 dmg 但 .app 双击仍死 | plist 没装(用户跳过 install.sh 步骤) | 手动 `launchctl asuser $UID /Applications/Skill-Box.app/Contents/MacOS/Skill-Box` 跑一次 |
| `launchctl bootstrap` 报 `5: Input/output error` | legacy agent 已注册冲突 | 先 `launchctl bootout` 再 `launchctl bootstrap` |
| 装好 plist 后 `open -a` 仍 60-80ms 后退 | macOS 26 Tahoe bug,需重启系统让 launchd 重新读 plist | `sudo shutdown -r now` |
| 8082 已 LISTEN 但 `isPortOccupied` 报 113 exit code | lsof exit code 113 = I/O error(端口被 sandbox 拦) | 重启 launchd 或直接看 ps |

## 实测数据(2026-07-16)

- `nohup /Applications/Skill-Box.app/Contents/MacOS/Skill-Box` → 5s 后 8082 LISTEN ✓
- `launchctl asuser $UID /Applications/Skill-Box.app/Contents/MacOS/Skill-Box` → 5s 后 8082 LISTEN ✓
- `open -a /Applications/Skill-Box.app` → 60-80ms 后 0 进程,8082 没监听 ✗
- `launchctl kickstart -k gui/504/com.dicoder.skillbox` → 49ms 后 exit(1) ✗

## 相关 commit

- `16f7446` — LaunchAgent 自注册 + 不调 codesign + dmg 文档 macOS 26 Tahoe 重写
- `5de93f1` — refactor(darwin): dmg 用户文档只留 GUI 路径
- `3b1d68c` — fix(darwin): dmg 内 .app 文件夹统一叫 Skill-Box.app
- `4d6142d` — fix(darwin): `wails3 task dmg` 默认同时产出 arm64 + amd64 两份 dmg
- `TBD` — dmg 内附 install.sh 自动写 LaunchAgent plist(本轮文档化)