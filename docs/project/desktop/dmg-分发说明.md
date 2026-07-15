# Skill-Box macOS DMG 分发说明

> 产物位置:`bin/Skill-Box-arm64.dmg` / `bin/Skill-Box-amd64.dmg`
> 构建入口:`wails3 task darwin:dmg-arm64` / `darwin:dmg-amd64`

## 用户安装流程(就这几步)

```
1. 双击 dmg,把 Skill-Box.app 拖到 /Applications
2. 在 /Applications 找到 Skill-Box.app,双击
3. macOS 弹「无法验证开发者」或一闪而过
4. 打开「系统设置 → 隐私与安全性」
5. 滚到最下面,看到「已阻止使用 Skill-Box,因为它来自身份不明的开发者」
6. 点右边的「仍要打开」按钮
7. 再双击 Skill-Box.app,启动成功
```

**之后每次双击直接启动**,macOS 已把 app 加进 LaunchServices 例外清单。

**这就是 Apple 给非 Developer ID 应用的官方免费入口**,所有非 Apple Developer ID 签名的开源 macOS dmg(happytools、Tauri、Electron-builder 自签)用户都是这么走的。

## 架构选择

| 你的 Mac | 用哪个 dmg |
|---|---|
| Apple Silicon (M1/M2/M3/M4) | `Skill-Box-arm64.dmg` |
| Intel Mac | `Skill-Box-amd64.dmg`(没装 Rosetta 的话,arm64 dmg 启动会失败) |

不需要 universal dmg —— arm64 native 比 Rosetta 转译更稳。

## 当前产物

| 文件 | 架构 | 内含 .app | 签名 |
|---|---|---|---|
| `bin/Skill-Box-arm64.dmg` | arm64 (Apple Silicon) | `Skill-Box.app` | ad-hoc + entitlements |
| `bin/Skill-Box-amd64.dmg` | x86_64 (Intel Mac) | `Skill-Box.app` | ad-hoc + entitlements |

**两个 dmg 里 .app 文件夹统一叫 `Skill-Box.app`** —— 拖到 /Applications 后,Finder / Dock 看到的就是干净名字,不带架构后缀。

**dmg 内容只有这些**:
- `Skill-Box.app`
- `Applications → /Applications` 软链(让用户能拖到 /Applications)
- `.DS_Store`(Finder 布局)

**没有 README、没有 install.sh**。

## 想「双击即开,无任何提示」

走 Apple Developer ID + notarize($99/年 Apple Developer Program)。本 dmg 是 ad-hoc 签名,**只能靠用户手动走 GUI 兜底**。

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

升级后 dmg 用户流程只剩「拖 → 双击 → 启动」,**没有 GUI 兜底那一步**。

## 排错速查

| 现象 | 根因 | 解决 |
|---|---|---|
| dmg 内找不到 `Applications` 软链 | `ln -sfn /Applications` 失败 | 检查 `/Applications` 在宿主是否存在 |
| dmg 双击后 app 图标没布局 | `.DS_Store` 没写入 dmg 根 | 检查 `scripts/build-dmg.sh` AppleScript 是否 `delay 1` + `close winOpts` |
| codesign: `bundle format is ambiguous` | binary 被 `cp` 后未 strip | `go build -ldflags="-w -s"` 已 strip |
| `Apple partition list` 而不是 `zlib compressed` | dmg 转 UDZO 失败 | 检查 `hdiutil convert` 输出 |
| 用户**双击闪退 / 弹「无法验证开发者」** | macOS 26 Tahoe amfi 拒 ad-hoc signed binary | **正常现象**,用户走「系统设置 → 隐私与安全 → 仍要打开」 |

## 相关 commit

- `3b1d68c` — dmg 内 .app 统一叫 Skill-Box.app + binary entitlements
- `f8c059c` — dmg 分发说明文档 + 官方 wails v3 对照
- `4d6142d` — `wails3 task dmg` 默认同时产出 arm64 + amd64 两份 dmg
- `2801b0a` — dmg 内不写 install.sh 和 README,只留 .app(用户明确不要 install.sh)