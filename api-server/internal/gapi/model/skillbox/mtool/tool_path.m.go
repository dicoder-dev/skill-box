package mtool

import (
	"fmt"
	"sort"

	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/dbops"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// ToolPathModel 工具路径子表 model。
type ToolPathModel struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func NewToolPathModel(dbWrite_, dbRead_ *gorm.DB) *ToolPathModel {
	return &ToolPathModel{dbWrite: dbWrite_, dbRead: dbRead_}
}

func (s *ToolPathModel) Create(dtoCreate *entity.ToolPath) (*entity.ToolPath, error) {
	err := dbops.Create(dtoCreate, s.dbWrite)
	if err != nil {
		return nil, err
	}
	if dtoCreate == nil || dtoCreate.ID <= 0 {
		return nil, fmt.Errorf("create tool_path: empty result")
	}
	return dtoCreate, nil
}

func (s *ToolPathModel) FindOne(wheres []*where.Condition) (*entity.ToolPath, error) {
	out := new(entity.ToolPath)
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

func (s *ToolPathModel) FindList(wheres []*where.Condition, extra *where.Extra) ([]*entity.ToolPath, uint, error) {
	var list []*entity.ToolPath
	err := dbops.FindList(&dbops.FindListConfig{
		Conditions:    wheres,
		Db:            s.dbRead,
		Extra:         extra,
		NewEntityList: &list,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := dbops.GetTotal(wheres, new(entity.ToolPath), s.dbRead)
	if err != nil {
		return []*entity.ToolPath{}, 0, err
	}
	return list, uint(total), nil
}

func (s *ToolPathModel) Update(wheres []*where.Condition, dtoUpdate *entity.ToolPath, columnsCfg ...string) error {
	return dbops.Update(&dbops.UpdateConfNew{
		Wheres:           wheres,
		NewEntity:        new(entity.ToolPath),
		Db:               s.dbWrite,
		UpdateColumnsCfg: columnsCfg,
		DataUpdate:       dtoUpdate,
	})
}

func (s *ToolPathModel) Delete(wheres []*where.Condition) error {
	return dbops.Delete(&dbops.DeleteConfig{
		Wheres:     wheres,
		Db:         s.dbWrite,
		SoftDelete: false,
		NewEntity:  new(entity.ToolPath),
	})
}

func (s *ToolPathModel) DeleteByID(id uint) error {
	return s.Delete(where.New(FieldPathID, "=", id).Conditions())
}

// DeleteByToolID 删一个工具的全部 path(级联删时使用)。
func (s *ToolPathModel) DeleteByToolID(toolID uint) error {
	return s.Delete(where.New(FieldPathToolID, "=", toolID).Conditions())
}

// FindByToolID 查一个工具的全部 path,按 (scope, category, path_order) 排序。
func (s *ToolPathModel) FindByToolID(toolID uint) ([]*entity.ToolPath, error) {
	list, _, err := s.FindList(where.New(FieldPathToolID, "=", toolID).Conditions(), nil)
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

// FindAllByToolIDs 批量查多个工具的 path(避免 N+1),返回 map[toolID][]ToolPath。
func (s *ToolPathModel) FindAllByToolIDs(toolIDs []uint) (map[uint][]*entity.ToolPath, error) {
	if len(toolIDs) == 0 {
		return map[uint][]*entity.ToolPath{}, nil
	}
	conds := where.New(FieldPathToolID, "IN", toolIDs).Conditions()
	list, _, err := s.FindList(conds, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[uint][]*entity.ToolPath, len(toolIDs))
	for _, p := range list {
		out[p.ToolID] = append(out[p.ToolID], p)
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