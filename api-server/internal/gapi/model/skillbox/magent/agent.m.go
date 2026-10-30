package magent

import (
	"fmt"
	"sort"

	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/dbops"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// Model Agent主表 model,带 FindByAgentID / ListWithPaths 等业务方法。
type Model struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewModel(dbWrite_, dbRead_ *gorm.DB) *Model {
	return &Model{dbWrite: dbWrite_, dbRead: dbRead_}
}

// standard CRUD — 与 mpackage 其它模块保持一致。

func (s *Model) Create(dtoCreate *entity.Agent) (*entity.Agent, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create agent: empty result")
	}
	return dtoCreate, nil
}

func (s *Model) FindOne(wheres []*where.Condition) (*entity.Agent, error) {
	out := new(entity.Agent)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    wheres,
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone agent: not found")
	}
	return out, nil
}

func (s *Model) FindOneByID(id uint) (*entity.Agent, error) {
	return s.FindOne(where.New(FieldID, "=", id).Conditions())
}

func (s *Model) FindByAgentID(agentID string) (*entity.Agent, error) {
	return s.FindOne(where.New(FieldAgentID, "=", agentID).Conditions())
}

func (s *Model) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.Agent, uint, error) {
	var list []*entity.Agent
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.Agent), s.dbRead)
	if err != nil {
		return []*entity.Agent{}, 0, err
	}
	return list, uint(total), nil
}

func (s *Model) Update(wheres []*where.Condition, dtoUpdate *entity.Agent, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.Agent),
		Db:               s.dbWrite,
		UpdateColumnsCfg: columnsCfg,
		DataUpdate:       dtoUpdate,
	})
}

func (s *Model) Delete(wheres []*where.Condition) error {
	return dbops.Delete(&dbops.DeleteConfig{
		Wheres:     wheres,
		Db:         s.dbWrite,
		SoftDelete: false,
		NewEntity:  new(entity.Agent),
	})
}

func (s *Model) DeleteByID(id uint) error {
	return s.Delete(where.New(FieldID, "=", id).Conditions())
}

// Count 表行数(0 表示未初始化过,启动期 seed 用此判断)。
func (s *Model) Count() (int64, error) {
	return dbops.GetTotal(nil, new(entity.Agent), s.dbRead)
}

// ListAllEnabled 列出所有 enabled=true 的Agent(给 Registry Reload 用)。
// 排除系统关闭(enabled=false)行,但保留 is_system=true 的也走 enabled 开关。
func (s *Model) ListAllEnabled() ([]*entity.Agent, error) {
	list, _, err := s.FindList(where.New(FieldEnabled, "=", true).Conditions(), nil)
	if err != nil {
		return nil, err
	}
	// sort_order 升序,保证注册顺序稳定
	sort.Slice(list, func(i, j int) bool {
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return list[i].ID < list[j].ID
	})
	return list, nil
}