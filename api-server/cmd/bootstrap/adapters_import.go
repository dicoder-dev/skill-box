package bootstrap

// adapters_import.go 在 import 阶段触发各子 adapter 包的自注册。
//
// 历史背景(2026-06-23):Trae / Codex / Claude / OpenCode / Cursor 各子包在自己的
// init() 里调用 skilladapter.Register(...),但生产代码路径里没有任何位置 import
// 这些子包,Go 编译器会整个去掉未引用的包,导致 defaultRegistry 永远是空的,
// skillimporter.Scan 的 adapter 列表为空 → 扫描结果 0 个 skill。
//
// 单元测试 adapters_integration_test.go 里有 blank import,所以测试能跑通,
// 但生产二进制从未触发注册。修复:在本文件 blank import 所有子包,bootstrap
// 是 gapi / web / skill-box 三个入口都会过的地方,在这里 import 等价于"全局
// 生效一次"。新加 adapter 时,只要在这里加一行 blank import 即可。
//
// Please do not move the placeholders below, otherwise it will cause the
// generation tool to fail to replace them automatically.
import (
	_ "ginp-api/internal/skilladapter/claude"
	_ "ginp-api/internal/skilladapter/codex"
	_ "ginp-api/internal/skilladapter/cursor"
	_ "ginp-api/internal/skilladapter/opencode"
	_ "ginp-api/internal/skilladapter/trae"
	//{{placeholder_adapter_import}}//
)
