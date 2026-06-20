package dbops

import (
	"ginp-api/pkg/where"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateWithDb 创建
func Create(Entity any, db *gorm.DB) error {
	result := db.Create(Entity)
	return result.Error
}

// FindOneWithDb 查找一个
func FindOne(findConf *FindOneConfig) error {
	whereStr, whereValues, err := where.ConvertToGormWhere(findConf.Wheres)
	if err != nil {
		return err
	}
	tx := findConf.Db.Where(whereStr, whereValues...)
	if findConf.getSoftDelData {
		//Unscoped()，如果关联模型被软删除了，它们将被包括在查询结果中
		tx = tx.Unscoped()
	}

	//关联查询
	if len(findConf.RelationList) > 0 {
		for i := 0; i < len(findConf.RelationList); i++ {
			item := findConf.RelationList[i]
			if item.Wheres != nil {
				whereValuesRelation, _ := where.ConvertToGormWhere2(item.Wheres)
				tx = tx.Preload(item.RelationName, whereValuesRelation...)
			} else {
				tx = tx.Preload(item.RelationName)
			}
		}
	}

	tx = tx.Find(findConf.NewEntity)

	return tx.Error
}

// 查询关联查询的数据--1.支持关联查询 2.参数使用结构体形式更灵活 3.不再使用dto,直接返回实体本体，如果json时不需要返回字段则json标签填入 - 即可
func FindList(findConf *FindListConfig) error {
	findConf.Db, _ = formatDbExtra(findConf.Db, findConf.Extra) //组装附加条件
	db, err := formatDbWhere(findConf.Db, findConf.Conditions)  //组装wheres
	if err != nil {
		return err
	}
	if findConf.GetSoftDelData {
		//Unscoped()，如果关联模型被软删除了，它们将被包括在查询结果中
		db = db.Unscoped()
	}

	//开始查询，根据需要关联查询的数量，返回不同的结果
	if len(findConf.RelationList) > 0 {
		for i := 0; i < len(findConf.RelationList); i++ {
			item := findConf.RelationList[i]
			if item.Wheres != nil {
				whereValuesRelation, _ := where.ConvertToGormWhere2(item.Wheres)
				db = db.Preload(item.RelationName, whereValuesRelation...)
			} else {
				db = db.Preload(item.RelationName)
			}
		}
	}

	//构造要查询的字段
	if len(findConf.Fields) > 0 {
		db = db.Select(findConf.Fields)
	}

	err = db.Find(findConf.NewEntityList).Error
	if err != nil {
		return err
	}
	return nil
}

// 删除
func Delete(delConf *DeleteConfig) error {
	whereStr, whereValues, err := where.ConvertToGormWhere(delConf.Wheres)
	if err != nil {
		return err
	}

	tx := delConf.Db.
		Where(whereStr, whereValues...)
	if !delConf.SoftDelete {
		//Unscoped可以在执行时忽略软删除功能，实现真实删除
		tx = tx.Unscoped()
	}

	//加载关联删除
	if len(delConf.RelationList) > 0 {
		for i := 0; i < len(delConf.RelationList); i++ {
			tx = tx.Select(delConf.RelationList[i])
		}
	}
	err = tx.Delete(delConf.NewEntity).Error
	if err != nil {
		return err
	}
	return nil
}

// CreateBatch 批量创建,WhenErrorUpdate表示遇到冲突时是否更新记录
func CreateBatch(tableName string, newDtoCreateList any, WhenErrorUpdate bool, db *gorm.DB) error {
	//开启事务
	db = db.Begin()
	// entityInfo := newEntity                              //为分配一个内存空间
	// utils.DtoToEntity(newDtoCreateList, entityInfo)      //将dto转成entity
	tx := db.Clauses(clause.OnConflict{DoNothing: true}) //遇到冲突时，不更新记录
	// tx = tx
	if WhenErrorUpdate {
		tx = db.Clauses(clause.OnConflict{UpdateAll: true}) //遇到冲突时，更新记录
	}
	if err := tx.Table(tableName).Create(newDtoCreateList).Error; err != nil {
		db.Rollback()
		return err
	}
	if err := db.Commit().Error; err != nil {
		db.Rollback()
		return err
	}
	return nil
}

// Update   指定配置更新
// 注意：2025-03-21.如果传入的updateColumnsCfg为空，则会调用Save方法，save方法要求主键存在字段
// 要特别注意零值问题，updateColumnsCfg可以指定要更新的字段类型或者字段名，如果不传入则默认更新非零值字段。
// updateColumnsCfg只传入一个值时会先跟update_config里面的几个常量进行匹配,
// 匹配到了则按照里面的规则进行更新，如UpdateColumnsNotZeroAndNumber表示更新所有非0值和整型的字段
// 否则会按照其是一个指定字段来更新，传入多个值是会认为是手动指定要更新字段。
func Update(updateCfg *UpdateConfNew) error {
	// updateEntity := newEntity //为分配一个内存空间

	updateColumns := make([]string, 0) //要更新的字段
	if len(updateCfg.UpdateColumnsCfg) > 0 {
		if len(updateCfg.UpdateColumnsCfg) == 1 && arrContains(UpdateTypList, updateCfg.UpdateColumnsCfg[0]) {
			//按照指定的更新规则来更新
			updateColumns = GetUpdateColumns(updateCfg.DataUpdate, updateCfg.UpdateColumnsCfg[0])
		} else {
			//多个值 或1个值但是不属于类型指定 则组装更新的字段列表
			// 修复后的代码
			updateColumns = append(updateColumns, updateCfg.UpdateColumnsCfg...)
		}
	}

	whereStr, whereValues, err := where.ConvertToGormWhere(updateCfg.Wheres)
	if err != nil {
		return err
	}

	dbx := updateCfg.Db.Model(updateCfg.NewEntity).
		Where(whereStr, whereValues...)
	// .Preload(clause.Associations) //预加载全部

	if len(updateColumns) == 0 {
		//不指定字段，gorm内部会默认更新所有非零值数据
		//dbx = dbx.Save(updateCfg.DataUpdate)
		dbx = dbx.Updates(updateCfg.DataUpdate)
	} else {
		//指定要更新的字段（解决零值问题）
		dbx = dbx.Select(updateColumns).Updates(updateCfg.DataUpdate)
	}

	if dbx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return dbx.Error
}

// GetTotal 获取符合某个条件的数量
func GetTotal(wheres []*where.Condition, newEntity any, db *gorm.DB) (int64, error) {
	var count int64
	whereStr, whereValues, err := where.ConvertToGormWhere(wheres)
	if err != nil {
		return -1, err
	}

	err = db.
		Model(newEntity).Where(whereStr, whereValues...).
		Count(&count).Error
	if err != nil {
		return -1, err
	}
	return count, nil
}
