package skilltester

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
)

// 默认超时 / 日志截断。
const (
	defaultScriptTimeoutSec = 60
	maxLogBytes             = 32 * 1024
)

// scriptWhitelistRE 允许的脚本后缀(显式白名单,避免误跑非脚本文件)。
var scriptWhitelistRE = regexp.MustCompile(`(?i)\.(sh|bash|zsh|py|js|ts)$`)

// shellInjRE 拒绝命令中的 shell 注入字符(命令模式)。
var shellInjRE = regexp.MustCompile(`[;&|<>$\\\n\r]`)

// ScriptSummary 脚本执行详情。
type ScriptSummary struct {
	Command    string `json:"command"`
	WorkDir    string `json:"work_dir"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Skipped    bool   `json:"skipped,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// RunScript 跑 skill 自带脚本或自定义 command。
//
// 顺序:Options.ScriptCommand 非空 -> 走自定义;否则按文件名找 test.sh / test.py / test。
// 找不到任何脚本 = skipped(不视为 failed,常见情况是 skill 本身没带测试)。
func RunScript(c skilladapter.Canonical, workDir string, opts Options) CheckResult {
	timeout := defaultScriptTimeoutSec
	if opts.ScriptTimeoutSec > 0 {
		timeout = opts.ScriptTimeoutSec
	}

	var (
		cmd  string
		args []string
		dir  string
	)

	// 1) 自定义 command
	if opts.ScriptCommand != "" {
		if shellInjRE.MatchString(opts.ScriptCommand) {
			return CheckResult{
				Check:   CheckScript,
				Status:  StatusErrored,
				Message: "custom command contains shell metacharacters",
			}
		}
		parts := strings.Fields(opts.ScriptCommand)
		cmd = parts[0]
		args = parts[1:]
	} else {
		// 2) 找 skill 自带 test 脚本
		var found string
		for _, name := range []string{"test.sh", "test.py", "test.js", "test"} {
			for _, f := range c.Files {
				if f.Path == name || f.Path == "scripts/"+name {
					found = f.Path
					break
				}
			}
			if found != "" {
				break
			}
		}
		if found == "" {
			// 3) 都没找到就 skipped
			summary := ScriptSummary{Skipped: true, Reason: "no test script found (test.sh / test.py / test.js / test)"}
			b, _ := json.Marshal(summary)
			return CheckResult{Check: CheckScript, Status: StatusSkipped, Message: summary.Reason, Detail: string(b)}
		}
		// 物理文件:写到 workDir 临时目录(沙盒隔离)
		tmp, err := os.MkdirTemp("", "skilltest-*")
		if err != nil {
			return CheckResult{Check: CheckScript, Status: StatusErrored, Message: "mkdir temp: " + err.Error()}
		}
		defer os.RemoveAll(tmp)
		for _, f := range c.Files {
			if !scriptWhitelistRE.MatchString(f.Path) {
				continue
			}
			dst := filepath.Join(tmp, f.Path)
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return CheckResult{Check: CheckScript, Status: StatusErrored, Message: "mkdir file: " + err.Error()}
			}
			if err := os.WriteFile(dst, []byte(f.Content), 0o755); err != nil {
				return CheckResult{Check: CheckScript, Status: StatusErrored, Message: "write file: " + err.Error()}
			}
		}
		// 跑找到的那个
		exe := filepath.Join(tmp, found)
		cmd = exe
		dir = tmp
		// 解释器
		switch {
		case strings.HasSuffix(found, ".py"):
			args = []string{exe}
			cmd = "python3"
		case strings.HasSuffix(found, ".js"):
			args = []string{exe}
			cmd = "node"
		}
	}

	// workDir 决定
	wd := dir
	if wd == "" {
		wd = workDir
	}
	if wd == "" {
		wd, _ = os.Getwd()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	start := time.Now()
	cctx := exec.CommandContext(ctx, cmd, args...)
	cctx.Dir = wd
	var stdout, stderr bytes.Buffer
	cctx.Stdout = &stdout
	cctx.Stderr = &stderr

	runErr := cctx.Run()
	summary := ScriptSummary{
		Command:    cmd + " " + strings.Join(args, " "),
		WorkDir:    wd,
		Stdout:     truncateBytes(stdout.String(), maxLogBytes),
		Stderr:     truncateBytes(stderr.String(), maxLogBytes),
		DurationMs: time.Since(start).Milliseconds(),
	}
	if cctx.ProcessState != nil {
		summary.ExitCode = cctx.ProcessState.ExitCode()
	}

	b, _ := json.Marshal(summary)
	res := CheckResult{Check: CheckScript, Detail: string(b)}
	switch {
	case runErr == nil && summary.ExitCode == 0:
		res.Status = StatusPassed
		res.Message = fmt.Sprintf("exit 0 in %dms", summary.DurationMs)
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		res.Status = StatusErrored
		res.Message = fmt.Sprintf("timeout after %ds", timeout)
	default:
		res.Status = StatusFailed
		res.Message = fmt.Sprintf("exit %d: %s", summary.ExitCode, firstLine(stderr.String()))
	}
	return res
}

func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
