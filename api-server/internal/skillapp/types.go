// Package skillapp 提供"把 canonical skill 落到目标工具"的核心能力。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.3 + 5.1 节):
//   - Apply 走 adapter.Apply(c, targetDir);失败时根据 pre-snapshot 整体回滚
//   - pre-snapshot 存到 entity.SkillApply.PreSnapshot(包含 "目标目录是否存在 /
//     apply 前的文件清单 / apply 后的文件清单"),Undo 时据此恢复
//   - 批量 Apply 是多个单 Apply 的有序串联,任一失败整体回滚已成功的
//   - Undo 只能对 status=applied 的记录操作;status=rolled_back / failed 不能重做
//   - v1 只支持文本文件(整段读写),二进制文件 P1 再补
package skillapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
)

// 业务错误。
var (
	ErrEmptySkill      = errors.New("skillapp: skill is empty")
	ErrEmptyTools      = errors.New("skillapp: no tools specified")
	ErrToolNotFound    = errors.New("skillapp: tool adapter not found")
	ErrEmptyFiles      = errors.New("skillapp: skill has no files")
	ErrAlreadyRolled   = errors.New("skillapp: already rolled back")
	ErrApplyNotFound   = errors.New("skillapp: apply record not found")
	ErrTargetNotInSnap = errors.New("skillapp: target path mismatch on undo")
)

// Status 状态枚举。
const (
	StatusApplied    = "applied"
	StatusRolledBack = "rolled_back"
	StatusFailed     = "failed"
)

// ApplyStatus 状态值集合(校验用)。
var allStatuses = map[string]bool{
	StatusApplied:    true,
	StatusRolledBack: true,
	StatusFailed:     true,
}

// FileSnapshot 记录"目标目录里一个文件的状态"。
// v1 假设都是文本,直接存 content;二进制后续加 Encoding 字段。
type FileSnapshot struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"` // apply 前是否已存在
	Content string `json:"content,omitempty"`
}

// PreSnapshot apply 前的快照。
type PreSnapshot struct {
	TargetExisted bool           `json:"target_existed"` // 目标目录 apply 前是否存在
	Files         []FileSnapshot `json:"files"`          // apply 前存在的文件
	// PostFiles apply 后的文件清单(为 Undo 时知道"apply 加了哪些"用)
	PostFiles []string `json:"post_files"`
}

// MarshalJSON 序列化成 DB 存的字符串。
func (s *PreSnapshot) Marshal() string {
	if s == nil {
		return ""
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// UnmarshalPreSnapshot 解析。
func UnmarshalPreSnapshot(s string) (*PreSnapshot, error) {
	if strings.TrimSpace(s) == "" {
		return &PreSnapshot{}, nil
	}
	var ps PreSnapshot
	if err := json.Unmarshal([]byte(s), &ps); err != nil {
		return nil, fmt.Errorf("skillapp: parse pre_snapshot: %w", err)
	}
	return &ps, nil
}

// IsStatusValid 校验状态字符串合法。
func IsStatusValid(s string) bool {
	return allStatuses[s]
}

// ApplyInput 单次 apply 的入参。
// 2026-06-24 改造:不再用 SkillID 数字 ID,改用 SkillName 字符串作为唯一键。
type ApplyInput struct {
	SkillName string                    // 来自 sskill 的 Canonical.Manifest.Name
	Scope     string                    // global / project
	ProjectID uint                      // scope=project 时必填
	Tools     []string                  // 目标工具 ID 列表
	Canonical *skilladapter.Canonical   // 来自 sskill.Get
}

// ApplyOutput 单次 apply 的产出(每个 tool 一行 entity.SkillApply)。
type ApplyOutput struct {
	Applies []*AppliedItem `json:"applies"`
}

// AppliedItem 单个 tool 的 apply 结果。
type AppliedItem struct {
	Tool       string    `json:"tool"`
	TargetPath string    `json:"target_path"`
	Status     string    `json:"status"`          // applied / failed
	ApplyID    uint      `json:"apply_id,omitempty"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// BatchApplyInput 批量 apply(多 skill × 多 tool 笛卡尔积)。
type BatchApplyInput struct {
	Items     []ApplyInput
	// Atomic 任一失败时是否回滚已成功的;v1 强制 true(批量本意就是原子)
	Atomic bool
}
