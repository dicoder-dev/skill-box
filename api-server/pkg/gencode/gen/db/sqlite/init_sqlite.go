// Package dbinit
// @Author: zhangdi
// @File: fixed
// @Version: 1.0.0
// @Date: 2023/11/22 15:15
package sqlite

import (
	"ginp-api/pkg/filehelper"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 固定的数据 不跟随年份的改变而改变，也就是不管哪一年都在同一库同一个文件中保存
func InitdDb(dbPath string) {
	db, err := newDbInstance(dbPath)
	if err != nil {
		log.Println(err.Error())
	}
	fixedDb = db
}

// NewDbInstance 初始化连接Sqlite
func newDbInstance(dbPath string) (*gorm.DB, error) {
	//判断文件夹是否存在，不存在先创建
	baseDir := filepath.Dir(dbPath)
	if !filehelper.FileExists(baseDir) {
		err := os.MkdirAll(baseDir, 0644)
		if err != nil {
			return nil, err
		}
	}
	if !filehelper.FileExists(dbPath) {
		file, err := os.Create(dbPath)
		if err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
	}
	//连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=encoding=UTF-8"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 调整最大打开连接数以匹配您的池大小
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	//SetMaxIdleConns:用于设置连接池中空闲连接的最大数量。在使用GORM进行数据库操作时，每次GORM调用完毕后，
	//数据库连接将变为空闲状态，等待下一次使用。因此，当空闲连接达到最大值时，
	//多余的连接就会被关闭。这样可以避免浪费资源和过多的数据库连接
	sqlDB.SetMaxIdleConns(10)

	//SetMaxOpenConns:用于设置同时打开的最大连接数（包括空闲和正在使用的连接）。
	//如果当前连接数已满，则进入等待状态，并在空闲连接没有足够可用时创建新连接。
	//通过适当地调整这些参数，可以更好地平衡数据库连接的使用和性能。
	//但应该注意的是，在这里设置的最大连接数不应该超过数据库的实际最大连接数限制。
	sqlDB.SetMaxOpenConns(20)

	//迁移表结构

	return db, nil
}
