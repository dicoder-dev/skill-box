package toolspecs

import (
	"ginp-api/internal/skilladapter"

	"gorm.io/gorm"
)

// ReloadAllFromDB 从 DB 拉一次全部 enabled 工具,转成 BaseAdapter,
// 重新注册到 skilladapter.DefaultRegistry()。
//
// 2026-06-30 二改:此函数替代原 init() 自动注册;启动期 / 用户在 UI 改
// 完工具元数据后,业务层(stool Service)主动调一次,Registry 内容整体
// 替换(包括删除用户已删的工具)。
//
// 调用方:
//   - cmd/bootstrap/start_db.go:AutoMigrate + EnsureSeeded 之后,启 HTTP 前
//   - internal/gapi/service/tool/stool.Service.Reload:用户改完工具
//
// 失败语义:DB 错误透传;不允许"半 reload 状态"。
func ReloadAllFromDB(db *gorm.DB) error {
	specs, err := LoadAllFromDB(db)
	if err != nil {
		return err
	}
	adapters := make([]skilladapter.Adapter, 0, len(specs))
	for _, spec := range specs {
		adapters = append(adapters, NewSpecAdapter(spec))
	}
	skilladapter.DefaultRegistry().Reload(adapters)
	return nil
}
