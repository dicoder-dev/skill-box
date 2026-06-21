package mskilltestresult

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

func (s *Model) Create(dto *entity.SkillTestResult) (*entity.SkillTestResult, error) {
	if err := dbops.Create(dto, s.dbWrite); err != nil {
		return nil, err
	}
	if dto == nil || dto.ID <= 0 {
		return nil, fmt.Errorf("create skill test result: empty result")
	}
	return dto, nil
}

func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.SkillTestResult, uint, error) {
	var list []*entity.SkillTestResult
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.SkillTestResult), s.dbRead)
	if err != nil {
		return []*entity.SkillTestResult{}, 0, err
	}
	return list, uint(total), nil
}

func (s *Model) Delete(wheres []*where.Condition) error {
	return dbops.Delete(&dbops.DeleteConfig{
		Wheres:     wheres,
		Db:         s.dbWrite,
		SoftDelete: false,
		NewEntity:  new(entity.SkillTestResult),
	})
}
