package bootstrap

// adapters_import.go 在 import 阶段触发 toolspecs 包的导入(blank import)。
//
// 历史背景(2026-06-23):Trae / Codex / Claude / OpenCode / Cursor 各子包在自己的
// init() 里调用 skilladapter.Register(...),但生产代码路径里没有任何位置 import
// 这些子包,Go 编译器会整个去掉未引用的包,导致 defaultRegistry 永远是空的,
// skillimporter.Scan 的 adapter 列表为空 → 扫描结果 0 个 skill。
//
// 2026-06-30 二改:工具元数据从 yaml embed 改成 DB(由 toolseed.EnsureSeeded
// 在 start_db.go 启动期 seed);toolspecs 包不再 init() 自动注册,改由
// start_db.go 显式调 toolspecs.ReloadAllFromDB(dbs.GetWriteDb())。
// 但仍保留 blank import,确保包被链接进二进制(虽然它没副作用)。
//
// Please do not move the placeholders below, otherwise it will cause the
// generation tool to fail to replace them automatically.
import (
	_ "ginp-api/internal/skilladapter/toolspecs"
	//{{placeholder_adapter_import}}//
)
