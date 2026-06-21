// Package sproject 提供 Project 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 6.1 节):
//   - Project 是 skill 的容器之一(全局域不归 Project 管)
//   - 字段约束:alias 唯一、root_path 唯一、name 非空
//   - 物理根存在性:不强校验(允许"声明项目但暂未 git clone"的占位语义),
//     由 Apply / Scan 阶段按需报错
package sproject

import (
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/gapi/entity"
	mproject "ginp-api/internal/gapi/model/skillbox/mproject"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误(sentinel),controller 可用 errors.Is 判断。
var (
	ErrEmptyName   = errors.New("project: name is empty")
	ErrEmptyAlias  = errors.New("project: alias is empty")
	ErrEmptyRoot   = errors.New("project: root_path is empty")
	ErrAliasExists = errors.New("project: alias already exists")
	ErrRootExists  = errors.New("project: root_path already exists")
	ErrNotFound    = errors.New("project: not found")
)

// Service 业务服务。dbWrite / dbRead 来自 dbs.GetWriteDb / GetReadDb。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func New(dbWrite, dbRead *gorm.DB) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead}
}

func (s *Service) model() *mproject.Model {
	return mproject.NewModel(s.dbWrite, s.dbRead)
}

// Create 创建项目;alias / root_path 重复时返回 sentinel error。
func (s *Service) Create(in *entity.Project) (*entity.Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Alias = strings.TrimSpace(in.Alias)
	in.RootPath = strings.TrimSpace(in.RootPath)
	if in.Name == "" {
		return nil, ErrEmptyName
	}
	if in.Alias == "" {
		return nil, ErrEmptyAlias
	}
	if in.RootPath == "" {
		return nil, ErrEmptyRoot
	}
	if _, err := s.model().FindOne(where.New(mproject.FieldAlias, "=", in.Alias).Conditions()); err == nil {
		return nil, ErrAliasExists
	}
	if _, err := s.model().FindOne(where.New(mproject.FieldRootPath, "=", in.RootPath).Conditions()); err == nil {
		return nil, ErrRootExists
	}
	out, err := s.model().Create(in)
	if err != nil {
		return nil, fmt.Errorf("project: create: %w", err)
	}
	return out, nil
}

// Update 按 id 更新;alias / root_path 重复时返回 sentinel error。
func (s *Service) Update(id uint, in *entity.Project) (*entity.Project, error) {
	if id == 0 {
		return nil, ErrNotFound
	}
	cur, err := s.model().FindOneById(id)
	if err != nil {
		return nil, ErrNotFound
	}
	if in.Name != "" {
		cur.Name = strings.TrimSpace(in.Name)
	}
	if in.Alias != "" {
		newAlias := strings.TrimSpace(in.Alias)
		if newAlias != cur.Alias {
			if _, err := s.model().FindOne(where.New(mproject.FieldAlias, "=", newAlias).Conditions()); err == nil {
				return nil, ErrAliasExists
			}
			cur.Alias = newAlias
		}
	}
	if in.RootPath != "" {
		newRoot := strings.TrimSpace(in.RootPath)
		if newRoot != cur.RootPath {
			if _, err := s.model().FindOne(where.New(mproject.FieldRootPath, "=", newRoot).Conditions()); err == nil {
				return nil, ErrRootExists
			}
			cur.RootPath = newRoot
		}
	}
	if in.Description != "" {
		cur.Description = in.Description
	}
	if err := s.model().Update(where.New(mproject.FieldID, "=", id).Conditions(), cur); err != nil {
		return nil, fmt.Errorf("project: update: %w", err)
	}
	return cur, nil
}

// GetByID 按主键查;不存在返回 ErrNotFound。
func (s *Service) GetByID(id uint) (*entity.Project, error) {
	out, err := s.model().FindOneById(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return out, nil
}

// ListQuery 列表查询参数。
type ListQuery struct {
	Keyword string // 模糊匹配 name
	Page    int    // 1-based;0 表示不分页
	Size    int
}

// ListResult 列表结果。
type ListResult struct {
	Items []*entity.Project `json:"items"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

func (s *Service) List(q ListQuery) (*ListResult, error) {
	var conds []*where.Condition
	if k := strings.TrimSpace(q.Keyword); k != "" {
		conds = append(conds, where.New(mproject.FieldName, "LIKE", "%"+k+"%").Conditions()...)
	}
	page := q.Page
	size := q.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	items, total, err := s.model().FindList(conds, &where.Extra{
		PageNum:       page,
		PageSize:      size,
		OrderByColumn: mproject.FieldUpdatedAt,
		OrderByDesc:   true,
	})
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(total), Page: page, Size: size}, nil
}

// Delete 按 id 删;不存在返回 ErrNotFound。
func (s *Service) Delete(id uint) error {
	if id == 0 {
		return ErrNotFound
	}
	if _, err := s.model().FindOneById(id); err != nil {
		return ErrNotFound
	}
	if err := s.model().DeleteById(id); err != nil {
		return fmt.Errorf("project: delete: %w", err)
	}
	return nil
}
