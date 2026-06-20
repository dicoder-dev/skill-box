package sqlite

import (
	"fmt"

	"gorm.io/gorm"
)

var (
	//Mutex   sync.Mutex       // 定义一个互斥锁对象
	//	dbs     map[int]*gorm.DB //多数据库 按需创建
	fixedDb *gorm.DB //固定数据
)

func GetReadDb() (db *gorm.DB, err error) {
	if fixedDb == nil {
		return nil, fmt.Errorf("数据库尚未初始化，请先调用 sqlite.InitdDb()")
	}
	return fixedDb, nil
}

func GetWriteDb() (db *gorm.DB, err error) {
	if fixedDb == nil {
		return nil, fmt.Errorf("数据库尚未初始化，请先调用 sqlite.InitdDb()")
	}
	return fixedDb, nil
}
