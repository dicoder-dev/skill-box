package mskillapply

import (
	"fmt"
	"strings"

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
func (s *Model) Create(dtoCreate *entity.SkillApply) (*entity.SkillApply, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create skill_apply: empty result")
	}
	return dtoCreate, nil
}

// FindOne 查询一条数据。
func (s *Model) FindOne(wheres []*where.Condition) (*entity.SkillApply, error) {
	out := new(entity.SkillApply)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    wheres,
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone skill_apply: not found")
	}
	return out, nil
}

// FindOneById 按主键查。
func (s *Model) FindOneById(id uint) (*entity.SkillApply, error) {
	return s.FindOne(where.New(FieldID, "=", id).Conditions())
}

// FindLatestByKey 按 (scope, project_id, name) 找最近一条(applied_at desc)。
// uniqueIndex 约束下,同一键只可能有一行,函数名仍叫 Latest 是为将来扩 tool 列预留。
func (s *Model) FindLatestByKey(scope string, projectID uint, name string) (*entity.SkillApply, error) {
	conds := []*where.Condition{}
	if sc := strings.TrimSpace(scope); sc != "" {
		conds = append(conds, where.New(FieldScope, "=", sc).Conditions()...)
	}
	if name = strings.TrimSpace(name); name == "" {
		return nil, fmt.Errorf("findlatest skill_apply: empty name")
	}
	conds = append(conds, where.New(FieldProjectID, "=", projectID).Conditions()...)
	conds = append(conds, where.New(FieldName, "=", name).Conditions()...)
	items, _, err := s.FindList(conds, &where.Extra{
		PageNum: 1, PageSize: 1,
		OrderByColumn: FieldAppliedAt, OrderByDesc: true,
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil // 不存在不算错
	}
	return items[0], nil
}

// FindList 查询列表。
func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.SkillApply, uint, error) {
	var list []*entity.SkillApply
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.SkillApply), s.dbRead)
	if err != nil {
		return []*entity.SkillApply{}, 0, err
	}
	return list, uint(total), nil
}

// Update 按条件更新。
func (s *Model) Update(wheres []*where.Condition, dtoUpdate *entity.SkillApply, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.SkillApply),
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
		NewEntity:  new(entity.SkillApply),
	})
}

// DeleteById 按主键删除。
func (s *Model) DeleteById(id uint) error {
	return s.Delete(where.New(FieldID, "=", id).Conditions())
}

// GetTotal 按条件统计。
func (s *Model) GetTotal(wheres []*where.Condition) (int64, error) {
	return dbops.GetTotal(wheres, new(entity.SkillApply), s.dbRead)
}
