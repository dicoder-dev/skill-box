# Updater manifest schema (运维指南)

> 本文件给运营 / CI 维护者看。代码逻辑与字段含义见 `api-server/internal/gapi/service/updater/ssupdater/manifest.go`。

## 1. 一句话

把一份 JSON manifest 上传到 GitHub release 的一个 **asset**,文件名叫 `manifest.json`,同时挂一份到 Gitea mirror 同一路径。前端从多源数组里按网络可达性挑源。

## 2. 文件位置

| 源 | URL |
| --- | --- |
| GitHub | `https://github.com/<owner>/skill-box/releases/download/v<VERSION>/manifest.json` |
| Gitea | `https://gitea.example.com/<owner>/skill-box/releases/download/v<VERSION>/manifest.json` |

> Gitea 的实际 URL 由公司自行调整,只要求"与 GitHub 同形,公开匿名可访问"。前端按 urls 数组顺序超时降级。

## 3. Schema

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
        "https://github.com/.../skill-box_1.2.3_darwin_arm64.zip",
        "https://gitea.example.com/.../skill-box_1.2.3_darwin_arm64.zip"
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
| `notes` | object | 多语言,key 命中 `__APP_RUNTIME__.lang`,找不到 fallback 到 en-US |
| `assets[]` | array | 每条一个平台二进制 |
| `assets[].os` | string | `darwin` / `windows` / `linux` |
| `assets[].arch` | string | `amd64` / `arm64` |
| `assets[].kind` | string | `app`(mac)/ `installer`(win nsis)/ `appimage`(linux) |
| `assets[].size` | int | 字节数 |
| `assets[].sha256` | string | **强约束**,前端下载完会校验,失败不替身 |
| `assets[].urls` | array of string | 多源,**至少 1 个**;前端按 index 顺序 / HEAD 探测选 |

## 4. 一行校验

```bash
jq -e '
  .version | test("^[0-9]+\\.[0-9]+\\.[0-9]+(-[a-zA-Z0-9.]+)?$")
' manifest.json
```

`-e` 在校验通过时回 `true`,失败 exit code 非 0。CI 用作 fail-fast gate。

## 5. 强制规则

- **必须** 含 `sha256`(MVP 没有公钥分发,只信 SHA256)。
- **必须** `urls.length >= 1`,**强烈建议** >= 2(GitHub + Gitea 两个源)。
- `version` 与 `assets[].kind=installer` 的具体文件名约定:**当前是 `<appName>_<version>_<os>_<arch>.<ext>`**(脚本 release.sh 生成),后续改了 release 流程要同步这条约定。
- **不要** 把 manifest 路径直接变成 `releases/latest/index.json` 之类的简化形式 —— GitHub release 不会自动生成 JSON 索引,这条路不通,必须自己 attach 一个叫 `manifest.json` 的 asset。

## 6. Phase 2 预留

签名机制(`minisign` 公钥)后续单独迭代,MVP 用 SHA256 强校验兜底。manifest 预留空字段即可(`"signature": ""`)。
