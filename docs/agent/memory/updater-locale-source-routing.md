---
name: updater-locale-source-routing
description: 应用自升级按用户语言分源(中文 Gitee / 英文 GitHub)的选源设计、指针文件约定与 Gitee 侧的坑
metadata:
  type: project
---

# 应用更新按语言分源(Gitee / GitHub)

**2026-10-30 落地**。目标:国内用户从 Gitee 拉更新,英文用户从 GitHub 拉。

## 设计要点

- **选源逻辑单点**:`api-server/pkg/updatersrc/`(放 pkg 不放 internal,因为桌面端模块
  `skill-box/desktop/services` 也要用,internal 跨模块不可 import)。
  - `PreferredIDs(locale)`:zh-CN → [gitee, github];其余 → [github, gitee]。
  - `ManifestURLs(sources, locale)`:按语言排好序的 manifest url 列表(顺序 = 降级顺序)。
- **语言从哪来**:前端 `GET /api/desktop/update/check?lang=zh-CN|en-US` 传,后端存进
  `RequestUpdateCheck.Lang` / `RequestUpdateDownload.Lang`。**不要指望 Accept-Language**,
  桌面端 webview 那条链路不可靠。
- **两站各一份 manifest 指针**(不是一份 manifest 带两个 host 的 urls):
  `build/updater/manifest.github.json` / `manifest.gitee.json`,除 `assets[].urls` 指向
  各自站点外内容一致,发版时由 `scripts/release.sh` 生成并提交。
  → 好处:选源对整条链路生效(manifest 与安装包同站),不会出现"manifest 从 A 站取、
  安装包从 B 站拉"的半截状态;兜底换源时安装包地址自动跟着换。
- **manifest 缓存键必须含源地址列表**(不只是 channel):两个源的 manifest 内容不同
  (urls 指向不同站),只按 channel 缓存会让"切语言后拿到上一个源的 manifest"。
  见 `ssupdater.cacheKey`。

## 坑

- **Gitee 附件单文件 100MB 上限**:`bin/Skill-Box` ~85MB,Windows 安装包更大 →
  macOS/Windows 统一压 zip 上传;仍超限的包只进 GitHub(Gitee 那份 manifest 不放该 asset)。
- **Gitee 默认分支可能叫 `master`**:raw 地址里的分支名写错直接 404,
  `GITEE_BRANCH` 可覆盖(默认 main)。
- **Gitee 没有 `releases/latest/download/<file>` 重定向的保证**:版本指针统一放分支 raw。
- **Gitee raw 会 302 跳转**到真实文件地址,Go http.Client 默认跟随,不用特殊处理。
- **release.sh 历史 bug(已修)**:manifest 里 `assets[].urls` 写成 `[]`、`arch` 写死 `amd64`
  → 下载必失败 / arm64 机器匹配不到 asset。现在按实际产物生成。
- **Gitee release 附件直链格式与 GitHub 同形**:
  `https://gitee.com/<owner>/<repo>/releases/download/<tag>/<file>`。

## 相关

- 运维文档:`docs/agent/project/updater-manifest.md`
- 发布脚本:`scripts/release.sh`(用 curl 调 API,不再依赖 `gh` CLI)
