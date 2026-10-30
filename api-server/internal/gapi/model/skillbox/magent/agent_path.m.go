package magent

import (
	"fmt"
	"sort"

	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/dbops"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// AgentPathModel Agent路径子表 model。
type AgentPathModel struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewAgentPathModel(dbWrite_, dbRead_ *gorm.DB) *AgentPathModel {
	return &AgentPathModel{dbWrite: dbWrite_, dbRead: dbRead_}
}

func (s *AgentPathModel) Create(dtoCreate *entity.AgentPath) (*entity.AgentPath, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create tool_path: empty result")
	}
	return dtoCreate, nil
}

func (s *AgentPathModel) FindOne(wheres []*where.Condition) (*entity.AgentPath, error) {
	out := new(entity.AgentPath)
	err := dbops.FindOne(&dbops.FindOneConfig{
		Wheres:    wheres,
		Db:        s.dbRead,
		NewEntity: out,
	})
	if err != nil {
		return nil, err
	}
	if out.ID <= 0 {
		return nil, fmt.Errorf("findone tool_path: not found")
	}
	return out, nil
}

func (s *AgentPathModel) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.AgentPath, uint, error) {
	var list []*entity.AgentPath
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.AgentPath), s.dbRead)
	if err != nil {
		return []*entity.AgentPath{}, 0, err
	}
	return list, uint(total), nil
}

func (s *AgentPathModel) Update(wheres []*where.Condition, dtoUpdate *entity.AgentPath, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.AgentPath),
		Db:               s.dbWrite,
		UpdateColumnsCfg: columnsCfg,
		DataUpdate:       dtoUpdate,
	})
}

func (s *AgentPathModel) Delete(wheres []*where.Condition) error {
	return dbops.Delete(&dbops.DeleteConfig{
		Wheres:     wheres,
		Db:         s.dbWrite,
		SoftDelete: false,
		NewEntity:  new(entity.AgentPath),
	})
}

func (s *AgentPathModel) DeleteByID(id uint) error {
	return s.Delete(where.New(FieldPathID, "=", id).Conditions())
}

// DeleteByAgentID 删一个Agent的全部 path(级联删时使用)。
func (s *AgentPathModel) DeleteByAgentID(agentID uint) error {
	return s.Delete(where.New(FieldPathAgentID, "=", agentID).Conditions())
}

// FindByAgentID 查一个Agent的全部 path,按 (scope, category, path_order) 排序。
func (s *AgentPathModel) FindByAgentID(agentID uint) ([]*entity.AgentPath, error) {
	list, _, err := s.FindList(where.New(FieldPathAgentID, "=", agentID).Conditions(), nil)
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Scope != list[j].Scope {
			return list[i].Scope < list[j].Scope
		}
		if list[i].Category != list[j].Category {
			return list[i].Category < list[j].Category
		}
		if list[i].PathOrder != list[j].PathOrder {
			return list[i].PathOrder < list[j].PathOrder
		}
		return list[i].ID < list[j].ID
	})
	return list, nil
}

// FindAllByAgentIDs 批量查多个Agent的 path(避免 N+1),返回 map[agentID][]AgentPath。
func (s *AgentPathModel) FindAllByAgentIDs(agentIDs []uint) (map[uint][]*entity.AgentPath, error) {
	if len(agentIDs) == 0 {
		return map[uint][]*entity.AgentPath{}, nil
	}
	conds := where.New(FieldPathAgentID, "IN", agentIDs).Conditions()
	list, _, err := s.FindList(conds, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[uint][]*entity.AgentPath, len(agentIDs))
	for _, p := range list {
		out[p.AgentID] = append(out[p.AgentID], p)
	}
	// 各自内部排好序
	for k := range out {
		ps := out[k]
		sort.Slice(ps, func(i, j int) bool {
			if ps[i].Scope != ps[j].Scope {
				return ps[i].Scope < ps[j].Scope
			}
			if ps[i].Category != ps[j].Category {
				return ps[i].Category < ps[j].Category
			}
			return ps[i].PathOrder < ps[j].PathOrder
		})
	}
	return out, nil
}