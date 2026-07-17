// Package toolseed - builtin_icons_embed.go
//
// 把 9 个内置工具的真实图标(从官方源下载)用 //go:embed 嵌到 Go 二进制,
// seed 阶段写到 ~/.skill-box/tool-icons/ 让前端自托管接口能立即服务出来。
//
// 为什么不直接用前端 assets?
//   - Go //go:embed 只能 embed 本包同 module 内的实体文件;
//     frontend/src/assets 不在 api-server module 内,无法跨 module embed。
//   - 这里在 api-server internal 下再放一份,Go 直接读本目录 embed.FS。
//
// 为什么不直接 embed 到 stool/toolicon 包?
//   - toolseed 是唯一一个"知道内置图标长啥样"的层;让 seed 同时管文件存在性
//     比让 icon 包自己 embed 更清晰。
//   - 用户上传走的是另一条路径(stool + ctool 上传),不在 embed 范围。
package toolseed

import "embed"

//go:embed builtin-icons/*
var builtinIconsFS embed.FS

// builtinIconNames 内置图标的文件名 — 与 builtin.go 中内置工具的 IconFile 字段对应。
// 任何修改这里都要同步改 builtin.go 中 bt.IconFile 字段。
//
// 2026-07-03 更新:从临时占位图标换成联网搜集到的官方源:
//   - claude.ico    Anthropic 官方
//   - codex.png     OpenAI 官方(Codex 是 OpenAI 产品)
//   - cursor.ico    cursor.com 官方 favicon(多尺寸)
//   - opencode.svg  sst/opencode 仓库 brand 资源(嵌套方框)
//   - trae.ico      trae.com.cn 国内 CDN favicon
//   - antigravity.ico antigravity.google 官方 favicon
//   - cline.png     cline/cline 仓库 assets/icons/icon.png
//   - codebuddy.png + .svg  codebuddy.tencent.com 真 favicon
//   - jetbrains.ico jetbrains.com 官方 favicon
//
// 2026-07-18 大扩:内置工具从 9 增到 17,新增图标(已联网抓到):
//   - copilot.ico   github.com 站点 favicon
//   - windsurf.ico  windsurf.com 官方 favicon(原 codeium.com)
//   - goose.ico     block.github.io/goose 官方 icon(对应 block/goose)
//   - roo.ico       roomote.dev 官方 favicon(对应 RooCodeInc/Roo-Code)
//   - continue.png  continuedev/continue 仓库 docs/static/img/logo.svg
//
// 2026-07-18 暂时回退:
//   - windsurf / goose / hermes 的官方真 logo 暂时拿不到(自动抓的 webfetch
//     缓存 hash 撞了,sha 一致证明是同一张图,显然不对),删 IconFile 让前端
//     走 mdi 兜底。
//   - codebuddy.png 是早期的占位 HTML 文件(<!DOCTYPE html>...开头),不是
//     真 PNG;前端实际是 codebuddy.svg,这里 builtinIconNames 撤掉 codebuddy.png
//     避免 embed.FS 把它读进去 writeBuiltinIcons 时报错。
//
// 用户手动提供的话:文件名后缀须在 allowedExts 白名单内(.png/.svg/.jpg/.jpeg/
// .webp/.ico/.gif),下载下来 cp 到 builtin-icons/ + 改 builtin.go IconFile +
// builtin_icons_embed.go builtinIconNames,启动时 upsertBuiltinTools 刷新老 DB。
var builtinIconNames = []string{
	"claude.ico",
	"codex.png",
	"cursor.ico",
	"opencode.svg",
	"trae.ico",
	"antigravity.ico",
	"cline.png",
	"codebuddy.svg",
	"jetbrains.ico",
	// 2026-07-18 联网抓
	"copilot.ico",
	"roo.ico",
	"continue.png",
	// 2026-07-18 用户手动提供
	"openclaw.svg",
	"aider.svg",
}
