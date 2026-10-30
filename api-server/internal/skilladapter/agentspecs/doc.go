// Package toolspecs 提供"工具元数据"的数据驱动加载层。
//
// 设计动机:
//
//	改造前(2026-06 之前):每个工具(Codex / Claude / OpenCode / Cursor / Trae)
//	各占一个 Go 子包,内部 80 行 boilerplate(Tools map / SystemPaths /
//	DisplayName / IconEmoji),只差 path 跟 display_name。新加一个工具要:
//	  1. 新建子包 + 写 adapter.go
//	  2. 在 init() 注册到默认 registry
//	  3. 重新 go build
//	  4. 写完发现路径错了,改 yaml 后又要再 build
//	→ 迭代极慢,工具元数据藏在 Go 代码里,运维/产品改不动。
//
//	改造后(2026-06-30 起):本包把"工具是什么"从代码中分离出来,以 YAML 数据文件
//	存到 specs/<tool>.yaml,Go 端只负责按 ToolSpec 喂给 BaseAdapter。
//	新增一个工具 = 在 specs/ 里加一个 yaml,不需要 Go 代码变动、不需要 build。
//
// 关键边界:
//
//   - 本包不实现 Adapter 接口 — 那是 skilladapter 的事;本包只产出
//     *ToolSpec,然后由 skilladapter.NewSpecAdapter 转换。
//   - 本包不决定路径是否 "~/" 缩写 — ExpandPath 在 skilladapter 内部完成;
//     YAML 里写 "~/.agents/skills" 仅是声明,运行时展开。
//   - 本包启动时通过 init() 调用 skilladapter.Register — 与原 5 个 adapter
//     子包 init() 同等地位,只此一处。
package toolspecs
