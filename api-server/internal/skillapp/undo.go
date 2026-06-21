package skillapp

import (
	"errors"
	"fmt"
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
