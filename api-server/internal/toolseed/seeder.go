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

// EnsureSeeded 启动期调用:若 e_tool 表空,seed 9 个默认工具 + 内置图标写盘。
//
// 判定:用 e_tool.Count(),为 0 才 seed;为 >0 直接返回 nil(已初始化)。
// 失败:DB 错误透传,seed 写入失败透传(包事务内回滚)。
// 图标写盘:在 DB 事务外执行(写文件失败不影响 DB;反之 DB 失败就别写文件)。
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
		// 即使跳过 seed,也要同步内置图标到磁盘 + 刷新系统工具的 icon_file 字段
		// (2026-07-03 加:之前占位图标名如 cursor.png 已被替换成 cursor.ico,
		// 老 DB 里 icon_file 字段还指向不存在的旧名,前端会拿到 404)
		writeBuiltinIcons()
		if err := refreshSystemIconFiles(dbWrite); err != nil {
			log.Printf("toolseed: refreshSystemIconFiles: %v", err)
		}
		return nil
	}
	log.Printf("toolseed: seeding %d default tools", len(builtins))
	if err := runSeedInTx(dbWrite); err != nil {
		return fmt.Errorf("toolseed: seed: %w", err)
	}
	writeBuiltinIcons() // best-effort,失败也只是图标回退 mdi
	log.Printf("toolseed: seeded %d default tools", len(builtins))
	return nil
}

// refreshSystemIconFiles 把系统工具的 icon_file 字段刷成 builtins 里最新的值。
// 只在 "已初始化但内置图标文件名变了" 时生效;老字段值跟新字段值一样时不做无谓 update。
// 系统工具 is_system=true 不让走 stool.Update(防止误改),所以这里直接走 model 层写 DB。
func refreshSystemIconFiles(db *gorm.DB) error {
	toolM := mtool.NewModel(db, db)
	updated := 0
	for _, bt := range builtins {
		cur, err := toolM.FindByToolID(bt.ToolID)
		if err != nil {
			// 系统工具应该都在(已经过 EnsureSeeded),找不到就跳过
			continue
		}
		if cur.IconFile == bt.IconFile {
			continue
		}
		if tx := db.Model(cur).Update(mtool.FieldIconFile, bt.IconFile); tx.Error != nil {
			log.Printf("toolseed: refresh %s icon_file: %v", bt.ToolID, tx.Error)
			continue
		}
		updated++
		log.Printf("toolseed: refresh %s icon_file: %q → %q", bt.ToolID, cur.IconFile, bt.IconFile)
	}
	if updated > 0 {
		log.Printf("toolseed: refreshed %d system tool icon_file fields", updated)
	}
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
