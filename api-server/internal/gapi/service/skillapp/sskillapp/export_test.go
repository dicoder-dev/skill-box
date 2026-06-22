package sskillapp

import (
	"ginp-api/internal/gapi/entity"

	"gorm.io/gorm"
)

// WriteMarketSkillForTest 写一条 market_skill,供 *_test.go 端到端验证 CheckUpdates。
func (s *Service) WriteMarketSkillForTest(m *entity.MarketSkill) {
	s.marketSkillModel().Create(m)
}

// GetDBForTest 暴露 dbWrite 给 *_test.go 用(白盒 export),用于断言 audit_log。
func (s *Service) GetDBForTest() *gorm.DB {
	return s.dbWrite
}
