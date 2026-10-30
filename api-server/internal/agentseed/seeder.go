package toolseed

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/mtool"
	"ginp-api/internal/gapi/service/tool/toolicon"

	"gorm.io/gorm"
)

// ErrAlreadySeeded DB 里已有工具,跳过 seed(非错误;只是"无需再 seed")。
var ErrAlreadySeeded = errors.New("toolseed: already initialized, skip seed")

// EnsureSeeded 启动期调用:e_tool 表非空时,upsert 所有内置工具行;
// e_tool 表为空时,第一次 seed 全部内置工具。
//
// 2026-07-18 升级:从"只刷已有系统工具的软字段"升级为"增量 upsert"。
// 判定逻辑:
//   - DB 空 → runSeedInTx,事务内批量 insert 全量 builtins
//   - DB 非空 → upsertBuiltinTools,事务内每条 builtin:
//       *  DB 已存在(按 tool_id 查) → 更新软字段(sort_order / display_name /
//          mdi_icon / icon_file / maturity / note),保留用户改过的 is_system /
//          enabled / 独立字段
//       *  DB 不存在 → insert 新工具行(is_system=true,enabled=true,sort_order
//          来自 builtins)+ 写 paths
// 这样 builtin.go 新增条目后,老用户重启一次就自动补上新增内置。
// 用户自定义工具(is_system=false)不受影响。
//
// 图标写盘独立于 DB(失败仅 log),refresh 软字段 / 新增条目都共用。
func EnsureSeeded(dbWrite, dbRead *gorm.DB) error {
	m := mtool.NewModel(dbWrite, dbRead)
	count, err := m.Count()
	if err != nil {
		return fmt.Errorf("toolseed: count tools: %w", err)
	}
	if count == 0 {
		log.Printf("toolseed: seeding %d default tools (empty DB)", len(builtins))
		if err := runSeedInTx(dbWrite); err != nil {
			return fmt.Errorf("toolseed: seed: %w", err)
		}
		writeBuiltinIcons() // best-effort,失败也只是图标回退 mdi
		log.Printf("toolseed: seeded %d default tools", len(builtins))
		return nil
	}
	// DB 非空:upsert + 图标写盘
	if err := upsertBuiltinTools(dbWrite); err != nil {
		return fmt.Errorf("toolseed: upsert: %w", err)
	}
	writeBuiltinIcons()
	log.Printf("toolseed: upsert complete (DB had %d rows)", count)
	return nil
}

