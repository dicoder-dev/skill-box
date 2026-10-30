# Updater manifest schema (运维指南)

> 本文件给运营 / CI 维护者看。代码逻辑与字段含义见 `api-server/internal/gapi/service/updater/ssupdater/manifest.go`。
> 选源规则见 `api-server/pkg/updatersrc/updatersrc.go`。

## 1. 一句话

**两个站各托管一份 manifest 指针文件,客户端按用户当前语言选站**:
中文用户走 Gitee,英文用户走 GitHub;选的哪一份 manifest,安装包就从同一个站下载。
两个文件内容除 `assets[].urls` 指向各自站点外完全一致,由 `scripts/release.sh` 发版时生成并提交。

## 2. 文件位置(版本指针)

| 源 | manifest 指针 | 安装包 |
| --- | --- | --- |
| GitHub | `https://raw.githubusercontent.com/dicoder-dev/skill-box/main/build/updater/manifest.github.json` | `https://github.com/dicoder-dev/skill-box/releases/download/v<VERSION>/<file>` |
| Gitee | `https://gitee.com/dicoder/skill-box/raw/main/build/updater/manifest.gitee.json` | `https://gitee.com/dicoder/skill-box/releases/download/v<VERSION>/<file>` |

仓库里对应的文件是 `build/updater/manifest.github.json` / `build/updater/manifest.gitee.json`
(发版前是占位内容,`version` 为 `0.0.0`、`assets` 为空)。

> 客户端不知道最新版本号,所以 manifest 必须放在**固定 URL**(分支 raw),不能放
> `releases/latest/download/manifest.json` 这种形式 —— GitHub 支持该重定向,Gitee 不保证,
> 两边统一用分支 raw 更稳。

## 3. 选源规则

| 用户语言 | 首选源 | 兜底源 |
| --- | --- | --- |
| `zh-CN` | Gitee | GitHub |
| `en-US`(及未知语言) | GitHub | Gitee |

- 语言由前端 `GET /api/desktop/update/check?lang=zh-CN|en-US` 传入,后端按语言排序 manifest 源;
- 首选源拉不到 manifest(超时 / 404 / 解析失败)时自动降级到兜底源;
- 兜底源命中时,安装包地址也会跟着换站(因为 manifest 里 `urls` 本来就是该站自己的地址)。

## 4. Schema

```json
{
  "channel": "stable",
  "version": "1.2.3",
  "released_at": "2026-07-14T12:00:00Z",
  "min_supported": "1.0.0",
  "notes": {
    "zh-CN": "修复若干 bug",
    "en-US": "Several bug fixes"
  },
  "assets": [
    {
      "os": "darwin",
      "arch": "arm64",
      "kind": "app",
      "size": 12345678,
      "sha256": "<sha256-hex>",
      "urls": [
        "https://gitee.com/dicoder/skill-box/releases/download/v1.2.3/skill-box_1.2.3_darwin_universal.zip"
      ]
    }
  ]
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `channel` | string | `stable` / `beta`(MVP 只两个) |
| `version` | string | **严格 semver** `MAJOR.MINOR.PATCH[-prerelease]`,代码用 `golang.org/x/mod/semver` 解析 |
| `released_at` | string | RFC3339 时间戳,UI 显示用 |
| `min_supported` | string | 低于此版本前端加重提示(黄色 banner) |
| `notes` | object | 多语言,key 命中前端 locale,找不到 fallback 到 en-US |
| `assets[]` | array | 每条一个平台二进制 |
| `assets[].os` | string | `darwin` / `windows` / `linux` |
| `assets[].arch` | string | `amd64` / `arm64` |
| `assets[].kind` | string | `app`(mac zip / win zip)/ `appimage`(linux) |
| `assets[].size` | int | 字节数 |
| `assets[].sha256` | string | **强约束**,前端下载完会校验,失败不替身 |
| `assets[].urls` | array of string | 下载地址列表,**顺序即降级顺序**;单源时只有一个元素 |

## 5. 一行校验

```bash
jq -e '
  .version | test("^[0-9]+\\.[0-9]+\\.[0-9]+(-[a-zA-Z0-9.]+)?$")
