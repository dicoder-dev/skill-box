package skilltester

import (
	"time"

	"ginp-api/internal/skilladapter"
)

// Tester 主入口,聚合三个 check 产出 Report。
type Tester struct {
	// WorkDir 脚本执行的工作目录(空 = caller 不指定,由 RunScript 兜底到 cwd)。
	WorkDir string
}

// New 构造一个 Tester。
func New() *Tester { return &Tester{} }

// Run 顺序跑:static -> script -> ai。返回 3 个 CheckResult + 聚合 Report。
//
// 参数:
//   - c    skill 全文(从 store 读出来的 Canonical)
//   - opts 测试参数
//   - walker AI 走查(可空;空时 ai check 走 skipped)
func (t *Tester) Run(c skilladapter.Canonical, opts Options, walker *AIWalker) Report {
	started := time.Now()
	results := make([]CheckResult, 0, 3)
	results = append(results, Lint(c))
	results = append(results, RunScript(c, t.WorkDir, opts))
	results = append(results, RunAIWalk(c, walker, opts))
	return aggregate(results, started)
}

// aggregate 聚合 3 个 check 成 Report。
func aggregate(results []CheckResult, started time.Time) Report {
	finished := time.Now()
	status := StatusPassed
	summary := make([]string, 0, len(results))
	failedCount := 0
	erroredCount := 0
	skippedCount := 0
	for _, r := range results {
		switch r.Status {
		case StatusFailed:
			failedCount++
			status = StatusFailed
		case StatusErrored:
			erroredCount++
			if status == StatusPassed {
				status = StatusErrored
			}
		case StatusSkipped:
			skippedCount++
		}
		summary = append(summary, r.Check+":"+r.Status)
	}
	// 全部 skipped -> 整体 skipped(避免 static passed 看起来"通过"但实际啥也没跑)
	if failedCount == 0 && erroredCount == 0 && skippedCount == len(results) {
		status = StatusSkipped
	}
	return Report{
		Status:     status,
		Summary:    joinSummary(summary),
		Results:    results,
		StartedAt:  started,
		FinishedAt: finished,
	}
}

func joinSummary(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}