// upsertBuiltinTools 单事务内对每条 builtin 做"在 → 刷软字段 / 不在 → insert"两路处理。
//
// 保留不动:
//   - 用户对系统工具改过的 is_system / enabled(后者是用户开关语义)
//   - 用户对系统工具改过的图标(icon_file 字段值不为空 + 与 builtins 不同时不刷;
//      但 builtins 从空改为非空时刷,因为 builtin 优先;用户上传图片的兜底路径仍由
//      ctool upload 覆盖,这里只是把"系统内置默认 icon"刷成内置图标名)
//   - 用户对系统工具改过的 tool_id(unique 不可变,也不刷)
//
// 刷(差异才刷):
//   - sort_order / display_name / mdi_icon / maturity / note
//
// 新增:builtins[i] 不在 DB → insert 一行 + 它的 paths。
func upsertBuiltinTools(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		toolM := mtool.NewModel(tx, tx)
		pathM := mtool.NewToolPathModel(tx, tx)
		inserted := 0
		updated := 0
		for _, bt := range builtins {
			cur, err := toolM.FindByToolID(bt.ToolID)
			if err != nil {
				// 找不到 = 内置工具 DB 缺失 → 新增
				tool := &entity.Tool{
					ToolID:      bt.ToolID,
					DisplayName: bt.DisplayName,
					MdiIcon:     bt.MdiIcon,
					IconFile:    bt.IconFile,
					Maturity:    bt.Maturity,
					Note:        bt.Note,
					IsSystem:    true,
					Enabled:     true,
					SortOrder:   bt.SortOrder,
				}
				created, err := toolM.Create(tool)
				if err != nil {
					return fmt.Errorf("upsert insert %s: %w", bt.ToolID, err)
				}
				for _, p := range bt.Paths {
					if _, err := pathM.Create(&entity.ToolPath{
						ToolID:    created.ID,
						Scope:     p.Scope,
						Category:  p.Category,
						Path:      p.Path,
						PathOrder: p.PathOrder,
					}); err != nil {
						return fmt.Errorf("upsert insert %s path %s: %w", bt.ToolID, p.Path, err)
					}
				}
				inserted++
				log.Printf("toolseed: inserted new builtin tool %q (sort_order=%d)", bt.ToolID, bt.SortOrder)
				continue
			}
			// 已存在:差异才刷 5 个软字段
			n := 0
			if cur.SortOrder != bt.SortOrder {
				if err := tx.Model(cur).Update(mtool.FieldSortOrder, bt.SortOrder).Error; err != nil {
					return fmt.Errorf("upsert update %s sort_order: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s sort_order: %d → %d", bt.ToolID, cur.SortOrder, bt.SortOrder)
				n++
			}
			if cur.DisplayName != bt.DisplayName {
				if err := tx.Model(cur).Update(mtool.FieldDisplayName, bt.DisplayName).Error; err != nil {
					return fmt.Errorf("upsert update %s display_name: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s display_name: %q → %q", bt.ToolID, cur.DisplayName, bt.DisplayName)
				n++
			}
			if cur.MdiIcon != bt.MdiIcon {
				if err := tx.Model(cur).Update(mtool.FieldMdiIcon, bt.MdiIcon).Error; err != nil {
					return fmt.Errorf("upsert update %s mdi_icon: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s mdi_icon: %q → %q", bt.ToolID, cur.MdiIcon, bt.MdiIcon)
				n++
			}
			if cur.Maturity != bt.Maturity {
				if err := tx.Model(cur).Update(mtool.FieldMaturity, bt.Maturity).Error; err != nil {
					return fmt.Errorf("upsert update %s maturity: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s maturity: %q → %q", bt.ToolID, cur.Maturity, bt.Maturity)
				n++
			}
			if cur.Note != bt.Note {
				if err := tx.Model(cur).Update(mtool.FieldNote, bt.Note).Error; err != nil {
					return fmt.Errorf("upsert update %s note: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s note: updated", bt.ToolID)
				n++
			}
			// icon_file 单独规则:仅当 builtins.IconFile 非空 且 DB 为空时刷。
			// 避免把用户上传的自定义 icon 给覆盖回去(用户的非空 icon_file 保留)。
			if bt.IconFile != "" && cur.IconFile == "" {
				if err := tx.Model(cur).Update(mtool.FieldIconFile, bt.IconFile).Error; err != nil {
					return fmt.Errorf("upsert update %s icon_file: %w", bt.ToolID, err)
				}
				log.Printf("toolseed: refresh %s icon_file: %q → %q", bt.ToolID, cur.IconFile, bt.IconFile)
				n++
			}
			updated += n
		}
		log.Printf("toolseed: upsertBuiltinTools inserted=%d updated=%d (no-op fields skipped)", inserted, updated)
		return nil
	})
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
				IconFile:    bt.IconFile,
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

// writeBuiltinIcons 把 builtin-icons/*.{png,svg,ico} 从 embed.FS 写到
// ~/.skill-box/tool-icons/<name>。独立于 DB 事务,失败仅 log 警告 —
// 不阻塞启动,前端会用 mdi_icon 兜底。
func writeBuiltinIcons() {
	dir, err := toolicon.Dir()
	if err != nil {
		log.Printf("toolseed: writeBuiltinIcons dir: %v", err)
		return
	}
	for _, name := range builtinIconNames {
		// 安全检查:builtinIconNames 是包内硬编码列表,无需再次校验
		// 但走 ValidIconFileName 多一道防御
		if !toolicon.ValidIconFileName(name) {
			continue
		}
		data, err := builtinIconsFS.ReadFile(filepath.ToSlash(filepath.Join("builtin-icons", name)))
		if err != nil {
			log.Printf("toolseed: read embedded %s: %v", name, err)
			continue
		}
		out := filepath.Join(dir, name)
		if err := os.WriteFile(out, data, 0o644); err != nil {
			log.Printf("toolseed: write %s: %v", name, err)
			continue
		}
	}
	log.Printf("toolseed: wrote %d builtin icons to %s", len(builtinIconNames), dir)
}
