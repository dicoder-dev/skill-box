package mmarketskill

import (
	"fmt"

	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/dbops"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// Model 走 dbops 通用 DAO。
type Model struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewModel(dbWrite_, dbRead_ *gorm.DB) *Model {
	return &Model{dbWrite: dbWrite_, dbRead: dbRead_}
}

// Upsert 按 (source_id, remote_id) 写入;存在则刷新(覆盖式)。
func (s *Model) Upsert(row *entity.MarketSkill) error {
	if row == nil {
		return fmt.Errorf("marketskill: nil row")
	}
	conds := []*where.Condition{}
	conds = append(conds, where.New(FieldSourceID, "=", row.SourceID).Conditions()...)
	conds = append(conds, where.New(FieldRemoteID, "=", row.RemoteID).Conditions()...)
	exist, err := s.FindOne(conds)
	if err == nil && exist != nil && exist.ID > 0 {
		row.ID = exist.ID
		return s.Update(where.New(FieldID, "=", exist.ID).Conditions(), row)
	}
	_, cerr := s.Create(row)
	return cerr
}

// Create 创建。
func (s *Model) Create(dtoCreate *entity.MarketSkill) (*entity.MarketSkill, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create market_skill: empty result")
	}
	return dtoCreate, nil
}

// FindOne 单条。
func (s *Model) FindOne(wheres []*where.Condition) (*entity.MarketSkill, error) {
	out := new(entity.MarketSkill)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    wheres,
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone market_skill: not found")
	}
	return out, nil
}

// FindOneById 按主键查。
func (s *Model) FindOneById(id uint) (*entity.MarketSkill, error) {
	return s.FindOne(where.New(FieldID, "=", id).Conditions())
}

// FindList 列表 + 分页。
func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.MarketSkill, uint, error) {
	var list []*entity.MarketSkill
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.MarketSkill), s.dbRead)
	if err != nil {
		return []*entity.MarketSkill{}, 0, err
	}
	return list, uint(total), nil
}

// DeleteBySource 清空某 source 的缓存(刷新前调用)。
func (s *Model) DeleteBySource(sourceID uint) error {
	return s.Delete(where.New(FieldSourceID, "=", sourceID).Conditions())
}

// DeleteBySourceAndRemote 删一行(刷新时旧条目清理用)。
func (s *Model) DeleteBySourceAndRemote(sourceID uint, remoteID string) error {
	conds := []*where.Condition{}
	conds = append(conds, where.New(FieldSourceID, "=", sourceID).Conditions()...)
	conds = append(conds, where.New(FieldRemoteID, "=", remoteID).Conditions()...)
	return s.Delete(conds)
}

// Update 按条件更新。
func (s *Model) Update(wheres []*where.Condition, dtoUpdate *entity.MarketSkill, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.MarketSkill),
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
		NewEntity:  new(entity.MarketSkill),
	})
}

// GetTotal 统计。
func (s *Model) GetTotal(wheres []*where.Condition) (int64, error) {
	return dbops.GetTotal(wheres, new(entity.MarketSkill), s.dbRead)
}
