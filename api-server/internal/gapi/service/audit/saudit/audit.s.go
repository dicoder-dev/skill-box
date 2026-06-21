// Package saudit 提供 audit_log 域的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.6 节):
//   - List:分页 + 过滤(actor / action / target_type)
//   - Stats:总记录数 + 按 action 分布 + 按 actor 分布
//   - Write:本包只暴露 Write 给同进程内其他 service(apply / undo / import / rollback 等),
//     controller 不接写入端点
package saudit

import (
	"fmt"

	"ginp-api/internal/gapi/entity"
	mauditlog "ginp-api/internal/gapi/model/skillbox/mauditlog"
	"ginp-api/pkg/where"
	"gorm.io/gorm"
)

type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

func New(dbWrite, dbRead *gorm.DB) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead}
}

func (s *Service) model() *mauditlog.Model {
	return mauditlog.NewModel(s.dbWrite, s.dbRead)
}

// ListQuery 列表入参。
type ListQuery struct {
	Actor      string
	Action     string
	TargetType string
	Page       int
	Size       int
}

// ListResult 列表出参。
type ListResult struct {
	Items []*entity.AuditLog `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

// List 拉一页日志,按 ID desc。
func (s *Service) List(q ListQuery) (*ListResult, error) {
	conds := []*where.Condition{}
	if q.Actor != "" {
		conds = append(conds, where.New(mauditlog.FieldActor, "=", q.Actor).Conditions()...)
	}
	if q.Action != "" {
		conds = append(conds, where.New(mauditlog.FieldAction, "=", q.Action).Conditions()...)
	}
	if q.TargetType != "" {
		conds = append(conds, where.New(mauditlog.FieldTargetType, "=", q.TargetType).Conditions()...)
	}
	page, size := q.Page, q.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	items, total, err := s.model().FindList(conds, &where.Extra{
		PageNum: page, PageSize: size, OrderByColumn: mauditlog.FieldID, OrderByDesc: true,
	})
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: int64(total), Page: page, Size: size}, nil
}

// Stats 概览。
type Stats struct {
	Total    int64            `json:"total"`
	ByAction map[string]int64 `json:"by_action"`
	ByActor  map[string]int64 `json:"by_actor"`
}

// GetStats 拉统计。
func (s *Service) GetStats() (*Stats, error) {
	total, err := s.model().GetTotal(nil)
	if err != nil {
		return nil, err
	}
	byAction, err := s.groupCount(mauditlog.FieldAction)
	if err != nil {
		return nil, err
	}
	byActor, err := s.groupCount(mauditlog.FieldActor)
	if err != nil {
		return nil, err
	}
	return &Stats{Total: total, ByAction: byAction, ByActor: byActor}, nil
}

// groupCount 内部 helper:对 audit_log 按 field group by 计数。
// 直接用 GORM,绕开 mauditlog.Model(避免给 model 加专用方法)。
func (s *Service) groupCount(field string) (map[string]int64, error) {
	type row struct {
		K string `gorm:"column:k"`
		N int64  `gorm:"column:n"`
	}
	var rows []row
	q := s.dbRead.Model(&entity.AuditLog{}).
		Select(fmt.Sprintf("%s AS k, COUNT(*) AS n", field)).
		Group(field)
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.K] = r.N
	}
	return out, nil
}

// WriteInput 写一条日志的入参。
type WriteInput struct {
	Actor      string
	Action     string
	TargetType string
	TargetID   uint
	Payload    string
}

// Write 同进程其他 service 用;不在路由层暴露。
func (s *Service) Write(in WriteInput) (*entity.AuditLog, error) {
	row := &entity.AuditLog{
		Actor:      in.Actor,
		Action:     in.Action,
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Payload:    in.Payload,
	}
	return s.model().Create(row)
}