' manifest.json
```

`-e` 在校验通过时回 `true`,失败 exit code 非 0。CI 用作 fail-fast gate。

## 6. 强制规则

- **必须** 含 `sha256`(MVP 没有公钥分发,只信 SHA256)。
- **必须** `urls.length >= 1`。
- `version` 与文件名约定:**`<appName>_<version>_<os>_<arch>.<ext>`**(release.sh 生成)。
- **不要** 把 manifest 路径直接变成 `releases/latest/index.json` 之类的简化形式 —— GitHub release 不会自动生成 JSON 索引,这条路不通,必须自己生成一份 manifest 并提交到分支。

## 7. 发布流程(`scripts/release.sh`)

```bash
VERSION=1.2.3 CHANNEL=stable \
GITHUB_TOKEN=xxx GITEE_TOKEN=yyy \
GITEE_REPO=dicoder/skill-box GITEE_BRANCH=main \
./scripts/release.sh
```

脚本做这些事:

1. 校验 semver + git 工作树干净;
2. sed 替换各平台版本号(dry-run 模式跳过);
3. build 三平台;
4. 整理安装包:macOS `.app` → zip、Windows `.exe` → zip、Linux AppImage 直接用;
5. 算 SHA256,生成 `build/updater/manifest.{github,gitee}.json` 两份;
6. 上传 GitHub release + Gitee release(API,不走 `gh` CLI);
7. 提交并推送两份 manifest 指针到 GitHub + Gitee 两个远端;
8. HEAD 校验每个下载地址,404 直接失败。

可用 env:

| env | 默认 | 说明 |
| --- | --- | --- |
| `DRY_RUN` | `0` | 只跑到第 5 步,不推不传(演练用) |
| `NO_PUSH` | `0` | 跑完但不调上传 API |
| `GITHUB_REPO` | `dicoder-dev/skill-box` | |
| `GITHUB_TOKEN` | 空 | 必填(除非 DRY_RUN / NO_PUSH) |
| `GITEE_REPO` | `dicoder/skill-box` | 必须是**已存在的公开仓库** |
| `GITEE_USER` | `dicoder` | 推代码时拼 URL 用 |
| `GITEE_TOKEN` | 空 | 必填(除非 DRY_RUN / NO_PUSH) |
| `GITEE_BRANCH` | `main` | **必须是 Gitee 仓库的实际分支**,否则 raw 404 |
| `GITEE_MAX_ATTACH_MB` | `100` | Gitee 单附件上限,超限的包只进 GitHub |

## 8. 已知坑

- **Gitee 附件单文件 100MB 上限**:`bin/Skill-Box` 约 85MB,Windows 安装包更大,
  所以 macOS / Windows 都压缩成 zip 再上传;仍然超限的包只进 GitHub
  (Gitee 那份 manifest 里直接不放这个 asset)。
- **Gitee 默认分支不叫 main**:Gitee 新建仓库默认分支可能是 `master`,
  raw 地址里的分支名写错直接 404。用 `GITEE_BRANCH` 覆盖,并确认仓库已 push 过内容。
- **Gitee raw 会 302 跳转**到实际文件地址,后端 http.Client 默认跟随跳转,不用额外处理。
- **manifest 缓存 5 分钟,缓存键含源地址列表**:切语言后拿到的是新源的 manifest,
  不会出现"中文用户拿到 GitHub 地址的安装包"。
- **历史 bug(已修)**:`release.sh` 早期版本把 `assets[].urls` 写成空数组、`arch` 写死 `amd64`,
  导致下载必失败 / arm64 机器匹配不到 asset。现在由脚本按实际产物生成。

## 9. Phase 2 预留

签名机制(`minisign` 公钥)后续单独迭代,MVP 用 SHA256 强校验兜底。manifest 预留空字段即可(`"signature": ""`)。
