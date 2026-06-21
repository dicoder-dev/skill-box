package mskilltestrun

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
	return &Model{dbWrite: dbWrite_, dbRead: dbRead_}
}

func (s *Model) Create(dto *entity.SkillTestRun) (*entity.SkillTestRun, error) {
	if err := dbops.Create(dto, s.dbWrite); err != nil {
		return nil, err
	}
	if dto == nil || dto.ID <= 0 {
		return nil, fmt.Errorf("create skill test run: empty result")
	}
	return dto, nil
}

func (s *Model) FindOneById(id uint) (*entity.SkillTestRun, error) {
	out := new(entity.SkillTestRun)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    where.New(FieldID, "=", id).Conditions(),
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone skill test run: not found")
	}
	return out, nil
}

func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.SkillTestRun, uint, error) {
	var list []*entity.SkillTestRun
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.SkillTestRun), s.dbRead)
	if err != nil {
		return []*entity.SkillTestRun{}, 0, err
	}
	return list, uint(total), nil
}

func (s *Model) Update(wheres []*where.Condition, dto *entity.SkillTestRun, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.SkillTestRun),
		Db:               s.dbWrite,
		UpdateColumnsCfg: columnsCfg,
		DataUpdate:       dto,
	})
}
