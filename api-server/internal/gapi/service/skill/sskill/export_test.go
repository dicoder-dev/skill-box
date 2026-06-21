package sskill

import "gorm.io/gorm"

// GetDBForTest 暴露 dbWrite 给 *_test.go 用(白盒 export)。
func (s *Service) GetDBForTest() *gorm.DB {
	return s.dbWrite
}
