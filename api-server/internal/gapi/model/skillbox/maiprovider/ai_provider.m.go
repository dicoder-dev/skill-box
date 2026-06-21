package maiprovider

import (
	"fmt"
	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/dbops"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

type Model struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewModel(dbWrite_, dbRead_ *gorm.DB) *Model {
	return &Model{
		dbWrite: dbWrite_,
		dbRead:  dbRead_,
	}
}

// Create 创建数据。
func (s *Model) Create(dtoCreate *entity.AIProvider) (*entity.AIProvider, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create ai_provider: empty result")
	}
	return dtoCreate, nil
}

// FindOne 查询一条数据。
func (s *Model) FindOne(wheres []*where.Condition) (*entity.AIProvider, error) {
	out := new(entity.AIProvider)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    wheres,
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone ai_provider: not found")
	}
	return out, nil
}

// FindOneById 按主键查。
func (s *Model) FindOneById(id uint) (*entity.AIProvider, error) {
	return s.FindOne(where.New(FieldID, "=", id).Conditions())
}

// FindList 查询列表。
func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.AIProvider, uint, error) {
	var list []*entity.AIProvider
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.AIProvider), s.dbRead)
	if err != nil {
		return []*entity.AIProvider{}, 0, err
	}
	return list, uint(total), nil
}

// Update 按条件更新。
func (s *Model) Update(wheres []*where.Condition, dtoUpdate *entity.AIProvider, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.AIProvider),
		Db:               s.dbWrite,
		UpdateColumnsCfg: columnsCfg,
		DataUpdate:       dtoUpdate,
	})
}

// Delete 按条件删除。
func (s *Model) Delete(wheres []*where.Condition) error {
	return dbops.Delete(&dbops.DeleteConfig{
		Wheres:     wheres,
		Db:         s.dbWrite,
		SoftDelete: false,
		NewEntity:  new(entity.AIProvider),
	})
}

// DeleteById 按主键删除。
func (s *Model) DeleteById(id uint) error {
	return s.Delete(where.New(FieldID, "=", id).Conditions())
}

// GetTotal 按条件统计。
func (s *Model) GetTotal(wheres []*where.Condition) (int64, error) {
	return dbops.GetTotal(wheres, new(entity.AIProvider), s.dbRead)
}
