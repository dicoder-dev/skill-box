package sskillaudit

import "gorm.io/gorm"

// GetDBForTest 暴露 dbWrite 给 *_test.go 用(白盒 export),用于断言 audit_log。
func (s *Service) GetDBForTest() *gorm.DB {
	return s.dbWrite
}
