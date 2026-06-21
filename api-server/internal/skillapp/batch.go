package skillapp

import (
	"fmt"
	"sync"

	"ginp-api/internal/skilladapter"
)

// BatchApplier 批量 apply + 原子回滚。
//
// 语义:遍历 (skill × tool) 笛卡尔积,任一失败 → 整体回滚已成功的。
// 成功 / 失败的明细都返回给 caller(回滚结果用同一组 tool_id + target_path)。
type BatchApplier struct {
	applier *Applier
}

// NewBatchApplier 构造。
func NewBatchApplier(a *Applier) *BatchApplier {
	return &BatchApplier{applier: a}
}

// BatchItem 单条 apply 入参(单 skill 单 tool 的笛卡尔基元)。
type BatchItem struct {
	SkillID   uint
	SkillName string
	Scope     string
	ProjectID uint
	Tool      string
	Canonical *skilladapter.Canonical
}

// BatchOutput 批量结果。
type BatchOutput struct {
	Items   []BatchItemResult `json:"items"`
	AllOK   bool              `json:"all_ok"`
	RolledBack bool           `json:"rolled_back"`
}

// BatchItemResult 单条结果。
type BatchItemResult struct {
	BatchItem
	Result *ApplyResult `json:"result"`
	Error  string       `json:"error,omitempty"`
}

// Apply 跑批量;Atomic=true 时任一失败 → 整体回滚。
// 不论成功 / 失败,items 都会按输入顺序填好,前端可逐条展示。
func (b *BatchApplier) Apply(items []BatchItem, atomic bool) *BatchOutput {
	out := &BatchOutput{AllOK: true, RolledBack: false}
	successIdx := []int{} // 已成功的 items 下标(回滚时用)
	for i, it := range items {
		res, err := b.applier.ApplyOne(ApplyInput{
			Scope:     it.Scope,
			ProjectID: it.ProjectID,
			Tools:     []string{it.Tool},
			Canonical: it.Canonical,
		})
		bir := BatchItemResult{BatchItem: it, Result: res}
		if err != nil {
			bir.Error = err.Error()
			out.AllOK = false
		} else {
			successIdx = append(successIdx, i)
		}
		out.Items = append(out.Items, bir)
		if err != nil && atomic {
			// 整体回滚已成功的
			out.RolledBack = b.rollback(out.Items, successIdx)
			return out
		}
	}
	return out
}

// rollback 用 pre-snapshot 把已成功的恢复;返回是否全部回滚成功。
// 任何单条回滚失败只记 error 到对应 result,不影响其他回滚。
func (b *BatchApplier) rollback(items []BatchItemResult, idx []int) bool {
	// 这里 "rollback" 等价于"按 pre-snapshot 恢复 target 目录";
	// v1 借用 applier 的内部 restoreFromSnapshot(暂时简单 expose)。
	var wg sync.WaitGroup
	ok := true
	for _, i := range idx {
		bir := items[i]
		if bir.Result == nil || bir.Result.PreSnapshot == nil {
			continue
		}
		wg.Add(1)
		go func(targetDir, snapJSON string) {
			defer wg.Done()
			snap, perr := UnmarshalPreSnapshot(snapJSON)
			if perr != nil {
				snap = bir.Result.PreSnapshot
			}
			_ = snap
			if rerr := restoreForUndo(targetDir, bir.Result.PreSnapshot); rerr != nil {
				ok = false
			}
		}(bir.Result.TargetPath, bir.Result.PreSnapshot.Marshal())
	}
	wg.Wait()
	return ok
}

// restoreForUndo 给 service / undo.go 用,直接 walk PreSnapshot 恢复。
func restoreForUndo(targetDir string, pre *PreSnapshot) error {
	return restoreFromSnapshot(targetDir, pre)
}

// ApplyWithItems 公共入口(防止 caller 拼错 BatchItem 字段)。
func (b *BatchApplier) ApplyWithItems(items []BatchItem, atomic bool) *BatchOutput {
	if len(items) == 0 {
		return &BatchOutput{AllOK: true}
	}
	return b.Apply(items, atomic)
}

// Suppress unused import 防止 goimports 删掉 sync。
var _ = fmt.Sprintf
