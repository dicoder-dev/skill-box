package sskillapp

import (
	"ginp-api/internal/gapi/entity"
)

// WriteMarketSkillForTest 写一条 market_skill,供 *_test.go 端到端验证 CheckUpdates。
func (s *Service) WriteMarketSkillForTest(m *entity.MarketSkill) {
	s.marketSkillModel().Create(m)
}
