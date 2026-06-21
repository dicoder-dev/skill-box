// Package skilltester 跑 skill 的"测试":
//   1. 静态 lint(frontmatter 完整性 / 文件路径合法性 / 必填字段 / 长度阈值 / 硬编码 secret 扫描)
//   2. 脚本执行(skill 自带 test.sh / 自定义 command,带超时)
//   3. AI 走查(走 aiengine + safety_check preset,无 provider / 无 key 时降级为 skipped)
//
// 整体设计见 docs/project/需求规划.md 第 6.4 节。
package skilltester

import "time"

// Check 类型。
const (
	CheckStatic = "static"
	CheckScript = "script"
	CheckAI     = "ai"
)

// Status 状态。
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusErrored = "errored"
	StatusSkipped = "skipped"
)

// Trigger 触发方式。
const (
	TriggerManual = "manual"
	TriggerAuto   = "auto"
)

// CheckResult 单个 check 的结果(同时落到 skill_test_results 表)。
type CheckResult struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Message string `json:"message"`
	// Detail 是 JSON 字符串(具体内容因 check 而异,见各 .go 文件)
	Detail string `json:"detail,omitempty"`
}

// Report 一次完整 run 的聚合(由 Tester.Test 返回)。
type Report struct {
	// RunID 对应 entity.SkillTestRun.ID(run 入库后的主键)
	RunID uint `json:"run_id"`
	// Status 聚合状态:passed / failed / errored / skipped
	Status string `json:"status"`
	// Summary 一句话总结
	Summary string `json:"summary"`
	// Results 各 check 详细结果
	Results []CheckResult `json:"results"`
	// StartedAt / FinishedAt
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// Options 测试参数。
type Options struct {
	// ScriptCommand 自定义执行命令(为空时走 skill 自带 test.sh,再退化为 test.py / test)
	// 不允许包含 ; | & > < $( 等 shell 注入字符。
	ScriptCommand string
	// ScriptWorkDir 工作目录(空 = 用 skill 在 store 里的物理目录)
	ScriptWorkDir string
	// ScriptTimeoutSec 单次脚本执行超时秒数(默认 60)
	ScriptTimeoutSec int
	// AIProvider 走查用 provider name(空 = 按 priority 选)
	AIProvider string
	// AIPreset 走查用 preset(默认 safety_check)
	AIPreset string
	// Trigger manual / auto
	Trigger string
}
