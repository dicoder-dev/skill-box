package toolspecs

import (
	"log"

	"ginp-api/internal/skilladapter"
)

// init 在启动期把全部 spec 转成 BaseAdapter,注册到 default registry。
//
// 2026-06-30 改造:本包替代 5 个旧 adapter 子包(claude/codex/cursor/opencode/trae)
// 的 init() 注册,成为 adapter 唯一入口。新加工具 = 在 specs/ 加 yaml。
//
// 失败语义:LoadAll 内部 panic — spec 文件合法性是构建期硬约束,服务
// 起来后再报就太晚了。NormalInit() 失败只 log,不影响其它启动路径(测试)。
func init() {
	if err := RegisterAll(); err != nil {
		log.Fatalf("toolspecs: init failed: %v", err)
	}
}

// RegisterAll 显式入口(供测试 / 二次注册用),正常流程由 init() 调用。
func RegisterAll() error {
	specs, err := LoadAll()
	if err != nil {
		return err
	}
	for _, spec := range specs {
		skilladapter.Register(NewSpecAdapter(spec))
	}
	return nil
}