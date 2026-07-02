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

// ApplyMode 落盘模式(2026-07-02 增)。
//
//   - ModeCopy:    把 canonical 逐文件拷贝到目标目录(原行为,占磁盘空间)。
//   - ModeSymlink: 目标目录整体做成软链接,指向 skillstore 源 skill 根(零占用,
//                  改源即生效)。具体落盘由 adapter 的 ApplyLink 实现。
//
// 与 settings.ApplyMode 常量值一致(都是 "copy"/"symlink"),便于跨包统一判断。
const (
	ModeCopy    = "copy"
	ModeSymlink = "symlink"
)

// IsModeValid 校验 mode 字符串合法。
func IsModeValid(m string) bool {
	return m == ModeCopy || m == ModeSymlink
}

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
// 2026-07-02 增:TargetWasSymlink 用于 symlink 模式,记录 apply 前 targetDir
// 是否本身就是 symlink —— 切回 copy 模式或 Undo 时,需判断"原 target 是普通
// 目录(含原内容)还是软链(可能是外部安装,不该被覆盖)"。
type PreSnapshot struct {
	TargetExisted    bool           `json:"target_existed"`     // 目标目录 apply 前是否存在
	TargetWasSymlink bool           `json:"target_was_symlink"` // 目标 apply 前是否为 symlink(symlink 模式专用)
	Files            []FileSnapshot `json:"files"`              // apply 前存在的文件(copy 模式填)
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
// 2026-06-29 增:ProjectRoot = project scope 的真实项目根目录(由 caller 从
// sproject.Service 查 entity.Project.RootPath 得到);不为空时,apply 把
// <Tool.Project.Rel> 直接拼到 ProjectRoot 下;为空时退回"占位实现"
// (home/.skillbox/projects/<ProjectID>/) — 占位逻辑已废,production 必须传。
type ApplyInput struct {
	SkillName   string                  // 来自 sskill 的 Canonical.Manifest.Name
	Scope       string                  // global / project
	ProjectID   uint                    // scope=project 时必填
	ProjectRoot string                  // scope=project 时的项目 root_path(从 sproject 查)
	Tools       []string                // 目标工具 ID 列表
	Canonical   *skilladapter.Canonical // 来自 sskill.Get
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
