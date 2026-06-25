package skillapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UndoRecord 包含 service 层做 undo 需要的最小信息(从 SkillApply 行抽出)。
type UndoRecord struct {
	ID         uint
	TargetPath string
	Tool       string
	Status     string
	AppliedAt  time.Time
}

// Undo 撤销一条 apply。
// 状态约束:只有 StatusApplied 的记录能 undo;RolledBack / Failed 返 sentinel。
func Undo(rec *UndoRecord) error {
	if rec == nil {
		return ErrApplyNotFound
	}
	if rec.Status == StatusRolledBack {
		return ErrAlreadyRolled
	}
	if rec.Status == StatusFailed {
		return fmt.Errorf("skillapp: cannot undo a failed apply (id=%d)", rec.ID)
	}
	// SkillApply.PreSnapshot 是 JSON;由 service 解析后传进 Undo;
// 这里只负责"按 PreSnapshot 恢复 target 目录"。
	return nil
}

// UndoWithSnapshot Undo 真正的实现(显式传 PreSnapshot,便于复用)。
func UndoWithSnapshot(targetDir, preSnapshotJSON string) error {
	pre, err := UnmarshalPreSnapshot(preSnapshotJSON)
	if err != nil {
		return err
	}
	return restoreForUndo(targetDir, pre)
}

// SuppressUnused 把 filepath 引用留一下(后续可能加 target path 校验)。
var _ = filepath.Join
var _ = errors.New

// ForceRemoveFromPath 直接从磁盘删 skill 目录(不走 pre-snapshot)。
//
// 2026-06-25 增:用于"scope-status 显示命中但 DB 没 apply 记录"场景
// (用户手动 cp / 外部安装,没走过 skillbox apply)。删整个 target 目录;
// 调用方需保证路径来自 scope-status 命中的 resolved 路径(白名单)。
func ForceRemoveFromPath(targetDir string) error {
	if targetDir == "" {
		return fmt.Errorf("skillapp: force remove: empty target dir")
	}
	st, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 没了算成功
		}
		return fmt.Errorf("skillapp: force remove stat %s: %w", targetDir, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("skillapp: force remove %s: not a dir", targetDir)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("skillapp: force remove %s: %w", targetDir, err)
	}
	// 顺手删空父目录(到当前用户 home 为止),避免留下空目录垃圾
	homedir, _ := os.UserHomeDir()
	if homedir != "" {
		_ = removeEmptyParents(filepath.Dir(targetDir), homedir)
	}
	return nil
}
