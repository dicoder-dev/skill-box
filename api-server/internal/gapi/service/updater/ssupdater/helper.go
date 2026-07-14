// Package ssupdater 包级 helper.go 集中描述替身脚本的执行顺序硬规则。
//
// 真实脚本由 desktop/assets/updater/embed.go 用 //go:embed 注入,
// 运行时由 desktop/wails_app.go 的 BootstrapHooks.UpdaterSpawnHelper 取出 fork。
// 本文件是"协议层"——任何对 SpawnHelper 调用顺序的修改都必须先读这一段。
// 包级,与其他 4 个文件一致(manifest.go / compare.go / download.go / verify.go 都是 ssupdater)。
package ssupdater

// 脚本协议(三个平台统一约定,顺序敏感):
//
// 1) helper <DEST> <TARGET_INSTALL_DIR> <OS> <ARCH> <OLD_VERSION>
//    - DEST: 已下载文件(zip / .app / installer / AppImage)的本地绝对路径
//    - TARGET_INSTALL_DIR: 桌面端"安装目录"(macOS: /Applications/SkillBox.app;
//      Windows: 与 desktop.exe 同目录;Linux: ~/.local/bin/skill-box 或 AppImage 自身)
//    - OS / ARCH: darwin/arm64 等,辅助 helper 决定解压位置
//    - OLD_VERSION: 升级前版本,作为 env SKILLBOX_UPDATER_FROM 透传给新进程
//
// 2) helper 必须 sleep 2s 等待父进程(wails app)释放端口 / 释放可执行文件锁;
//
// 3) 失败自包含回滚:任何一步失败,返回非 0,并把 .bak 还原。
//
// 4) helper 启动新二进制前,把 OLD_VERSION 写到 env SKILLBOX_UPDATER_FROM,
//    desktop.startupAsync 协程会读这个 env 决定是否弹"升级成功/失败"通知。

// SpawnOrder 给 controller 参考的"先 helper 再 Quit"的伪代码。
const SpawnOrder = `
1) 检查 dbs.IsDesktop() == false → 501
2) 调 SpawnHelper(HelperBundle{...}.Args())    // 在 desktop 钩子里 exec.Cmd.Start() 不 Wait
3) helper.Start() 返回 err != nil → 500,**不调 AppQuit**
4) helper.Start() == nil → AppQuit() 退出 Wails 主循环
`

// ErrHelperSpawnFailed controller 返 500 时携带。
var ErrHelperSpawnFailed = "updater: helper spawn failed"
