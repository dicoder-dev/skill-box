package toolseed

import (
	"errors"
	"fmt"
	"log"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/mtool"

	"gorm.io/gorm"
)

// ErrAlreadySeeded DB 里已有工具,跳过 seed(非错误;只是"无需再 seed")。
var ErrAlreadySeeded = errors.New("toolseed: already initialized, skip seed")

// EnsureSeeded 启动期调用:若 e_tool 表空,seed 9 个默认工具。
//
// 判定:用 e_tool.Count(),为 0 才 seed;为 >0 直接返回 nil(已初始化)。
// 失败:DB 错误透传,seed 写入失败透传(包事务内回滚)。
//
// 调用方:cmd/bootstrap/start_db.go 在 AutoMigrate 之后启 HTTP 之前。
func EnsureSeeded(dbWrite, dbRead *gorm.DB) error {
	m := mtool.NewModel(dbWrite, dbRead)
	count, err := m.Count()
	if err != nil {
		return fmt.Errorf("toolseed: count tools: %w", err)
	}
	if count > 0 {
		log.Printf("toolseed: skip (e_tool already has %d rows)", count)
		return nil
	}
	log.Printf("toolseed: seeding %d default tools", len(builtins))
	if err := runSeedInTx(dbWrite); err != nil {
		return fmt.Errorf("toolseed: seed: %w", err)
	}
	log.Printf("toolseed: seeded %d default tools", len(builtins))
	return nil
}

// runSeedInTx 把 9 个默认工具 + paths 写进 DB,事务内。
// 系统工具 IsSystem=true;Maturity 原样落库;Path 保留 ~/ 形式不展开。
func runSeedInTx(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		toolM := mtool.NewModel(tx, tx)
		pathM := mtool.NewToolPathModel(tx, tx)
		for _, bt := range builtins {
			tool := &entity.Tool{
				ToolID:      bt.ToolID,
				DisplayName: bt.DisplayName,
				MdiIcon:     bt.MdiIcon,
				Maturity:    bt.Maturity,
				Note:        bt.Note,
				IsSystem:    true, // seed 出的全是系统工具
				Enabled:     true,
				SortOrder:   bt.SortOrder,
			}
			created, err := toolM.Create(tool)
			if err != nil {
				return fmt.Errorf("seed %s: %w", bt.ToolID, err)
			}
			for _, p := range bt.Paths {
				if _, err := pathM.Create(&entity.ToolPath{
					ToolID:    created.ID,
					Scope:     p.Scope,
					Category:  p.Category,
					Path:      p.Path,
					PathOrder: p.PathOrder,
				}); err != nil {
					return fmt.Errorf("seed %s path %s: %w", bt.ToolID, p.Path, err)
				}
			}
		}
		return nil
	})
}
