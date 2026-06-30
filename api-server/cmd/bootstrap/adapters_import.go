package bootstrap

// adapters_import.go 在 import 阶段触发 toolspecs 包的自注册。
//
// 历史背景(2026-06-23):Trae / Codex / Claude / OpenCode / Cursor 各子包在自己的
// init() 里调用 skilladapter.Register(...),但生产代码路径里没有任何位置 import
// 这些子包,Go 编译器会整个去掉未引用的包,导致 defaultRegistry 永远是空的,
// skillimporter.Scan 的 adapter 列表为空 → 扫描结果 0 个 skill。
//
// 2026-06-30 改造:全部 5 个 adapter 子包删除,统一由 toolspecs 包在 init() 里
// 加载 specs/*.yaml → NewSpecAdapter → skilladapter.Register。
// 旧版本需要在三个入口(blank import)各 blank import 5 个子包,新版本只需要
// blank import 一次 toolspecs。
//
// 新加 adapter 时:在 internal/skilladapter/toolspecs/specs/ 加一个 yaml 文件,
// 不需要再改本文件;但本文件仍需保留 blank import 触发 toolspecs.init()。
//
// Please do not move the placeholders below, otherwise it will cause the
// generation tool to fail to replace them automatically.
import (
	_ "ginp-api/internal/skilladapter/toolspecs"
	//{{placeholder_adapter_import}}//
)
