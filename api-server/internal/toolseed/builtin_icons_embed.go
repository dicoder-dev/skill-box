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
var builtinIconNames = []string{
	"claude.ico",
	"codex.png",
	"cursor.png",
	"opencode.png",
	"trae.png",
	"antigravity.png",
	"cline.png",
	"codebuddy.svg", // codebuddy.svg + codebuddy.png 二选一,seed 只放 svg
	"codebuddy.png",
	"jetbrains.ico",
}
