package cmarket

import (
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
)

// newService 旧工厂(2026-06-30 保留):仅用于老 Install 端点,不带 apply 能力。
func newService() *smarket.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	return smarket.New(ww, rr, func() (*sskill.Service, error) {
		store, err := sskill.NewStore()
		if err != nil {
			return nil, err
		}
		return sskill.New(store), nil
	})
}

// newServiceV2 工厂(2026-06-30 增):注入 sskillapp + sproject,供 install-v2
// 和源管理端点使用;scope=project 时 sproject 让 sskillapp 能把 project_id
// 解析成真实项目根路径。
func newServiceV2() *smarket.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	skillSvcFactory := func() (*sskill.Service, error) {
		store, err := sskill.NewStore()
		if err != nil {
			return nil, err
		}
		return sskill.New(store), nil
	}
	skillApp := sskillapp.New(ww, rr, skillSvcFactory).
		WithProjectService(sproject.New(ww, rr))
	return smarket.NewWithApply(ww, rr, skillSvcFactory, skillApp)
}
