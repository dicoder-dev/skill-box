// Package commitmsg 为 skillversion.AutoCommitAndPush 提供"自动 commit message"生成器。
//
// 2026-07-18 增:之前 store.Save 末尾 hardcode `skill(store): update <rel>` 作 message,
// 用户不满意 — 该格式对所有 skill 一致,无法反映实际改动。现在统一通过本包生成:
//
//   - LLMEnabled=true: 调 LLM 让模型看 diff 生成 conventional commit message;
//                      模型错误/超时 → 降级到模板路径(commit 仍落盘 + 推记录
//                      错误,不阻断 store.Save)
//   - LLMEnabled=false: 用固定模板(按时间戳 / 按文件名 / op+文件名 三选)
//
// 设计要点:
//   - LLMGenerate 函数由 caller 注入(避免本包依赖 blades / aiengine)。
//   - LLM 开关不能乱开:caller 在注入前应当已测试 provider 可用 + api_key 非空。
//   - 模板实现为零外部依赖(fmt + time + path)。
package commitmsg

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Source 标识返回 message 的来源。
type Source string

const (
	SourceLLM       Source = "llm"        // LLM 成功产出
	SourceLLMFailed Source = "llm-failed" // 用户开了 LLM 但调用失败/降级
	SourceTemplate  Source = "template"   // 模板路径
)

// Template 模板风格(LLMEnabled=false 或降级时使用)。
type Template string

const (
	TemplateTimestamp Template = "timestamp" // "skill(store): 2026-07-18T10:30:00Z update"
	TemplateFilename  Template = "filename"  // "skill(store): update frontend/xxx/SKILL.md"
	TemplateOpFiles   Template = "op_files"  // "skill(store): update SKILL.md + 2 files"
)

// LLMGenerate 是 caller 注入的 LLM 调用闭包 — 输入 prompt,返回 single-line commit message。
//
// 调用规范:prompt 应当让模型输出"只一行 conventional commit message,无解释无引号"。
// caller 在闭包内负责:
//   - 选 provider(按 priority 升序)
//   - 拼 messages + 限时
//   - 流式消费 resp,拼字符串
//   - 返错时返非空 error
type LLMGenerate func(ctx context.Context, prompt string) (string, error)

// Options 调用 snapshot。
type Options struct {
	LLMEnabled   bool
	Template     Template
	Op           string
	Paths        []string // 相对 repo root 的改动文件路径
	LLM          LLMGenerate
	LLMTimeoutMs int // LLM 调用超时;0 = 默认 25s
	Now          time.Time
}

// ----------------------------------------------------------------------------
// 2026-07-18 增:全局 LLM 生成器注册器 — 解耦 skillstore 与 cskillversion 包。
//
// 调用方(例如 cskillversion init() )调用 SetGlobalLLMGenerator 注入一次。
// store.store 包不能反向 import gapi/controller 包,通过全局注册器拿。
// ----------------------------------------------------------------------------

var globalMu sync.RWMutex
var globalLLM LLMGenerate

// SetGlobalLLMGenerator 注册全局 LLM 生成器;传 nil 清空(走模板路径)。
func SetGlobalLLMGenerator(fn LLMGenerate) {
	globalMu.Lock()
	globalLLM = fn
	globalMu.Unlock()
}

func globalLLMGenerator() LLMGenerate {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLLM
}

// Result 总是有 Message(降级保证),Source 标识来源。
type Result struct {
	Message string
	Source  Source
	Err     error
}

// Generate 生成 commit message。失败/不可用都会通过降级返回一条 Result。
func Generate(ctx context.Context, opts Options) Result {
	if opts.Op == "" {
		opts.Op = "update"
	}
	if opts.Template == "" {
		opts.Template = TemplateFilename
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.LLMTimeoutMs <= 0 {
		opts.LLMTimeoutMs = 25_000
	}
	// 2026-07-18:调用方未传 LLM 时回退到全局注册器 — 让 skillstore 等
	// 不便持有 aiengine 实例的包也能走 LLM 路径。controller 包 init()
	// 时调用 SetGlobalLLMGenerator 注册一次即可。
	if opts.LLM == nil {
		opts.LLM = globalLLMGenerator()
	}

	if opts.LLMEnabled && opts.LLM != nil {
		cctx, cancel := context.WithTimeout(ctx, time.Duration(opts.LLMTimeoutMs)*time.Millisecond)
		defer cancel()
		msg, err := opts.LLM(cctx, buildCommitPrompt(opts.Op, opts.Paths))
		if err == nil && msg != "" {
			clean := sanitizeLLMOutput(msg)
			if clean != "" {
				return Result{Message: clean, Source: SourceLLM}
			}
			err = errors.New("commitmsg: model output invalid after sanitize")
		}
		// 降级
		return Result{
			Message: fromTemplate(opts.Template, opts.Op, opts.Paths, opts.Now),
			Source:  SourceLLMFailed,
			Err:     err,
		}
	}

	return Result{Message: fromTemplate(opts.Template, opts.Op, opts.Paths, opts.Now), Source: SourceTemplate}
}

// sanitizeLLMOutput 把模型输出清洗成"单行 + 长度合理 + 去掉包裹引号"。
func sanitizeLLMOutput(raw string) string {
	s := strings.TrimSpace(raw)
	// 去包裹引号
	if len(s) >= 2 && (s[0] == '`' && s[len(s)-1] == '`' || s[0] == '"' && s[len(s)-1] == '"') {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	// 取第一行
	if idx := strings.IndexAny(s, "\r\n"); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}

func buildCommitPrompt(op string, paths []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Action: %s\n", op)
	if len(paths) == 0 {
		b.WriteString("Files: (none provided)\n")
	} else {
		fmt.Fprintf(&b, "Files (%d):\n", len(paths))
		max := 25
		for i, p := range paths {
			if i >= max {
				fmt.Fprintf(&b, "  ... and %d more\n", len(paths)-max)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	b.WriteString("\nReturn ONE conventional commit message line: <type>(<scope>): <desc>.\n")
	b.WriteString("Allowed types: feat, fix, chore, docs, refactor, test, perf, style.\n")
	b.WriteString("Output ONLY the line, no markdown, no quotes.\n")
	return b.String()
}

func fromTemplate(t Template, op string, paths []string, now time.Time) string {
	switch t {
	case TemplateTimestamp:
		return fmt.Sprintf("skill(store): %s %s", now.Format("2006-01-02T15:04:05Z"), op)
	case TemplateOpFiles, TemplateFilename:
		switch {
		case len(paths) == 0:
			return fmt.Sprintf("skill(store): %s", op)
		case len(paths) == 1:
			return fmt.Sprintf("skill(store): %s %s", op, filepath.ToSlash(paths[0]))
		default:
			return fmt.Sprintf("skill(store): %s %s + %d files", op, filepath.ToSlash(paths[0]), len(paths)-1)
		}
	default:
		return fmt.Sprintf("skill(store): %s", op)
	}
}
